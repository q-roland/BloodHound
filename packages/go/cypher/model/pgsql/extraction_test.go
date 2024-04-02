package pgsql_test

import (
	"github.com/specterops/bloodhound/cypher/frontend"
	"github.com/specterops/bloodhound/cypher/model/pgsql"
	"github.com/specterops/bloodhound/cypher/model/pgsql/format"
	"github.com/specterops/bloodhound/cypher/model/pgsql/visualization"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func mustWritePUML(t *testing.T, expression pgsql.Expression) {
	graph, err := visualization.SQLToDigraph(expression)
	require.Nil(t, err)

	fout, err := os.OpenFile("/home/zinic/graph.puml", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	defer fout.Close()

	require.Nil(t, err)
	require.Nil(t, visualization.GraphToPUMLDigraph(graph, fout))
}

func TestExtract(t *testing.T) {
	regularQuery, err := frontend.ParseCypher(frontend.NewContext(), "match (s), (e) where s.name = s.other + 1 / s.last and s.value = 1234 and not s.test and e.value = 1234 and e.comp = s.comp return s")
	require.Nil(t, err)

	sqlAST, err := pgsql.TranslateCypherExpression(regularQuery.SingleQuery.SinglePartQuery.ReadingClauses[0].Match.Where.Expressions[0])
	require.Nil(t, err)

	extractedAST, err := pgsql.Extract([]pgsql.Expression{pgsql.Identifier("s"), pgsql.CompoundIdentifier{"s", "properties"}}, sqlAST)
	require.Nil(t, err)

	mustWritePUML(t, extractedAST)

	output, err := format.FormatExpression(extractedAST)
	require.Nil(t, err)
	require.Equal(t, "s.properties -> 'name' = s.properties -> 'other' + 1 / s.properties -> 'last' and s.properties -> 'value' = 1234 and not s.properties -> 'test'", output.Value)
}
