package clause

// With is a placeholder type without clause methods. It does not currently implement
// Interface or build a common-table expression; use a supported custom expression or
// explicit SQL for a WITH query.
type With struct{}
