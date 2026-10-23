package clause

// Limit encodes row count and offset. A nil count or negative count omits LIMIT; a
// positive offset emits OFFSET. Merge rules preserve an old count for an incoming nil
// or zero count and preserve an old offset for incoming zero; use negative values to
// cancel them.
type Limit struct {
	// Limit points to the row count. Nil or negative omits LIMIT; a fresh zero renders
	// LIMIT 0, but zero does not replace an existing count during merging.
	Limit *int
	// Offset is the number of skipped rows. Only positive values render; a negative input
	// clears a previous offset during merging.
	Offset int
}

// Name returns LIMIT, the statement registry key for this clause.
func (limit Limit) Name() string {
	return "LIMIT"
}

// Build emits LIMIT for a non-negative pointed count and OFFSET only when positive,
// binding both numeric values. It includes its own keywords.
func (limit Limit) Build(builder Builder) {
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

// MergeClause clears the outer keyword and merges paging values. Incoming nil/zero
// count preserves an existing non-nil count; zero offset preserves a positive offset,
// while a negative offset clears it.
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
