package clause

// Joins joins clause
type Joins struct {
	Name  string
	Query string
	Vars  []interface{}
}
