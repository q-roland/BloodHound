package pgsql

import (
	"fmt"
	"github.com/specterops/bloodhound/cypher/model"
)

type CancelableErrorHandler interface {
	Done() bool
	Error() error
	SetDone()
	SetError(err error)
	SetErrorf(format string, args ...any)
}

type cancelableErrorHandler struct {
	done bool
	err  error
}

func (s *cancelableErrorHandler) Done() bool {
	return s.done
}

func (s *cancelableErrorHandler) SetDone() {
	s.done = true
}

func (s *cancelableErrorHandler) SetError(err error) {
	s.err = err
	s.done = true
}

func (s *cancelableErrorHandler) SetErrorf(format string, args ...any) {
	s.SetError(fmt.Errorf(format, args...))
}

func (s *cancelableErrorHandler) Error() error {
	return s.err
}

func NewCancelableErrorHandler() CancelableErrorHandler {
	return &cancelableErrorHandler{}
}

type Visitor interface {
	CancelableErrorHandler

	Visit(expression Expression)
}

type HierarchicalVisitor[E any] interface {
	CancelableErrorHandler

	Enter(expression E)
	Visit(expression E)
	Exit(expression E)
}

type WalkCursor[E any] struct {
	Expression  E
	Branches    []E
	BranchIndex int
}

func (s *WalkCursor[E]) NumBranchesRemaining() int {
	return len(s.Branches) - s.BranchIndex
}

func (s *WalkCursor[E]) IsFirstVisit() bool {
	return s.BranchIndex == 0
}

func (s *WalkCursor[E]) HasNext() bool {
	return s.BranchIndex < len(s.Branches)
}

func (s *WalkCursor[E]) NextBranch() E {
	nextBranch := s.Branches[s.BranchIndex]
	s.BranchIndex += 1

	return nextBranch
}

func SetBranches[E any, T any](cursor *WalkCursor[E], branches ...T) error {
	for _, branch := range branches {
		if eTypedBranch, isEType := any(branch).(E); !isEType {
			var emptyE E
			return fmt.Errorf("branch type %T does not convert to %T", branch, emptyE)
		} else {
			cursor.Branches = append(cursor.Branches, eTypedBranch)
		}
	}

	return nil
}

type WalkOrder int

func (s WalkOrder) Visiting(visitor Visitor) OrderedVisitor {
	return OrderedVisitor{
		order:   s,
		visitor: visitor,
	}
}

const (
	WalkOrderPrefix WalkOrder = iota
	WalkOrderPostfix
)

type OrderedVisitor struct {
	order   WalkOrder
	visitor Visitor
}

func (s OrderedVisitor) Done() bool {
	return s.visitor.Done()
}

func (s OrderedVisitor) Error() error {
	return s.visitor.Error()
}

func (s OrderedVisitor) Enter(expression Expression) {
	if s.order == WalkOrderPrefix {
		s.visitor.Visit(expression)
	}
}

func (s OrderedVisitor) Visit(expression Expression) {}

func (s OrderedVisitor) Exit(expression Expression) {
	if s.order == WalkOrderPostfix {
		s.visitor.Visit(expression)
	}
}

func Walk[E any](expression E, visitor HierarchicalVisitor[E], cursorConstructor func(expression E) (*WalkCursor[E], error)) error {
	var stack []*WalkCursor[E]

	if cursor, err := cursorConstructor(expression); err != nil {
		return err
	} else {
		stack = append(stack, cursor)
	}

	for len(stack) > 0 && !visitor.Done() {
		var (
			nextExpressionNode = stack[len(stack)-1]
			isFirstVisit       = nextExpressionNode.IsFirstVisit()
		)

		if isFirstVisit {
			visitor.Enter(nextExpressionNode.Expression)
		}

		if nextExpressionNode.HasNext() {
			if !isFirstVisit {
				visitor.Visit(nextExpressionNode.Expression)
			}

			if cursor, err := cursorConstructor(nextExpressionNode.NextBranch()); err != nil {
				return err
			} else {
				stack = append(stack, cursor)
			}
		} else {
			visitor.Exit(nextExpressionNode.Expression)
			stack = stack[0 : len(stack)-1]
		}
	}

	return visitor.Error()
}

func PgSQLWalk(expression Expression, visitor HierarchicalVisitor[Expression]) error {
	return Walk(expression, visitor, newSQLWalkCursor)
}

func CypherWalk(expression model.Expression, visitor HierarchicalVisitor[model.Expression]) error {
	return Walk(expression, visitor, newCypherWalkCursor)
}
