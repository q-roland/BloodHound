package fold

import (
	"fmt"
	"github.com/specterops/bloodhound/cypher/model/pgsql"
	"github.com/specterops/bloodhound/cypher/model/pgsql/visualization"
)

type Extractor struct {
	conjoinedConstraintsByKey map[string]*pgsql.Tree
	dependentExpression       pgsql.Expression
	operatorDeps              []*pgsql.IdentifierDependencies
	err                       error
	done                      bool
}

func (s *Extractor) setError(err error) {
	s.err = err
	s.done = true
}

func (s *Extractor) setErrorf(format string, args ...any) {
	s.err = fmt.Errorf(format, args...)
	s.done = true
}

func (s *Extractor) Enter(expression pgsql.Expression) {
	switch typedExpression := expression.(type) {
	case pgsql.Identifier:
		for _, operatorDeps := range s.operatorDeps {
			operatorDeps.Track(typedExpression.String())
		}

	case pgsql.CompoundIdentifier:
		for _, operatorDeps := range s.operatorDeps {
			operatorDeps.Track(typedExpression.String())
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
			if typedOperator == "and" {
				typedExpression.Rewritten = true

				if len(typedExpression.LOperDependencies.Identifiers) == 1 {
					fmt.Printf("Left operand references only a single bound identifier")
				} else if len(typedExpression.LOperDependencies.Identifiers) > 1 {
					fmt.Printf("Left operand references only a multiple bound identifiers")
				}

				if len(typedExpression.ROperDependencies.Identifiers) == 1 {
					fmt.Printf("Right operand references only a single bound identifier")
				} else if len(typedExpression.ROperDependencies.Identifiers) > 1 {
					fmt.Printf("Right operand references only a multiple bound identifiers")
				}

				rewrite := true

				switch typedLeftOper := typedExpression.LeftOperand.(type) {
				case *pgsql.BinaryExpression:
					rewrite = !typedLeftOper.Rewritten
				}

				if rewrite {
					leftDepKey := typedExpression.LOperDependencies.Key()

					visualization.MustWritePUML(typedExpression.LeftOperand, "/home/zinic/digraphs/stage.puml")

					if tree, hasTree := s.conjoinedConstraintsByKey[leftDepKey]; hasTree {
						if err := tree.And(typedExpression.LeftOperand); err != nil {
							s.setError(err)
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

					visualization.MustWritePUML(typedExpression.RightOperand, "/home/zinic/digraphs/stage.puml")

					if tree, hasTree := s.conjoinedConstraintsByKey[rightDepKey]; hasTree {
						if err := tree.And(typedExpression.RightOperand); err != nil {
							s.setError(err)
						}
					} else {
						newTree := &pgsql.Tree{}
						newTree.Push(typedExpression.RightOperand)

						s.conjoinedConstraintsByKey[rightDepKey] = newTree
					}
				}
			}

		default:
			s.setErrorf("unknown operator type: %T", typedExpression)
		}
	}
}

func (s *Extractor) Done() bool {
	return s.done
}

func (s *Extractor) Error() error {
	return s.err
}

func Extract(targets []pgsql.Expression, expression pgsql.Expression) (pgsql.Expression, error) {
	extractor := &Extractor{
		conjoinedConstraintsByKey: map[string]*pgsql.Tree{},
	}

	if err := pgsql.Walk(expression, extractor); err != nil {
		return nil, err
	}

	for key, tree := range extractor.conjoinedConstraintsByKey {
		visualization.MustWritePUML(tree.Root(), fmt.Sprintf("/home/zinic/digraphs/%s.puml", key))
	}

	return nil, nil
}
