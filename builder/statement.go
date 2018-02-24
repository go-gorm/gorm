package builder

// Column column type
type Column = string

// Statement GORM statement
type Statement struct {
	Dest        interface{}   // Insert, Select, Update, Delete
	Table       string        // Insert, Select, Update, Delete
	Select      []interface{} // Select
	Joins       []Join        // Select
	GroupBy     GroupBy       // Select
	OrderBy     OrderBy       // Select
	Preload     []Column      // Select
	Limit       Limit         // Select, Update
	Where       []Condition   // Select, Update, Delete
	Assignments []Assignment  // Insert, Update
	Returnnings []Column      // Insert, Update, Delete
}

// Condition query condition statement interface
type Condition interface {
	// ToSQL()
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

// OrderBy order by statement
type OrderBy []OrderByColumn

// OrderByColumn column used for order
type OrderByColumn struct {
	Name string
	Asc  bool
}

// Limit limit statement
type Limit struct {
	Limit  int
	Offset int
}

// Assignment assign statement
type Assignment struct {
	Column Column
	Value  interface{}
}
