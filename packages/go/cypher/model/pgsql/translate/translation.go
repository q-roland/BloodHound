package translate

import (
	"fmt"
	"github.com/specterops/bloodhound/cypher/model"
	"github.com/specterops/bloodhound/cypher/model/pgsql"
	"github.com/specterops/bloodhound/cypher/model/pgsql/fold"
)

func propertyLookupToBinaryExpression(propertyLookup *model.PropertyLookup) (*pgsql.BinaryExpression, error) {
	// Property lookups become a binary expression tree of JSON operators
	if propertyLookupAtom, err := model.ExpressionAs[*model.Variable](propertyLookup.Atom); err != nil {
		return nil, err
	} else {
		return &pgsql.BinaryExpression{
			LeftOperand:  pgsql.CompoundIdentifier{pgsql.Identifier(propertyLookupAtom.Symbol), "properties"},
			Operator:     pgsql.OperatorJSONField,
			RightOperand: pgsql.AsLiteral(propertyLookup.Symbols[0]),
		}, nil
	}
}

type Translator struct {
	pgsql.CancelableErrorHandler

	tree *pgsql.Tree
}

func (s *Translator) Enter(expression model.Expression) {
	switch expression.(type) {
	case *model.Disjunction, *model.Conjunction:
		s.tree.Push(&pgsql.BinaryExpression{})
	}
}

func (s *Translator) Visit(expression model.Expression) {
	switch expression.(type) {
	case *model.Comparison, *model.ArithmeticExpression:
		s.tree.Push(&pgsql.BinaryExpression{
			LeftOperand: s.tree.Pop(),
		})

	case *model.Disjunction:
		if err := s.tree.ContinueBinaryExpression(pgsql.OperatorOr, s.tree.Pop()); err != nil {
			s.SetError(err)
		}

	case *model.Conjunction:
		if err := s.tree.ContinueBinaryExpression(pgsql.OperatorAnd, s.tree.Pop()); err != nil {
			s.SetError(err)
		}
	}
}

func (s *Translator) Exit(expression model.Expression) {
	switch typedExpression := expression.(type) {
	case *model.Negation:
		s.tree.Push(&pgsql.UnaryExpression{
			Operator: pgsql.OperatorNot,
			Operand:  s.tree.Pop(),
		})

	case *model.Literal:
		var literal pgsql.Literal

		if strLiteral, isStr := typedExpression.Value.(string); isStr {
			// Cypher parser wraps string literals with ' characters - unwrap them first
			literal = pgsql.AsLiteral(strLiteral[1 : len(strLiteral)-1])
			literal.Null = typedExpression.Null
		} else {
			literal = pgsql.AsLiteral(typedExpression.Value)
			literal.Null = typedExpression.Null
		}

		s.tree.Push(literal)

	case *model.PropertyLookup:
		if propertyLookupBE, err := propertyLookupToBinaryExpression(typedExpression); err != nil {
			s.SetError(err)
		} else {
			s.tree.Push(propertyLookupBE)
		}

	case *model.PartialComparison:
		if err := s.tree.ContinueBinaryExpression(pgsql.Operator(typedExpression.Operator), s.tree.Pop()); err != nil {
			s.SetError(err)
		}

	case *model.PartialArithmeticExpression:
		if err := s.tree.ContinueBinaryExpression(pgsql.Operator(typedExpression.Operator), s.tree.Pop()); err != nil {
			s.SetError(err)
		}

	case *model.Disjunction:
		if err := s.tree.ContinueBinaryExpression(pgsql.OperatorOr, s.tree.Pop()); err != nil {
			s.SetError(err)
		}

		s.tree.Ascend(typedExpression.Len() - 2)

	case *model.Conjunction:
		if err := s.tree.ContinueBinaryExpression(pgsql.OperatorAnd, s.tree.Pop()); err != nil {
			s.SetError(err)
		}

		s.tree.Ascend(typedExpression.Len() - 2)
	}
}

func NewTranslator() *Translator {
	return &Translator{
		CancelableErrorHandler: pgsql.NewCancelableErrorHandler(),
		tree:                   &pgsql.Tree{},
	}
}

func TranslateCypherExpression(expression model.Expression) (pgsql.Expression, error) {
	translator := NewTranslator()

	if err := pgsql.CypherWalk(expression, translator); err != nil {
		return nil, err
	}

	return translator.tree.Root(), nil
}

func translateReadingClauses(readingClauses []*model.ReadingClause) (pgsql.Query, error) {
	var (
		query = pgsql.Query{
			CommonTableExpressions: &pgsql.With{},
		}

		existingBindings = map[string]struct{}{}
	)

	for _, readingClause := range readingClauses {
		var (
			conjoinedConstraintsByKey map[string]*pgsql.Tree
			whereAST                  *pgsql.BinaryExpression
		)

		if readingClause.Match.Where != nil && len(readingClause.Match.Where.Expressions) > 0 {
			// TODO: Refactor the cypher Where AST node out of being an expression list
			if whereASTExpr, err := TranslateCypherExpression(readingClause.Match.Where.Expressions[0]); err != nil {
				return query, err
			} else if conjoinedConstraintsByKey, err = fold.FragmentExpressionTree(whereASTExpr); err != nil {
				return query, err
			} else {
				whereAST = whereASTExpr.(*pgsql.BinaryExpression)
			}
		}

		for _, patternPart := range readingClause.Match.Pattern {
			var (
				nextCTE = pgsql.CommonTableExpression{
					Query: pgsql.Query{},
				}
			)

			for _, patternElement := range patternPart.PatternElements {
				selectStmt := pgsql.Select{
					Projection: []pgsql.Projection{pgsql.Wildcard{}},
				}

				if nodePattern, isNodePattern := patternElement.AsNodePattern(); isNodePattern {
					if nodePattern.Binding != nil {
						if nodePatternBinding, err := model.ExpressionAs[*model.Variable](nodePattern.Binding); err != nil {
							return query, err
						} else {
							var (
								nodeIdentifier = pgsql.Identifier(nodePatternBinding.Symbol)
							)

							existingBindings[nodeIdentifier.String()] = struct{}{}

							nextCTE.Alias = pgsql.TableAlias{
								Name: nodeIdentifier,
							}

							if constraints, hasConstraints := conjoinedConstraintsByKey[nodeIdentifier.String()]; hasConstraints {
								selectStmt.From = append(selectStmt.From, pgsql.FromClause{
									Relation: pgsql.TableReference{
										Name:    pgsql.CompoundIdentifier{"node"},
										Binding: pgsql.AsOptionalIdentifier(nodeIdentifier),
									},
								})

								selectStmt.Where = constraints.Root()
							} else {
								selectStmt.From = append(selectStmt.From, pgsql.FromClause{
									Relation: pgsql.TableReference{
										Name:    pgsql.CompoundIdentifier{"node"},
										Binding: pgsql.AsOptionalIdentifier(nodeIdentifier),
									},
								})

								if whereAST != nil {
									var (
										combinedDeps = whereAST.CombinedDependencies()
										hasDeps      = true
									)

									for identifierDep := range combinedDeps {
										if _, hasDep := existingBindings[identifierDep]; !hasDep {
											hasDeps = false
											break
										}
									}

									if hasDeps {
										for identifierDep := range combinedDeps {
											if identifierDep != nodeIdentifier.String() {
												selectStmt.From = append(selectStmt.From, pgsql.FromClause{
													Relation: pgsql.TableReference{
														Name: pgsql.CompoundIdentifier{pgsql.Identifier(identifierDep)},
													},
												})
											}
										}

										selectStmt.Where = whereAST
										whereAST = nil
									}
								}
							}
						}
					}

					nextCTE.Query.Body = selectStmt
				}
			}

			// Add the CTE to the query
			query.CommonTableExpressions.Expressions = append(query.CommonTableExpressions.Expressions, nextCTE)
		}
	}

	return query, nil
}

func translateSinglePartQuery(singlePartQuery *model.SinglePartQuery) (pgsql.Statement, error) {
	if query, err := translateReadingClauses(singlePartQuery.ReadingClauses); err != nil {
		return query, err
	} else {

		if singlePartQuery.Return != nil {
			selectStmt := pgsql.Select{
				Projection: []pgsql.Projection{pgsql.Wildcard{}},
			}

			for _, untypedProjectionItem := range singlePartQuery.Return.Projection.Items {
				if projectItem, typeOK := untypedProjectionItem.(*model.ProjectionItem); !typeOK {
					return nil, fmt.Errorf("unsupported cypher projection type %T", untypedProjectionItem)
				} else {
					switch typedProjectionItemExpression := projectItem.Expression.(type) {
					case *model.Variable:
						selectStmt.From = append(selectStmt.From, pgsql.FromClause{
							Relation: pgsql.TableReference{
								Name: pgsql.CompoundIdentifier{pgsql.Identifier(typedProjectionItemExpression.Symbol)},
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
}

func Translate(cypherQuery *model.RegularQuery) (pgsql.Statement, error) {
	// Is this a single part query?
	if cypherQuery.SingleQuery.SinglePartQuery != nil {
		return translateSinglePartQuery(cypherQuery.SingleQuery.SinglePartQuery)
	}

	return nil, nil
}
