package clause

type ExprCaseCondition struct {
	When string
	Then string
	Vars []any
}

type ExprCaseElse struct {
	Then string
	Vars []any
}

type ExprCase struct {
	Cases []*ExprCaseCondition
	Else  *ExprCaseElse
}

func (expr ExprCase) Name() string {
	return "CASE"
}

func (expr ExprCase) Build(builder Builder) {
	var vars []any
	for idx, condition := range expr.Cases {
		if idx > 0 {
			_ = builder.WriteByte(' ')
		}
		_, _ = builder.WriteString("WHEN ")
		_, _ = builder.WriteString(condition.When)
		_, _ = builder.WriteString(" THEN ")
		_, _ = builder.WriteString(condition.Then)
		if len(condition.Vars) > 0 {
			vars = append(vars, condition.Vars...)
		}
	}

	if expr.Else != nil {
		elseExpr := expr.Else
		_, _ = builder.WriteString(" ELSE ")
		_, _ = builder.WriteString(elseExpr.Then)
		if len(elseExpr.Vars) > 0 {
			vars = append(vars, elseExpr.Vars...)
		}
	}
	_, _ = builder.WriteString(" END")

	clauseExpr := Expr{SQL: "", Vars: vars}
	clauseExpr.Build(builder)
}

func (expr ExprCase) MergeClause(clause *Clause) {
	clause.Expression = expr
}
