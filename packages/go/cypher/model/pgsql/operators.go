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
