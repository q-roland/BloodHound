package pgsql

import (
	"github.com/specterops/bloodhound/cypher/model"
)

type CriteriaBuilder struct {
	Root  model.Expression
	Stack []model.Expression
}

func (s *CriteriaBuilder) Push(expression model.Expression) {
	switch typedCursor := s.Stack[len(s.Stack)-1].(type) {
	case model.ExpressionList:
		typedCursor.Add(expression)
	}

	switch expression.(type) {
	case model.ExpressionList:
		s.Stack = append(s.Stack, expression)
	}
}

func (s *CriteriaBuilder) Pop() {
	s.Stack = s.Stack[:len(s.Stack)-1]
}

type InitialCriteria struct {
}

type Operator string

func (s Operator) String() string {
	return string(s)
}

func (s Operator) NodeType() string {
	return "operator"
}

type TableAlias struct {
	Name    Identifier
	Columns []Identifier
}

type Node interface {
	NodeType() string
}

// SetExpression
// Must resolve to one of the following types: Query, Select, SetOperation, Values
type SetExpression interface {
	Node
}

type Values struct {
	Values []Expression
}

type Expression interface {
	Node
}

// exists(<query>)
type Exists struct {
	Query Query
}

// [not] in (val1, val2, ...)
type InExpression struct {
	Expression Expression
	List       []Expression
	Negated    bool
}

// [not] in (<Select> ...)
type InSubquery struct {
	Expression Expression
	Query      Query
	Negated    bool
}

// <expr> [not] between <low> and <high>
type Between struct {
	Expression Expression
	Low        Expression
	High       Expression
	Negated    bool
}

type Literal struct {
	Value any
}

func (l Literal) NodeType() string {
	return "literal"
}

type Subquery struct {
	Query Query
}

// not <expr>
type UnaryExpression struct {
	Operator Operator
	Operand  Expression
}

// <expr> > <expr>
// table.column > 12345
type BinaryExpression struct {
	LeftOperand  Expression
	Operator     Operator
	RightOperand Expression
}

func (s BinaryExpression) NodeType() string {
	return "BinaryExpression"
}

// (<expr>)
type Parenthetical struct {
	Expression Expression
}

type JoinType string

type Join struct {
	Table     TableReference
	Condition model.Expression
}

func (s *Join) NodeType() string {
	return "join"
}

type StringLike interface {
	String() string
}

type Identifier string

func (s Identifier) String() string {
	return string(s)
}

type OptionalIdentifier *Identifier

func AsOptionalIdentifier(val Identifier) OptionalIdentifier {
	return &val
}

type CompoundIdentifier []Identifier

func (c CompoundIdentifier) NodeType() string {
	return "compound_identifier"
}

type TableReference struct {
	Name    Identifier
	Binding OptionalIdentifier
}

type FromClause struct {
	Relation TableReference
	Joins    *[]Join
}

type AliasedExpression struct {
	Expression Expression
	Binding    Identifier
}

type Wildcard struct{}

func (w Wildcard) NodeType() string {
	return "wildcard"
}

type QualifiedWildcard struct {
	Qualifier string
}

type Projection struct {
	Expression Expression
}

type Select struct {
	Distinct   bool
	Projection []Projection
	From       []FromClause
	Where      Expression
	GroupBy    []Expression
	Having     Expression
}

func (s Select) NodeType() string {
	return "select"
}

type SetOperation struct {
	Operator     Operator
	LeftOperand  SetExpression
	RightOperand SetExpression
	All          bool
	Distinct     bool
}

type CTE struct {
	Recursive bool
	Alias     TableAlias
	Query     Query
}

type Query struct {
	CTEs []*CTE
	Body SetExpression
}

func Walk(query *Query, visitor func(node Node)) {

}
