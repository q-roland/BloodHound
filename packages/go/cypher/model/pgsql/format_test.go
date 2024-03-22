package pgsql

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFormat_SingleQueries(t *testing.T) {
	query := Query{
		Body: Select{
			Distinct: false,
			Projection: []Projection{
				Wildcard{},
			},
			From: []FromClause{{
				Relation: TableReference{
					Name:    "table",
					Binding: AsOptionalIdentifier("t"),
				},
			}},
			Where: BinaryExpression{
				LeftOperand: CompoundIdentifier{"t", "col1"},
				Operator:    Operator(">"),
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

func TestFormat_CTEs(t *testing.T) {
	query := Query{
		CommonTableExpressions: &CommonTableExpressions{
			Recursive: true,
			Expressions: []CommonTableExpression{{
				Alias: TableAlias{
					Name: "expansion_1",
					Columns: []Identifier{
						"root_id",
						"next_id",
						"depth",
						"stop",
						"is_cycle",
						"path",
					},
				},
				Query: Query{
					Body: SetOperation{
						Operator: "union",
						All:      true,
						LeftOperand: Select{
							Projection: []Projection{
								CompoundIdentifier{"r", "start_id"},
								CompoundIdentifier{"r", "end_id"},
								Literal{
									Value: 1,
								},
								Literal{
									Value: false,
								},
								BinaryExpression{
									LeftOperand:  CompoundIdentifier{"r", "start_id"},
									Operator:     Operator("="),
									RightOperand: CompoundIdentifier{"r", "end_id"},
								},
								ArrayLiteral{
									Values: []Expression{
										CompoundIdentifier{"r", "id"},
									},
								},
							},

							From: []FromClause{{
								Relation: TableReference{
									Name:    "edge",
									Binding: AsOptionalIdentifier("r"),
								},

								Joins: []Join{{
									Table: TableReference{
										Name:    "node",
										Binding: AsOptionalIdentifier("a"),
									},
									JoinOperator: JoinOperator{
										JoinType: JoinTypeInner,
										Constraint: BinaryExpression{
											LeftOperand:  CompoundIdentifier{"a", "id"},
											Operator:     Operator("="),
											RightOperand: CompoundIdentifier{"r", "start_id"},
										},
									},
								}},
							}},

							Where: BinaryExpression{
								LeftOperand: CompoundIdentifier{"a", "kind_ids"},
								Operator: FunctionCall{
									Function: "operator",
									Parameters: []Expression{
										CompoundIdentifier{"pg_catalog", "&&"},
									},
								},
								RightOperand: ArrayLiteral{
									Values: []Expression{
										Literal{
											Value: 23,
										},
									},
									TypeHint: Int2Array,
								},
							},
						},
						RightOperand: Select{
							Projection: []Projection{
								CompoundIdentifier{"expansion_1", "root_id"},
								CompoundIdentifier{"r", "end_id"},
								BinaryExpression{
									LeftOperand: CompoundIdentifier{"expansion_1", "depth"},
									Operator:    Operator("+"),
									RightOperand: Literal{
										Value: 1,
									},
								},
								BinaryExpression{
									LeftOperand: CompoundIdentifier{"b", "kind_ids"},
									Operator: FunctionCall{
										Function: "operator",
										Parameters: []Expression{
											CompoundIdentifier{"pg_catalog", "&&"},
										},
									},
									RightOperand: ArrayLiteral{
										Values: []Expression{
											Literal{
												Value: 24,
											},
										},
										TypeHint: Int2Array,
									},
								},
								BinaryExpression{
									LeftOperand: CompoundIdentifier{"r", "id"},
									Operator:    Operator("="),
									RightOperand: FunctionCall{
										Function: "any",
										Parameters: []Expression{
											CompoundIdentifier{"expansion_1", "path"},
										},
									},
								},
								BinaryExpression{
									LeftOperand:  CompoundIdentifier{"expansion_1", "path"},
									Operator:     Operator("||"),
									RightOperand: CompoundIdentifier{"r", "id"},
								},
							},
							From: []FromClause{{
								Relation: TableReference{
									Name: "expansion_1",
								},
								Joins: []Join{{
									Table: TableReference{
										Name:    "edge",
										Binding: AsOptionalIdentifier("r"),
									},
									JoinOperator: JoinOperator{
										JoinType: JoinTypeInner,
										Constraint: BinaryExpression{
											LeftOperand:  CompoundIdentifier{"r", "start_id"},
											Operator:     Operator("="),
											RightOperand: CompoundIdentifier{"expansion_1", "next_id"},
										},
									},
								}, {
									Table: TableReference{
										Name:    "node",
										Binding: AsOptionalIdentifier("b"),
									},
									JoinOperator: JoinOperator{
										JoinType: JoinTypeInner,
										Constraint: BinaryExpression{
											LeftOperand:  CompoundIdentifier{"b", "id"},
											Operator:     Operator("="),
											RightOperand: CompoundIdentifier{"r", "end_id"},
										},
									},
								}},
							}},
							Where: BinaryExpression{
								LeftOperand: UnaryExpression{
									Operator: Operator("not"),
									Operand:  CompoundIdentifier{"expansion_1", "is_cycle"},
								},
								Operator: Operator("and"),
								RightOperand: UnaryExpression{
									Operator: Operator("not"),
									Operand:  CompoundIdentifier{"expansion_1", "stop"},
								},
							},
						},
					},
				},
			}},
		},
		Body: Select{
			Projection: []Projection{
				CompoundIdentifier{"a", "properties"},
				CompoundIdentifier{"b", "properties"},
			},
			From: []FromClause{{
				Relation: TableReference{
					Name: "expansion_1",
				},
				Joins: []Join{{
					Table: TableReference{
						Name:    "node",
						Binding: AsOptionalIdentifier("a"),
					},
					JoinOperator: JoinOperator{
						JoinType: JoinTypeInner,
						Constraint: BinaryExpression{
							LeftOperand:  CompoundIdentifier{"a", "id"},
							Operator:     Operator("="),
							RightOperand: CompoundIdentifier{"expansion_1", "root_id"},
						},
					},
				}, {
					Table: TableReference{
						Name:    "node",
						Binding: AsOptionalIdentifier("b"),
					},
					JoinOperator: JoinOperator{
						JoinType: JoinTypeInner,
						Constraint: BinaryExpression{
							LeftOperand:  CompoundIdentifier{"b", "id"},
							Operator:     Operator("="),
							RightOperand: CompoundIdentifier{"expansion_1", "next_id"},
						},
					},
				}},
			}},

			Where: BinaryExpression{
				LeftOperand: UnaryExpression{
					Operator: Operator("not"),
					Operand:  CompoundIdentifier{"expansion_1", "is_cycle"},
				},
				Operator:     Operator("and"),
				RightOperand: CompoundIdentifier{"expansion_1", "stop"},
			},
		},
	}

	formattedQuery, err := FormatQuery(query)
	require.Nil(t, err)
	require.Equal(t, "with recursive expansion_1(root_id, next_id, depth, stop, is_cycle, path) as (select r.start_id, r.end_id, 1, false, r.start_id = r.end_id, array[r.id] from edge r join node a on a.id = r.start_id where a.kind_ids operator(pg_catalog.&&) array[23]::int2[] union all select expansion_1.root_id, r.end_id, expansion_1.depth + 1, b.kind_ids operator(pg_catalog.&&) array[24]::int2[], r.id = any(expansion_1.path), expansion_1.path || r.id from expansion_1 join edge r on r.start_id = expansion_1.next_id join node b on b.id = r.end_id where not expansion_1.is_cycle and not expansion_1.stop) select a.properties, b.properties from expansion_1 join node a on a.id = expansion_1.root_id join node b on b.id = expansion_1.next_id where not expansion_1.is_cycle and expansion_1.stop", formattedQuery.Query)
}
