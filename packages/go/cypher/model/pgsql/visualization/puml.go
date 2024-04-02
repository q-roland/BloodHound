package visualization

import (
	"fmt"
	"io"
	"strings"
)

func WriteStrings(writer io.Writer, strings ...string) error {
	for _, str := range strings {
		if _, err := io.WriteString(writer, str); err != nil {
			return err
		}
	}

	return nil
}

func GraphToPUMLDigraph(graph Graph, writer io.Writer) error {
	if err := WriteStrings(writer, "@startuml\ndigraph syntaxTree {\nrankdir=BT\n\n"); err != nil {
		return err
	}

	if graph.Title != "" {
		if err := WriteStrings(writer, "label=\"", graph.Title, "\"\n\n"); err != nil {
			return err
		}
	}

	for _, node := range graph.Nodes {
		nodeLabel := strings.Join(node.Labels, ":")

		if value, hasValue := node.Properties["value"]; hasValue {
			nodeLabel = fmt.Sprintf("%v", value)
		}

		if err := WriteStrings(writer, node.ID, "[label=\"", nodeLabel, "\"]", "\n"); err != nil {
			return err
		}
	}

	if err := WriteStrings(writer, "\n"); err != nil {
		return err
	}

	for _, relationship := range graph.Relationships {
		if err := WriteStrings(writer, relationship.FromID, " -> ", relationship.ToID, "\n"); err != nil {
			return err
		}
	}

	return WriteStrings(writer, "}\n@enduml\n")
}
