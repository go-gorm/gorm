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
	// NOT: We don't auto-set limit here. We only rely on the final struct's Limit and Offset.
	// Any "auto offset => limit" logic is handled in MergeClause.

	if limit.Limit != nil && *limit.Limit >= 0 {
		builder.WriteString("LIMIT ")
		builder.AddVar(builder, *limit.Limit)
	}

	if limit.Offset > 0 {
		// Add space if LIMIT was set
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
		// 1) Merge offset
		if limit.Offset == 0 && v.Offset > 0 {
			limit.Offset = v.Offset
		} else if limit.Offset < 0 {
			// Negative offset => 0
			limit.Offset = 0
		}

		// 2) Merge limit
		if (limit.Limit == nil || *limit.Limit == 0) && v.Limit != nil {
			limit.Limit = v.Limit
		}

		// 3) If final limit is negative => treat it as nil (meaning "no limit")
		if limit.Limit != nil && *limit.Limit < 0 {
			limit.Limit = nil
		}
	}

	// 4) If offset > 0 but limit is still nil, set limit to math.MaxInt
	if limit.Offset > 0 && limit.Limit == nil {
		maxInt := math.MaxInt
		limit.Limit = &maxInt
	}

	clause.Expression = limit
}
