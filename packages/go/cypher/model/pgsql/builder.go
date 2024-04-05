package pgsql

import (
	"errors"
	"fmt"
)

func SearchT[T any](expressionBuilder *ExpressionBuilder, delegate func(index int, expression Expression) (T, bool)) (T, bool) {
	for idx := len(expressionBuilder.stack) - 1; idx >= 0; idx-- {
		if value, found := delegate(idx, expressionBuilder.stack[idx]); found {
			return value, true
		}
	}

	var empty T
	return empty, false
}

var (
	ErrOperatorAlreadyAssigned = errors.New("expression operator already assigned")
	ErrOperandAlreadyAssigned  = errors.New("expression operand already assigned")
)

type ExpressionBuilder struct {
	root  Expression
	stack []Expression
}

func (s *ExpressionBuilder) Depth() int {
	return len(s.stack)
}

func (s *ExpressionBuilder) Peek() Expression {
	return s.stack[len(s.stack)-1]
}

func (s *ExpressionBuilder) Assign(expression Expression) error {
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

func (s *ExpressionBuilder) Pop(depth int) {
	s.stack = s.stack[0 : len(s.stack)-depth]
}

func (s *ExpressionBuilder) PopAssign(depth int) error {
	for currentDepth := 0; currentDepth < depth; currentDepth++ {
		nextExpression := s.Peek()
		s.Pop(1)

		if err := s.Assign(nextExpression); err != nil {
			return err
		}
	}

	return nil
}

func (s *ExpressionBuilder) Push(expression Expression) {
	if s.root == nil {
		s.root = expression
	}

	s.stack = append(s.stack, expression)
}

func (s *ExpressionBuilder) PushAssign(expression Expression) error {
	if s.root != nil {
		if err := s.Assign(expression); err != nil {
			return err
		}
	}

	s.Push(expression)
	return nil
}

type ExpressionTreeBuilder struct {
	trees []*ExpressionBuilder
}

func (s *ExpressionTreeBuilder) Current() *ExpressionBuilder {
	return s.trees[len(s.trees)-1]
}

func (s *ExpressionTreeBuilder) PopTree() {
	s.trees = s.trees[0 : len(s.trees)-1]
}

type ReverseSearchFunc func(index int, expression Expression) bool

func (s *ExpressionTreeBuilder) CreateOffshoot(searchFunc ReverseSearchFunc) bool {
	tree := s.trees[len(s.trees)-1]

	for idx := len(tree.stack) - 1; idx >= 0; idx-- {
		nextExpression := tree.stack[idx]

		// This is the target
		if found := searchFunc(idx, nextExpression); found {
			descendingExpression := tree.stack[idx+1]

			switch typedExpression := tree.stack[idx].(type) {
			case *BinaryExpression:
				// Did we ascend from the left or right operand expression
				if typedExpression.LeftOperand == descendingExpression {
					typedExpression.LeftOperand = nil
					tree.stack = tree.stack[0:idx]
				} else {
					typedExpression.RightOperand = nil
					tree.stack = tree.stack[0:idx]
				}

			default:
				panic(fmt.Sprintf("can't offshoot from expression type: %T", descendingExpression))
			}

			s.trees = append(s.trees, &ExpressionBuilder{
				root:  descendingExpression,
				stack: []Expression{descendingExpression},
			})

			return true
		}
	}

	return false
}
