package pgsql

import "fmt"

func newSQLWalkCursor(expression Expression) (*WalkCursor[Expression], error) {
	switch typedExpression := expression.(type) {
	case CompoundIdentifier, Operator, Literal:
		return &WalkCursor[Expression]{
			Expression: expression,
		}, nil

	case *UnaryExpression:
		return &WalkCursor[Expression]{
			Expression: expression,
			Branches:   []Expression{typedExpression.Operator, typedExpression.Operand},
		}, nil

	case *BinaryExpression:
		return &WalkCursor[Expression]{
			Expression: expression,
			Branches:   []Expression{typedExpression.LeftOperand, typedExpression.Operator, typedExpression.RightOperand},
		}, nil

	default:
		return nil, fmt.Errorf("unable to negotiate sql type %T into a translation cursor", expression)
	}
}
