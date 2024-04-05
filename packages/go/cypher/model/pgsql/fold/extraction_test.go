package fold_test

import (
	"github.com/specterops/bloodhound/cypher/frontend"
	"github.com/specterops/bloodhound/cypher/model/pgsql"
	"github.com/specterops/bloodhound/cypher/model/pgsql/fold"
	"github.com/specterops/bloodhound/cypher/model/pgsql/format"
	"github.com/specterops/bloodhound/cypher/model/pgsql/visualization"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestExtract(t *testing.T) {
	regularQuery, err := frontend.ParseCypher(frontend.NewContext(), "match (s), (e) where s.name = 'test' and e.name = 'test' and s.value = '1234' return s, e")
	require.Nil(t, err)

	sqlAST, err := pgsql.TranslateCypherExpression(regularQuery.SingleQuery.SinglePartQuery.ReadingClauses[0].Match.Where.Expressions[0])
	require.Nil(t, err)
	visualization.MustWritePUML(sqlAST, "/home/zinic/graph.puml")

	extractedAST, err := fold.Extract([]pgsql.Expression{pgsql.Identifier("s"), pgsql.CompoundIdentifier{"s", "properties"}}, sqlAST)
	require.Nil(t, err)

	visualization.MustWritePUML(extractedAST, "/home/zinic/graph.puml")

	output, err := format.FormatExpression(extractedAST)
	require.Nil(t, err)
	require.Equal(t, "s.properties -> 'name' = s.properties -> 'other' + 1 / s.properties -> 'last' and s.properties -> 'value' = 1234 and not s.properties -> 'test'", output.Value)
}
