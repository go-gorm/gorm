package gorm

// OraDialect interface allows for each Ora driver to implement custom behaviours. Defining
// a required oracle dialect interface also allows us to assert the interface whenever we
// have oracle specific behaviours in the rest of gorm.
type OraDialect interface {
	// CreateWithReturningInto is called by gorm.createCallback(*Scope) and will create new entity while populating the identity ID into the primary key
	// different drivers will have different ways of handling this behaviour for Ora
	CreateWithReturningInto(*Scope)
}
