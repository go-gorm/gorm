// +build go1.8

package gorm

import "context"

func (s postgres) HasIndex(tableName string, indexName string) bool {
	var count int
	s.db.QueryRowContext(context.Background(), queryPostgresHasIndex, tableName, indexName).Scan(&count)
	return count > 0
}

func (s postgres) HasForeignKey(tableName string, foreignKeyName string) bool {
	var count int
	s.db.QueryRowContext(context.Background(), queryPostgresHasForeignKey, tableName, foreignKeyName).Scan(&count)
	return count > 0
}

func (s postgres) HasTable(tableName string) bool {
	var count int
	s.db.QueryRowContext(context.Background(), queryPostgresHasTable, tableName).Scan(&count)
	return count > 0
}

func (s postgres) HasColumn(tableName string, columnName string) bool {
	var count int
	s.db.QueryRowContext(context.Background(), queryPostgresHasColumn, tableName, columnName).Scan(&count)
	return count > 0
}

func (s postgres) CurrentDatabase() (name string) {
	s.db.QueryRowContext(context.Background(), queryPostgresCurrentDatabase).Scan(&name)
	return
}
