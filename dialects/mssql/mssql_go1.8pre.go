// +build !go1.8

package mssql

import "fmt"

func (s mssql) HasIndex(tableName string, indexName string) bool {
	var count int
	s.db.QueryRow(queryMSSQLHasIndex, indexName, tableName).Scan(&count)
	return count > 0
}

func (s mssql) RemoveIndex(tableName string, indexName string) error {
	_, err := s.db.Exec(fmt.Sprintf(queryMSSQLRemoveIndex, indexName, s.Quote(tableName)))
	return err
}

func (s mssql) HasTable(tableName string) bool {
	var count int
	s.db.QueryRow(queryMSSQLHasTable, tableName, s.CurrentDatabase()).Scan(&count)
	return count > 0
}

func (s mssql) HasColumn(tableName string, columnName string) bool {
	var count int
	s.db.QueryRow(queryMSSQLHasColumn, s.CurrentDatabase(), tableName, columnName).Scan(&count)
	return count > 0
}

func (s mssql) CurrentDatabase() (name string) {
	s.db.QueryRow(queryMSSQLCurrentDatabase).Scan(&name)
	return
}
