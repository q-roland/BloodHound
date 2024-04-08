package pgsql

import (
	"fmt"
	"github.com/specterops/bloodhound/cypher/model"
)

func newCypherWalkCursor(expression model.Expression) (*WalkCursor[model.Expression], error) {
	cursor := &WalkCursor[model.Expression]{
		Expression: expression,
	}

	switch typedExpression := expression.(type) {
	// Types with no AST branches
	case *model.PropertyLookup, *model.Literal, model.Operator:
		return cursor, nil

	case *model.ArithmeticExpression:
		return &WalkCursor[model.Expression]{
			Expression: expression,
			Branches:   append([]model.Expression{typedExpression.Left}, MustSliceAs[model.Expression](typedExpression.Partials)...),
		}, nil

	case *model.PartialArithmeticExpression:
		return &WalkCursor[model.Expression]{
			Expression:  expression,
			Branches:    []model.Expression{typedExpression.Operator, typedExpression.Right},
			BranchIndex: 0,
		}, nil

	case *model.PartialComparison:
		return &WalkCursor[model.Expression]{
			Expression:  expression,
			Branches:    []model.Expression{typedExpression.Operator, typedExpression.Right},
			BranchIndex: 0,
		}, nil

	case *model.Negation:
		return cursor, SetBranches(cursor, typedExpression.Expression)

	case *model.Conjunction:
		return cursor, SetBranches(cursor, typedExpression.Expressions...)

	case *model.Disjunction:
		return cursor, SetBranches(cursor, typedExpression.Expressions...)

	case *model.Comparison:
		return &WalkCursor[model.Expression]{
			Expression: expression,
			Branches:   append([]model.Expression{typedExpression.Left}, MustSliceAs[model.Expression](typedExpression.Partials)...),
		}, nil

	default:
		return nil, fmt.Errorf("unable to negotiate cypher model type %T into a translation cursor", expression)
	}
}
