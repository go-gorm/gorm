package callbacks

import (
	"reflect"
	"sort"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ConvertMapToValuesForCreate builds one INSERT row from mapValue. It sorts input keys,
// maps recognized model field names to database columns, and honors Select/Omit and
// create permissions. Explicit nil and zero values are retained. It does not fill model
// defaults or timestamps; supply at most one key per database column to avoid duplicate
// columns.
func ConvertMapToValuesForCreate(stmt *gorm.Statement, mapValue map[string]interface{}) (values clause.Values) {
	values.Columns = make([]clause.Column, 0, len(mapValue))
	selectColumns, restricted := stmt.SelectAndOmitColumns(true, false)

	keys := make([]string, 0, len(mapValue))
	for k := range mapValue {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		value := mapValue[k]
		if stmt.Schema != nil {
			if field := stmt.Schema.LookUpField(k); field != nil {
				k = field.DBName
			}
		}

		if v, ok := selectColumns[k]; (ok && v) || (!ok && !restricted) {
			values.Columns = append(values.Columns, clause.Column{Name: k})
			if len(values.Values) == 0 {
				values.Values = [][]interface{}{{}}
			}

			values.Values[0] = append(values.Values[0], value)
		}
	}
	return
}

// ConvertSliceOfMapToValuesForCreate builds INSERT rows with a shared, sorted column
// set from mapValues, honoring schema mappings and create selections. A key missing in
// a row becomes nil in that column, not a database DEFAULT expression. An empty slice
// records ErrEmptySlice. It does not fill model defaults or timestamps; avoid multiple
// keys mapping to the same column.
func ConvertSliceOfMapToValuesForCreate(stmt *gorm.Statement, mapValues []map[string]interface{}) (values clause.Values) {
	columns := make([]string, 0, len(mapValues))

	// when the length of mapValues is zero,return directly here
	// no need to call stmt.SelectAndOmitColumns method
	if len(mapValues) == 0 {
		stmt.AddError(gorm.ErrEmptySlice)
		return
	}

	var (
		result                    = make(map[string][]interface{}, len(mapValues))
		selectColumns, restricted = stmt.SelectAndOmitColumns(true, false)
	)

	for idx, mapValue := range mapValues {
		for k, v := range mapValue {
			if stmt.Schema != nil {
				if field := stmt.Schema.LookUpField(k); field != nil {
					k = field.DBName
				}
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
	values.Columns = make([]clause.Column, len(columns))
	for idx, column := range columns {
		values.Columns[idx] = clause.Column{Name: column}

		for i, v := range result[column] {
			if len(values.Values[i]) == 0 {
				values.Values[i] = make([]interface{}, len(columns))
			}

			values.Values[i][idx] = v
		}
	}
	return
}

// hasReturning reports whether the dialector supports RETURNING and the statement
// contains that clause. Returning all columns uses the normal scan mode; a non-wildcard
// projection uses ScanUpdate to preserve destination fields not returned.
func hasReturning(tx *gorm.DB, supportReturning bool) (bool, gorm.ScanMode) {
	if supportReturning {
		if c, ok := tx.Statement.Clauses["RETURNING"]; ok {
			returning, _ := c.Expression.(clause.Returning)
			if len(returning.Columns) == 0 || (len(returning.Columns) == 1 && returning.Columns[0].Name == "*") {
				return true, 0
			}
			return true, gorm.ScanUpdate
		}
	}
	return false, 0
}

// checkMissingWhereConditions records ErrMissingWhereClause when no qualifying WHERE
// clause is present, unless AllowGlobalUpdate is enabled or db already has an error.
// When soft deletion added its own predicate, that predicate alone does not satisfy the
// guard. This structural check does not determine whether a supplied condition is
// selective or safe.
func checkMissingWhereConditions(db *gorm.DB) {
	if !db.AllowGlobalUpdate && db.Error == nil {
		where, withCondition := db.Statement.Clauses["WHERE"]
		if withCondition {
			if _, withSoftDelete := db.Statement.Clauses["soft_delete_enabled"]; withSoftDelete {
				whereClause, _ := where.Expression.(clause.Where)
				withCondition = len(whereClause.Exprs) > 1
			}
		}
		if !withCondition {
			db.AddError(gorm.ErrMissingWhereClause)
		}
		return
	}
}

// visitMap records addressable model identities while traversing association values. It
// prevents recursion through already visited objects, not duplicate database operations
// across independent requests.
type visitMap = map[reflect.Value]bool

// loadOrStoreVisitMap checks and records addressable struct/interface values after one
// pointer dereference. For slices and arrays it visits every element and returns true
// only if all elements were already present (including an empty collection).
// Unaddressable values cannot be remembered. The map is shared within one association
// traversal and is not concurrency-safe.
func loadOrStoreVisitMap(visitMap *visitMap, v reflect.Value) (loaded bool) {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		loaded = true
		for i := 0; i < v.Len(); i++ {
			if !loadOrStoreVisitMap(visitMap, v.Index(i)) {
				loaded = false
			}
		}
	case reflect.Struct, reflect.Interface:
		if v.CanAddr() {
			p := v.Addr()
			if _, ok := (*visitMap)[p]; ok {
				return true
			}
			(*visitMap)[p] = true
		}
	}

	return
}
