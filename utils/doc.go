// Package utils contains helpers shared by GORM's implementation, dialectors,
// and tests. The package is importable, but its conversions, identifier heuristics,
// and comparisons follow GORM's internal conventions rather than general-purpose
// serialization or SQL semantics.
//
// # Identity and association helpers
//
// ToStringKey combines association identity components into a textual map key.
// It unwraps driver.Valuer values and uses special zero-value formatting; it is
// not collision-free encoding or a persistent identifier format. ToString accepts
// strings and built-in integer types, returning an empty string for unsupported
// types. AssertEqual compares Go values and optional driver.Valuer results; it
// does not reproduce a database's equality or collation rules.
//
// NestedRelationName, JoinNestedRelationNames, and SplitNestedRelationName use
// double underscores for joined-association aliases and scan paths. They do not
// quote SQL identifiers. IsInvalidDBNameChar is an identifier-splitting heuristic,
// not an authorization check or SQL-injection defense. CheckTruth implements tag
// truthiness: a non-empty value other than case-insensitive "false" is true,
// including "0".
//
// # Call sites and slices
//
// CallerFrame and FileWithLineNum locate a relevant application or test frame in
// a bounded stack walk, skipping GORM implementation and generated source files.
// Their output depends on compiled source paths and stack depth; it is diagnostic
// information rather than a stable source identifier.
//
// RTrimSlice returns a view of the original backing array. It neither copies the
// retained elements nor clears removed references. Callers own synchronization
// when the same array is accessed or modified concurrently.
package utils
