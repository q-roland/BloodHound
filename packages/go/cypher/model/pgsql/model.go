package pgsql

type FormattingLiteral string

func (s FormattingLiteral) Expression() Expression {
	return s
}

func (s FormattingLiteral) NodeType() string {
	return "formatting_literal"
}

func (s FormattingLiteral) String() string {
	return string(s)
}

type TableAlias struct {
	Name    Identifier
	Columns []Identifier
}

type Values struct {
	Values []Expression
}

func (s Values) Expression() Expression {
	return s
}

func (s Values) SetExpression() SetExpression {
	return s
}

func (s Values) NodeType() string {
	return "values"
}

type Case struct {
	Operand    Expression
	Conditions []Expression
	Then       []Expression
	Else       Expression
}

// [not] exists(<query>)
type Exists struct {
	Query   Query
	Negated bool
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
	Null     bool
	TypeHint DataType
}

func (s Literal) Expression() Expression {
	return s
}

func (s Literal) Projection() Projection {
	return s
}

func AsLiteral(value any) Literal {
	return Literal{
		Value: value,
	}
}

func (s Literal) NodeType() string {
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

func (s UnaryExpression) Expression() Expression {
	return s
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

func (s BinaryExpression) Expression() Expression {
	return s
}

func (s BinaryExpression) Projection() Projection {
	return s
}

func (s BinaryExpression) NodeType() string {
	return "binary_expression"
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

func (s FunctionCall) Expression() Expression {
	return s
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

type Identifier string

func (s Identifier) Projection() Projection {
	return s
}

func (s Identifier) Expression() Expression {
	return s
}

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

func (s CompoundIdentifier) Strings() []string {
	strCopy := make([]string, len(s))

	for idx, identifier := range s {
		strCopy[idx] = identifier.String()
	}

	return strCopy
}

func (s CompoundIdentifier) Copy() CompoundIdentifier {
	copyInst := make(CompoundIdentifier, len(s))
	copy(copyInst, s)

	return copyInst
}

func (s CompoundIdentifier) Expression() Expression {
	return s
}

func (s CompoundIdentifier) Projection() Projection {
	return s
}

func (s CompoundIdentifier) NodeType() string {
	return "compound_identifier"
}

type TableReference struct {
	Name    CompoundIdentifier
	Binding OptionalIdentifier
}

func (s TableReference) Expression() Expression {
	return s
}

func (s TableReference) NodeType() string {
	return "table_reference"
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

func (s Wildcard) Expression() Expression {
	return s
}

func (s Wildcard) Projection() Projection {
	return s
}

func (s Wildcard) NodeType() string {
	return "wildcard"
}

type QualifiedWildcard struct {
	Qualifier string
}

type ArrayLiteral struct {
	Values   []Expression
	TypeHint DataType
}

func (s ArrayLiteral) Expression() Expression {
	return s
}

func (s ArrayLiteral) Projection() Projection {
	return s
}

func (s ArrayLiteral) NodeType() string {
	return "array"
}

type MatchedUpdate struct {
	Predicate   Expression
	Assignments []Assignment
}

func (s MatchedUpdate) NodeType() string {
	return "matched_update"
}

func (s MatchedUpdate) Expression() Expression {
	return s
}

func (s MatchedUpdate) MergeAction() MergeAction {
	return s
}

type MatchedDelete struct {
	Predicate Expression
}

func (s MatchedDelete) NodeType() string {
	return "matched_delete"
}

func (s MatchedDelete) Expression() Expression {
	return s
}

func (s MatchedDelete) MergeAction() MergeAction {
	return s
}

type UnmatchedAction struct {
	Predicate Expression
	Columns   []Identifier
	Values    Values
}

func (s UnmatchedAction) NodeType() string {
	return "unmatched_action"
}

func (s UnmatchedAction) Expression() Expression {
	return s
}

func (s UnmatchedAction) MergeAction() MergeAction {
	return s
}

type Merge struct {
	Into       bool
	Table      TableReference
	Source     TableReference
	JoinTarget Expression
	Actions    []MergeAction
}

func (s Merge) NodeType() string {
	return "merge"
}

func (s Merge) Statement() Statement {
	return s
}

type ConflictTarget struct {
	Columns    []Identifier
	Constraint CompoundIdentifier
}

func (s ConflictTarget) NodeType() string {
	return "conflict_target"
}

func (s ConflictTarget) Expression() Expression {
	return s
}

type DoNothing struct{}

type DoUpdate struct {
	Assignments []Assignment
	Where       Expression
}

func (s DoUpdate) NodeType() string {
	return "do_update"
}

func (s DoUpdate) Expression() Expression {
	return s
}

func (s DoUpdate) ConflictAction() ConflictAction {
	return s
}

type OnConflict struct {
	Target *ConflictTarget
	Action ConflictAction
}

func (s OnConflict) NodeType() string {
	return "on_conflict"
}

func (s OnConflict) Expression() Expression {
	return s
}

type Insert struct {
	Table      CompoundIdentifier
	Columns    []Identifier
	OnConflict *OnConflict
	Source     *Query
	Returning  []Projection
}

func (s Insert) Statement() Statement {
	return s
}

func (s Insert) NodeType() string {
	return "insert"
}

// <identifier> = <value>
type Assignment struct {
	Identifier Identifier
	Value      Expression
}

func (s Assignment) Expression() Expression {
	return s
}

func (s Assignment) NodeType() string {
	return "assignment"
}

type Delete struct {
	Table TableReference
	Where Expression
}

func (s Delete) Statement() Statement {
	return s
}

func (s Delete) NodeType() string {
	return "delete"
}

type Update struct {
	Table       TableReference
	Assignments []Assignment
	Where       Expression
}

func (s Update) Statement() Statement {
	return s
}

func (s Update) NodeType() string {
	return "update"
}

type Select struct {
	Distinct   bool
	Projection []Projection
	From       []FromClause
	Where      Expression
	GroupBy    []Expression
	Having     Expression
}

func (s Select) Expression() Expression {
	return s
}

func (s Select) SetExpression() SetExpression {
	return s
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

func (s SetOperation) Expression() Expression {
	return s
}

func (s SetOperation) SetExpression() SetExpression {
	return s
}

func (s SetOperation) NodeType() string {
	return "set_operation"
}

type CommonTableExpression struct {
	Alias        TableAlias
	Materialized *Materialized
	Query        Query
}

type Materialized struct {
	Materialized bool
}

func (s Materialized) Expression() Expression {
	return s
}

func (s Materialized) SetExpression() SetExpression {
	return s
}

func (s Materialized) NodeType() string {
	return "materialized"
}

type With struct {
	Recursive   bool
	Expressions []CommonTableExpression
}

// [with <CTE>] select * from table;
type Query struct {
	CommonTableExpressions *With
	Body                   SetExpression
}

func (s Query) Expression() Expression {
	return s
}

func (s Query) SetExpression() SetExpression {
	return s
}

func (s Query) Statement() Statement {
	return s
}

func (s Query) NodeType() string {
	return "query"
}

func Walk(query *Query, visitor func(node SyntaxNode)) {

}
