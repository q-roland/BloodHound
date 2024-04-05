package pgsql

import (
	"fmt"
)

type Visitor interface {
	Enter(expression Expression)
	Exit(expression Expression)
	Done() bool
	Error() error
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

func WalkExpression(expression Expression, visitor Visitor) error {
	var stack []*WalkCursor[Expression]

	if cursor, err := newSQLWalkCursor(expression); err != nil {
		return err
	} else {
		stack = append(stack, cursor)
	}

	for len(stack) > 0 && !visitor.Done() {
		nextExpressionNode := stack[len(stack)-1]

		if nextExpressionNode.IsFirstVisit() {
			visitor.Enter(nextExpressionNode.Expression)
		}

		if nextExpressionNode.HasNext() {
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
