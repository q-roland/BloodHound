package pgsql_test

import (
	"github.com/specterops/bloodhound/cypher/frontend"
	"github.com/specterops/bloodhound/cypher/model/pgsql"
	"github.com/specterops/bloodhound/cypher/model/pgsql/format"
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

		sqlStatement, err := pgsql.Translate(regularQuery)
		require.Nil(t, err)

		formattedQuery, err := format.FormatStatement(sqlStatement)
		require.Nil(t, err)

		require.Equalf(t, expectedSQL, formattedQuery.Value, "Test case for cypher query: '%s' failed to match.", cypherQuery)
	}
}

func TestTranslateCypherExpression(t *testing.T) {
	regularQuery, err := frontend.ParseCypher(frontend.NewContext(), "match (s), (e) where s.name = s.other + 1 / s.last and s.value = 1234 and not s.test and e.value = 1234 and e.comp = s.comp return s")
	require.Nil(t, err)

	sqlAST, err := pgsql.TranslateCypherExpression(regularQuery.SingleQuery.SinglePartQuery.ReadingClauses[0].Match.Where.Expressions[0])
	require.Nil(t, err)

	output, err := format.FormatExpression(sqlAST)
	require.Nil(t, err)
	require.Equal(t, "s.properties -> 'name' = s.properties -> 'other' + 1 / s.properties -> 'last' and s.properties -> 'value' = 1234 and not s.properties -> 'test' and e.properties -> 'value' = 1234 and e.properties -> 'comp' = s.properties -> 'comp'", output.Value)
}

func TestTranslateWhereClause(t *testing.T) {
	//regularQuery, err := frontend.ParseCypher(frontend.NewContext(), "match (s) where s.name = 123 and s.other = 'yes' and not s.bool_value return s")

	//
	//sql, err := FormatStatement(Query{
	//	Body: Select{
	//		Projection: []Projection{Wildcard{}},
	//		From: []FromClause{{
	//			Relation: TableReference{
	//				Name:    CompoundIdentifier{"node"},
	//				Binding: AsOptionalIdentifier("s"),
	//			},
	//		}},
	//		Where: sqlAST,
	//	},
	//})
	//
	//require.Nil(t, err)
	//require.Equal(t, "select * from node s where s.properties -> 'name' = s.properties -> 'other' + 1 / s.properties -> 'last' and s.properties -> 'value' = 1234 and not s.properties -> 'test'", sql.Query)
}
