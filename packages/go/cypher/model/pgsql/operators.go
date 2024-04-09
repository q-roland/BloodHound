package pgsql

type Operator string

func (s Operator) Expression() Expression {
	return s
}

func (s Operator) String() string {
	return string(s)
}

func (s Operator) NodeType() string {
	return "operator"
}

const (
	OperatorEquals           Operator = "="
	OperatorAnd           Operator = "and"
	OperatorOr            Operator = "or"
	OperatorNot           Operator = "not"
	OperatorJSONField     Operator = "->"
	OperatorJSONTextField Operator = "->>"
)
