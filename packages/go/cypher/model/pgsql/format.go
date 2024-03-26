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

func formatLiteral(builder FormattedQueryBuilder, literal Literal) error {
	switch typedLiteral := literal.Value.(type) {
	case uint:
		builder.Write(strconv.FormatUint(uint64(typedLiteral), 10))
	case uint8:
		builder.Write(strconv.FormatUint(uint64(typedLiteral), 10))
	case uint16:
		builder.Write(strconv.FormatUint(uint64(typedLiteral), 10))
	case uint32:
		builder.Write(strconv.FormatUint(uint64(typedLiteral), 10))
	case uint64:
		builder.Write(strconv.FormatUint(typedLiteral, 10))
	case int:
		builder.Write(strconv.FormatInt(int64(typedLiteral), 10))
	case int8:
		builder.Write(strconv.FormatInt(int64(typedLiteral), 10))
	case int16:
		builder.Write(strconv.FormatInt(int64(typedLiteral), 10))
	case int32:
		builder.Write(strconv.FormatInt(int64(typedLiteral), 10))
	case int64:
		builder.Write(strconv.FormatInt(typedLiteral, 10))
	case string:
		builder.Write("'", typedLiteral, "'")
	case bool:
		builder.Write(strconv.FormatBool(typedLiteral))
	default:
		return fmt.Errorf("unsupported literal type: %T", literal.Value)
	}

	return nil
}

func formatExpression(builder FormattedQueryBuilder, rootExpr Expression) error {
	exprStack := []SyntaxNode{
		rootExpr,
	}

	for len(exprStack) > 0 {
		nextExpr := exprStack[len(exprStack)-1]
		exprStack = exprStack[:len(exprStack)-1]

		switch typedNextExpr := nextExpr.(type) {
		case Wildcard:
			builder.Write("*")

		case Literal:
			if err := formatLiteral(builder, typedNextExpr); err != nil {
				return err
			}

		case Materialized:
			if typedNextExpr.Materialized {
				exprStack = append(exprStack, FormattingLiteral("materialized"))
			} else {
				exprStack = append(exprStack, FormattingLiteral("not materialized"))
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
			builder.Write(typedNextExpr)

		case Identifier:
			builder.Write(typedNextExpr)

		case CompoundIdentifier:
			for idx := len(typedNextExpr) - 1; idx >= 0; idx-- {
				exprStack = append(exprStack, typedNextExpr[idx])

				if idx > 0 {
					exprStack = append(exprStack, FormattingLiteral("."))
				}
			}

		case FormattingLiteral:
			builder.Write(typedNextExpr)

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

		case TableReference:
			if typedNextExpr.Binding != nil {
				exprStack = append(exprStack, *typedNextExpr.Binding, FormattingLiteral(" "))
			}

			exprStack = append(exprStack, typedNextExpr.Name)

		case Assignment:
			exprStack = append(exprStack,
				typedNextExpr.Value,
				FormattingLiteral(" "),
				Operator("="),
				FormattingLiteral(" "),
				typedNextExpr.Identifier)

		case Values:
			exprStack = append(exprStack, FormattingLiteral(")"))

			for idx := len(typedNextExpr.Values) - 1; idx >= 0; idx-- {
				exprStack = append(exprStack, typedNextExpr.Values[idx])

				if idx > 0 {
					exprStack = append(exprStack, FormattingLiteral(", "))
				}
			}

			exprStack = append(exprStack, FormattingLiteral("values ("))

		case ArrayLiteral:
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

			exprStack = append(exprStack, FormattingLiteral("array["))

		case DoUpdate:
			if typedNextExpr.Where != nil {
				exprStack = append(exprStack, typedNextExpr.Where)
				exprStack = append(exprStack, FormattingLiteral(" where "))
			}

			if len(typedNextExpr.Assignments) > 0 {
				for idx := len(typedNextExpr.Assignments) - 1; idx >= 0; idx-- {
					exprStack = append(exprStack, typedNextExpr.Assignments[idx])
				}

				exprStack = append(exprStack, FormattingLiteral(" set "))
			}

			exprStack = append(exprStack, FormattingLiteral("do update"))

		case ConflictTarget:
			if len(typedNextExpr.Columns) > 0 {
				if len(typedNextExpr.Constraint) > 0 {
					return fmt.Errorf("conflict target has both columns and an 'on constraint' expression set")
				}

				exprStack = append(exprStack, FormattingLiteral(")"))

				for idx := len(typedNextExpr.Columns) - 1; idx >= 0; idx-- {
					exprStack = append(exprStack, typedNextExpr.Columns[idx])

					if idx > 0 {
						exprStack = append(exprStack, FormattingLiteral(", "))
					}
				}

				exprStack = append(exprStack, FormattingLiteral("("))
			}

			if len(typedNextExpr.Constraint) > 0 {
				if len(typedNextExpr.Columns) > 0 {
					return fmt.Errorf("conflict target has both columns and an 'on constraint' expression set")
				}

				exprStack = append(exprStack, typedNextExpr.Constraint, FormattingLiteral("on constraint "))
			}

		default:
			return fmt.Errorf("unsupported expression type: %T", nextExpr)
		}
	}

	return nil
}

func formatSelect(builder FormattedQueryBuilder, selectStmt Select) error {
	builder.Write("select ")

	for idx, projection := range selectStmt.Projection {
		if idx > 0 {
			builder.Write(", ")
		}

		if err := formatExpression(builder, projection); err != nil {
			return err
		}
	}

	builder.Write(" ")

	if len(selectStmt.From) > 0 {
		builder.Write("from ")

		for idx, fromClause := range selectStmt.From {
			if idx > 0 {
				builder.Write(", ")
			}

			if err := formatExpression(builder, fromClause.Relation); err != nil {
				return err
			}

			if len(fromClause.Joins) > 0 {
				builder.Write(" ")

				for idx, join := range fromClause.Joins {
					if idx > 0 {
						builder.Write(" ")
					}

					switch join.JoinOperator.JoinType {
					case JoinTypeInner:
						// A bare join keyword is also an alias for an inner join

					case JoinTypeLeftOuter:
						builder.Write("left outer ")

					case JoinTypeRightOuter:
						builder.Write("right outer ")

					case JoinTypeFullOuter:
						builder.Write("full outer ")

					default:
						return fmt.Errorf("unsupported join type: %d", join.JoinOperator.JoinType)
					}

					builder.Write("join ")

					if err := formatExpression(builder, join.Table); err != nil {
						return err
					}

					builder.Write(" on ")

					if err := formatExpression(builder, join.JoinOperator.Constraint); err != nil {
						return err
					}
				}
			}
		}
	}

	if selectStmt.Where != nil {
		builder.Write(" where ")

		if err := formatExpression(builder, selectStmt.Where); err != nil {
			return err
		}
	}

	return nil
}

func formatTableAlias(builder FormattedQueryBuilder, tableAlias TableAlias) error {
	builder.Write(tableAlias.Name)

	if len(tableAlias.Columns) > 0 {
		builder.Write("(")

		for idx, column := range tableAlias.Columns {
			if idx > 0 {
				builder.Write(", ")
			}

			if err := formatExpression(builder, column); err != nil {
				return err
			}
		}

		builder.Write(")")
	}

	return nil
}

func formatCommonTableExpressions(builder FormattedQueryBuilder, commonTableExpressions With) error {
	builder.Write("with ")

	if commonTableExpressions.Recursive {
		builder.Write("recursive ")
	}

	for idx, commonTableExpression := range commonTableExpressions.Expressions {
		if idx > 0 {
			builder.Write(", ")
		}

		if err := formatTableAlias(builder, commonTableExpression.Alias); err != nil {
			return err
		}

		builder.Write(" as ")

		if commonTableExpression.Materialized != nil {
			if err := formatExpression(builder, *commonTableExpression.Materialized); err != nil {
				return err
			}

			builder.Write(" ")
		}

		builder.Write("(")

		if err := formatSetExpression(builder, commonTableExpression.Query); err != nil {
			return err
		}

		builder.Write(")")
	}

	// Leave a trailing space after formatting CTEs for the subsequent query body
	builder.Write(" ")

	return nil
}

func formatSetExpression(builder FormattedQueryBuilder, expression SetExpression) error {
	switch typedSetExpression := expression.(type) {
	case Query:
		if typedSetExpression.CommonTableExpressions != nil {
			if err := formatCommonTableExpressions(builder, *typedSetExpression.CommonTableExpressions); err != nil {
				return err
			}
		}

		return formatSetExpression(builder, typedSetExpression.Body)

	case Select:
		return formatSelect(builder, typedSetExpression)

	case SetOperation:
		if typedSetExpression.All && typedSetExpression.Distinct {
			return fmt.Errorf("set operation for query may not be both ALL and DISTINCT")
		}

		if err := formatSetExpression(builder, typedSetExpression.LeftOperand); err != nil {
			return err
		}

		builder.Write(" ")

		if err := formatExpression(builder, typedSetExpression.Operator); err != nil {
			return err
		}

		builder.Write(" ")

		if typedSetExpression.All {
			builder.Write("all ")
		}

		if typedSetExpression.Distinct {
			builder.Write("distinct ")
		}

		if err := formatSetExpression(builder, typedSetExpression.RightOperand); err != nil {
			return err
		}

	case Values:
		if err := formatExpression(builder, typedSetExpression); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unsupported set expression type %T", expression)
	}

	return nil
}

func formatMergeStatement(builder FormattedQueryBuilder, merge Merge) error {
	builder.Write("merge ")

	if merge.Into {
		builder.Write("into ")
	}

	if err := formatExpression(builder, merge.Table); err != nil {
		return err
	}

	builder.Write(" using ")

	if err := formatExpression(builder, merge.Source); err != nil {
		return err
	}

	builder.Write(" on ")

	if err := formatExpression(builder, merge.JoinTarget); err != nil {
		return err
	}

	builder.Write(" ")

	for idx, mergeAction := range merge.Actions {
		if idx > 0 {
			builder.Write(" ")
		}

		builder.Write("when ")

		switch typedMergeAction := mergeAction.(type) {
		case MatchedUpdate:
			builder.Write("matched")

			// Predicate is optional
			if typedMergeAction.Predicate != nil {
				builder.Write(" and ")

				if err := formatExpression(builder, typedMergeAction.Predicate); err != nil {
					return err
				}
			}

			builder.Write(" then update set ")

			for idx, assignment := range typedMergeAction.Assignments {
				if idx > 0 {
					builder.Write(", ")
				}

				if err := formatExpression(builder, assignment); err != nil {
					return err
				}
			}

		case MatchedDelete:
			builder.Write("matched")

			// Predicate is optional
			if typedMergeAction.Predicate != nil {
				builder.Write(" and ")

				if err := formatExpression(builder, typedMergeAction.Predicate); err != nil {
					return err
				}
			}

			builder.Write(" then delete")

		case UnmatchedAction:
			builder.Write("not matched")

			// Predicate is optional
			if typedMergeAction.Predicate != nil {
				builder.Write(" and ")

				if err := formatExpression(builder, typedMergeAction.Predicate); err != nil {
					return err
				}
			}

			builder.Write(" then insert (")

			for idx, column := range typedMergeAction.Columns {
				if idx > 0 {
					builder.Write(", ")
				}

				if err := formatExpression(builder, column); err != nil {
					return err
				}
			}

			builder.Write(") ")

			if err := formatExpression(builder, typedMergeAction.Values); err != nil {
				return err
			}

		default:
			return fmt.Errorf("unknown merge action type: %T", mergeAction)
		}
	}

	return nil
}

func formatInsertStatement(builder FormattedQueryBuilder, insert Insert) error {
	builder.Write("insert into ")

	if err := formatExpression(builder, insert.Table); err != nil {
		return err
	}

	if len(insert.Columns) > 0 {
		builder.Write(" (")

		for idx, column := range insert.Columns {
			if idx > 0 {
				builder.Write(", ")
			}

			builder.Write(column)
		}

		builder.Write(")")
	}

	builder.Write(" ")

	if insert.Source != nil {
		if err := formatSetExpression(builder, *insert.Source); err != nil {
			return err
		}
	}

	if insert.OnConflict != nil {
		builder.Write(" on conflict ")

		if insert.OnConflict.Target != nil {
			if err := formatExpression(builder, *insert.OnConflict.Target); err != nil {
				return err
			}

			builder.Write(" ")
		}

		if err := formatExpression(builder, insert.OnConflict.Action); err != nil {
			return err
		}
	}

	if len(insert.Returning) > 0 {
		builder.Write(" returning ")

		for idx, projection := range insert.Returning {
			if idx > 0 {
				builder.Write(", ")
			}

			if err := formatExpression(builder, projection); err != nil {
				return err
			}
		}
	}

	return nil
}

func formatUpdateStatement(builder FormattedQueryBuilder, update Update) error {
	builder.Write("update ")

	if err := formatExpression(builder, update.Table); err != nil {
		return err
	}

	builder.Write(" set ")

	for idx, assignment := range update.Assignments {
		if idx > 0 {
			builder.Write(", ")
		}

		if err := formatExpression(builder, assignment); err != nil {
			return err
		}
	}

	if update.Where != nil {
		builder.Write(" where ")

		if err := formatExpression(builder, update.Where); err != nil {
			return err
		}
	}

	return nil
}

func formatDeleteStatement(builder FormattedQueryBuilder, delete Delete) error {
	builder.Write("delete from ")

	if err := formatExpression(builder, delete.Table); err != nil {
		return err
	}

	if delete.Where != nil {
		builder.Write(" where ")

		if err := formatExpression(builder, delete.Where); err != nil {
			return err
		}
	}

	return nil
}

func FormatStatement(statement Statement) (FormattedQuery, error) {
	builder := NewFormattedQueryBuilder()

	switch typedStatement := statement.(type) {
	case Merge:
		if err := formatMergeStatement(builder, typedStatement); err != nil {
			return FormattedQuery{}, err
		}

	case Query:
		if err := formatSetExpression(builder, typedStatement); err != nil {
			return FormattedQuery{}, err
		}

	case Insert:
		if err := formatInsertStatement(builder, typedStatement); err != nil {
			return FormattedQuery{}, err
		}

	case Update:
		if err := formatUpdateStatement(builder, typedStatement); err != nil {
			return FormattedQuery{}, err
		}

	case Delete:
		if err := formatDeleteStatement(builder, typedStatement); err != nil {
			return FormattedQuery{}, err
		}

	default:
		return FormattedQuery{}, fmt.Errorf("unsupported PgSQL statement type: %T", statement)
	}

	return builder.Build(), nil
}
