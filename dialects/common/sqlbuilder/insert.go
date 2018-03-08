package sqlbuilder

import (
	"bytes"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/model"
)

// BuildInsertSQL build insert SQL
func BuildInsertSQL(tx *gorm.DB) (s *bytes.Buffer, args []interface{}, defaultFieldsSlice [][]*model.Field, err error) {
	var (
		dialect         = tx.Dialect()
		assignmentsChan = GetAssignmentFields(tx)
		tableNameChan   = GetTable(tx)
	)
	defer close(tableNameChan)

	s = bytes.NewBufferString("INSERT INTO ")
	s.WriteString(dialect.Quote(<-tableNameChan))

	if assignments := <-assignmentsChan; len(assignments) > 0 {
		columns := []string{}
		defaultFields := []*model.Field{}

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
					if field.HasDefaultValue {
						defaultFields = append(defaultFields, field)
					}
				} else {
					args = append(args, field.Value.Interface())
				}
			}

			defaultFieldsSlice = append(defaultFieldsSlice, append([]*model.Field{primaryField}, defaultFields...))
			valueBuffer.WriteString(")")
		}
		s.WriteString(") ")

		_, err = valueBuffer.WriteTo(s)
	} else {
		s.WriteString(" DEFAULT VALUES")
	}

	return
}
