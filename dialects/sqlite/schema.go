package sqlite

import (
	"strings"
)

// AutoMigrate auto migrate database
func (dialect *Dialect) AutoMigrate(value interface{}) (err error) {
	// create table

	// create missed column

	// safe upgrade some fields (like size, change data type)

	// create missed foreign key

	// create missed index
	return nil
}

// HasTable check if has table in current schema
func (dialect *Dialect) HasTable(tableName string) bool {
	var count int
	currentDatabase, tableName := currentDatabaseAndTable(dialect, tableName)
	_ = dialect.DB.QueryRow("SELECT count(*) FROM INFORMATION_SCHEMA.TABLES WHERE table_schema = ? AND table_name = ?", currentDatabase, tableName).Scan(&count)
	return count > 0
}

// CreateTable create table for value
func (dialect *Dialect) CreateTable(value interface{}) error {
	// s := schema.Parse(value)
	return nil
}

// CurrentDatabase get current database name
func (dialect *Dialect) CurrentDatabase() (name string) {
	_ = dialect.DB.QueryRow("SELECT DATABASE()").Scan(&name)
	return
}

func currentDatabaseAndTable(dialect *Dialect, tableName string) (string, string) {
	if strings.Contains(tableName, ".") {
		splitStrings := strings.SplitN(tableName, ".", 2)
		return splitStrings[0], splitStrings[1]
	}
	return dialect.CurrentDatabase(), tableName
}
