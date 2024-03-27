package pgsql

import (
	"github.com/specterops/bloodhound/cypher/frontend"
	"github.com/specterops/bloodhound/cypher/model"
	"github.com/stretchr/testify/require"
	"testing"
)

var testCases = map[string]string{
	"match (s) return s":                       "with s as (select * from node s) select * from s",
	"match (s) where s.name = '1234' return s": "",
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
	regularQuery, err := frontend.ParseCypher(frontend.NewContext(), "match (s) where s.name = s.other + 123 and s.other = 'yes' and s.action = 'yeet' and not s.reject return s")
	require.Nil(t, err)

	_, err = ctbe(regularQuery.SingleQuery.SinglePartQuery.ReadingClauses[0].Match.Where.Expressions[0].(*model.Conjunction))
	require.Nil(t, err)
}
