package pgsql

import (
	"errors"
	"fmt"
	"github.com/specterops/bloodhound/cypher/model"
)

type TranslationNode[E any] struct {
	Expression  E
	Branches    []E
	BranchIndex int
	Tag         any
}

func newCypherTranslationCursor(expression model.Expression) (*TranslationNode[model.Expression], error) {
	cursor := &TranslationNode[model.Expression]{
		Expression: expression,
	}

	switch typedExpression := expression.(type) {
	// Types with no AST branches
	case *model.PropertyLookup, *model.Literal, model.Operator:
		return cursor, nil

	case *model.ArithmeticExpression:
		return &TranslationNode[model.Expression]{
			Expression: expression,
			Branches:   append([]model.Expression{typedExpression.Left}, MustSliceAs[model.Expression](typedExpression.Partials)...),
		}, nil

	case *model.PartialArithmeticExpression:
		return &TranslationNode[model.Expression]{
			Expression:  expression,
			Branches:    []model.Expression{typedExpression.Operator, typedExpression.Right},
			BranchIndex: 0,
		}, nil

	case *model.PartialComparison:
		return &TranslationNode[model.Expression]{
			Expression:  expression,
			Branches:    []model.Expression{typedExpression.Operator, typedExpression.Right},
			BranchIndex: 0,
		}, nil

	case *model.Negation:
		return cursor, SetBranches(cursor, typedExpression.Expression)

	case *model.Conjunction:
		return cursor, SetBranches(cursor, typedExpression.Expressions...)

	case *model.Comparison:
		return &TranslationNode[model.Expression]{
			Expression: expression,
			Branches:   append([]model.Expression{typedExpression.Left}, MustSliceAs[model.Expression](typedExpression.Partials)...),
		}, nil

	default:
		return nil, fmt.Errorf("unable to negotiate cypher model type %T into a translation cursor", expression)
	}
}

func (s *TranslationNode[E]) NumBranchesRemaining() int {
	return len(s.Branches) - s.BranchIndex
}

func (s *TranslationNode[E]) IsFirstVisit() bool {
	return s.BranchIndex == 0
}

func (s *TranslationNode[E]) HasNext() bool {
	return s.BranchIndex < len(s.Branches)
}

func (s *TranslationNode[E]) NextBranch() E {
	nextBranch := s.Branches[s.BranchIndex]
	s.BranchIndex += 1

	return nextBranch
}

func SetBranches[E any, T any](cursor *TranslationNode[E], branches ...T) error {
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

func propertyLookupToBinaryExpression(propertyLookup *model.PropertyLookup) (*BinaryExpression, error) {
	// Property lookups become a binary expression tree of JSON operators
	if propertyLookupAtom, err := model.ExpressionAs[*model.Variable](propertyLookup.Atom); err != nil {
		return nil, err
	} else {
		return &BinaryExpression{
			LeftOperand:  CompoundIdentifier{Identifier(propertyLookupAtom.Symbol), "properties"},
			Operator:     Operator("->"),
			RightOperand: AsLiteral(propertyLookup.Symbols[0]),
		}, nil
	}
}

type SQLBuilder struct {
	root  Expression
	stack []Expression
}

func (s *SQLBuilder) Depth() int {
	return len(s.stack)
}

func (s *SQLBuilder) Peek() Expression {
	return s.stack[len(s.stack)-1]
}

var (
	ErrOperatorAlreadyAssigned = errors.New("expression operator already assigned")
	ErrOperandAlreadyAssigned  = errors.New("expression operand already assigned")
)

func (s *SQLBuilder) Assign(expression Expression) error {
	switch assignmentTarget := s.Peek().(type) {
	case *UnaryExpression:
		if _, isOperator := expression.(Operator); isOperator {
			if assignmentTarget.Operator != nil {
				return ErrOperatorAlreadyAssigned
			}

			assignmentTarget.Operator = expression
		} else {
			if assignmentTarget.Operand != nil {
				return ErrOperandAlreadyAssigned
			}

			assignmentTarget.Operand = expression
		}

	case *BinaryExpression:
		if _, isOperator := expression.(Operator); isOperator {
			if assignmentTarget.Operator != nil {
				return ErrOperatorAlreadyAssigned
			}

			assignmentTarget.Operator = expression
		} else if assignmentTarget.LeftOperand == nil {
			assignmentTarget.LeftOperand = expression
		} else if assignmentTarget.RightOperand == nil {
			assignmentTarget.RightOperand = expression
		} else {
			return ErrOperandAlreadyAssigned
		}
	}

	return nil
}

func (s *SQLBuilder) Pop(depth int) {
	s.stack = s.stack[0 : len(s.stack)-depth]
}

func (s *SQLBuilder) PopAssign(depth int) error {
	for currentDepth := 0; currentDepth < depth; currentDepth++ {
		nextExpression := s.Peek()
		s.Pop(1)

		if err := s.Assign(nextExpression); err != nil {
			return err
		}
	}

	return nil
}

func (s *SQLBuilder) Push(expression Expression) {
	if s.root == nil {
		s.root = expression
	}

	s.stack = append(s.stack, expression)
}

func (s *SQLBuilder) PushAssign(expression Expression) error {
	if s.root != nil {
		if err := s.Assign(expression); err != nil {
			return err
		}
	}

	s.Push(expression)
	return nil
}

func TranslateCypherExpression(conjunction model.Expression) (Expression, error) {
	var (
		sqlBuilder  = &SQLBuilder{}
		cypherStack []*TranslationNode[model.Expression]
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

func newSQLTranslationCursor(expression Expression) (*TranslationNode[Expression], error) {
	switch typedExpression := expression.(type) {
	case CompoundIdentifier, Operator, Literal:
		return &TranslationNode[Expression]{
			Expression: expression,
		}, nil

	case *UnaryExpression:
		return &TranslationNode[Expression]{
			Expression: expression,
			Branches:   []Expression{typedExpression.Operator, typedExpression.Operand},
		}, nil

	case *BinaryExpression:
		return &TranslationNode[Expression]{
			Expression: expression,
			Branches:   []Expression{typedExpression.LeftOperand, typedExpression.Operator, typedExpression.RightOperand},
		}, nil

	default:
		return nil, fmt.Errorf("unable to negotiate sql type %T into a translation cursor", expression)
	}
}

type ExtractionTag struct {
	Matched bool
}

func ExpressionMatches(expression Expression, matchers []Expression) (bool, error) {
	switch typedExpression := expression.(type) {
	case Identifier:
		for _, matcher := range matchers {
			switch typedMatcher := matcher.(type) {
			case Identifier:
				if typedExpression == typedMatcher {
					return true, nil
				}
			}
		}

	case CompoundIdentifier:
		for _, matcher := range matchers {
			switch typedMatcher := matcher.(type) {
			case CompoundIdentifier:
				matches := len(typedExpression) == len(typedMatcher)

				if matches {
					for idx, expressionIdentifier := range typedExpression {
						if expressionIdentifier != typedMatcher[idx] {
							matches = false
							break
						}
					}

					if matches {
						return true, nil
					}
				}
			}
		}
	default:
		return false, fmt.Errorf("unable to match for expression type %T", expression)
	}

	return false, nil
}

func extractSingle(extractionTargets []Expression, expression Expression) (Expression, error) {
	var (
		stack   []*TranslationNode[Expression]
		builder = &SQLBuilder{}
	)

	if cursor, err := newSQLTranslationCursor(expression); err != nil {
		return nil, err
	} else {
		stack = append(stack, cursor)
	}

	for len(stack) > 0 {
		nextExpressionNode := stack[len(stack)-1]

		switch typedNextExpression := nextExpressionNode.Expression.(type) {
		case Literal:
			if err := builder.Assign(typedNextExpression); err != nil {
				return nil, err
			}

			stack = stack[0 : len(stack)-1]

		case Identifier:
			switch typedTag := nextExpressionNode.Tag.(type) {
			case *ExtractionTag:
				if !typedTag.Matched {
					if matches, err := ExpressionMatches(typedNextExpression, extractionTargets); err != nil {
						return nil, err
					} else {
						typedTag.Matched = matches
					}
				}
			}

			if err := builder.Assign(typedNextExpression); err != nil {
				return nil, err
			}

			stack = stack[0 : len(stack)-1]

		case CompoundIdentifier:
			switch typedTag := nextExpressionNode.Tag.(type) {
			case *ExtractionTag:
				if !typedTag.Matched {
					if matches, err := ExpressionMatches(typedNextExpression, extractionTargets); err != nil {
						return nil, err
					} else {
						typedTag.Matched = matches
					}
				}
			}

			if err := builder.Assign(typedNextExpression.Copy()); err != nil {
				return nil, err
			}

			stack = stack[0 : len(stack)-1]

		case Operator:
			if err := builder.Assign(typedNextExpression); err != nil {
				return nil, err
			}

			stack = stack[0 : len(stack)-1]

		case *UnaryExpression:
			if nextExpressionNode.IsFirstVisit() {
				// Assign an extraction tag to this node if it doesn't have one set
				if nextExpressionNode.Tag == nil {
					nextExpressionNode.Tag = &ExtractionTag{
						Matched: false,
					}
				}

				// Push, don't assign since we need to figure out if this is relevant to our search criteria
				builder.Push(&UnaryExpression{})
			}

			if nextExpressionNode.HasNext() {
				if cursor, err := newSQLTranslationCursor(nextExpressionNode.NextBranch()); err != nil {
					return nil, err
				} else {
					// Inherit the current extraction tag and append to the descent stack
					cursor.Tag = nextExpressionNode.Tag
					stack = append(stack, cursor)
				}
			} else {
				stack = stack[0 : len(stack)-1]

				if builder.Depth() > 1 {
					switch typedTag := nextExpressionNode.Tag.(type) {
					case *ExtractionTag:
						if typedTag.Matched {
							if err := builder.PopAssign(1); err != nil {
								return nil, err
							}
						} else {
							builder.Pop(1)
						}
					}
				}
			}

		case *BinaryExpression:
			if nextExpressionNode.IsFirstVisit() {
				// Assign an extraction tag to this node if it doesn't have one set
				if nextExpressionNode.Tag == nil {
					nextExpressionNode.Tag = &ExtractionTag{
						Matched: false,
					}
				}

				// Push, don't assign since we need to figure out if this is relevant to our search criteria
				builder.Push(&BinaryExpression{})
			}

			if nextExpressionNode.HasNext() {
				if cursor, err := newSQLTranslationCursor(nextExpressionNode.NextBranch()); err != nil {
					return nil, err
				} else {
					// Inherit the current extraction tag and append to the descent stack
					cursor.Tag = nextExpressionNode.Tag
					stack = append(stack, cursor)
				}
			} else {
				stack = stack[0 : len(stack)-1]

				if builder.Depth() > 1 {
					switch typedTag := nextExpressionNode.Tag.(type) {
					case *ExtractionTag:
						if typedTag.Matched {
							if err := builder.PopAssign(1); err != nil {
								return nil, err
							}
						} else {
							builder.Pop(1)
						}
					}
				}
			}

		default:
			return nil, fmt.Errorf("unsupported expression type for binding constraint extraction: %T", nextExpressionNode.Expression)
		}
	}

	return builder.root, nil
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
