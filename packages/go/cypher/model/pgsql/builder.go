package pgsql

import "errors"

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
