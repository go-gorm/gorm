package callbacks

import (
	"sort"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
)

// SelectAndOmitColumns get select and omit columns, select -> true, omit -> false
func SelectAndOmitColumns(stmt *gorm.Statement) (map[string]bool, bool) {
	results := map[string]bool{}

	// select columns
	for _, column := range stmt.Selects {
		if column == "*" {
			for _, dbName := range stmt.Schema.DBNames {
				results[dbName] = true
			}
			return results, true
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

	return results, len(stmt.Selects) > 0
}

// ConvertMapToValues convert map to values
func ConvertMapToValues(stmt *gorm.Statement, mapValue map[string]interface{}) (values clause.Values) {
	columns := make([]string, 0, len(mapValue))
	selectColumns, restricted := SelectAndOmitColumns(stmt)

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

// ConvertSliceOfMapToValues convert slice of map to values
func ConvertSliceOfMapToValues(stmt *gorm.Statement, mapValues []map[string]interface{}) (values clause.Values) {
	var (
		columns                   = []string{}
		result                    = map[string][]interface{}{}
		selectColumns, restricted = SelectAndOmitColumns(stmt)
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
