package clause

import (
	"math"
)

// Limit represents a limit clause
type Limit struct {
	Limit  *int
	Offset int
}

// Name returns the name of the clause ("LIMIT")
func (limit Limit) Name() string {
	return "LIMIT"
}

// Build constructs the LIMIT clause
func (limit Limit) Build(builder Builder) {
	// If only offset is defined and limit is nil, set limit to math.MaxInt
	if limit.Limit == nil && limit.Offset > 0 {
		maxInt := math.MaxInt
		limit.Limit = &maxInt
	}

	if limit.Limit != nil && *limit.Limit >= 0 {
		builder.WriteString("LIMIT ")
		builder.AddVar(builder, *limit.Limit)
	}
	if limit.Offset > 0 {
		if limit.Limit != nil && *limit.Limit >= 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString("OFFSET ")
		builder.AddVar(builder, limit.Offset)
	}
}

// MergeClause merges two limit clauses
func (limit Limit) MergeClause(clause *Clause) {
	clause.Name = ""

	if v, ok := clause.Expression.(Limit); ok {
		if (limit.Limit == nil || *limit.Limit == 0) && v.Limit != nil {
			limit.Limit = v.Limit
		}

		if limit.Offset == 0 && v.Offset > 0 {
			limit.Offset = v.Offset
		} else if limit.Offset < 0 {
			limit.Offset = 0
		}
	}

	clause.Expression = limit
}
