package pgsql

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFormat_All(t *testing.T) {
	// select * from table t where t.col1 > 1
	query := Query{
		Body: Select{
			Distinct: false,
			Projection: []Projection{{
				Expression: Wildcard{},
			}},
			From: []FromClause{{
				Relation: TableReference{
					Name:    "table",
					Binding: AsOptionalIdentifier("t"),
				},
			}},
			Where: BinaryExpression{
				LeftOperand: CompoundIdentifier{"t", "col1"},
				Operator:    ">",
				RightOperand: Literal{
					Value: 1,
				},
			},
		},
	}

	formattedQuery, err := FormatQuery(query)
	require.Nil(t, err)
	require.Equal(t, "select * from table t where t.col1 > 1", formattedQuery.Query)
}
