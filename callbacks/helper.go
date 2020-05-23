package callbacks

import (
	"sort"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
)

// SelectAndOmitColumns get select and omit columns, select -> true, omit -> false
func SelectAndOmitColumns(stmt *gorm.Statement, requireCreate, requireUpdate bool) (map[string]bool, bool) {
	results := map[string]bool{}
	notRestricted := false

	// select columns
	for _, column := range stmt.Selects {
		if column == "*" {
			notRestricted = true
			for _, dbName := range stmt.Schema.DBNames {
				results[dbName] = true
			}
			break
		}

		if field := stmt.Schema.LookUpField(column); field != nil {
			results[field.DBName] = true
		} else {
			results[column] = true
		}
	}

	// omit columns
	for _, omit := range stmt.Omits {
		if field := stmt.Schema.LookUpField(omit); field != nil {
			results[field.DBName] = false
		} else {
			results[omit] = false
		}
	}

	if stmt.Schema != nil {
		for _, field := range stmt.Schema.Fields {
			name := field.DBName
			if name == "" {
				name = field.Name
			}

			if requireCreate && !field.Creatable {
				results[name] = false
			} else if requireUpdate && !field.Updatable {
				results[name] = false
			}
		}
	}

	return results, !notRestricted && len(stmt.Selects) > 0
}

// ConvertMapToValuesForCreate convert map to values
func ConvertMapToValuesForCreate(stmt *gorm.Statement, mapValue map[string]interface{}) (values clause.Values) {
	columns := make([]string, 0, len(mapValue))
	selectColumns, restricted := SelectAndOmitColumns(stmt, true, false)

	var keys []string
	for k, _ := range mapValue {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		value := mapValue[k]
		if field := stmt.Schema.LookUpField(k); field != nil {
			k = field.DBName
		}

		if v, ok := selectColumns[k]; (ok && v) || (!ok && !restricted) {
			columns = append(columns, k)
			values.Values[0] = append(values.Values[0], value)
		}
	}
	return
}

// ConvertSliceOfMapToValuesForCreate convert slice of map to values
func ConvertSliceOfMapToValuesForCreate(stmt *gorm.Statement, mapValues []map[string]interface{}) (values clause.Values) {
	var (
		columns                   = []string{}
		result                    = map[string][]interface{}{}
		selectColumns, restricted = SelectAndOmitColumns(stmt, true, false)
	)

	for idx, mapValue := range mapValues {
		for k, v := range mapValue {
			if field := stmt.Schema.LookUpField(k); field != nil {
				k = field.DBName
			}

			if _, ok := result[k]; !ok {
				if v, ok := selectColumns[k]; (ok && v) || (!ok && !restricted) {
					result[k] = make([]interface{}, len(mapValues))
					columns = append(columns, k)
				} else {
					continue
				}
			}

			result[k][idx] = v
		}
	}

	sort.Strings(columns)
	values.Values = make([][]interface{}, len(mapValues))
	for idx, column := range columns {
		for i, v := range result[column] {
			if i == 0 {
				values.Values[i] = make([]interface{}, len(columns))
			}
			values.Values[i][idx] = v
		}
	}
	return
}
