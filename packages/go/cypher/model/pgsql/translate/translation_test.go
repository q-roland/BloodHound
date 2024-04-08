package translate_test

import (
	"github.com/specterops/bloodhound/cypher/frontend"
	"github.com/specterops/bloodhound/cypher/model/pgsql/format"
	"github.com/specterops/bloodhound/cypher/model/pgsql/translate"
	"github.com/stretchr/testify/require"
	"testing"
)

var testCases = map[string]string{
	"match (s) return s":                                   "with s as (select * from node s) select * from s",
	"match (s) where s.name = '1234' return s":             "with s as (select * from node s where s.properties -> 'name' = '1234') select * from s",
	"match (s), (e) where s.name = '1234' return s":        "with s as (select * from node s where s.properties -> 'name' = '1234'), e as (select * from node e) select * from s",
	"match (s:A), (e:B) where s.name = e.name return s, e": "with s as (select * from node s), e as (select * from node e, s where s.properties -> 'name' = e.properties -> 'name') select * from s, e",

	"match (s), (e) where s.name = '1234' and e.other = 1234 return s": "with s as (select * from node s where s.properties -> 'name' = '1234'), e as (select * from node e where e.properties -> 'other' = 1234) select * from s",
	//
	"match (n), (k) where n.name = '1234' and k.name = '1234' match (e) where e.name = n.name return k, e": "with n as (select * from node n where n.properties -> 'name' = '1234'), k as (select * from node k where k.properties -> 'name' = '1234'), e as (select * from node e, n where e.properties -> 'name' = n.properties -> 'name') select * from k, e",

	//

}

func TestTranslate(t *testing.T) {
	for cypherQuery, expectedSQL := range testCases {
		regularQuery, err := frontend.ParseCypher(frontend.NewContext(), cypherQuery)
		require.Nil(t, err)

		sqlStatement, err := translate.Translate(regularQuery)
		require.Nil(t, err)

		formattedQuery, err := format.FormatStatement(sqlStatement)
		require.Nil(t, err)

		require.Equalf(t, expectedSQL, formattedQuery.Value, "Test case for cypher query: '%s' failed to match.", cypherQuery)
	}
}

func TestTranslateCypherExpression(t *testing.T) {
	regularQuery, err := frontend.ParseCypher(frontend.NewContext(), "match p = (s)-[r:RT1*..]->(e) where s.name = '1234' and e.selected and e.end_id in r.eligible return p")
	require.Nil(t, err)

	sqlAST, err := translate.Translate(regularQuery)
	require.Nil(t, err)

	output, err := format.FormatStatement(sqlAST)
	require.Nil(t, err)
	require.Equal(t, "s.properties -> 'name' = s.properties -> 'other' + 1 / s.properties -> 'last' and s.properties -> 'value' = 1234 and not s.properties -> 'test' and e.properties -> 'value' = 1234 and e.properties -> 'comp' = s.properties -> 'comp' or e.properties -> 'comp' and s.properties -> 'other_property' = '1234'", output.Value)
}
