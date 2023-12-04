package clause

type LockingStrength string

const (
	LockingStrengthUpdate = LockingStrength("UPDATE")
	LockingStrengthShare  = LockingStrength("SHARE")
)

type LockingOptions string

const (
	LockingOptionsSkipLocked = LockingOptions("SKIP LOCKED")
	LockingOptionsNoWait     = LockingOptions("NOWAIT")
)

type Locking struct {
	Strength LockingStrength
	Table    Table
	Options  LockingOptions
}

// Name where clause name
func (locking Locking) Name() string {
	return "FOR"
}

// Build build where clause
func (locking Locking) Build(builder Builder) {
	builder.WriteString(string(locking.Strength))
	if locking.Table.Name != "" {
		builder.WriteString(" OF ")
		builder.WriteQuoted(locking.Table)
	}

	if locking.Options != "" {
		builder.WriteByte(' ')
		builder.WriteString(string(locking.Options))
	}
}

// MergeClause merge order by clauses
func (locking Locking) MergeClause(clause *Clause) {
	clause.Expression = locking
}
