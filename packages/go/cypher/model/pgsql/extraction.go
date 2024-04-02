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

type Extractor struct {
	builder *ExpressionBuilder
	targets []Expression
	err     error
	done    bool
}

func (s *Extractor) setError(err error) {
	s.err = err
	s.done = true
}

func (s *Extractor) Enter(expression Expression) {
	switch expression.(type) {
	case *BinaryExpression:
		s.builder.Push(&BinaryExpression{})

	default:
		s.setError(fmt.Errorf("unsupported expression type for binding constraint extraction: %T", expression))
	}
}

func (s *Extractor) Exit(expression Expression) {
}

func (s *Extractor) Done() bool {
	return s.done
}

func (s *Extractor) Error() error {
	return s.err
}

func Extract(targets []Expression, expression Expression) (Expression, error) {
	extractor := &Extractor{
		builder: &ExpressionBuilder{},
		targets: targets,
	}

	return extractor.builder.root, WalkExpression(expression, extractor)
}
