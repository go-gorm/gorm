// +build go1.8

package gorm

import (
	"context"
	"fmt"
)

func (s mysql) RemoveIndex(tableName string, indexName string) error {
	_, err := s.db.ExecContext(context.Background(), fmt.Sprintf(queryMySQLRemoveIndex, indexName, s.Quote(tableName)))
	return err
}

func (s mysql) HasForeignKey(tableName string, foreignKeyName string) bool {
	var count int
	s.db.QueryRowContext(context.Background(), queryMySQLHasForeignKey, s.CurrentDatabase(), tableName, foreignKeyName).Scan(&count)
	return count > 0
}

func (s mysql) CurrentDatabase() (name string) {
	s.db.QueryRowContext(context.Background(), queryMySQLCurrentDatabase).Scan(&name)
	return
}
