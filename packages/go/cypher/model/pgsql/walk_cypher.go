package pgsql

import (
	"fmt"
	"github.com/specterops/bloodhound/cypher/model"
)

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

func newCypherTranslationCursor(expression model.Expression) (*WalkCursor[model.Expression], error) {
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

	case *model.Comparison:
		return &WalkCursor[model.Expression]{
			Expression: expression,
			Branches:   append([]model.Expression{typedExpression.Left}, MustSliceAs[model.Expression](typedExpression.Partials)...),
		}, nil

	default:
		return nil, fmt.Errorf("unable to negotiate cypher model type %T into a translation cursor", expression)
	}
}
