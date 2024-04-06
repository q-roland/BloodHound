package fold_test

import (
	"github.com/specterops/bloodhound/cypher/frontend"
	"github.com/specterops/bloodhound/cypher/model/pgsql"
	"github.com/specterops/bloodhound/cypher/model/pgsql/fold"
	"github.com/specterops/bloodhound/cypher/model/pgsql/format"
	"github.com/stretchr/testify/require"
	"testing"
)

type TestCase struct {
	Cypher                 string
	ExpectedSQLExpressions []string
}

func TestExtract(t *testing.T) {
	testCases := []TestCase{{
		Cypher: "match (s), (e) where s.name = 'test' and e.name = 'test' and s.value = e.value return s, e",
		ExpectedSQLExpressions: []string{
			"s.properties -> 'name' = 'test'",
			"e.properties -> 'name' = 'test'",
			"s.properties -> 'value' = e.properties -> 'value'",
		},
	}}

	for _, testCase := range testCases {
		regularQuery, err := frontend.ParseCypher(frontend.NewContext(), testCase.Cypher)
		require.Nil(t, err)

		sqlAST, err := pgsql.TranslateCypherExpression(regularQuery.SingleQuery.SinglePartQuery.ReadingClauses[0].Match.Where.Expressions[0])
		require.Nil(t, err)

		conjoinedConstraintsByKey, err := fold.FragmentExpressionTree([]pgsql.Expression{pgsql.Identifier("s"), pgsql.CompoundIdentifier{"s", "properties"}}, sqlAST)
		require.Nil(t, err)

		for _, conjoinedConstraints := range conjoinedConstraintsByKey {
			formatted, err := format.FormatExpression(conjoinedConstraints.Root())
			require.Nil(t, err)

			matches := false
			for _, matcher := range testCase.ExpectedSQLExpressions {
				if formatted.Value == matcher {
					matches = true
					break
				}
			}

			if !matches {
				t.Fatalf("Unable to match formatted expression: \"%s\" for test case.", formatted.Value)
			}
		}
	}
}
