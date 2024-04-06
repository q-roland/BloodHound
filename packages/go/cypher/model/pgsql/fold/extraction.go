package fold

import (
	"fmt"
	"github.com/specterops/bloodhound/cypher/model/pgsql"
	"slices"
	"strings"
)

type ExtractionTag struct {
	Matched bool
}

func ExpressionMatches(expression pgsql.Expression, matchers []pgsql.Expression) (bool, error) {
	switch typedExpression := expression.(type) {
	case pgsql.Identifier:
		for _, matcher := range matchers {
			switch typedMatcher := matcher.(type) {
			case pgsql.Identifier:
				if typedExpression == typedMatcher {
					return true, nil
				}
			}
		}

	case pgsql.CompoundIdentifier:
		for _, matcher := range matchers {
			switch typedMatcher := matcher.(type) {
			case pgsql.CompoundIdentifier:
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

type Dependencies struct {
	Identifiers map[string]struct{}
}

func (s *Dependencies) Track(identifier string) {
	s.Identifiers[identifier] = struct{}{}
}

func (s *Dependencies) Key() string {
	depSlice := make([]string, 0, len(s.Identifiers))

	for key := range s.Identifiers {
		depSlice = append(depSlice, key)
	}

	slices.Sort(depSlice)
	return strings.Join(depSlice, "")
}

type Extractor struct {
	treeBuilder  *pgsql.TreeBuilder
	dependencies []*Dependencies
	err          error
	done         bool
}

func (s *Extractor) setError(err error) {
	s.err = err
	s.done = true
}

func (s *Extractor) Enter(expression pgsql.Expression) {
	switch typedExpression := expression.(type) {
	case pgsql.Operator:
	case pgsql.Literal:
		if err := s.treeBuilder.AssignExpression(typedExpression); err != nil {
			s.setError(err)
		}

	case pgsql.Identifier:
		identifier := typedExpression.String()

		for _, dependency := range s.dependencies {
			dependency.Track(identifier)
		}

		if err := s.treeBuilder.AssignExpression(typedExpression); err != nil {
			s.setError(err)
		}

	case pgsql.CompoundIdentifier:
		identifier := typedExpression.String()

		for _, dependency := range s.dependencies {
			dependency.Track(identifier)
		}

		if err := s.treeBuilder.AssignExpression(typedExpression); err != nil {
			s.setError(err)
		}

	case *pgsql.UnaryExpression:
		s.dependencies = append(s.dependencies, &Dependencies{
			Identifiers: make(map[string]struct{}),
		})

		if err := s.treeBuilder.PushExpression(&pgsql.UnaryExpression{}); err != nil {
			s.setError(err)
		}

	case *pgsql.BinaryExpression:
		s.dependencies = append(s.dependencies, &Dependencies{
			Identifiers: make(map[string]struct{}),
		})

		if err := s.treeBuilder.PushExpression(&pgsql.BinaryExpression{
			Operator: typedExpression.Operator,
		}); err != nil {
			s.setError(err)
		}

	default:
		s.setError(fmt.Errorf("unsupported expression type for binding constraint extraction: %T", expression))
	}
}

func (s *Extractor) Exit(expression pgsql.Expression) {
	switch expression.(type) {
	case *pgsql.BinaryExpression:
		var (
			expressionDeps   = s.dependencies[len(s.dependencies)-1]
			expressionDepKey = expressionDeps.Key()
			differs          = false
		)

		s.dependencies = s.dependencies[0 : len(s.dependencies)-1]

		for idx := len(s.dependencies) - 1; idx >= 0; idx-- {
			previousDepKey := s.dependencies[idx].Key()

			if differs = previousDepKey != expressionDepKey; differs {
				break
			}
		}

		if differs {
			if err := s.treeBuilder.Offshoot(); err != nil {
				s.setError(err)
			}
		}

		s.treeBuilder.PopExpression()

	case *pgsql.UnaryExpression:
		//s.builder.Pop(1)
	}
}

func (s *Extractor) Done() bool {
	return s.done
}

func (s *Extractor) Error() error {
	return s.err
}

func Extract(targets []pgsql.Expression, expression pgsql.Expression) (pgsql.Expression, error) {
	treeBuilder := &pgsql.TreeBuilder{}
	treeBuilder.Push(&pgsql.Tree{})

	extractor := &Extractor{
		treeBuilder: treeBuilder,
	}

	return nil, pgsql.WalkExpression(expression, extractor)
}
