package pgsql

import (
	"fmt"
)

type CancelableErrorHandler interface {
	Done() bool
	Error() error
}

type Visitor interface {
	CancelableErrorHandler

	Visit(expression Expression)
}

type HierarchicalVisitor interface {
	CancelableErrorHandler

	Enter(expression Expression)
	Visit(expression Expression)
	Exit(expression Expression)
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

func Walk(expression Expression, visitor HierarchicalVisitor) error {
	var stack []*WalkCursor[Expression]

	if cursor, err := newSQLWalkCursor(expression); err != nil {
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

			if cursor, err := newSQLWalkCursor(nextExpressionNode.NextBranch()); err != nil {
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
