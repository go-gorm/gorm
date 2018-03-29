package sqlite

import (
	"fmt"
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

// CreateTable create table for value
func (dialect *Dialect) CreateTable(value interface{}) error {
	// s := schema.Parse(value)
	return nil
}

// HasTable check if has table
func (dialect *Dialect) HasTable(tableName string) bool {
	var count int
	_ = dialect.DB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&count)
	return count > 0
}

// HasColumn check if has column in table
func (dialect *Dialect) HasColumn(tableName string, columnName string) bool {
	var count int
	_ = dialect.DB.QueryRow(fmt.Sprintf("SELECT count(*) FROM sqlite_master WHERE tbl_name = ? AND (sql LIKE '%%\"%v\" %%' OR sql LIKE '%%%v %%');\n", columnName, columnName), tableName).Scan(&count)
	return count > 0
}

// HasIndex check if has index in table
func (dialect *Dialect) HasIndex(tableName string, indexName string) bool {
	var count int
	_ = dialect.DB.QueryRow(fmt.Sprintf("SELECT count(*) FROM sqlite_master WHERE tbl_name = ? AND sql LIKE '%%INDEX %v ON%%'", indexName), tableName).Scan(&count)
	return count > 0
}

// HasForeignKey check if has foreign key in table
func (dialect *Dialect) HasForeignKey(tableName string, foreignKeyName string) bool {
	return false
}

// RemoveIndex remove index from table
func (dialect *Dialect) RemoveIndex(tableName string, indexName string) error {
	_, err := dialect.DB.Exec(fmt.Sprintf("DROP INDEX %v", indexName))
	return err
}

// ModifyColumn modify column
func (dialect *Dialect) ModifyColumn(tableName string, columnName string, typ string) error {
	_, err := dialect.DB.Exec(fmt.Sprintf("ALTER TABLE %v ALTER COLUMN %v TYPE %v", tableName, columnName, typ))
	return err
}

// CurrentDatabase get current database name
func (dialect *Dialect) CurrentDatabase() (name string) {
	var (
		ifaces   = make([]interface{}, 3)
		pointers = make([]*string, 3)
		i        int
	)
	for i = 0; i < 3; i++ {
		ifaces[i] = &pointers[i]
	}
	if err := dialect.DB.QueryRow("PRAGMA database_list").Scan(ifaces...); err != nil {
		return
	}
	if pointers[1] != nil {
		name = *pointers[1]
	}
	return
}
