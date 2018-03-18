package sqlschema

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/jinzhu/gorm/schema"
)

// SQLSchema sql schema
type SQLSchema struct {
	*schema.Schema
	Fields []*Field
}

// Field SQLSchema Field
type Field struct {
	*schema.Field
	DataType  sql.NullString
	Precision sql.NullInt64
	Length    sql.NullInt64
	Nullable  sql.NullBool
	Unique    sql.NullBool
}

// Parse parse sql schema
func Parse(dest interface{}) *SQLSchema {
	s := schema.Parse(dest)

	sqlSchema := &SQLSchema{
		Schema: s,
	}

	for _, f := range s.Fields {
		if f.IsNormal {
			field := &Field{Field: f}

			if s, ok := field.TagSettings["TYPE"]; ok {
				_ = field.DataType.Scan(s)
			}

			if num, ok := field.TagSettings["SIZE"]; ok {
				n, err := strconv.Atoi(num)
				if err != nil {
					fmt.Println(err)
				}
				_ = field.Length.Scan(n)
			}

			if num, ok := field.TagSettings["PRECISION"]; ok {
				n, err := strconv.Atoi(num)
				if err != nil {
					fmt.Println(err)
				}
				_ = field.Precision.Scan(n)
			}

			if _, ok := field.TagSettings["NOT NULL"]; !ok {
				_ = field.Nullable.Scan(true)
			}

			if _, ok := field.TagSettings["UNIQUE"]; ok {
				_ = field.Nullable.Scan(true)
			}

			sqlSchema.Fields = append(sqlSchema.Fields, field)
		}
	}

	return sqlSchema
}
