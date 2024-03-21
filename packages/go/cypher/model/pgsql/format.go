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
	case bool:
		formattedQueryBuilder.Write(strconv.FormatBool(typedLiteral))
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
			formattedQueryBuilder.Write("*")

		case Literal:
			if err := formatLiteral(formattedQueryBuilder, typedNextExpr); err != nil {
				return err
			}

		case FunctionCall:
			exprStack = append(exprStack, FormattingLiteral(")"))

			for idx := len(typedNextExpr.Parameters) - 1; idx >= 0; idx-- {
				exprStack = append(exprStack, typedNextExpr.Parameters[idx])

				if idx > 0 {
					exprStack = append(exprStack, FormattingLiteral(","))
				}
			}

			if typedNextExpr.Distinct {
				exprStack = append(exprStack, FormattingLiteral("distinct "))
			}

			exprStack = append(exprStack, FormattingLiteral("("))
			exprStack = append(exprStack, typedNextExpr.Function)

		case Operator:
			formattedQueryBuilder.Write(typedNextExpr)

		case Identifier:
			formattedQueryBuilder.Write(typedNextExpr)

		case CompoundIdentifier:
			for idx := len(typedNextExpr) - 1; idx >= 0; idx-- {
				exprStack = append(exprStack, typedNextExpr[idx])

				if idx > 0 {
					exprStack = append(exprStack, FormattingLiteral("."))
				}
			}

		case FormattingLiteral:
			formattedQueryBuilder.Write(typedNextExpr)

		case UnaryExpression:
			exprStack = append(exprStack,
				typedNextExpr.Operand,
				FormattingLiteral(" "),
				typedNextExpr.Operator,
			)

		case BinaryExpression:
			// Push the operands and operator onto the stack in reverse order
			exprStack = append(exprStack,
				typedNextExpr.RightOperand,
				FormattingLiteral(" "),
				typedNextExpr.Operator,
				FormattingLiteral(" "),
				typedNextExpr.LeftOperand,
			)

		case ArrayLiteral:
			formattedQueryBuilder.Write("array[")

			if typedNextExpr.TypeHint != UnsetDataType {
				exprStack = append(exprStack, FormattingLiteral(typedNextExpr.TypeHint.String()), FormattingLiteral("::"))
			}

			exprStack = append(exprStack, FormattingLiteral("]"))

			for idx := len(typedNextExpr.Values) - 1; idx >= 0; idx-- {
				exprStack = append(exprStack, typedNextExpr.Values[idx])

				if idx > 0 {
					exprStack = append(exprStack, FormattingLiteral(", "))
				}
			}

		default:
			return fmt.Errorf("unsupported expression type: %T", nextExpr)
		}
	}

	return nil
}

func formatSelect(formattedQueryBuilder FormattedQueryBuilder, selectStmt Select) error {
	formattedQueryBuilder.Write("select ")

	for idx, projection := range selectStmt.Projection {
		if idx > 0 {
			formattedQueryBuilder.Write(", ")
		}

		if err := formatExpression(formattedQueryBuilder, projection); err != nil {
			return err
		}
	}

	formattedQueryBuilder.Write(" ")

	if len(selectStmt.From) > 0 {
		formattedQueryBuilder.Write("from ")

		for idx, fromClause := range selectStmt.From {
			if idx > 0 {
				formattedQueryBuilder.Write(", ")
			}

			formattedQueryBuilder.Write(fromClause.Relation.Name)

			if fromClause.Relation.Binding != nil {
				formattedQueryBuilder.Write(" ", *fromClause.Relation.Binding)
			}

			formattedQueryBuilder.Write(" ")

			if len(fromClause.Joins) > 0 {
				for idx, join := range fromClause.Joins {
					if idx > 0 {
						formattedQueryBuilder.Write(" ")
					}

					switch join.JoinOperator.JoinType {
					case JoinTypeInner:
						// A bare join keyword is also an alias for an inner join

					case JoinTypeLeftOuter:
						formattedQueryBuilder.Write("left outer ")

					case JoinTypeRightOuter:
						formattedQueryBuilder.Write("right outer ")

					case JoinTypeFullOuter:
						formattedQueryBuilder.Write("full outer ")

					default:
						return fmt.Errorf("unsupported join type: %d", join.JoinOperator.JoinType)
					}

					formattedQueryBuilder.Write("join ", join.Table.Name, " ")

					if join.Table.Binding != nil {
						formattedQueryBuilder.Write(*join.Table.Binding, " ")
					}

					formattedQueryBuilder.Write("on ")

					if err := formatExpression(formattedQueryBuilder, join.JoinOperator.Constraint); err != nil {
						return err
					}
				}
			}
		}
	}

	if selectStmt.Where != nil {
		formattedQueryBuilder.Write(" where ")

		if err := formatExpression(formattedQueryBuilder, selectStmt.Where); err != nil {
			return err
		}
	}

	return nil
}

func formatTableAlias(formattedQueryBuilder FormattedQueryBuilder, tableAlias TableAlias) error {
	formattedQueryBuilder.Write(tableAlias.Name)

	if len(tableAlias.Columns) > 0 {
		formattedQueryBuilder.Write("(")

		for idx, column := range tableAlias.Columns {
			if idx > 0 {
				formattedQueryBuilder.Write(", ")
			}

			if err := formatExpression(formattedQueryBuilder, column); err != nil {
				return err
			}
		}

		formattedQueryBuilder.Write(")")
	}

	return nil
}

func formatCommonTableExpressions(formattedQueryBuilder FormattedQueryBuilder, commonTableExpressions CommonTableExpressions) error {
	formattedQueryBuilder.Write("with ")

	if commonTableExpressions.Recursive {
		formattedQueryBuilder.Write("recursive ")
	}

	for idx, commonTableExpression := range commonTableExpressions.Expressions {
		if idx > 0 {
			formattedQueryBuilder.Write(", ")
		}

		if err := formatTableAlias(formattedQueryBuilder, commonTableExpression.Alias); err != nil {
			return err
		}

		formattedQueryBuilder.Write(" as (")

		if err := formatSetExpression(formattedQueryBuilder, commonTableExpression.Query); err != nil {
			return err
		}

		formattedQueryBuilder.Write(")")
	}

	// Leave a trailing space after formatting CTEs for the subsequent query body
	formattedQueryBuilder.Write(" ")

	return nil
}

func formatSetExpression(formattedQueryBuilder FormattedQueryBuilder, expression SetExpression) error {
	switch typedSetExpression := expression.(type) {
	case Query:
		if typedSetExpression.CommonTableExpressions != nil {
			if err := formatCommonTableExpressions(formattedQueryBuilder, *typedSetExpression.CommonTableExpressions); err != nil {
				return err
			}
		}

		return formatSetExpression(formattedQueryBuilder, typedSetExpression.Body)

	case Select:
		return formatSelect(formattedQueryBuilder, typedSetExpression)

	case SetOperation:
		if typedSetExpression.All && typedSetExpression.Distinct {
			return fmt.Errorf("set operation for query may not be both ALL and DISTINCT")
		}

		if err := formatSetExpression(formattedQueryBuilder, typedSetExpression.LeftOperand); err != nil {
			return err
		}

		formattedQueryBuilder.Write(" ")

		if err := formatExpression(formattedQueryBuilder, typedSetExpression.Operator); err != nil {
			return err
		}

		formattedQueryBuilder.Write(" ")

		if typedSetExpression.All {
			formattedQueryBuilder.Write("all ")
		}

		if typedSetExpression.Distinct {
			formattedQueryBuilder.Write("distinct ")
		}

		if err := formatSetExpression(formattedQueryBuilder, typedSetExpression.RightOperand); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unsupported set expression type %T", expression)
	}

	return nil
}

func FormatQuery(query Query) (FormattedQuery, error) {
	formattedQueryBuilder := NewFormattedQueryBuilder()

	if err := formatSetExpression(formattedQueryBuilder, query); err != nil {
		return FormattedQuery{}, err
	}

	return formattedQueryBuilder.Build(), nil
}
