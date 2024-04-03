package pgsql

import "fmt"

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

type IdentifierAnnotations struct {
	IdentifierDependencies         map[string]Identifier
	CompoundIdentifierDependencies map[string]CompoundIdentifier
}

type Extractor struct {
	annotations []*IdentifierAnnotations
	targets     []Expression
	err         error
	done        bool
}

func (s *Extractor) pushAnnotation() {
	s.annotations = append(s.annotations, &IdentifierAnnotations{
		IdentifierDependencies:         map[string]Identifier{},
		CompoundIdentifierDependencies: map[string]CompoundIdentifier{},
	})
}

func (s *Extractor) popAnnotation() *IdentifierAnnotations {
	annotations := s.annotations[len(s.annotations)-1]
	s.annotations = s.annotations[0 : len(s.annotations)-1]

	return annotations
}

func (s *Extractor) trackIdentifierDependency(identifier Identifier) {
	for _, annotation := range s.annotations {
		annotation.IdentifierDependencies[identifier.String()] = identifier
	}
}

func (s *Extractor) trackCompoundIdentifierDependency(identifier CompoundIdentifier) {
	for _, annotation := range s.annotations {
		annotation.CompoundIdentifierDependencies[identifier.String()] = identifier
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
		s.trackCompoundIdentifierDependency(typedExpression)

	case *UnaryExpression, *BinaryExpression:
		s.pushAnnotation()

	default:
		s.setError(fmt.Errorf("unsupported expression type for binding constraint extraction: %T", expression))
	}
}

func (s *Extractor) Exit(expression Expression) {
	switch expression.(type) {
	case *BinaryExpression, *UnaryExpression:
		s.popAnnotation()
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
		targets: targets,
	}

	return nil, WalkExpression(expression, extractor)
}
