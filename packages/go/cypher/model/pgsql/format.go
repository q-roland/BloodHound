package pgsql

import (
	"fmt"
	"strconv"
	"strings"
)

type FormattedQuery struct {
	Query      string
	Parameters map[string]any
}

type FormattedQueryBuilder struct {
	QueryBuilder *strings.Builder
	Parameters   map[string]any
}

func NewFormattedQueryBuilder() FormattedQueryBuilder {
	return FormattedQueryBuilder{
		QueryBuilder: &strings.Builder{},
		Parameters:   map[string]any{},
	}
}

func (s FormattedQueryBuilder) Write(values ...any) {
	for _, value := range values {
		switch typedValue := value.(type) {
		case string:
			s.QueryBuilder.WriteString(typedValue)

		case StringLike:
			s.QueryBuilder.WriteString(typedValue.String())

		default:
			panic(fmt.Sprintf("invalid write parameter type: %T", value))
		}
	}
}

func (s FormattedQueryBuilder) Build() FormattedQuery {
	return FormattedQuery{
		Query:      s.QueryBuilder.String(),
		Parameters: s.Parameters,
	}
}

func formatLiteral(formattedQueryBuilder FormattedQueryBuilder, literal Literal) error {
	switch typedLiteral := literal.Value.(type) {
	case uint:
		formattedQueryBuilder.Write(strconv.FormatUint(uint64(typedLiteral), 10))
	case uint8:
		formattedQueryBuilder.Write(strconv.FormatUint(uint64(typedLiteral), 10))
	case uint16:
		formattedQueryBuilder.Write(strconv.FormatUint(uint64(typedLiteral), 10))
	case uint32:
		formattedQueryBuilder.Write(strconv.FormatUint(uint64(typedLiteral), 10))
	case uint64:
		formattedQueryBuilder.Write(strconv.FormatUint(typedLiteral, 10))
	case int:
		formattedQueryBuilder.Write(strconv.FormatInt(int64(typedLiteral), 10))
	case int8:
		formattedQueryBuilder.Write(strconv.FormatInt(int64(typedLiteral), 10))
	case int16:
		formattedQueryBuilder.Write(strconv.FormatInt(int64(typedLiteral), 10))
	case int32:
		formattedQueryBuilder.Write(strconv.FormatInt(int64(typedLiteral), 10))
	case int64:
		formattedQueryBuilder.Write(strconv.FormatInt(typedLiteral, 10))
	case string:
		formattedQueryBuilder.Write("'", typedLiteral, "'")
	default:
		return fmt.Errorf("unsupported literal type: %T", literal.Value)
	}

	return nil
}

func formatExpression(formattedQueryBuilder FormattedQueryBuilder, rootExpr Expression) error {
	exprStack := []Expression{
		rootExpr,
	}

	for len(exprStack) > 0 {
		nextExpr := exprStack[len(exprStack)-1]
		exprStack = exprStack[:len(exprStack)-1]

		switch typedNextExpr := nextExpr.(type) {
		case Wildcard:
			formattedQueryBuilder.Write("* ")

		case Literal:
			if err := formatLiteral(formattedQueryBuilder, typedNextExpr); err != nil {
				return err
			}

			formattedQueryBuilder.Write(" ")

		case Operator:
			formattedQueryBuilder.Write(typedNextExpr, " ")

		case CompoundIdentifier:
			for idx, nextIdentifier := range typedNextExpr {
				if idx > 0 {
					formattedQueryBuilder.Write(".")
				}

				formattedQueryBuilder.Write(nextIdentifier)
			}

			formattedQueryBuilder.Write(" ")

		case BinaryExpression:
			// Push the operands and operator onto the stack in reverse order
			exprStack = append(exprStack, typedNextExpr.RightOperand, typedNextExpr.Operator, typedNextExpr.LeftOperand)
		}
	}

	return nil
}

func formatSelect(formattedQueryBuilder FormattedQueryBuilder, selectStmt Select) error {
	formattedQueryBuilder.Write("select ")

	for _, projection := range selectStmt.Projection {
		if err := formatExpression(formattedQueryBuilder, projection.Expression); err != nil {
			return err
		}
	}

	for _, fromClause := range selectStmt.From {
		formattedQueryBuilder.Write("from ", fromClause.Relation.Name)

		if fromClause.Relation.Binding != nil {
			formattedQueryBuilder.Write(" ", *fromClause.Relation.Binding)
		}

		formattedQueryBuilder.Write(" ")
	}

	if selectStmt.Where != nil {
		formattedQueryBuilder.Write("where ")

		if err := formatExpression(formattedQueryBuilder, selectStmt.Where); err != nil {
			return err
		}
	}

	return nil
}

func FormatQuery(query Query) (FormattedQuery, error) {
	formattedQueryBuilder := NewFormattedQueryBuilder()

	switch typedBody := query.Body.(type) {
	case Select:
		if err := formatSelect(formattedQueryBuilder, typedBody); err != nil {
			return FormattedQuery{}, err
		}

	default:
		return formattedQueryBuilder.Build(), fmt.Errorf("unsupported query body type %T", query.Body)
	}

	return formattedQueryBuilder.Build(), nil
}
