package clause

import "strings"

type Condition Builder
type AddConditions []Condition

func (cs AddConditions) Build(builder BuilderInterface) {
	for idx, c := range cs {
		if idx > 0 {
			builder.Write(" AND ")
		}
		c.Build(builder)
	}
}

type ORConditions []Condition

func (cs ORConditions) Build(builder BuilderInterface) {
	for idx, c := range cs {
		if idx > 0 {
			builder.Write(" OR ")
		}
		c.Build(builder)
	}
}

type NotConditions []Condition

func (cs NotConditions) Build(builder BuilderInterface) {
	for idx, c := range cs {
		if idx > 0 {
			builder.Write(" AND ")
		}

		if negationBuilder, ok := c.(NegationBuilder); ok {
			negationBuilder.NegationBuild(builder)
		} else {
			builder.Write(" NOT ")
			c.Build(builder)
		}
	}
}

// Raw raw sql for where
type Raw struct {
	SQL    string
	Values []interface{}
}

func (raw Raw) Build(builder BuilderInterface) {
	sql := raw.SQL
	for _, v := range raw.Values {
		sql = strings.Replace(sql, " ? ", " "+builder.AddVar(v)+" ", 1)
	}
	builder.Write(sql)
}

// IN Whether a value is within a set of values
type IN struct {
	Column interface{}
	Values []interface{}
}

func (in IN) Build(builder BuilderInterface) {
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

func (in IN) NegationBuild(builder BuilderInterface) {
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

func (eq Eq) Build(builder BuilderInterface) {
	builder.WriteQuoted(eq.Column)

	if eq.Value == nil {
		builder.Write(" IS NULL")
	} else {
		builder.Write(" = ", builder.AddVar(eq.Value))
	}
}

func (eq Eq) NegationBuild(builder BuilderInterface) {
	Neq{eq.Column, eq.Value}.Build(builder)
}

// Neq not equal to for where
type Neq struct {
	Column interface{}
	Value  interface{}
}

func (neq Neq) Build(builder BuilderInterface) {
	builder.WriteQuoted(neq.Column)

	if neq.Value == nil {
		builder.Write(" IS NOT NULL")
	} else {
		builder.Write(" <> ", builder.AddVar(neq.Value))
	}
}

func (neq Neq) NegationBuild(builder BuilderInterface) {
	Eq{neq.Column, neq.Value}.Build(builder)
}

// Gt greater than for where
type Gt struct {
	Column interface{}
	Value  interface{}
}

func (gt Gt) Build(builder BuilderInterface) {
	builder.WriteQuoted(gt.Column)
	builder.Write(" > ", builder.AddVar(gt.Value))
}

func (gt Gt) NegationBuild(builder BuilderInterface) {
	Lte{gt.Column, gt.Value}.Build(builder)
}

// Gte greater than or equal to for where
type Gte struct {
	Column interface{}
	Value  interface{}
}

func (gte Gte) Build(builder BuilderInterface) {
	builder.WriteQuoted(gte.Column)
	builder.Write(" >= ", builder.AddVar(gte.Value))
}

func (gte Gte) NegationBuild(builder BuilderInterface) {
	Lt{gte.Column, gte.Value}.Build(builder)
}

// Lt less than for where
type Lt struct {
	Column interface{}
	Value  interface{}
}

func (lt Lt) Build(builder BuilderInterface) {
	builder.WriteQuoted(lt.Column)
	builder.Write(" < ", builder.AddVar(lt.Value))
}

func (lt Lt) NegationBuild(builder BuilderInterface) {
	Gte{lt.Column, lt.Value}.Build(builder)
}

// Lte less than or equal to for where
type Lte struct {
	Column interface{}
	Value  interface{}
}

func (lte Lte) Build(builder BuilderInterface) {
	builder.WriteQuoted(lte.Column)
	builder.Write(" <= ", builder.AddVar(lte.Value))
}

func (lte Lte) NegationBuild(builder BuilderInterface) {
	Gt{lte.Column, lte.Value}.Build(builder)
}

// Like whether string matches regular expression
type Like struct {
	Column interface{}
	Value  interface{}
}

func (like Like) Build(builder BuilderInterface) {
	builder.WriteQuoted(like.Column)
	builder.Write(" LIKE ", builder.AddVar(like.Value))
}

func (like Like) NegationBuild(builder BuilderInterface) {
	builder.WriteQuoted(like.Column)
	builder.Write(" NOT LIKE ", builder.AddVar(like.Value))
}

// Map
type Map map[interface{}]interface{}

func (m Map) Build(builder BuilderInterface) {
	// TODO
}

func (m Map) NegationBuild(builder BuilderInterface) {
	// TODO
}

// Attrs
type Attrs struct {
	Value  interface{}
	Select []string
	Omit   []string
}

func (attrs Attrs) Build(builder BuilderInterface) {
	// TODO
	// builder.WriteQuoted(like.Column)
	// builder.Write(" LIKE ", builder.AddVar(like.Value))
}

func (attrs Attrs) NegationBuild(builder BuilderInterface) {
	// TODO
}

// ID
type ID struct {
	Value []interface{}
}

func (id ID) Build(builder BuilderInterface) {
	if len(id.Value) == 1 {
	}
	// TODO
	// builder.WriteQuoted(like.Column)
	// builder.Write(" LIKE ", builder.AddVar(like.Value))
}

func (id ID) NegationBuild(builder BuilderInterface) {
	// TODO
}
