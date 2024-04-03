package visualization

import (
	"fmt"
	"github.com/specterops/bloodhound/cypher/model/pgsql"
	"github.com/specterops/bloodhound/cypher/model/pgsql/format"
	"strconv"
	"strings"
)

type SQLVisualizer struct {
	Graph  Graph
	stack  []Node
	nextID int
	done   bool
	err    error
}

func (s *SQLVisualizer) getNextID(prefix string) string {
	nextID := s.nextID
	s.nextID += 1

	return prefix + strconv.Itoa(nextID)
}

func (s *SQLVisualizer) setError(err error) {
	s.err = err
	s.done = true
}

func (s *SQLVisualizer) Enter(expression pgsql.Expression) {
	nextNode := Node{
		ID:         s.getNextID("n"),
		Labels:     []string{expression.NodeType()},
		Properties: map[string]any{},
	}

	switch typedExpression := expression.(type) {
	case pgsql.Operator:
		nextNode.Properties["value"] = typedExpression

	case pgsql.Identifier:
		nextNode.Properties["value"] = typedExpression

	case pgsql.CompoundIdentifier:
		nextNode.Properties["value"] = strings.Join(typedExpression.Strings(), ".")

	case pgsql.Literal:
		nextNode.Properties["value"] = fmt.Sprintf("%v::%T", typedExpression.Value, typedExpression.Value)
	}

	s.Graph.Nodes = append(s.Graph.Nodes, nextNode)

	if len(s.stack) > 0 {
		s.Graph.Relationships = append(s.Graph.Relationships, Relationship{
			ID:     s.getNextID("r"),
			FromID: nextNode.ID,
			ToID:   s.stack[len(s.stack)-1].ID,
		})
	}

	s.stack = append(s.stack, nextNode)
}

func (s *SQLVisualizer) Exit(expression pgsql.Expression) {
	s.stack = s.stack[0 : len(s.stack)-1]
}

func (s *SQLVisualizer) Done() bool {
	return s.done
}

func (s *SQLVisualizer) Error() error {
	return s.err
}

func SQLToDigraph(expression pgsql.Expression) (Graph, error) {
	visualizer := &SQLVisualizer{}

	if title, err := format.FormatExpression(expression); err != nil {
		return Graph{}, err
	} else {
		visualizer.Graph.Title = title.Value
	}

	return visualizer.Graph, pgsql.WalkExpression(expression, visualizer)
}
