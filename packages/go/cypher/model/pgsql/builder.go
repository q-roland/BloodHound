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

func Assign(assignmentTarget, branch Expression) error {
	switch typedAssignmentTarget := assignmentTarget.(type) {
	case *UnaryExpression:
		if _, isOperator := branch.(Operator); isOperator {
			if typedAssignmentTarget.Operator != nil {
				return ErrOperatorAlreadyAssigned
			}

			typedAssignmentTarget.Operator = branch
		} else {
			if typedAssignmentTarget.Operand != nil {
				return ErrOperandAlreadyAssigned
			}

			typedAssignmentTarget.Operand = branch
		}

	case *BinaryExpression:
		if _, isOperator := branch.(Operator); isOperator {
			if typedAssignmentTarget.Operator != nil {
				return ErrOperatorAlreadyAssigned
			}

			typedAssignmentTarget.Operator = branch
		} else if typedAssignmentTarget.LeftOperand == nil {
			typedAssignmentTarget.LeftOperand = branch
		} else if typedAssignmentTarget.RightOperand == nil {
			typedAssignmentTarget.RightOperand = branch
		} else {
			return ErrOperandAlreadyAssigned
		}

	default:
		return fmt.Errorf("unable to assign branch %T to assignment target %T", branch, assignmentTarget)
	}

	return nil
}

type Tree struct {
	root  Expression
	stack []Expression
}

func (s *Tree) Depth() int {
	return len(s.stack)
}

func (s *Tree) Peek() Expression {
	return s.stack[len(s.stack)-1]
}

func (s *Tree) Pop() {
	s.stack = s.stack[:len(s.stack)-1]
}

func (s *Tree) Assign(expression Expression) error {
	return Assign(s.stack[len(s.stack)-1], expression)
}

func (s *Tree) Push(expression Expression) error {
	if len(s.stack) == 0 {
		s.root = expression
	} else if err := s.Assign(expression); err != nil {
		return err
	}

	s.stack = append(s.stack, expression)
	return nil
}

func (s *Tree) Root() Expression {
	return s.root
}

type TreeBuilder struct {
	trees []*Tree
}

func IsOperator(expression Expression, matcher Operator) bool {
	if operator, isOperator := expression.(Operator); isOperator {
		return operator == matcher
	}

	return false
}

func (s *TreeBuilder) Offshoot() error {
	tree := s.trees[len(s.trees)-1]

	for idx := len(tree.stack) - 1; idx >= 0; idx-- {
		switch nextExpression := tree.stack[idx].(type) {
		case *BinaryExpression:
			if IsOperator(nextExpression.Operator, "and") {
				// cleave here
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
					return fmt.Errorf("can't offshoot from expression type: %T", descendingExpression)
				}
			}
		}
	}

	return nil
}

func (s *TreeBuilder) Peek() *Tree {
	return s.trees[len(s.trees)-1]
}

func (s *TreeBuilder) Push(tree *Tree) {
	s.trees = append(s.trees, tree)
}

func (s *TreeBuilder) Pop() *Tree {
	tree := s.Peek()
	s.trees = s.trees[:len(s.trees)-1]

	return tree
}

func (s *TreeBuilder) AssignExpression(expression Expression) error {
	return s.Peek().Assign(expression)
}

func (s *TreeBuilder) PushExpression(expression Expression) error {
	if len(s.trees) == 0 {
		s.trees = append(s.trees, &Tree{})
	}

	return s.Peek().Push(expression)
}

func (s *TreeBuilder) PopExpression() {
	currentTree := s.trees[len(s.trees)-1]
	currentTree.Pop()

	if currentTree.Depth() == 0 {
		s.trees = s.trees[:len(s.trees)-1]
	}
}
