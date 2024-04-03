package pgsql

import (
	"fmt"
	"github.com/specterops/bloodhound/cypher/model"
)

// TODO: Try to refactor this to a walk vistior
func TranslateCypherExpression(conjunction model.Expression) (Expression, error) {
	var (
		sqlBuilder  = &ExpressionBuilder{}
		cypherStack []*WalkCursor[model.Expression]
	)

	if cypherRootCursor, err := newCypherTranslationCursor(conjunction); err != nil {
		return nil, err
	} else {
		cypherStack = append(cypherStack, cypherRootCursor)
	}

	for len(cypherStack) > 0 {
		nextCypherNode := cypherStack[len(cypherStack)-1]

		switch typedCypherExpression := nextCypherNode.Expression.(type) {
		case model.Operator:
			if err := sqlBuilder.Assign(Operator(typedCypherExpression.String())); err != nil {
				return nil, err
			}

			// Pop this element from the cyper translation stack
			cypherStack = cypherStack[0 : len(cypherStack)-1]

		case *model.Negation:
			if nextCypherNode.IsFirstVisit() {
				if err := sqlBuilder.PushAssign(&UnaryExpression{
					Operator: Operator("not"),
				}); err != nil {
					return nil, err
				}
			}

			if nextCypherNode.HasNext() {
				if nextCypherTranslationCursor, err := newCypherTranslationCursor(nextCypherNode.NextBranch()); err != nil {
					return nil, err
				} else {
					cypherStack = append(cypherStack, nextCypherTranslationCursor)
				}
			} else {
				cypherStack = cypherStack[0 : len(cypherStack)-1]
				sqlBuilder.Pop(1)
			}

		case *model.Literal:
			literal := AsLiteral(typedCypherExpression.Value)
			literal.Null = typedCypherExpression.Null

			if err := sqlBuilder.Assign(literal); err != nil {
				return nil, err
			}

			// Pop this element from the cyper translation stack
			cypherStack = cypherStack[0 : len(cypherStack)-1]

		case *model.PropertyLookup:
			if propertyLookupBE, err := propertyLookupToBinaryExpression(typedCypherExpression); err != nil {
				return nil, err
			} else if err := sqlBuilder.Assign(propertyLookupBE); err != nil {
				return nil, err
			}

			// Pop this element from the cyper translation stack
			cypherStack = cypherStack[0 : len(cypherStack)-1]

		case *model.PartialComparison, *model.PartialArithmeticExpression:
			if nextCypherNode.HasNext() {
				nextBranch := nextCypherNode.NextBranch()

				switch nextBranch.(type) {
				case *model.PartialComparison, *model.PartialArithmeticExpression:
					// Each partial is represented as a nested binary expression
					if err := sqlBuilder.PushAssign(&BinaryExpression{}); err != nil {
						return nil, err
					}
				}

				if nextCypherTranslationCursor, err := newCypherTranslationCursor(nextBranch); err != nil {
					return nil, err
				} else {
					cypherStack = append(cypherStack, nextCypherTranslationCursor)
				}
			} else {
				cypherStack = cypherStack[0 : len(cypherStack)-1]
			}

		case *model.Comparison, *model.ArithmeticExpression:
			if nextCypherNode.IsFirstVisit() {
				if err := sqlBuilder.PushAssign(&BinaryExpression{}); err != nil {
					return nil, err
				}
			}

			if nextCypherNode.HasNext() {
				if nextCypherTranslationCursor, err := newCypherTranslationCursor(nextCypherNode.NextBranch()); err != nil {
					return nil, err
				} else {
					cypherStack = append(cypherStack, nextCypherTranslationCursor)
				}
			} else {
				// Pop this element from the cyper translation stack
				cypherStack = cypherStack[0 : len(cypherStack)-1]

				// Pop from the SQL stack all pushed expressions
				sqlBuilder.Pop(len(nextCypherNode.Branches) - 1)
			}

		case *model.Conjunction:
			if nextCypherNode.HasNext() {
				if nextCypherTranslationCursor, err := newCypherTranslationCursor(nextCypherNode.NextBranch()); err != nil {
					return nil, err
				} else {
					cypherStack = append(cypherStack, nextCypherTranslationCursor)
				}

				// If we still have more elements to address we need additional binary expressions to represent
				// the conjunction
				if nextCypherNode.HasNext() {
					if err := sqlBuilder.PushAssign(&BinaryExpression{
						Operator: Operator("and"),
					}); err != nil {
						return nil, err
					}
				}
			} else {
				// Pop this element from the cyper translation stack
				cypherStack = cypherStack[0 : len(cypherStack)-1]

				// Pop from the SQL stack all pushed expressions
				sqlBuilder.Pop(len(nextCypherNode.Branches) - 1)
			}

		default:
			return nil, fmt.Errorf("unknown translation type: %T", nextCypherNode.Expression)
		}
	}

	return sqlBuilder.root, nil
}

func translateSinglePartQuery(singlePartQuery *model.SinglePartQuery) (Statement, error) {
	query := Query{
		CommonTableExpressions: &With{},
	}

	if len(singlePartQuery.ReadingClauses) > 0 {
		if singlePartQuery.ReadingClauses[0].Match != nil {
			currentMatch := singlePartQuery.ReadingClauses[0].Match

			if currentMatch.Where != nil && len(currentMatch.Where.Expressions) > 0 {
				// TODO: Refactor the cypher Where AST node out of being an expression list
				if _, err := TranslateCypherExpression(currentMatch.Where.Expressions[0]); err != nil {
					return nil, err
				} else {

				}
			}

			for _, patternPart := range currentMatch.Pattern {
				nextCTE := CommonTableExpression{
					Query: Query{},
				}

				for _, patternElement := range patternPart.PatternElements {
					selectStmt := Select{
						Projection: []Projection{Wildcard{}},
					}

					if nodePattern, isNodePattern := patternElement.AsNodePattern(); isNodePattern {
						if nodePattern.Binding != nil {
							if nodePatternBinding, err := model.ExpressionAs[*model.Variable](nodePattern.Binding); err != nil {
								return nil, err
							} else {
								nodeIdentifier := Identifier(nodePatternBinding.Symbol)

								nextCTE.Alias = TableAlias{
									Name: nodeIdentifier,
								}

								selectStmt.From = append(selectStmt.From, FromClause{
									Relation: TableReference{
										Name:    CompoundIdentifier{"node"},
										Binding: AsOptionalIdentifier(nodeIdentifier),
									},
								})
							}
						}

						nextCTE.Query.Body = selectStmt
					}
				}

				// Add the CTE to the query
				query.CommonTableExpressions.Expressions = append(query.CommonTableExpressions.Expressions, nextCTE)
			}
		}
	}

	if singlePartQuery.Return != nil {
		selectStmt := Select{
			Projection: []Projection{Wildcard{}},
		}

		for _, untypedProjectionItem := range singlePartQuery.Return.Projection.Items {
			if projectItem, typeOK := untypedProjectionItem.(*model.ProjectionItem); !typeOK {
				return nil, fmt.Errorf("unsupported cypher projection type %T", untypedProjectionItem)
			} else {
				switch typedProjectionItemExpression := projectItem.Expression.(type) {
				case *model.Variable:
					selectStmt.From = append(selectStmt.From, FromClause{
						Relation: TableReference{
							Name: CompoundIdentifier{Identifier(typedProjectionItemExpression.Symbol)},
						},
					})

				default:
					return nil, fmt.Errorf("unsupported cypher projection type expression %T", projectItem.Expression)
				}
			}
		}

		query.Body = selectStmt
	}

	return query, nil
}

func Translate(cypherQuery *model.RegularQuery) (Statement, error) {
	// Is this a single part query?
	if cypherQuery.SingleQuery.SinglePartQuery != nil {
		return translateSinglePartQuery(cypherQuery.SingleQuery.SinglePartQuery)
	}

	return nil, nil
}
