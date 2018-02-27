package sqlite

import (
	"bytes"
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/model"

	// import sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

// Dialect Sqlite3 Dialect for GORM
type Dialect struct {
}

// Quote quote for value
func (dialect Dialect) Quote(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}

// Insert insert
func (dialect *Dialect) Insert(tx *gorm.DB) (err error) {
	var (
		args            []interface{}
		assignmentsChan = model.GetAssignments(tx)
		tableNameChan   = model.GetTable(tx)
	)

	s := bytes.NewBufferString("INSERT INTO ")
	tableName := <-tableNameChan
	s.WriteString(tableName)

	if assignments := <-assignmentsChan; len(assignments) > 0 {
		columns := []string{}

		// Write columns (table.column1, table.column2, table.column3)
		s.WriteString(" (")

		// Write values (v1, v2, v3), (v2-1, v2-2, v2-3)
		valueBuffer := bytes.NewBufferString("VALUES ")

		for idx, fields := range assignments {
			if idx != 0 {
				s.WriteString(", ")
			}
			valueBuffer = bytes.NewBufferString(" (")

			for j, field := range fields {
				if idx == 0 {
					columns = append(columns, field.Field.DBName)
					if j != 0 {
						s.WriteString(", ")
					}
					s.WriteString(dialect.Quote(tableName))
					s.WriteString(".")
					s.WriteString(dialect.Quote(field.Field.DBName))
				}

				if j != 0 {
					valueBuffer.WriteString(", ")
				}
				valueBuffer.WriteString("?")

				if field.IsBlank {
					args = append(args, field.Field.DefaultValue)
				} else {
					args = append(args, field.Value.Interface())
				}
			}
			valueBuffer = bytes.NewBufferString(") ")
		}
		s.WriteString(") ")

		_, err = valueBuffer.WriteTo(s)
	} else {
		s.WriteString(" DEFAULT VALUES")
	}

	return
}

// Query query
func (*Dialect) Query(tx *gorm.DB) error {
	return nil
}

// Update update
func (*Dialect) Update(tx *gorm.DB) error {
	return nil
}

// Delete delete
func (*Dialect) Delete(tx *gorm.DB) error {
	return nil
}
