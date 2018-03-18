package gorm

import (
	"sync"
)

// Column column type
type Column = string

// Statement GORM statement
type Statement struct {
	Dest        interface{}          // Insert, Select, Update, Delete
	Table       interface{}          // Insert, Select, Update, Delete
	Select      SelectColumn         // Insert, Select, Update
	Omit        []Column             // Insert, Select, Update
	Joins       []Join               // Select
	GroupBy     GroupBy              // Select
	OrderBy     OrderBy              // Select
	Preload     []Column             // Select
	Limit       Limit                // Select, Update
	Conditions  []ConditionInterface // Select, Update, Delete
	Assignments []Assignment         // Insert, Update
	Returnnings []Column             // Insert, Update, Delete
	Settings    sync.Map
}

// ConditionInterface query condition statement interface
type ConditionInterface interface{}

// Settings settings
type Settings map[string]interface{}

// DefaultValue default value type
type DefaultValue string

// SelectColumn select columns
type SelectColumn struct {
	Columns []string
	Args    []interface{}
}

// Join join statement
type Join struct {
	Table        string
	LocalField   string
	ForeignField string
	Conditions   []ConditionInterface
}

// GroupBy group by statement
type GroupBy struct {
	Columns []string
	Having  []ConditionInterface
}

// OrderCondition order condition, could be string or sql expr
type OrderCondition interface{}

// OrderBy order by statement
type OrderBy []OrderCondition

// OrderByColumn column used for order
type OrderByColumn struct {
	Name string
	Asc  bool
}

// Limit limit statement
type Limit struct {
	Limit  *int64
	Offset *int64
}

// Assignment assign statement
type Assignment struct {
	Column Column
	Value  interface{}
}

// Clone clone current statement
func (stmt *Statement) Clone() *Statement {
	newStatement := *stmt
	return &newStatement
}

// BuildCondition build condition
func (stmt *Statement) BuildCondition(query interface{}, args ...interface{}) ConditionInterface {
	if sql, ok := query.(string); ok {
		return Raw{SQL: sql, Args: args}
	}

	andConds := And([]ConditionInterface{ConditionInterface(query)})
	for _, arg := range args {
		andConds = append(andConds, ConditionInterface(arg))
	}
	return andConds
}

// AddConditions add conditions
func (stmt *Statement) AddConditions(conds ...ConditionInterface) {
	stmt.Conditions = append(stmt.Conditions, conds...)
}

////////////////////////////////////////////////////////////////////////////////
// Comparison Operators
////////////////////////////////////////////////////////////////////////////////

// Raw raw sql
type Raw struct {
	SQL  string
	Args []interface{} // TODO NamedArg
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
type And []ConditionInterface

// Not TRUE if condition is false
type Not []ConditionInterface

// Or TRUE if any of the conditions is TRUE
type Or []ConditionInterface
