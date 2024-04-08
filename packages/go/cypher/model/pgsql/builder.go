package pgsql

import (
	"errors"
	"fmt"
)

var (
	ErrOperatorAlreadyAssigned = errors.New("expression operator already assigned")
	ErrOperandAlreadyAssigned  = errors.New("expression operand already assigned")
)

func PeekAs[T any](tree *Tree) (T, error) {
	expression := tree.Peek()

	if value, isT := expression.(T); isT {
		return value, nil
	}

	var emptyT T
	return emptyT, fmt.Errorf("unable to convert expression %T to type %T", expression, emptyT)
}

type Tree struct {
	stack []Expression
}

func (s *Tree) Depth() int {
	return len(s.stack)
}

func (s *Tree) Root() Expression {
	return s.stack[0]
}

func (s *Tree) Peek() Expression {
	return s.stack[len(s.stack)-1]
}

func (s *Tree) Ascend(depth int) {
	s.stack = s.stack[:len(s.stack)-depth]
}

func (s *Tree) Pop() Expression {
	expression := s.Peek()
	s.stack = s.stack[:len(s.stack)-1]

	return expression
}

func (s *Tree) ContinueBinaryExpression(operator Operator, operand Expression) error {
	if assignmentTarget, err := PeekAs[*BinaryExpression](s); err != nil {
		return err
	} else if assignmentTarget.LeftOperand == nil {
		assignmentTarget.LeftOperand = operand
		assignmentTarget.Operator = operator
	} else if assignmentTarget.RightOperand == nil {
		assignmentTarget.RightOperand = operand
		assignmentTarget.Operator = operator
	} else {
		assignmentTarget.RightOperand = &BinaryExpression{
			Operator:     operator,
			LeftOperand:  assignmentTarget.RightOperand,
			RightOperand: operand,
		}

		s.Push(assignmentTarget.RightOperand)
	}

	return nil
}

func (s *Tree) Push(expression Expression) {
	s.stack = append(s.stack, expression)
}
