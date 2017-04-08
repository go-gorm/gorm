// +build !go1.8

package gorm

import "fmt"

func (s sqlite3) HasIndex(tableName string, indexName string) bool {
	var count int
	s.db.QueryRow(fmt.Sprintf(querySQLite3HasIndex, indexName), tableName).Scan(&count)
	return count > 0
}

func (s sqlite3) HasTable(tableName string) bool {
	var count int
	s.db.QueryRow(querySQLite3HasTable, tableName).Scan(&count)
	return count > 0
}

func (s sqlite3) HasColumn(tableName string, columnName string) bool {
	var count int
	s.db.QueryRow(fmt.Sprintf(querySQLite3HasColumn, columnName, columnName), tableName).Scan(&count)
	return count > 0
}

func (s sqlite3) CurrentDatabase() (name string) {
	var (
		ifaces   = make([]interface{}, 3)
		pointers = make([]*string, 3)
		i        int
	)
	for i = 0; i < 3; i++ {
		ifaces[i] = &pointers[i]
	}
	if err := s.db.QueryRow(querySQLite3CurrentDatabase).Scan(ifaces...); err != nil {
		return
	}
	if pointers[1] != nil {
		name = *pointers[1]
	}
	return
}
