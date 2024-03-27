package pgsql

import (
	"fmt"
	"github.com/specterops/bloodhound/cypher/model"
)

type TranslationCursor[E any] struct {
	Expression  E
	Branches    []E
	BranchIndex int
}

func (s *TranslationCursor[E]) NumBranchesRemaining() int {
	return len(s.Branches) - s.BranchIndex
}

func (s *TranslationCursor[E]) IsFirstVisit() bool {
	return s.BranchIndex == 0
}

func (s *TranslationCursor[E]) HasNext() bool {
	return s.BranchIndex < len(s.Branches)
}

func (s *TranslationCursor[E]) NextBranch() E {
	nextBranch := s.Branches[s.BranchIndex]
	s.BranchIndex += 1

	return nextBranch
}

func SliceAs[T any, TS []T, F any, FS []F](fs FS) (TS, error) {
	ts := make(TS, len(fs))

	for idx := 0; idx < len(fs); idx++ {
		if tTyped, isTType := any(fs[idx]).(T); !isTType {
			var emptyT T
			return nil, fmt.Errorf("slice type %T does not convert to %T", fs[idx], emptyT)
		} else {
			ts[idx] = tTyped
		}
	}

	return ts, nil
}

func MustSliceAs[T any, TS []T, F any, FS []F](fs FS) TS {
	if ts, err := SliceAs[T](fs); err != nil {
		panic(err.Error())
	} else {
		return ts
	}
}

func SetBranches[E any, T any](cursor *TranslationCursor[E], branches ...T) error {
	for _, branch := range branches {
		if eTypedBranch, isEType := any(branch).(E); !isEType {
			var emptyE E
			return fmt.Errorf("branch type %T does not convert to %T", branch, emptyE)
		} else {
			cursor.Branches = append(cursor.Branches, eTypedBranch)
		}
	}

	return nil
}

func newCypherTranslationCursor(expression model.Expression) (*TranslationCursor[model.Expression], error) {
	cursor := &TranslationCursor[model.Expression]{
		Expression: expression,
	}

	switch typedExpression := expression.(type) {
	// Types with no AST branches
	case *model.PropertyLookup, *model.Literal:
		return cursor, nil

	case *model.PartialComparison:
		return &TranslationCursor[model.Expression]{
			Expression:  expression,
			Branches:    []model.Expression{typedExpression.Operator, typedExpression.Right},
			BranchIndex: 0,
		}, nil

	case *model.Negation:
		return cursor, SetBranches(cursor, typedExpression.Expression)

	case *model.Conjunction:
		return cursor, SetBranches(cursor, typedExpression.Expressions...)

	case *model.Comparison:
		return &TranslationCursor[model.Expression]{
			Expression: expression,
			Branches:   append([]model.Expression{typedExpression.Left}, MustSliceAs[model.Expression](typedExpression.Partials)...),
		}, nil

	default:
		return nil, fmt.Errorf("unable to negotiate cypher model type %T into a translation cursor", expression)
	}
}

func propertyLookupToBinaryExpression(propertyLookup *model.PropertyLookup) (*BinaryExpression, error) {
	// Property lookups become a binary expression tree of JSON operators
	if propertyLookupAtom, err := model.ExpressionAs[*model.Variable](propertyLookup.Atom); err != nil {
		return nil, err
	} else {
		var (
			propertyLookupBE = &BinaryExpression{
				LeftOperand: Identifier(propertyLookupAtom.Symbol),
				Operator:    Operator("->"),
			}

			currentBE = propertyLookupBE
		)

		// TODO: The cypher grammar doesn't support nested property lookups, this should be simplified
		for idx, nextIdentifier := range propertyLookup.Symbols {
			if idx+1 == len(propertyLookup.Symbols) {
				currentBE.RightOperand = Identifier(nextIdentifier)
			} else {
				nextBE := &BinaryExpression{
					LeftOperand:  Identifier(nextIdentifier),
					Operator:     Operator("->"),
					RightOperand: nil,
				}

				currentBE.RightOperand = nextBE
				currentBE = nextBE
			}
		}

		return propertyLookupBE, nil
	}
}

func translateOperatorExpression(expression model.Expression) (Operator, error) {
	if operator, isOperator := expression.(model.Operator); !isOperator {
		return "", fmt.Errorf("unable to negotiate expression type %T to a cypher operator", expression)
	} else {
		return Operator(operator.String()), nil
	}
}

func assignOperatorToPrecedingExpression(precedingExpression, assignment Expression) error {
	switch typedAssignmentTarget := precedingExpression.(type) {
	case *UnaryExpression:
		if typedAssignmentTarget.Operand != nil {
			return fmt.Errorf("preceding unary expression operator is already populated")
		}

		typedAssignmentTarget.Operand = assignment

	case *BinaryExpression:
		if typedAssignmentTarget.Operator != nil {
			return fmt.Errorf("preceding binary expression operator is already populated")
		}

		typedAssignmentTarget.Operator = assignment
	}

	return nil
}

func assignToPrecedingExpression(precedingExpression, assignment Expression) error {
	switch typedAssignmentTarget := precedingExpression.(type) {
	case *UnaryExpression:
		if typedAssignmentTarget.Operand != nil {
			return fmt.Errorf("preceding unary expression is already populated")
		}

		typedAssignmentTarget.Operand = assignment

	case *BinaryExpression:
		if typedAssignmentTarget.LeftOperand == nil {
			typedAssignmentTarget.LeftOperand = assignment
		} else if typedAssignmentTarget.RightOperand == nil {
			typedAssignmentTarget.RightOperand = assignment
		} else {
			return fmt.Errorf("preceding binary expression is already populated")
		}
	}

	return nil
}

func ctbe(conjunction model.Expression) (Expression, error) {
	var (
		sqlStack    []*TranslationCursor[Expression]
		cypherStack []*TranslationCursor[model.Expression]
	)

	if cypherRootCursor, err := newCypherTranslationCursor(conjunction); err != nil {
		return nil, err
	} else {
		cypherStack = append(cypherStack, cypherRootCursor)
	}

	for len(cypherStack) > 0 {
		nextCypherNode := cypherStack[len(cypherStack)-1]

		switch typedCypherExpression := nextCypherNode.Expression.(type) {
		case *model.Literal:
			literal := AsLiteral(typedCypherExpression.Value)
			literal.Null = typedCypherExpression.Null

			if err := assignToPrecedingExpression(sqlStack[len(sqlStack)-1].Expression, literal); err != nil {
				return nil, err
			}

			// Pop this element from the cyper translation stack
			cypherStack = cypherStack[0 : len(cypherStack)-1]

		case *model.PartialComparison:
			if nextCypherNode.IsFirstVisit() {
				// Look at the previous SQL statement and assign the operator
				if translatedOperator, err := translateOperatorExpression(nextCypherNode.NextBranch()); err != nil {
					return nil, err
				} else if err := assignOperatorToPrecedingExpression(sqlStack[len(sqlStack)-1].Expression, translatedOperator); err != nil {
					return nil, err
				}
			} else if nextCypherNode.HasNext() {
				// Push the next operand onto the cypher translation stack
				if nextCypherTranslationCursor, err := newCypherTranslationCursor(nextCypherNode.NextBranch()); err != nil {
					return nil, err
				} else {
					cypherStack = append(cypherStack, nextCypherTranslationCursor)
				}
			} else {
				cypherStack = cypherStack[0 : len(cypherStack)-1]
			}

		case *model.Negation:
			if nextCypherNode.IsFirstVisit() {
				// If this is the first expression, create a binary expression to begin
				// translating the expression tree
				sqlStack = append(sqlStack, &TranslationCursor[Expression]{
					Expression: &UnaryExpression{
						Operator: Operator("not"),
					},
				})

				// Push the hand operand onto the cypher translation stack
				if nextCypherTranslationCursor, err := newCypherTranslationCursor(nextCypherNode.NextBranch()); err != nil {
					return nil, err
				} else {
					cypherStack = append(cypherStack, nextCypherTranslationCursor)
				}
			} else {
				translated := sqlStack[len(sqlStack)-1].Expression
				sqlStack = sqlStack[0 : len(sqlStack)-1]

				if err := assignToPrecedingExpression(sqlStack[len(sqlStack)-1].Expression, translated); err != nil {
					return nil, err
				}

				cypherStack = cypherStack[0 : len(cypherStack)-1]
			}

		case *model.PropertyLookup:
			if propertyLookupBE, err := propertyLookupToBinaryExpression(typedCypherExpression); err != nil {
				return nil, err
			} else if err := assignToPrecedingExpression(sqlStack[len(sqlStack)-1].Expression, propertyLookupBE); err != nil {
				return nil, err
			}

			// Pop this element from the cyper translation stack
			cypherStack = cypherStack[0 : len(cypherStack)-1]

		case *model.Comparison:
			if nextCypherNode.IsFirstVisit() {
				// If this is the first expression, create a binary expression to begin
				// translating the expression tree
				sqlStack = append(sqlStack, &TranslationCursor[Expression]{
					Expression: &BinaryExpression{},
				})

				// Push the left-hand operand onto the cypher translation stack
				if nextCypherTranslationCursor, err := newCypherTranslationCursor(nextCypherNode.NextBranch()); err != nil {
					return nil, err
				} else {
					cypherStack = append(cypherStack, nextCypherTranslationCursor)
				}
			} else if nextCypherNode.HasNext() {
				// Push the next operand onto the cypher translation stack
				if nextCypherTranslationCursor, err := newCypherTranslationCursor(nextCypherNode.NextBranch()); err != nil {
					return nil, err
				} else {
					cypherStack = append(cypherStack, nextCypherTranslationCursor)
				}
			} else {
				translated := sqlStack[len(sqlStack)-1].Expression
				sqlStack = sqlStack[0 : len(sqlStack)-1]

				if err := assignToPrecedingExpression(sqlStack[len(sqlStack)-1].Expression, translated); err != nil {
					return nil, err
				}

				// Pop this element from the cyper translation stack
				cypherStack = cypherStack[0 : len(cypherStack)-1]
			}

		case *model.Conjunction:
			if nextCypherNode.IsFirstVisit() {
				// If this is the first expression, create a binary expression to begin
				// translating the expression tree
				sqlStack = append(sqlStack, &TranslationCursor[Expression]{
					Expression: &BinaryExpression{
						Operator: Operator("and"),
					},
				})

				// Push the left-hand operand onto the cypher translation stack
				if nextCypherTranslationCursor, err := newCypherTranslationCursor(nextCypherNode.NextBranch()); err != nil {
					return nil, err
				} else {
					cypherStack = append(cypherStack, nextCypherTranslationCursor)
				}
			} else if nextCypherNode.HasNext() {
				// Push the next operand onto the cypher translation stack
				if nextCypherTranslationCursor, err := newCypherTranslationCursor(nextCypherNode.NextBranch()); err != nil {
					return nil, err
				} else {
					cypherStack = append(cypherStack, nextCypherTranslationCursor)
				}

				// If there are remaining conjoined expressions, create and assign the next nested binary expression
				if nextCypherNode.HasNext() {
					nextBinaryExpression := &BinaryExpression{
						Operator: Operator("and"),
					}

					if err := assignToPrecedingExpression(sqlStack[len(sqlStack)-1].Expression, nextBinaryExpression); err != nil {
						return nil, err
					}

					sqlStack = append(sqlStack, &TranslationCursor[Expression]{
						Expression: nextBinaryExpression,
					})
				}
			} else {
				// Pop this element from the cyper translation stack
				cypherStack = cypherStack[0 : len(cypherStack)-1]

				// Pop from the SQL stack all conjoined expressions. The length of the conjoined expressions is
				// reduced by to 2 for normalizing both for 0 index and for the depth of the nested binary expressions
				nestedDepth := len(typedCypherExpression.Expressions) - 2
				sqlStack = sqlStack[0 : len(sqlStack)-nestedDepth]
			}

		default:
			return nil, fmt.Errorf("unknown translation type: %T", nextCypherNode.Expression)
		}
	}

	return sqlStack[0].Expression, nil
}

func ConjunctionToBinaryExpression(conjunction *model.Conjunction) (BinaryExpression, error) {
	var (
		rootBinaryExpression = &BinaryExpression{}
		//cypherModelStack     = append([]model.Expression{}, conjunction)
		//sqlModelStack        = []Expression{rootBinaryExpression}
	)

	//for len(cypherModelStack) > 0 {
	//	nextCypherExpression := cypherModelStack[len(cypherModelStack)-1]
	//	cypherModelStack = cypherModelStack[0 : len(cypherModelStack)-1]
	//
	//	switch typedCypherExpression := nextCypherExpression.(type) {
	//	case *model.Conjunction:
	//		for _, conjoinedExpression := range typedCypherExpression.Expressions {
	//			nextExpr := BinaryExpression{}
	//
	//			switch typedSQLExpression := sqlModelStack[len(sqlModelStack)-1].(type) {
	//			case BinaryExpression:
	//				if typedSQLExpression.LeftOperand == nil {
	//					typedSQLExpression.LeftOperand = nextExpr
	//					sqlModelStack = append(sqlModelStack, &nextExpr)
	//				} else {
	//					typedSQLExpression.RightOperand = nextExpr
	//					sqlModelStack = append(sqlModelStack, &nextExpr)
	//				}
	//			}
	//		}
	//
	//	case *model.Comparison:
	//		switch typedSQLExpression := sqlModelStack[len(sqlModelStack)-1].(type) {
	//		case BinaryExpression:
	//			if typedSQLExpression.LeftOperand == nil {
	//				typedSQLExpression.LeftOperand = BinaryExpression{}
	//			}
	//		}
	//
	//	default:
	//		return rootBinaryExpression, fmt.Errorf("unknown expression: %T", conjunction.Expressions[0])
	//	}
	//}

	return *rootBinaryExpression, nil
}

func TranslateWhereClause(cypherWhere *model.Where) (Expression, error) {
	switch typedExpression := cypherWhere.Expressions[0].(type) {
	case *model.Conjunction:
		return ConjunctionToBinaryExpression(typedExpression)

	default:
		return nil, fmt.Errorf("unknown type: %T", cypherWhere.Expressions[0])
	}
}

func translateSinglePartQuery(singlePartQuery *model.SinglePartQuery) (Statement, error) {
	query := Query{
		CommonTableExpressions: &With{},
	}

	if len(singlePartQuery.ReadingClauses) > 0 {
		if singlePartQuery.ReadingClauses[0].Match != nil {
			currentMatch := singlePartQuery.ReadingClauses[0].Match

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

				if currentMatch.Where != nil {
					// ?
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
