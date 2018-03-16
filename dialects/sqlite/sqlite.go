package sqlite

import (
	"bytes"
	"database/sql"
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/common/sqlbuilder"
	"github.com/jinzhu/gorm/model"
)

// Dialect Sqlite3 Dialect for GORM
type Dialect struct {
	DB *sql.DB
}

// Quote quote for value
func (dialect Dialect) Quote(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}

// Insert insert
func (dialect *Dialect) Insert(tx *gorm.DB) (err error) {
	var (
		args            []interface{}
		assignmentsChan = sqlbuilder.GetAssignmentFields(tx)
		tableNameChan   = sqlbuilder.GetTable(tx)
		primaryFields   []*model.Field
	)

	s := bytes.NewBufferString("INSERT INTO ")
	s.WriteString(dialect.Quote(<-tableNameChan))

	if assignments := <-assignmentsChan; len(assignments) > 0 {
		columns := []string{}

		// Write columns (column1, column2, column3)
		s.WriteString(" (")

		// Write values (v1, v2, v3), (v2-1, v2-2, v2-3)
		valueBuffer := bytes.NewBufferString("VALUES ")

		for idx, fields := range assignments {
			var primaryField *model.Field
			if idx != 0 {
				valueBuffer.WriteString(",")
			}
			valueBuffer.WriteString(" (")

			for j, field := range fields {
				if field.Field.IsPrimaryKey && primaryField == nil || field.Field.DBName == "id" {
					primaryField = field
				}

				if idx == 0 {
					columns = append(columns, field.Field.DBName)
					if j != 0 {
						s.WriteString(", ")
					}
					s.WriteString(dialect.Quote(field.Field.DBName))
				}

				if j != 0 {
					valueBuffer.WriteString(", ")
				}
				valueBuffer.WriteString("?")

				if field.IsBlank {
					args = append(args, nil)
				} else {
					args = append(args, field.Value.Interface())
				}
			}

			primaryFields = append(primaryFields, primaryField)
			valueBuffer.WriteString(")")
		}
		s.WriteString(") ")

		_, err = valueBuffer.WriteTo(s)
	} else {
		s.WriteString(" DEFAULT VALUES")
	}

	result, err := dialect.DB.Exec(s.String(), args...)

	if err == nil {
		var lastInsertID int64
		tx.RowsAffected, _ = result.RowsAffected()
		lastInsertID, err = result.LastInsertId()
		if len(primaryFields) == int(tx.RowsAffected) {
			startID := lastInsertID - tx.RowsAffected + 1
			for i, primaryField := range primaryFields {
				tx.AddError(primaryField.Set(startID + int64(i)))
			}
		}
	}
	return
}

// Query query
func (dialect *Dialect) Query(tx *gorm.DB) (err error) {
	var (
		args           []interface{}
		tableNameChan  = sqlbuilder.GetTable(tx)
		conditionsChan = sqlbuilder.BuildConditions(tx, tx.Statement.Conditions)
		groupChan      = sqlbuilder.BuildGroupCondition(tx)
		orderChan      = sqlbuilder.BuildOrderCondition(tx)
		limitChan      = sqlbuilder.BuildLimitCondition(tx)
	)

	s := bytes.NewBufferString("SELECT ")

	// FIXME quote, add table
	columns := tx.Statement.Select.Columns
	if len(columns) > 0 {
		args = append(args, tx.Statement.Select.Args...)
	} else {
		columns = []string{"*"}
	}

	for idx, column := range columns {
		if idx != 0 {
			s.WriteString(",")
		}
		s.WriteString(column)
	}

	s.WriteString(" FROM ")
	s.WriteString(dialect.Quote(<-tableNameChan))

	// Join SQL
	for _, join := range tx.Statement.Joins {
		if join.Table == "" {
			builder := <-sqlbuilder.BuildConditions(tx, join.Conditions)
			_, err = builder.SQL.WriteTo(s)
			args = append(args, builder.Args...)
		}
		// FIXME
	}

	if len(tx.Statement.Conditions) > 0 {
		s.WriteString(" WHERE ")
		builder := <-conditionsChan
		_, err = builder.SQL.WriteTo(s)
		args = append(args, builder.Args...)
	}

	if builder := <-groupChan; builder != nil {
		_, err = builder.SQL.WriteTo(s)
		args = append(args, builder.Args)
	}

	if builder := <-orderChan; builder != nil {
		_, err = builder.SQL.WriteTo(s)
		args = append(args, builder.Args)
	}

	if builder := <-limitChan; builder != nil {
		_, err = builder.SQL.WriteTo(s)
		args = append(args, builder.Args)
	}
	return
}

// Update update
func (*Dialect) Update(tx *gorm.DB) error {
	return nil
}

// Delete delete
func (*Dialect) Delete(tx *gorm.DB) error {
	return nil
}
