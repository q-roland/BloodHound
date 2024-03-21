package pgsql

type FormattingLiteral string

func (s FormattingLiteral) NodeType() string {
	return "formatting_literal"
}

func (s FormattingLiteral) String() string {
	return string(s)
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

type SyntaxNode interface {
	NodeType() string
}

// SetExpression
// Must resolve to one of the following types: Query, Select, SetOperation, Values
type SetExpression interface {
	SyntaxNode
}

type Values struct {
	Values []Expression
}

type Expression interface {
	SyntaxNode
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
	Value    any
	TypeHint DataType
}

func (l Literal) NodeType() string {
	return "literal"
}

type Subquery struct {
	Query Query
}

// not <expr>
type UnaryExpression struct {
	Operator Expression
	Operand  Expression
}

func (s UnaryExpression) NodeType() string {
	return "unary_expression"
}

// <expr> > <expr>
// table.column > 12345
type BinaryExpression struct {
	LeftOperand  Expression
	Operator     Expression
	RightOperand Expression
}

func (s BinaryExpression) NodeType() string {
	return "BinaryExpression"
}

// (<expr>)
type Parenthetical struct {
	Expression Expression
}

type JoinType int

const (
	JoinTypeInner JoinType = iota
	JoinTypeLeftOuter
	JoinTypeRightOuter
	JoinTypeFullOuter
)

type JoinOperator struct {
	JoinType   JoinType
	Constraint Expression
}

type OrderBy struct {
	Expression Expression
	Ascending  bool
}

type WindowFrameUnit int

const (
	WindowFrameUnitRows WindowFrameUnit = iota
	WindowFrameUnitRange
	WindowFrameUnitGroups
)

type WindowFrameBoundaryType int

const (
	WindowFrameBoundaryTypeCurrentRow WindowFrameBoundaryType = iota
	WindowFrameBoundaryTypePreceding
	WindowFrameBoundaryTypeFollowing
)

type WindowFrameBoundary struct {
	BoundaryType    WindowFrameBoundaryType
	BoundaryLiteral *Literal
}

type WindowFrame struct {
	Unit          WindowFrameUnit
	StartBoundary WindowFrameBoundary
	EndBoundary   *WindowFrameBoundary
}

type Window struct {
	PartitionBy []Expression
	OrderBy     []OrderBy
	WindowFrame *WindowFrame
}

type FunctionCall struct {
	Distinct   bool
	Function   Identifier
	Parameters []Expression
	Over       *Window
}

func (s FunctionCall) NodeType() string {
	return "function_call"
}

type Join struct {
	Table        TableReference
	JoinOperator JoinOperator
}

func (s *Join) NodeType() string {
	return "join"
}

type StringLike interface {
	String() string
}

type Identifier string

func (s Identifier) NodeType() string {
	return "identifier"
}

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
	Joins    []Join
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

type ArrayLiteral struct {
	Values   []Expression
	TypeHint DataType
}

func (s ArrayLiteral) NodeType() string {
	return "array"
}

type Projection interface {
	SyntaxNode
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

func (s SetOperation) NodeType() string {
	return "set_operation"
}

type CommonTableExpression struct {
	Alias TableAlias
	Query Query
}

type CommonTableExpressions struct {
	Recursive   bool
	Expressions []CommonTableExpression
}

type Query struct {
	CommonTableExpressions *CommonTableExpressions
	Body                   SetExpression
}

func (s Query) NodeType() string {
	return "query"
}

func Walk(query *Query, visitor func(node SyntaxNode)) {

}
