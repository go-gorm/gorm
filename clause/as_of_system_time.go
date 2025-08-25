package clause

import (
	"fmt"
	"time"
)

// AsOfSystemTime represents CockroachDB's "AS OF SYSTEM TIME" clause
// This allows querying data as it existed at a specific point in time
type AsOfSystemTime struct {
	Timestamp time.Time
	Raw       string // For raw SQL expressions like "AS OF SYSTEM TIME '-1h'"
}

// Name returns the clause name
func (a AsOfSystemTime) Name() string {
	return "AS OF SYSTEM TIME"
}

// Build builds the "AS OF SYSTEM TIME" clause
func (a AsOfSystemTime) Build(builder Builder) {
	if a.Raw != "" {
		builder.WriteString("AS OF SYSTEM TIME ")
		builder.WriteString(a.Raw)
	} else if !a.Timestamp.IsZero() {
		builder.WriteString(fmt.Sprintf("AS OF SYSTEM TIME '%s'", a.Timestamp.Format("2006-01-02 15:04:05.000000")))
	}
}

// MergeClause merges the "AS OF SYSTEM TIME" clause
func (a AsOfSystemTime) MergeClause(clause *Clause) {
	clause.Expression = a
}
