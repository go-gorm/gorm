package clause

const (
	// LockingStrengthUpdate requests an UPDATE row lock on supporting databases.
	LockingStrengthUpdate = "UPDATE"
	// LockingStrengthShare requests a SHARE row lock on supporting databases.
	LockingStrengthShare = "SHARE"
	// LockingOptionsSkipLocked skips rows already locked by competing transactions where
	// supported. Results may omit otherwise matching rows.
	LockingOptionsSkipLocked = "SKIP LOCKED"
	// LockingOptionsNoWait asks a supporting database to fail instead of waiting for an
	// unavailable lock.
	LockingOptionsNoWait = "NOWAIT"
)

// Locking requests database row locks, optionally for a named table. Use it inside a
// transaction when locks must protect subsequent statements. Strength/Options and
// driver support are database-specific; SQLite and other engines may omit or reject
// this clause.
type Locking struct {
	// Strength is trusted lock syntax, commonly LockingStrengthUpdate or
	// LockingStrengthShare.
	Strength string
	// Table optionally limits the lock target using an OF table clause.
	Table Table
	// Options is trusted database syntax, commonly NOWAIT or SKIP LOCKED. Do not place
	// unvalidated user text here.
	Options string
}

// Name returns FOR, the statement registry key for this clause.
func (locking Locking) Name() string {
	return "FOR"
}

// Build writes trusted Strength, an optional quoted OF table, and trusted Options.
// Dialectors may override the syntax.
func (locking Locking) Build(builder Builder) {
	builder.WriteString(locking.Strength)
	if locking.Table.Name != "" {
		builder.WriteString(" OF ")
		builder.WriteQuoted(locking.Table)
	}

	if locking.Options != "" {
		builder.WriteByte(' ')
		builder.WriteString(locking.Options)
	}
}

// MergeClause replaces the previous locking clause rather than combining strengths or
// options.
func (locking Locking) MergeClause(clause *Clause) {
	clause.Expression = locking
}
