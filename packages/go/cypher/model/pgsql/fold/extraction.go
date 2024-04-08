package fold

import (
	"fmt"
	"github.com/specterops/bloodhound/cypher/model/pgsql"
	"github.com/specterops/bloodhound/cypher/model/pgsql/visualization"
)

type Extractor struct {
	pgsql.CancelableErrorHandler

	conjoinedConstraintsByKey map[string]*pgsql.Tree
	dependentExpression       pgsql.Expression
	operatorDeps              []*pgsql.IdentifierDependencies
}

func (s *Extractor) Enter(expression pgsql.Expression) {
	switch typedExpression := expression.(type) {
	case pgsql.Identifier:
		for _, operatorDeps := range s.operatorDeps {
			operatorDeps.Track(typedExpression.String())
		}

	case pgsql.CompoundIdentifier:
		for _, operatorDeps := range s.operatorDeps {
			operatorDeps.Track(typedExpression[0].String())
		}

	case *pgsql.BinaryExpression:
		s.operatorDeps = append(s.operatorDeps, pgsql.NewIdentifierDependencies())
		typedExpression.LOperDependencies = s.operatorDeps[len(s.operatorDeps)-1]
	}
}

func (s *Extractor) Visit(expression pgsql.Expression) {
	switch typedExpression := expression.(type) {
	case *pgsql.BinaryExpression:
		s.operatorDeps[len(s.operatorDeps)-1] = pgsql.NewIdentifierDependencies()
		typedExpression.ROperDependencies = s.operatorDeps[len(s.operatorDeps)-1]
	}
}

func (s *Extractor) Exit(expression pgsql.Expression) {
	switch typedExpression := expression.(type) {
	case *pgsql.BinaryExpression:
		s.operatorDeps = s.operatorDeps[:len(s.operatorDeps)-1]

		switch typedOperator := typedExpression.Operator.(type) {
		case pgsql.Operator:
			if typedOperator == pgsql.OperatorAnd {
				typedExpression.Rewritten = true

				rewrite := true

				switch typedLeftOper := typedExpression.LeftOperand.(type) {
				case *pgsql.BinaryExpression:
					rewrite = !typedLeftOper.Rewritten
				}

				if rewrite {
					leftDepKey := typedExpression.LOperDependencies.Key()

					if tree, hasTree := s.conjoinedConstraintsByKey[leftDepKey]; hasTree {
						if err := tree.ContinueBinaryExpression(pgsql.OperatorAnd, typedExpression.LeftOperand); err != nil {
							s.SetError(err)
						}
					} else {
						newTree := &pgsql.Tree{}
						newTree.Push(typedExpression.LeftOperand)

						s.conjoinedConstraintsByKey[leftDepKey] = newTree
					}
				}

				rewrite = true

				switch typedRightOper := typedExpression.RightOperand.(type) {
				case *pgsql.BinaryExpression:
					rewrite = !typedRightOper.Rewritten

				default:
				}

				if rewrite {
					rightDepKey := typedExpression.ROperDependencies.Key()

					if tree, hasTree := s.conjoinedConstraintsByKey[rightDepKey]; hasTree {
						if err := tree.ContinueBinaryExpression(pgsql.OperatorAnd, typedExpression.RightOperand); err != nil {
							s.SetError(err)
						}
					} else {
						newTree := &pgsql.Tree{}
						newTree.Push(typedExpression.RightOperand)

						s.conjoinedConstraintsByKey[rightDepKey] = newTree
					}
				}
			}

		default:
			s.SetErrorf("unknown operator type: %T", typedExpression)
		}
	}
}

func FragmentExpressionTree(expression pgsql.Expression) (map[string]*pgsql.Tree, error) {
	extractor := &Extractor{
		CancelableErrorHandler:    pgsql.NewCancelableErrorHandler(),
		conjoinedConstraintsByKey: map[string]*pgsql.Tree{},
	}

	if err := pgsql.PgSQLWalk(expression, extractor); err != nil {
		return nil, err
	}

	for key, tree := range extractor.conjoinedConstraintsByKey {
		visualization.MustWritePUML(tree.Root(), fmt.Sprintf("/home/zinic/digraphs/%s.puml", key))
	}

	return extractor.conjoinedConstraintsByKey, nil
}
