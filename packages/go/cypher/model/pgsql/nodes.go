package pgsql

type StringLike interface {
	String() string
}

type SyntaxNode interface {
	NodeType() string
}

type Statement interface {
	SyntaxNode
	Statement() Statement
}

type Expression interface {
	SyntaxNode
	Expression() Expression
}

type Projection interface {
	Expression
	Projection() Projection
}

type MergeAction interface {
	Expression
	MergeAction() MergeAction
}

type SetExpression interface {
	Expression
	SetExpression() SetExpression
}

type ConflictAction interface {
	Expression
	ConflictAction() ConflictAction
}
