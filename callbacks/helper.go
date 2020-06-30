package callbacks

import (
	"sort"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ConvertMapToValuesForCreate convert map to values
func ConvertMapToValuesForCreate(stmt *gorm.Statement, mapValue map[string]interface{}) (values clause.Values) {
	columns := make([]string, 0, len(mapValue))
	selectColumns, restricted := stmt.SelectAndOmitColumns(true, false)

	var keys []string
	for k := range mapValue {
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
		selectColumns, restricted = stmt.SelectAndOmitColumns(true, false)
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
