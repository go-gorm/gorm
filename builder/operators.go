package builder

////////////////////////////////////////////////////////////////////////////////
// Comparison Operators
////////////////////////////////////////////////////////////////////////////////

// Raw raw sql
type Raw struct {
	Value string
	Args  []interface{} // TODO NamedArg
}

// Eq equal to
type Eq struct {
	Column Column
	Value  interface{}
}

// Neq not equal to
type Neq struct {
	Column Column
	Value  interface{}
}

// Gt greater than
type Gt struct {
	Column Column
	Value  interface{}
}

// Gte greater than or equal to
type Gte struct {
	Column Column
	Value  interface{}
}

// Lt less than
type Lt struct {
	Column Column
	Value  interface{}
}

// Lte less than or equal to
type Lte struct {
	Column Column
	Value  interface{}
}

////////////////////////////////////////////////////////////////////////////////
// Logical Operators
////////////////////////////////////////////////////////////////////////////////

// And TRUE if all the conditions is TRUE
type And []Condition

// Not TRUE if condition is false
type Not Condition

// Or TRUE if any of the conditions is TRUE
type Or []Condition
