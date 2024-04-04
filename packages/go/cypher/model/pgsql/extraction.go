package pgsql

import (
	"fmt"
	"slices"
	"strings"
)

type ExtractionTag struct {
	Matched bool
}

func ExpressionMatches(expression Expression, matchers []Expression) (bool, error) {
	switch typedExpression := expression.(type) {
	case Identifier:
		for _, matcher := range matchers {
			switch typedMatcher := matcher.(type) {
			case Identifier:
				if typedExpression == typedMatcher {
					return true, nil
				}
			}
		}

	case CompoundIdentifier:
		for _, matcher := range matchers {
			switch typedMatcher := matcher.(type) {
			case CompoundIdentifier:
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

func (s *Dependencies) Key() string {
	depSlice := make([]string, 0, len(s.Identifiers))

	for key := range s.Identifiers {
		depSlice = append(depSlice, key)
	}

	slices.Sort(depSlice)
	return strings.Join(depSlice, "")
}

type Extractor struct {
	annotations     []*Dependencies
	expressionTrees *Builder
	err             error
	done            bool
}

func (s *Extractor) pushAnnotation() {
	s.annotations = append(s.annotations, &Dependencies{
		Identifiers: map[string]struct{}{},
	})
}

func (s *Extractor) popAnnotation() *Dependencies {
	annotations := s.annotations[len(s.annotations)-1]
	s.annotations = s.annotations[0 : len(s.annotations)-1]

	return annotations
}

func (s *Extractor) trackIdentifierDependency(identifier StringLike) {
	for _, annotation := range s.annotations {
		annotation.Identifiers[identifier.String()] = struct{}{}
	}
}

func (s *Extractor) setError(err error) {
	s.err = err
	s.done = true
}

func (s *Extractor) Enter(expression Expression) {
	switch typedExpression := expression.(type) {
	case Operator, Literal:
	case Identifier:
		s.trackIdentifierDependency(typedExpression)

	case CompoundIdentifier:
		s.trackIdentifierDependency(typedExpression)

	case *UnaryExpression, *BinaryExpression:
		s.pushAnnotation()

	default:
		s.setError(fmt.Errorf("unsupported expression type for binding constraint extraction: %T", expression))
	}
}

type Builder struct {
	trees map[string]*ExpressionBuilder
}

func (s *Builder) AppendUnaryExpression(expression *UnaryExpression, dependencies *Dependencies) error {
	var (
		depKey        = dependencies.Key()
		tree, hasTree = s.trees[depKey]
	)

	if !hasTree {
		tree = &ExpressionBuilder{}
		s.trees[depKey] = tree
	}

	return tree.PushAssign(expression)
}

func (s *Builder) AppendBinaryExpression(expression *BinaryExpression, dependencies *Dependencies) error {
	var (
		depKey        = dependencies.Key()
		tree, hasTree = s.trees[depKey]
	)

	if !hasTree {
		tree = &ExpressionBuilder{}
		s.trees[depKey] = tree
	}

	return tree.PushAssign(expression)
}

func (s *Extractor) Exit(expression Expression) {
	switch typedExpression := expression.(type) {
	case *BinaryExpression:
		if err := s.expressionTrees.AppendBinaryExpression(typedExpression, s.popAnnotation()); err != nil {
			s.setError(err)
		}

	case *UnaryExpression:
		if err := s.expressionTrees.AppendUnaryExpression(typedExpression, s.popAnnotation()); err != nil {
			s.setError(err)
		}
	}
}

func (s *Extractor) Done() bool {
	return s.done
}

func (s *Extractor) Error() error {
	return s.err
}

func Extract(targets []Expression, expression Expression) (Expression, error) {
	extractor := &Extractor{
		expressionTrees: &Builder{
			trees: map[string]*ExpressionBuilder{},
		},
	}

	return nil, WalkExpression(expression, extractor)
}
