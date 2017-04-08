// +build go1.8

package gorm

import (
	"context"
	"fmt"
)

func (s commonDialect) HasIndex(tableName string, indexName string) bool {
	var count int
	s.db.QueryRowContext(context.Background(), queryHasIndex, s.CurrentDatabase(), tableName, indexName).Scan(&count)
	return count > 0
}

func (s commonDialect) RemoveIndex(tableName string, indexName string) error {
	_, err := s.db.ExecContext(context.Background(), fmt.Sprintf(queryRemoveIndex, indexName))
	return err
}

func (s commonDialect) HasTable(tableName string) bool {
	var count int
	s.db.QueryRowContext(context.Background(), queryHasTable, s.CurrentDatabase(), tableName).Scan(&count)
	return count > 0
}

func (s commonDialect) HasColumn(tableName string, columnName string) bool {
	var count int
	s.db.QueryRowContext(context.Background(), queryHasColumn, s.CurrentDatabase(), tableName, columnName).Scan(&count)
	return count > 0
}

func (s commonDialect) CurrentDatabase() (name string) {
	s.db.QueryRowContext(context.Background(), queryCurrentDatabase).Scan(&name)
	return
}
