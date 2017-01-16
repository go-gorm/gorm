package gorm

import (
	"fmt"
	"reflect"

	"strings"
)

// Define callbacks for batch creating
func init() {
	DefaultCallback.CreateBatch().Register("gorm:begin_transaction", beginTransactionCallback)
	DefaultCallback.CreateBatch().Register("gorm:create_batch", createBatchCallback)
	DefaultCallback.CreateBatch().Register("gorm:commit_or_rollback_transaction", commitOrRollbackTransactionCallback)
}

// createCallback the callback used to insert data into database
func createBatchCallback(scope *Scope) {
	value := scope.IndirectValue()

	if value.Kind() != reflect.Slice {
		scope.Err(fmt.Errorf("createBatchCallback cannot be called for non-slice value, %+v given", value.Interface()))
		return
	}

	var (
		columns                      []string                        // one-dimensional array of strings containing columns
		blankColumnsWithDefaultValue []string                        // one-dimensional array of strings containing columns
		placeholders                 = make([][]string, value.Len()) // two-dimensional array of strings containing value placeholders
		structFields                 = scope.GetModelStruct().StructFields
	)

	// Filling up the columns
	for _, field := range fields(scope) {
		// We don't treat non-normal fields on batch operations (relationships, etc)
		if !field.IsNormal || field.IsIgnored {
			continue
		}

		if field.IsBlank && field.HasDefaultValue {
			blankColumnsWithDefaultValue = append(blankColumnsWithDefaultValue, scope.Quote(field.DBName))
			scope.InstanceSet("gorm:blank_columns_with_default_value", blankColumnsWithDefaultValue)
		} else if !field.IsPrimaryKey || !field.IsBlank {
			columns = append(columns, scope.Quote(field.DBName))
		}
	}

	// Filling up the placeholders
	for elementIndex := 0; elementIndex < value.Len(); elementIndex++ {
		valuePlaceholders := []string{}

		for _, structField := range structFields {
			// When inserting, the primary key is usually auto-increment
			if !structField.IsPrimaryKey && !structField.IsIgnored {
				fieldValue := reflect.Indirect(value.Index(elementIndex)).FieldByName(structField.Names[0]).Interface()
				valuePlaceholders = append(valuePlaceholders, scope.AddToVars(fieldValue))
			}
		}

		placeholders[elementIndex] = valuePlaceholders
	}

	var (
		returningColumn = "*"
		quotedTableName = scope.QuotedTableName()
		primaryField    = scope.PrimaryField()
		extraOption     string
	)

	if str, ok := scope.Get("gorm:insert_option"); ok {
		extraOption = fmt.Sprint(str)
	}

	if primaryField != nil {
		returningColumn = scope.Quote(primaryField.DBName)
	}

	lastInsertIDReturningSuffix := scope.Dialect().LastInsertIDReturningSuffix(quotedTableName, returningColumn)

	scope.Raw(fmt.Sprintf(
		"INSERT INTO %v (%v) VALUES %v%v%v",
		scope.QuotedTableName(),
		strings.Join(columns, ","),
		strings.Join(joinValuePlaceholders(placeholders), ","),
		addExtraSpaceIfExist(extraOption),
		addExtraSpaceIfExist(lastInsertIDReturningSuffix),
	))

	// Executing the query
	// TODO(drgomesp): Do we really need this check?
	if lastInsertIDReturningSuffix == "" || primaryField == nil {
		if result, err := scope.SQLDB().Exec(scope.SQL, scope.SQLVars...); scope.Err(err) == nil {
			// set rows affected count
			scope.db.RowsAffected, _ = result.RowsAffected()

			if firstInsertedID, err := result.LastInsertId(); scope.Err(err) == nil {
				fillPrimaryKeys(structFields, firstInsertedID, &value)
			}
		}
	}
}

func fillPrimaryKeys(structFields []*StructField, firstInsertedID int64, values *reflect.Value) {
	for _, structField := range structFields {
		for i := 0; i < values.Len(); i++ {
			field := reflect.Indirect(values.Index(i)).FieldByName(structField.Names[0])

			if field.IsValid() && field.CanSet() {
				if field.Kind() == reflect.Int64 || field.Kind() == reflect.Int32 || field.Kind() == reflect.Int8 || field.Kind() == reflect.Int {
					id := firstInsertedID + int64(i)

					if !field.OverflowInt(id) {
						field.SetInt(id)
					}
				}
			}
		}
	}
}

func joinValuePlaceholders(placeholders [][]string) []string {
	var valuePlaceholders []string

	for _, placeholder := range placeholders {
		valuePlaceholders = append(valuePlaceholders, fmt.Sprintf("(%s)", strings.Join(placeholder, ",")))
	}

	return valuePlaceholders
}

func fields(scope *Scope) []*Field {
	var (
		indirectScopeValue = scope.IndirectValue()
		structFields       = scope.GetModelStruct().StructFields
		fields             = make([]*Field, len(structFields))
	)

	for i, structField := range structFields {
		fieldValue := reflect.Indirect(indirectScopeValue.Index(0)).FieldByName(structField.Names[0])
		fields[i] = &Field{StructField: structField, Field: fieldValue, IsBlank: isBlank(fieldValue)}
	}

	return fields
}
