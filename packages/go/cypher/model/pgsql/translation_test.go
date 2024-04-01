package pgsql

import (
	"github.com/specterops/bloodhound/cypher/frontend"
	"github.com/stretchr/testify/require"
	"testing"
)

var testCases = map[string]string{
	"match (s) return s":                       "with s as (select * from node s) select * from s",
	"match (s) where s.name = '1234' return s": "with s as (select * from node s where s.properties -> 'name' = '1234') select * from s",
}

func TestTranslate(t *testing.T) {
	for cypherQuery, expectedSQL := range testCases {
		regularQuery, err := frontend.ParseCypher(frontend.NewContext(), cypherQuery)
		require.Nil(t, err)

		sqlStatement, err := Translate(regularQuery)
		require.Nil(t, err)

		formattedQuery, err := FormatStatement(sqlStatement)
		require.Nil(t, err)

		require.Equalf(t, expectedSQL, formattedQuery.Query, "Test case for cypher query: '%s' failed to match.", cypherQuery)
	}
}

func TestTranslateWhereClause(t *testing.T) {
	//regularQuery, err := frontend.ParseCypher(frontend.NewContext(), "match (s) where s.name = 123 and s.other = 'yes' and not s.bool_value return s")
	regularQuery, err := frontend.ParseCypher(frontend.NewContext(), "match (s) where s.name = s.other + 1 / s.last and s.value = 1234 and not s.test return s")
	require.Nil(t, err)

	sqlAST, err := TranslateCypherExpression(regularQuery.SingleQuery.SinglePartQuery.ReadingClauses[0].Match.Where.Expressions[0])
	require.Nil(t, err)

	extractedAST, err := extractSingle([]Expression{Identifier("s"), CompoundIdentifier{"s", "properties"}}, sqlAST)
	require.Nil(t, err)
	require.NotNil(t, extractedAST)

	sql, err := FormatStatement(Query{
		Body: Select{
			Projection: []Projection{Wildcard{}},
			From: []FromClause{{
				Relation: TableReference{
					Name:    CompoundIdentifier{"node"},
					Binding: AsOptionalIdentifier("s"),
				},
			}},
			Where: sqlAST,
		},
	})

	require.Nil(t, err)
	require.Equal(t, "select * from node s where s.properties -> 'name' = s.properties -> 'other' + 1 / s.properties -> 'last' and s.properties -> 'value' = 1234 and not s.properties -> 'test'", sql.Query)
}
