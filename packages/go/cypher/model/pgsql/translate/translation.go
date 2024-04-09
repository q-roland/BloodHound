package translate

import (
	"fmt"
	"github.com/specterops/bloodhound/cypher/model"
	"github.com/specterops/bloodhound/cypher/model/pgsql"
	"github.com/specterops/bloodhound/cypher/model/pgsql/fold"
	"github.com/specterops/bloodhound/dawgs/graph"
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

type TranslationContext struct {
	ExistingBindings      map[string]struct{}
	WhereAST              *pgsql.BinaryExpression
	ExtractedConjunctions map[string]*pgsql.Tree
}

type PatternPartTranslationContext struct {
	*TranslationContext
	PatternPart *model.PatternPart
}

type NodePatternElementTranslationContext struct {
	*PatternPartTranslationContext
	NodePatternElement *model.NodePattern
}

func translateNodePatternElement(translation *NodePatternElementTranslationContext) (pgsql.CommonTableExpression, error) {
	var (
		selectStmt = pgsql.Select{
			Projection: []pgsql.Projection{pgsql.Wildcard{}},
		}

		nextCTE = pgsql.CommonTableExpression{}
	)

	if nodePatternBinding, err := model.ExpressionAs[*model.Variable](translation.NodePatternElement.Binding); err != nil {
		return nextCTE, err
	} else {
		nodeIdentifier := pgsql.Identifier(nodePatternBinding.Symbol)

		translation.ExistingBindings[nodeIdentifier.String()] = struct{}{}

		nextCTE.Alias = pgsql.TableAlias{
			Name: nodeIdentifier,
		}

		selectStmt.From = append(selectStmt.From, pgsql.FromClause{
			Relation: pgsql.TableReference{
				Name:    pgsql.CompoundIdentifier{"node"},
				Binding: pgsql.AsOptionalIdentifier(nodeIdentifier),
			},
		})

		if constraints, hasConstraints := translation.ExtractedConjunctions[nodeIdentifier.String()]; hasConstraints {
			selectStmt.Where = constraints.Root()
		} else if translation.WhereAST != nil {
			var (
				combinedDeps = translation.WhereAST.CombinedDependencies()
				hasDeps      = true
			)

			for identifierDep := range combinedDeps {
				if _, hasDep := translation.ExistingBindings[identifierDep]; !hasDep {
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

				selectStmt.Where = translation.WhereAST
				translation.WhereAST = nil
			}
		}
	}

	nextCTE.Query.Body = selectStmt
	return nextCTE, nil
}

func cypherVariableAsIdentifier(expression model.Expression) (pgsql.Identifier, bool, error) {
	if expression == nil {
		return "", false, nil
	}

	variable, err := model.ExpressionAs[*model.Variable](expression)
	return pgsql.Identifier(variable.Symbol), true, err
}

func cypherBinding(expression model.Expression) (pgsql.Identifier, bool, error) {
	switch typedExpression := expression.(type) {
	case *model.NodePattern:
		return cypherVariableAsIdentifier(typedExpression.Binding)

	case *model.RelationshipPattern:
		return cypherVariableAsIdentifier(typedExpression.Binding)

	case *model.PatternPart:
		return cypherVariableAsIdentifier(typedExpression.Binding)

	default:
		return "", false, fmt.Errorf("unable to extract binding from expression type: %T", expression)
	}
}

func translateRelationshipPattern(translation *PatternPartTranslationContext) ([]pgsql.CommonTableExpression, error) {
	var (
		nextCTEs []pgsql.CommonTableExpression

		hasBoundNodes    = false
		patternPartBound = translation.PatternPart.Binding != nil
		numTraversals    = 0
	)

	for _, patternElement := range translation.PatternPart.PatternElements {
		if nodePattern, isNodePattern := patternElement.AsNodePattern(); isNodePattern {
			if nodePattern.Binding != nil {
				hasBoundNodes = true
			}
		} else {
			relationshipPattern, _ := patternElement.AsRelationshipPattern()

			if relationshipPattern.Range != nil {
				numTraversals += 1
			}
		}
	}

	if !patternPartBound {
		if numTraversals == 0 {
			if hasBoundNodes {
				for idx, patternElement := range translation.PatternPart.PatternElements {
					if relationshipPattern, isRelationshipPattern := patternElement.AsRelationshipPattern(); isRelationshipPattern {
						var (
							startNodePE *model.PatternElement
							endNodePE   *model.PatternElement
						)

						switch relationshipPattern.Direction {
						case graph.DirectionOutbound:
							// (start)-[]->(end)
							startNodePE = translation.PatternPart.PatternElements[idx-1]
							endNodePE = translation.PatternPart.PatternElements[idx+1]

						case graph.DirectionInbound:
							// (end)<-[]-(start)
							startNodePE = translation.PatternPart.PatternElements[idx+1]
							endNodePE = translation.PatternPart.PatternElements[idx-1]

						default:
							return nil, fmt.Errorf("unsupported direction: %d", relationshipPattern.Direction)
						}

						if startNode, isNodePattern := startNodePE.AsNodePattern(); !isNodePattern {
							return nil, fmt.Errorf("expected node pattern but found type: %T", startNodePE)
						} else if startNodeIdentifier, hasStartNodeIdentifier, err := cypherBinding(startNode); err != nil {
							return nil, err
						} else if endNode, isNodePattern := endNodePE.AsNodePattern(); !isNodePattern {
							return nil, fmt.Errorf("expected node pattern but found type: %T", startNodePE)
						} else if endNodeIdentifier, hasEndNodeIdentifier, err := cypherBinding(endNode); err != nil {
							return nil, err
						} else if relIdentifier, hasRelIdentifier, err := cypherVariableAsIdentifier(relationshipPattern.Binding); err != nil {
							return nil, err
						} else {
							if hasStartNodeIdentifier {
								translation.ExistingBindings[startNodeIdentifier.String()] = struct{}{}

								if nextCTE, err := translateNodePatternElement(&NodePatternElementTranslationContext{
									PatternPartTranslationContext: translation,
									NodePatternElement:            startNode,
								}); err != nil {
									return nil, err
								} else {
									nextCTEs = append(nextCTEs, nextCTE)
								}
							}

							if hasRelIdentifier {
								var (
									fromClauses = []pgsql.FromClause{{
										Relation: pgsql.TableReference{
											Name:    pgsql.CompoundIdentifier{"edge"},
											Binding: pgsql.AsOptionalIdentifier(relIdentifier),
										},
									}}

									whereClause pgsql.Expression
								)

								if constraints, hasConstraints := translation.ExtractedConjunctions[relIdentifier.String()]; hasConstraints {
									whereClause = constraints.Root()
								} else if translation.WhereAST != nil {
									var (
										combinedDeps = translation.WhereAST.CombinedDependencies()
										hasDeps      = true
									)

									for identifierDep := range combinedDeps {
										if _, hasDep := translation.ExistingBindings[identifierDep]; !hasDep {
											hasDeps = false
											break
										}
									}

									if hasDeps {
										for identifierDep := range combinedDeps {
											if identifierDep != relIdentifier.String() {
												fromClauses = append(fromClauses, pgsql.FromClause{
													Relation: pgsql.TableReference{
														Name: pgsql.CompoundIdentifier{pgsql.Identifier(identifierDep)},
													},
												})
											}
										}

										whereClause = translation.WhereAST
										translation.WhereAST = nil
									}
								}

								if hasStartNodeIdentifier {
									fromClauses = append(fromClauses, pgsql.FromClause{
										Relation: pgsql.TableReference{
											Name: pgsql.CompoundIdentifier{startNodeIdentifier},
										},
									})

									joinConstraint := &pgsql.BinaryExpression{
										Operator:     pgsql.OperatorEquals,
										RightOperand: pgsql.CompoundIdentifier{relIdentifier, "start_id"},
										LeftOperand:  pgsql.CompoundIdentifier{startNodeIdentifier, "id"},
									}

									if whereClause != nil {
										whereClause = &pgsql.BinaryExpression{
											Operator:     pgsql.OperatorAnd,
											LeftOperand:  joinConstraint,
											RightOperand: whereClause,
										}
									} else {
										whereClause = joinConstraint
									}
								}

								nextCTEs = append(nextCTEs, pgsql.CommonTableExpression{
									Alias: pgsql.TableAlias{
										Name: relIdentifier,
									},
									Query: pgsql.Query{
										Body: pgsql.Select{
											Projection: []pgsql.Projection{pgsql.Wildcard{}},
											From:       fromClauses,
											Where:      whereClause,
										},
									},
								})
							}

							if hasEndNodeIdentifier {
								var (
									fromClauses = []pgsql.FromClause{{
										Relation: pgsql.TableReference{
											Name:    pgsql.CompoundIdentifier{"node"},
											Binding: pgsql.AsOptionalIdentifier(endNodeIdentifier),
										},
									}}

									whereClause pgsql.Expression
								)

								if constraints, hasConstraints := translation.ExtractedConjunctions[endNodeIdentifier.String()]; hasConstraints {
									whereClause = constraints.Root()
								} else if translation.WhereAST != nil {
									var (
										combinedDeps = translation.WhereAST.CombinedDependencies()
										hasDeps      = true
									)

									for identifierDep := range combinedDeps {
										if _, hasDep := translation.ExistingBindings[identifierDep]; !hasDep {
											hasDeps = false
											break
										}
									}

									if hasDeps {
										for identifierDep := range combinedDeps {
											if identifierDep != endNodeIdentifier.String() {
												fromClauses = append(fromClauses, pgsql.FromClause{
													Relation: pgsql.TableReference{
														Name: pgsql.CompoundIdentifier{pgsql.Identifier(identifierDep)},
													},
												})
											}
										}

										whereClause = translation.WhereAST
										translation.WhereAST = nil
									}
								}

								if hasRelIdentifier {
									fromClauses = append(fromClauses, pgsql.FromClause{
										Relation: pgsql.TableReference{
											Name: pgsql.CompoundIdentifier{relIdentifier},
										},
									})

									joinConstraint := &pgsql.BinaryExpression{
										Operator:     pgsql.OperatorEquals,
										RightOperand: pgsql.CompoundIdentifier{relIdentifier, "end_id"},
										LeftOperand:  pgsql.CompoundIdentifier{endNodeIdentifier, "id"},
									}

									if whereClause != nil {
										whereClause = &pgsql.BinaryExpression{
											Operator:     pgsql.OperatorAnd,
											LeftOperand:  joinConstraint,
											RightOperand: whereClause,
										}
									} else {
										whereClause = joinConstraint
									}
								}

								nextCTEs = append(nextCTEs, pgsql.CommonTableExpression{
									Alias: pgsql.TableAlias{
										Name: endNodeIdentifier,
									},
									Query: pgsql.Query{
										Body: pgsql.Select{
											Projection: []pgsql.Projection{pgsql.Wildcard{}},
											From:       fromClauses,
											Where:      whereClause,
										},
									},
								})
							}
						}
					}
				}
			} else {
				// If the relationship query has no pattern binding, no traversal and no bound nodes author simple select
				// CTE chains
				for _, patternElement := range translation.PatternPart.PatternElements {
					if relationshipPattern, isRelationshipPattern := patternElement.AsRelationshipPattern(); isRelationshipPattern {
						var (
							nextCTE    = pgsql.CommonTableExpression{}
							selectStmt = pgsql.Select{
								Projection: []pgsql.Projection{pgsql.Wildcard{}},
							}
						)

						if relationshipPattern.Binding == nil {
							return nil, fmt.Errorf("expected relationship binding")
						} else if relationshipPatternBinding, err := model.ExpressionAs[*model.Variable](relationshipPattern.Binding); err != nil {
							return nil, err
						} else {
							var (
								relationshipIdentifier = pgsql.Identifier(relationshipPatternBinding.Symbol)
							)

							translation.ExistingBindings[relationshipIdentifier.String()] = struct{}{}

							nextCTE.Alias = pgsql.TableAlias{
								Name: relationshipIdentifier,
							}

							if constraints, hasConstraints := translation.ExtractedConjunctions[relationshipIdentifier.String()]; hasConstraints {
								selectStmt.From = append(selectStmt.From, pgsql.FromClause{
									Relation: pgsql.TableReference{
										Name:    pgsql.CompoundIdentifier{"edge"},
										Binding: pgsql.AsOptionalIdentifier(relationshipIdentifier),
									},
								})

								selectStmt.Where = constraints.Root()
							} else {
								selectStmt.From = append(selectStmt.From, pgsql.FromClause{
									Relation: pgsql.TableReference{
										Name:    pgsql.CompoundIdentifier{"edge"},
										Binding: pgsql.AsOptionalIdentifier(relationshipIdentifier),
									},
								})

								if translation.WhereAST != nil {
									var (
										combinedDeps = translation.WhereAST.CombinedDependencies()
										hasDeps      = true
									)

									for identifierDep := range combinedDeps {
										if _, hasDep := translation.ExistingBindings[identifierDep]; !hasDep {
											hasDeps = false
											break
										}
									}

									if hasDeps {
										for identifierDep := range combinedDeps {
											if identifierDep != relationshipIdentifier.String() {
												selectStmt.From = append(selectStmt.From, pgsql.FromClause{
													Relation: pgsql.TableReference{
														Name: pgsql.CompoundIdentifier{pgsql.Identifier(identifierDep)},
													},
												})
											}
										}

										selectStmt.Where = translation.WhereAST
										translation.WhereAST = nil
									}
								}
							}
						}

						nextCTE.Query.Body = selectStmt
						nextCTEs = append(nextCTEs, nextCTE)
					}
				}
			}
		}
	}

	return nextCTEs, nil
}

func translateNodePattern(translation *PatternPartTranslationContext) (pgsql.CommonTableExpression, error) {
	if numPatternElements := len(translation.PatternPart.PatternElements); numPatternElements != 1 {
		return pgsql.CommonTableExpression{}, fmt.Errorf("expected 1 node pattern element but found: %d", numPatternElements)
	} else if nodePattern, isNodePattern := translation.PatternPart.PatternElements[0].AsNodePattern(); !isNodePattern {
		return pgsql.CommonTableExpression{}, fmt.Errorf("expected node pattern but found: %T", translation.PatternPart.PatternElements[0])
	} else if nodePattern.Binding == nil {
		return pgsql.CommonTableExpression{}, fmt.Errorf("unbound node pattern")
	} else {
		return translateNodePatternElement(&NodePatternElementTranslationContext{
			PatternPartTranslationContext: translation,
			NodePatternElement:            nodePattern,
		})
	}
}

func translatePatternPart(translation *PatternPartTranslationContext) ([]pgsql.CommonTableExpression, error) {
	// If any of the pattern elements represent a relationship treat this as a traversal and translate accordingly
	if translation.PatternPart.HasRelationshipPattern() {
		return translateRelationshipPattern(translation)
	}

	if nodePattern, err := translateNodePattern(translation); err != nil {
		return nil, err
	} else {
		return []pgsql.CommonTableExpression{nodePattern}, nil
	}
}

func translateReadingClauses(readingClauses []*model.ReadingClause) (pgsql.Query, error) {
	var (
		query = pgsql.Query{
			CommonTableExpressions: &pgsql.With{},
		}

		existingBindings = map[string]struct{}{}
	)

	for _, readingClause := range readingClauses {
		translationContext := &TranslationContext{
			ExistingBindings: existingBindings,
		}

		if readingClause.Match.Where != nil && len(readingClause.Match.Where.Expressions) > 0 {
			// TODO: Refactor the cypher Where AST node out of being an expression list
			if whereASTExpr, err := TranslateCypherExpression(readingClause.Match.Where.Expressions[0]); err != nil {
				return query, err
			} else if extractedConjunctions, err := fold.FragmentExpressionTree(whereASTExpr); err != nil {
				return query, err
			} else {
				translationContext.ExtractedConjunctions = extractedConjunctions
				translationContext.WhereAST = whereASTExpr.(*pgsql.BinaryExpression)
			}
		}

		for _, patternPart := range readingClause.Match.Pattern {
			patternPartCtx := &PatternPartTranslationContext{
				TranslationContext: translationContext,
				PatternPart:        patternPart,
			}

			if nextCTEs, err := translatePatternPart(patternPartCtx); err != nil {
				return query, err
			} else {
				query.CommonTableExpressions.Expressions = append(query.CommonTableExpressions.Expressions, nextCTEs...)
			}
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
