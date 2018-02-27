package sqlite

import (
	"bytes"
	"database/sql"
	"fmt"

	"github.com/jinzhu/gorm"
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
		assignmentsChan = model.GetAssignments(tx)
		tableNameChan   = model.GetTable(tx)
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
			if idx != 0 {
				valueBuffer.WriteString(",")
			}
			valueBuffer.WriteString(" (")

			for j, field := range fields {
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
					if field.Field.HasDefaultValue {
						args = append(args, field.Field.DefaultValue)
					} else {
						args = append(args, nil)
					}
				} else {
					args = append(args, field.Value.Interface())
				}
			}
			valueBuffer.WriteString(")")
		}
		s.WriteString(") ")

		_, err = valueBuffer.WriteTo(s)
	} else {
		s.WriteString(" DEFAULT VALUES")
	}

	fmt.Println(s.String())
	fmt.Printf("%#v \n", args)
	if result, err := dialect.DB.Exec(s.String(), args...); err == nil {
		tx.RowsAffected, _ = result.RowsAffected()
	} else {
		fmt.Println(err)
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
