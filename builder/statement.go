package builder

import "sync"

// Column column type
type Column = string

// Statement GORM statement
type Statement struct {
	Dest        interface{}  // Insert, Select, Update, Delete
	Table       interface{}  // Insert, Select, Update, Delete
	Select      SelectColumn // Insert, Select, Update
	Omit        []Column     // Insert, Select, Update
	Joins       []Join       // Select
	GroupBy     GroupBy      // Select
	OrderBy     OrderBy      // Select
	Preload     []Column     // Select
	Limit       Limit        // Select, Update
	Conditions  []Condition  // Select, Update, Delete
	Assignments []Assignment // Insert, Update
	Returnnings []Column     // Insert, Update, Delete
	Settings    sync.Map
}

// Condition query condition statement interface
type Condition interface {
	// ToSQL()
}

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
	Conditions   []Condition
}

// GroupBy group by statement
type GroupBy struct {
	GroupByColumns []string
	Having         []Condition
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
	// FIXME fix settings
	return &newStatement
}

// BuildCondition build condition
func (stmt *Statement) BuildCondition(query interface{}, args ...interface{}) Condition {
	return nil
}

// AddConditions add conditions
func (stmt *Statement) AddConditions(conds ...Condition) {
	stmt.Conditions = append(stmt.Conditions, conds...)
}
