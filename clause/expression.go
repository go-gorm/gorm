package clause

import "strings"

// Expression expression interface
type Expression interface {
	Build(builder Builder)
}

// NegationExpressionBuilder negation expression builder
type NegationExpressionBuilder interface {
	NegationBuild(builder Builder)
}

// Expr raw expression
type Expr struct {
	SQL  string
	Vars []interface{}
}

// Build build raw expression
func (expr Expr) Build(builder Builder) {
	sql := expr.SQL
	for _, v := range expr.Vars {
		sql = strings.Replace(sql, " ?", " "+builder.AddVar(v), 1)
	}
	builder.Write(sql)
}

// IN Whether a value is within a set of values
type IN struct {
	Column interface{}
	Values []interface{}
}

func (in IN) Build(builder Builder) {
	builder.WriteQuoted(in.Column)

	switch len(in.Values) {
	case 0:
		builder.Write(" IN (NULL)")
	case 1:
		builder.Write(" = ", builder.AddVar(in.Values...))
	default:
		builder.Write(" IN (", builder.AddVar(in.Values...), ")")
	}
}

func (in IN) NegationBuild(builder Builder) {
	switch len(in.Values) {
	case 0:
	case 1:
		builder.Write(" <> ", builder.AddVar(in.Values...))
	default:
		builder.Write(" NOT IN (", builder.AddVar(in.Values...), ")")
	}
}

// Eq equal to for where
type Eq struct {
	Column interface{}
	Value  interface{}
}

func (eq Eq) Build(builder Builder) {
	builder.WriteQuoted(eq.Column)

	if eq.Value == nil {
		builder.Write(" IS NULL")
	} else {
		builder.Write(" = ", builder.AddVar(eq.Value))
	}
}

func (eq Eq) NegationBuild(builder Builder) {
	Neq{eq.Column, eq.Value}.Build(builder)
}

// Neq not equal to for where
type Neq Eq

func (neq Neq) Build(builder Builder) {
	builder.WriteQuoted(neq.Column)

	if neq.Value == nil {
		builder.Write(" IS NOT NULL")
	} else {
		builder.Write(" <> ", builder.AddVar(neq.Value))
	}
}

func (neq Neq) NegationBuild(builder Builder) {
	Eq{neq.Column, neq.Value}.Build(builder)
}

// Gt greater than for where
type Gt Eq

func (gt Gt) Build(builder Builder) {
	builder.WriteQuoted(gt.Column)
	builder.Write(" > ", builder.AddVar(gt.Value))
}

func (gt Gt) NegationBuild(builder Builder) {
	Lte{gt.Column, gt.Value}.Build(builder)
}

// Gte greater than or equal to for where
type Gte Eq

func (gte Gte) Build(builder Builder) {
	builder.WriteQuoted(gte.Column)
	builder.Write(" >= ", builder.AddVar(gte.Value))
}

func (gte Gte) NegationBuild(builder Builder) {
	Lt{gte.Column, gte.Value}.Build(builder)
}

// Lt less than for where
type Lt Eq

func (lt Lt) Build(builder Builder) {
	builder.WriteQuoted(lt.Column)
	builder.Write(" < ", builder.AddVar(lt.Value))
}

func (lt Lt) NegationBuild(builder Builder) {
	Gte{lt.Column, lt.Value}.Build(builder)
}

// Lte less than or equal to for where
type Lte Eq

func (lte Lte) Build(builder Builder) {
	builder.WriteQuoted(lte.Column)
	builder.Write(" <= ", builder.AddVar(lte.Value))
}

func (lte Lte) NegationBuild(builder Builder) {
	Gt{lte.Column, lte.Value}.Build(builder)
}

// Like whether string matches regular expression
type Like Eq

func (like Like) Build(builder Builder) {
	builder.WriteQuoted(like.Column)
	builder.Write(" LIKE ", builder.AddVar(like.Value))
}

func (like Like) NegationBuild(builder Builder) {
	builder.WriteQuoted(like.Column)
	builder.Write(" NOT LIKE ", builder.AddVar(like.Value))
}

// Map
type Map map[interface{}]interface{}

func (m Map) Build(builder Builder) {
	// TODO
}

func (m Map) NegationBuild(builder Builder) {
	// TODO
}
