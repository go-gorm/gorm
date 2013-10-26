package gorm

import (
	"strings"

	"fmt"
)

func (s *Orm) explain(value interface{}, operation string) {
	s.setModel(value)
	switch operation {
	case "Save":
		s.saveSql(value)
	case "Delete":
		s.deleteSql(value)
	}
	return
}

func (s *Orm) saveSql(value interface{}) {
	columns, values := modelValues(value)
	s.Sql = fmt.Sprintf(
		"INSERT INTO %v (%v) VALUES (%v)",
		s.TableName,
		strings.Join(columns, ","),
		valuesToBinVar(values),
	)
	s.SqlVars = values
	return
}

func (s *Orm) deleteSql(value interface{}) {
	s.Sql = fmt.Sprintf("DELETE FROM %v WHERE %v", s.TableName, s.whereSql)
	return
}

func (s *Orm) whereSql() (sql string) {
	sql = "1=1"
	return
}
