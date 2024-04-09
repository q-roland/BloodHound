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
	"match ()-[r]->() return r":                "with r as (select * from edge r) select * from r",
	"match (n), ()-[r]->() return n, r":        "with n as (select * from node n), r as (select * from edge r) select * from n, r",
	"match ()-[r]->(), ()-[e]->() return r, e": "with r as (select * from edge r), e as (select * from edge e) select * from r, e",

	//
	"match ()-[r]->() where r.value = 42 return r":                                                           "with r as (select * from edge r where r.properties -> 'value' = 42) select * from r",
	"match (n)-[r]->() where n.name = '123' return n, r":                                                     "with n as (select * from node n where n.properties -> 'name' = '123'), r as (select * from edge r, n where n.id = r.start_id) select * from n, r",
	"match (s)-[r]->(e) where s.name = '123' and e.name = '321' return s, r, e":                              "with s as (select * from node s where s.properties -> 'name' = '123'), r as (select * from edge r, s where s.id = r.start_id), e as (select * from node e, r where e.id = r.end_id and e.properties -> 'name' = '321') select * from s, r, e",
	"match (f), (s)-[r]->(e) where not f.bool_field and s.name = '123' and e.name = '321' return f, s, r, e": "with f as (select * from node f where not f.properties -> 'bool_field'), s as (select * from node s where s.properties -> 'name' = '123'), r as (select * from edge r, s where s.id = r.start_id), e as (select * from node e, r where e.id = r.end_id and e.properties -> 'name' = '321') select * from f, s, r, e",

	// TODO: Stopped here
	"match p = ()-[]->() return p": "",
}

func TestTranslate(t *testing.T) {
	for cypherQuery, expectedSQL := range testCases {
		if regularQuery, err := frontend.ParseCypher(frontend.NewContext(), cypherQuery); err != nil {
			t.Fatalf("Failed to compile cypher query: %s - %v", cypherQuery, err)
		} else if sqlStatement, err := translate.Translate(regularQuery); err != nil {
			t.Fatalf("Failed to translate cypher query: %s - %v", cypherQuery, err)
		} else if formattedQuery, err := format.FormatStatement(sqlStatement); err != nil {
			t.Fatalf("Failed to format SQL query: %v", err)
		} else {
			require.Equalf(t, expectedSQL, formattedQuery.Value, "Test case for cypher query: '%s' failed to match.", cypherQuery)
		}
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
