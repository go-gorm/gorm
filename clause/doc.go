// Package clause provides composable SQL expressions and named clauses for GORM.
// Applications normally pass values from this package to DB.Clauses or DB.Where;
// dialectors and plugins also use them to build and customize statements. Building
// an expression writes SQL and bound variables; it does not execute a query.
//
// # Choosing an expression
//
// Use structured values such as Eq, IN, Column, and Table when the structure is
// known. For example, with an initialized db and a destination users:
//
//	db.Where(clause.Eq{Column: clause.Column{Name: "active"}, Value: true}).Find(&users)
//	db.Clauses(clause.OnConflict{DoNothing: true}).Create(&users)
//
// The first example binds true as data and quotes the column through the dialect.
// The second selects conflict behavior; the database and dialector must support
// the resulting syntax. Locking, Returning, and other advanced clauses similarly
// depend on the database, transaction scope, and driver implementation.
//
// Use Expr or NamedExpr for SQL fragments that cannot be represented conveniently
// with structured expressions. Their SQL text is trusted syntax. Put untrusted
// data in Vars, and choose dynamic identifiers from application-controlled values.
// Placeholders bind values, not SQL keywords or identifier names. Raw fields and
// strings written directly to a Builder bypass identifier quoting.
//
// # Building and merging
//
// Expression defines the common Build operation. Interface adds a Name and a
// MergeClause operation, allowing DB.Clauses to store and combine entries under a
// statement clause name. Repeated clauses do not all behave alike: Where appends
// conditions, OnConflict replaces conflict options, and Limit has its own merge
// rules. Read the concrete MergeClause contract before composing reusable scopes.
//
// Statement callbacks choose clause order, and a dialector may replace rendering
// by name through a ClauseBuilder. A custom clause therefore needs both correct
// merge behavior and an execution path that includes its name in the build order.
// Implementations should use Builder.AddVar for data, Builder.WriteQuoted for
// identifiers, and Builder.AddError for construction failures.
//
// Clause values can contain slices, maps, or pointers that remain shared after a
// shallow copy. Treat a submitted expression as immutable, and construct separate
// values when modifying expressions concurrently. This package owns no database
// connections, transactions, rows, or prepared statements.
//
// See https://gorm.io/docs/sql_builder.html for the statement-building workflow
// and https://gorm.io/docs/security.html for SQL-input boundaries.
package clause
