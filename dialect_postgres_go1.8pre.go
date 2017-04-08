// +build !go1.8

package gorm

func (s postgres) HasIndex(tableName string, indexName string) bool {
	var count int
	s.db.QueryRow(queryPostgresHasIndex, tableName, indexName).Scan(&count)
	return count > 0
}

func (s postgres) HasForeignKey(tableName string, foreignKeyName string) bool {
	var count int
	s.db.QueryRow(queryPostgresHasForeignKey, tableName, foreignKeyName).Scan(&count)
	return count > 0
}

func (s postgres) HasTable(tableName string) bool {
	var count int
	s.db.QueryRow(queryPostgresHasTable, tableName).Scan(&count)
	return count > 0
}

func (s postgres) HasColumn(tableName string, columnName string) bool {
	var count int
	s.db.QueryRow(queryPostgresHasColumn, tableName, columnName).Scan(&count)
	return count > 0
}

func (s postgres) CurrentDatabase() (name string) {
	s.db.QueryRow(queryPostgresCurrentDatabase).Scan(&name)
	return
}
