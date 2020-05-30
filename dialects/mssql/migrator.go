package mssql

import (
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/migrator"
)

type Migrator struct {
	migrator.Migrator
}

func (m Migrator) HasTable(value interface{}) bool {
	var count int
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		return m.DB.Raw(
			"SELECT count(*) FROM INFORMATION_SCHEMA.tables WHERE table_name = ? AND table_catalog = ?",
			stmt.Table, m.CurrentDatabase(),
		).Row().Scan(&count)
	})
	return count > 0
}

func (m Migrator) HasIndex(value interface{}, name string) bool {
	var count int
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if idx := stmt.Schema.LookIndex(name); idx != nil {
			name = idx.Name
		}

		return m.DB.Raw(
			"SELECT count(*) FROM sys.indexes WHERE name=? AND object_id=OBJECT_ID(?)",
			name, stmt.Table,
		).Row().Scan(&count)
	})
	return count > 0
}

func (m Migrator) HasConstraint(value interface{}, name string) bool {
	var count int64
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		return m.DB.Raw(
			`SELECT count(*) FROM sys.foreign_keys as F inner join sys.tables as T on F.parent_object_id=T.object_id inner join information_schema.tables as I on I.TABLE_NAME = T.name WHERE F.name = ?  AND T.Name = ? AND I.TABLE_CATALOG = ?;`,
			name, stmt.Table, m.CurrentDatabase(),
		).Row().Scan(&count)
	})
	return count > 0
}

func (m Migrator) CurrentDatabase() (name string) {
	m.DB.Raw("SELECT DB_NAME() AS [Current Database]").Row().Scan(&name)
	return
}
