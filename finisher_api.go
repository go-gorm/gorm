package gorm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"hash/maphash"
	"reflect"
	"strings"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
)

// Create inserts value and returns a result whose Error and RowsAffected must be
// checked.
// Pass a pointer to a model or slice of models so generated primary keys and automatic
// timestamps can be written back. It runs create hooks and saves included associations;
// Select, Omit, and Session options control those behaviors. A positive CreateBatchSize
// splits slices into batches via CreateInBatches.
//
// Map inserts require Model or Table and do not run model hooks or save associations.
// Generated IDs may be written into the maps when the driver and statement support
// it; they do not populate a separate model instance supplied to Model.
// See https://gorm.io/docs/create.html.
func (db *DB) Create(value interface{}) (tx *DB) {
	if db.CreateBatchSize > 0 {
		return db.CreateInBatches(value, db.CreateBatchSize)
	}

	tx = db.getInstance()
	tx.Statement.Dest = value
	return tx.callbacks.Create().Execute(tx)
}

// CreateInBatches inserts a slice or array in chunks of at most batchSize records.
// batchSize must be positive. Generated values are copied back into the input elements,
// and RowsAffected accumulates successful batches. A failure stops later batches.
// When multiple batches are needed, a transaction wraps them unless
// SkipDefaultTransaction
// is enabled; without it earlier batches may persist after an error. For a non-slice
// value, this executes one normal create callback chain.
//
// RowsAffected counts batches reported successful before an error; it can include
// writes subsequently rolled back by an encompassing transaction. It is not a
// persisted-row count after rollback.
func (db *DB) CreateInBatches(value interface{}, batchSize int) (tx *DB) {
	reflectValue := reflect.Indirect(reflect.ValueOf(value))

	switch reflectValue.Kind() {
	case reflect.Slice, reflect.Array:
		var rowsAffected int64
		tx = db.getInstance()

		// the reflection length judgment of the optimized value
		reflectLen := reflectValue.Len()

		callFc := func(tx *DB) error {
			for i := 0; i < reflectLen; i += batchSize {
				ends := i + batchSize
				if ends > reflectLen {
					ends = reflectLen
				}

				subtx := tx.getInstance()
				batchSlice := reflect.New(reflectValue.Type())
				batchSlice.Elem().Set(reflectValue.Slice(i, ends))
				subtx.Statement.Dest = batchSlice.Interface()

				subtx.callbacks.Create().Execute(subtx)

				if subtx.Error != nil {
					return subtx.Error
				}

				resultSlice := reflect.Indirect(batchSlice)
				for j := 0; j < resultSlice.Len(); j++ {
					reflectValue.Index(i + j).Set(resultSlice.Index(j))
				}

				rowsAffected += subtx.RowsAffected
			}
			return nil
		}

		if tx.SkipDefaultTransaction || reflectLen <= batchSize {
			tx.AddError(callFc(tx.Session(&Session{})))
		} else {
			tx.AddError(tx.Transaction(callFc))
		}

		tx.RowsAffected = rowsAffected
	default:
		tx = db.getInstance()
		tx.Statement.Dest = value
		tx = tx.callbacks.Create().Execute(tx)
	}
	return
}

// Save persists all fields of value, including zero values, subject to field
// permissions.
// For a struct with a zero primary-key field it inserts. Otherwise it updates; when
// no explicit selection was supplied and the update affects no rows, it falls back
// to Create with an update-all conflict clause and skips hooks for that fallback.
// Slices use create callbacks with an update-all conflict clause unless one exists.
//
// Use Updates with an explicit condition when insertion is not intended. Do not
// combine Save with Model; that combination has undefined behavior. Save is absent
// from the generics API. See https://gorm.io/docs/update.html#Save-All-Fields.
func (db *DB) Save(value interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Dest = value

	reflectValue := reflect.Indirect(reflect.ValueOf(value))
	for reflectValue.Kind() == reflect.Ptr || reflectValue.Kind() == reflect.Interface {
		reflectValue = reflect.Indirect(reflectValue)
	}

	switch reflectValue.Kind() {
	case reflect.Slice, reflect.Array:
		if _, ok := tx.Statement.Clauses["ON CONFLICT"]; !ok {
			tx = tx.Clauses(clause.OnConflict{UpdateAll: true})
		}
		tx = tx.callbacks.Create().Execute(tx.Set("gorm:update_track_time", true))
	case reflect.Struct:
		if err := tx.Statement.Parse(value); err == nil && tx.Statement.Schema != nil {
			for _, pf := range tx.Statement.Schema.PrimaryFields {
				if _, isZero := pf.ValueOf(tx.Statement.Context, reflectValue); isZero {
					return tx.callbacks.Create().Execute(tx)
				}
			}
		}

		fallthrough
	default:
		selectedUpdate := len(tx.Statement.Selects) != 0
		// when updating, use all fields including those zero-value fields
		if !selectedUpdate {
			tx.Statement.Selects = append(tx.Statement.Selects, "*")
		}

		updateTx := tx.callbacks.Update().Execute(tx.Session(&Session{Initialized: true}))

		if updateTx.Error == nil && updateTx.RowsAffected == 0 && !updateTx.DryRun && !selectedUpdate {
			return tx.Session(&Session{SkipHooks: true}).Clauses(clause.OnConflict{UpdateAll: true}).Create(value)
		}

		return updateTx
	}

	return
}

// First retrieves at most one row, adding ascending primary-key ordering and any conds.
// It writes to dest and sets Error to ErrRecordNotFound when no row matches; test
// with errors.Is. Existing conditions and a non-zero destination primary key can
// also restrict the query. Pass a writable destination pointer and bind external
// values, for example First(&user, "id = ?", id). See Take for unordered lookup.
func (db *DB) First(dest interface{}, conds ...interface{}) (tx *DB) {
	tx = db.Limit(1).Order(clause.OrderByColumn{
		Column: clause.Column{Table: clause.CurrentTable, Name: clause.PrimaryKey},
	})
	if len(conds) > 0 {
		if exprs := tx.Statement.BuildCondition(conds[0], conds[1:]...); len(exprs) > 0 {
			tx.Statement.AddClause(clause.Where{Exprs: exprs})
		}
	}
	tx.Statement.RaiseErrorOnNotFound = true
	tx.Statement.Dest = dest
	return tx.callbacks.Query().Execute(tx)
}

// Take retrieves at most one row without adding an order, applying any conds and
// existing conditions. An existing Order is retained. It writes to a destination
// pointer and returns ErrRecordNotFound through Error when no row matches.
// Use First or an explicit Order when the selected row must be deterministic.
func (db *DB) Take(dest interface{}, conds ...interface{}) (tx *DB) {
	tx = db.Limit(1)
	if len(conds) > 0 {
		if exprs := tx.Statement.BuildCondition(conds[0], conds[1:]...); len(exprs) > 0 {
			tx.Statement.AddClause(clause.Where{Exprs: exprs})
		}
	}
	tx.Statement.RaiseErrorOnNotFound = true
	tx.Statement.Dest = dest
	return tx.callbacks.Query().Execute(tx)
}

// Last retrieves at most one row, adding descending primary-key ordering and any conds.
// It writes to a destination pointer and sets Error to ErrRecordNotFound if there
// is no match. It does not mean most recently created unless the key order has that
// meaning; use an explicit timestamp order when that is the requirement.
func (db *DB) Last(dest interface{}, conds ...interface{}) (tx *DB) {
	tx = db.Limit(1).Order(clause.OrderByColumn{
		Column: clause.Column{Table: clause.CurrentTable, Name: clause.PrimaryKey},
		Desc:   true,
	})
	if len(conds) > 0 {
		if exprs := tx.Statement.BuildCondition(conds[0], conds[1:]...); len(exprs) > 0 {
			tx.Statement.AddClause(clause.Where{Exprs: exprs})
		}
	}
	tx.Statement.RaiseErrorOnNotFound = true
	tx.Statement.Dest = dest
	return tx.callbacks.Query().Execute(tx)
}

// Find retrieves rows matching the current statement and optional conds into dest.
// Pass a pointer to a slice for multiple rows or a pointer to a struct for one result.
// Unlike First, Take, and Last, an empty result normally has nil Error and zero
// RowsAffected. A struct destination alone does not add LIMIT 1; use a single-row
// finisher when that is required. See https://gorm.io/docs/query.html.
func (db *DB) Find(dest interface{}, conds ...interface{}) (tx *DB) {
	tx = db.getInstance()
	if len(conds) > 0 {
		if exprs := tx.Statement.BuildCondition(conds[0], conds[1:]...); len(exprs) > 0 {
			tx.Statement.AddClause(clause.Where{Exprs: exprs})
		}
	}
	tx.Statement.Dest = dest
	return tx.callbacks.Query().Execute(tx)
}

// FindInBatches reads matching records into dest in batches and invokes fc for each
// non-empty batch. dest must be a pointer to a model slice and batchSize must be
// positive. Batch numbers start at 1; the callback's RowsAffected is that batch's size,
// while the returned handle accumulates rows across batches.
//
// Iteration advances with a greater-than predicate on the last record's prioritized
// primary key. Keep the traversal ordered by that key ascending: conflicting existing
// Order clauses, including descending key order, can skip or repeat records. Include
// the key when selecting columns, and do not change it in dest before the callback
// returns. A missing or zero continuation key produces ErrPrimaryKeyRequired.
//
// An error from the query or fc stops iteration. No transaction or snapshot is created
// automatically; concurrent writes can affect which rows are observed.
func (db *DB) FindInBatches(dest interface{}, batchSize int, fc func(tx *DB, batch int) error) *DB {
	var (
		tx = db.Order(clause.OrderByColumn{
			Column: clause.Column{Table: clause.CurrentTable, Name: clause.PrimaryKey},
		}).Session(&Session{})
		queryDB      = tx
		rowsAffected int64
		batch        int
	)

	// user specified offset or limit
	var totalSize int
	if c, ok := tx.Statement.Clauses["LIMIT"]; ok {
		if limit, ok := c.Expression.(clause.Limit); ok {
			if limit.Limit != nil {
				totalSize = *limit.Limit
			}

			if totalSize > 0 && batchSize > totalSize {
				batchSize = totalSize
			}

			// reset to offset to 0 in next batch
			tx = tx.Offset(-1).Session(&Session{})
		}
	}

	for {
		result := queryDB.Limit(batchSize).Find(dest)
		rowsAffected += result.RowsAffected
		batch++

		if result.Error == nil && result.RowsAffected != 0 {
			fcTx := result.Session(&Session{NewDB: true})
			fcTx.RowsAffected = result.RowsAffected
			tx.AddError(fc(fcTx, batch))
		} else if result.Error != nil {
			tx.AddError(result.Error)
		}

		if tx.Error != nil || int(result.RowsAffected) < batchSize {
			break
		}

		if totalSize > 0 {
			if totalSize <= int(rowsAffected) {
				break
			}
			if totalSize/batchSize == batch {
				batchSize = totalSize % batchSize
			}
		}

		// Optimize for-break
		resultsValue := reflect.Indirect(reflect.ValueOf(dest))
		if result.Statement.Schema.PrioritizedPrimaryField == nil {
			tx.AddError(ErrPrimaryKeyRequired)
			break
		}

		primaryValue, zero := result.Statement.Schema.PrioritizedPrimaryField.ValueOf(tx.Statement.Context, resultsValue.Index(resultsValue.Len()-1))
		if zero {
			tx.AddError(ErrPrimaryKeyRequired)
			break
		}
		queryDB = tx.Clauses(clause.Gt{Column: clause.Column{Table: clause.CurrentTable, Name: clause.PrimaryKey}, Value: primaryValue})
	}

	tx.RowsAffected = rowsAffected
	return tx
}

// assignInterfacesToValue applies equality conditions and non-zero struct fields to
// the current destination through schema setters. FirstOrInit and FirstOrCreate use
// it to seed records; arbitrary SQL predicates cannot be converted into field values.
func (db *DB) assignInterfacesToValue(values ...interface{}) {
	for _, value := range values {
		switch v := value.(type) {
		case []clause.Expression:
			for _, expr := range v {
				if eq, ok := expr.(clause.Eq); ok {
					switch column := eq.Column.(type) {
					case string:
						if field := db.Statement.Schema.LookUpField(column); field != nil {
							db.AddError(field.Set(db.Statement.Context, db.Statement.ReflectValue, eq.Value))
						}
					case clause.Column:
						if field := db.Statement.Schema.LookUpField(column.Name); field != nil {
							db.AddError(field.Set(db.Statement.Context, db.Statement.ReflectValue, eq.Value))
						}
					}
				} else if andCond, ok := expr.(clause.AndConditions); ok {
					db.assignInterfacesToValue(andCond.Exprs)
				}
			}
		case clause.Expression, map[string]string, map[interface{}]interface{}, map[string]interface{}:
			if exprs := db.Statement.BuildCondition(value); len(exprs) > 0 {
				db.assignInterfacesToValue(exprs)
			}
		default:
			if s, err := schema.Parse(value, db.cacheStore, db.NamingStrategy); err == nil {
				reflectValue := reflect.Indirect(reflect.ValueOf(value))
				switch reflectValue.Kind() {
				case reflect.Struct:
					for _, f := range s.Fields {
						if f.Readable {
							if v, isZero := f.ValueOf(db.Statement.Context, reflectValue); !isZero {
								if field := db.Statement.Schema.LookUpField(f.Name); field != nil {
									db.AddError(field.Set(db.Statement.Context, db.Statement.ReflectValue, v))
								}
							}
						}
					}
				}
			} else if len(values) > 0 {
				if exprs := db.Statement.BuildCondition(values[0], values[1:]...); len(exprs) > 0 {
					db.assignInterfacesToValue(exprs)
				}
				return
			}
		}
	}
}

// FirstOrInit queries for the first matching row, or initializes dest in memory
// when none matches. Use struct or map equality conditions for fields that must be
// copied into a new value; arbitrary SQL expressions cannot initialize those fields.
// Attrs applies only on a miss; Assign applies in both cases. This method does not
// insert or update database records. Check Error for query failures and RowsAffected
// for whether a database row was found.
func (db *DB) FirstOrInit(dest interface{}, conds ...interface{}) (tx *DB) {
	queryTx := db.Limit(1).Order(clause.OrderByColumn{
		Column: clause.Column{Table: clause.CurrentTable, Name: clause.PrimaryKey},
	})

	if tx = queryTx.Find(dest, conds...); tx.RowsAffected == 0 {
		if c, ok := tx.Statement.Clauses["WHERE"]; ok {
			if where, ok := c.Expression.(clause.Where); ok {
				tx.assignInterfacesToValue(where.Exprs)
			}
		}

		// initialize with attrs, conds
		if len(tx.Statement.attrs) > 0 {
			tx.assignInterfacesToValue(tx.Statement.attrs...)
		}
	}

	// initialize with attrs, conds
	if len(tx.Statement.assigns) > 0 {
		tx.assignInterfacesToValue(tx.Statement.assigns...)
	}
	return
}

// FirstOrCreate queries for a matching record and inserts dest when none is found.
// Use struct or map equality conditions to initialize a missing record. Attrs applies
// only on a miss; Assign also updates an existing row. Always check the returned Error.
//
// The lookup and possible insertion are separate operations, not an atomic upsert.
// Use a database unique constraint and appropriate conflict handling for concurrent
// callers. See clause.OnConflict and
// https://gorm.io/docs/advanced_query.html#FirstOrCreate.
func (db *DB) FirstOrCreate(dest interface{}, conds ...interface{}) (tx *DB) {
	tx = db.getInstance()
	queryTx := db.Session(&Session{}).Limit(1).Order(clause.OrderByColumn{
		Column: clause.Column{Table: clause.CurrentTable, Name: clause.PrimaryKey},
	})

	result := queryTx.Find(dest, conds...)
	if result.Error != nil {
		tx.Error = result.Error
		return tx
	}

	if result.RowsAffected == 0 {
		if c, ok := result.Statement.Clauses["WHERE"]; ok {
			if where, ok := c.Expression.(clause.Where); ok {
				result.assignInterfacesToValue(where.Exprs)
			}
		}

		// initialize with attrs, conds
		if len(db.Statement.attrs) > 0 {
			result.assignInterfacesToValue(db.Statement.attrs...)
		}

		// initialize with attrs, conds
		if len(db.Statement.assigns) > 0 {
			result.assignInterfacesToValue(db.Statement.assigns...)
		}

		return tx.Create(dest)
	} else if len(db.Statement.assigns) > 0 {
		exprs := tx.Statement.BuildCondition(db.Statement.assigns[0], db.Statement.assigns[1:]...)
		assigns := map[string]interface{}{}
		for i := 0; i < len(exprs); i++ {
			expr := exprs[i]

			if eq, ok := expr.(clause.AndConditions); ok {
				exprs = append(exprs, eq.Exprs...)
			} else if eq, ok := expr.(clause.Eq); ok {
				switch column := eq.Column.(type) {
				case string:
					assigns[column] = eq.Value
				case clause.Column:
					assigns[column.Name] = eq.Value
				}
			}
		}

		return tx.Model(dest).Updates(assigns)
	}

	return tx
}

// Update writes one column for rows selected by Model and the current conditions.
// It runs update hooks and automatic update-time handling. A non-zero model primary
// key restricts the update; without a condition the global-update guard returns
// ErrMissingWhereClause unless AllowGlobalUpdate is enabled.
//
//	result := db.Model(&User{}).Where("id = ?", id).Update("active", false)
//
// Check result.Error and result.RowsAffected. Use UpdateColumn to skip hooks and
// timestamps.
func (db *DB) Update(column string, value interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Dest = map[string]interface{}{column: value}
	return tx.callbacks.Update().Execute(tx)
}

// Updates writes fields from a struct or map using update callbacks.
// Struct inputs omit zero values unless fields are explicitly selected; map inputs
// include false, 0, and empty strings. Field permissions and Select/Omit still apply.
// Hooks and automatic update timestamps run unless disabled. Model and WHERE define
// target rows; a missing condition normally produces ErrMissingWhereClause.
//
//	db.Model(&user).Updates(map[string]interface{}{"active": false, "age": 0})
//
// Check the returned Error and RowsAffected. See https://gorm.io/docs/update.html.
func (db *DB) Updates(values interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Dest = values
	return tx.callbacks.Update().Execute(tx)
}

// UpdateColumn updates one column while skipping model hooks and automatic update
// timestamps.
// Model, conditions, field permissions, and the global-update guard still apply.
// The value may be an Expr for a database-side calculation. Check Error on the result.
func (db *DB) UpdateColumn(column string, value interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Dest = map[string]interface{}{column: value}
	tx.Statement.SkipHooks = true
	return tx.callbacks.Update().Execute(tx)
}

// UpdateColumns updates a struct or map while skipping model hooks and automatic
// update timestamps. Like Updates, struct zero values are omitted unless selected;
// use a map to write explicit zero values. Conditions and the global-update guard
// still apply. This is not a bypass for field permissions or database constraints.
func (db *DB) UpdateColumns(values interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Dest = values
	tx.Statement.SkipHooks = true
	return tx.callbacks.Update().Execute(tx)
}

// Delete removes rows selected by value's primary key, existing conditions, and conds.
// A field whose type installs soft-delete clauses, such as gorm.DeletedAt, normally
// changes deletion into an UPDATE setting its deletion time; Unscoped().Delete performs
// a physical DELETE. A field merely named DeletedAt, such as *time.Time, does not
// enable soft deletion. Hooks and selected association handling may run.
//
// A missing user condition normally returns ErrMissingWhereClause, including for soft
// deletion. Check Error and RowsAffected; deleting no rows is not ErrRecordNotFound.
// See https://gorm.io/docs/delete.html.
func (db *DB) Delete(value interface{}, conds ...interface{}) (tx *DB) {
	tx = db.getInstance()
	if len(conds) > 0 {
		if exprs := tx.Statement.BuildCondition(conds[0], conds[1:]...); len(exprs) > 0 {
			tx.Statement.AddClause(clause.Where{Exprs: exprs})
		}
	}
	tx.Statement.Dest = value
	return tx.callbacks.Delete().Execute(tx)
}

// Count writes the number of matching records to count using the current model or
// table.
// The default expression is count(*); Select of a single column counts non-NULL
// values and Distinct can count unique values of that column. With Group, count is
// the number of returned groups. Pass a non-nil *int64 and check the returned Error.
// Count changes the projection for execution; the scalar result is in *count,
// not generally in RowsAffected.
//
// Column counting recognizes a single column-like selection; it is not a general SQL
// expression parser. A selection beginning with count( is retained as supplied, while
// unsupported selections can fall back to count(*). Use an explicit trusted count(...)
// selection when counting an expression.
func (db *DB) Count(count *int64) (tx *DB) {
	tx = db.getInstance()
	if tx.Statement.Model == nil {
		tx.Statement.Model = tx.Statement.Dest
		defer func() {
			tx.Statement.Model = nil
		}()
	}

	if selectClause, ok := db.Statement.Clauses["SELECT"]; ok {
		defer func() {
			tx.Statement.Clauses["SELECT"] = selectClause
		}()
	} else {
		defer delete(tx.Statement.Clauses, "SELECT")
	}

	if len(tx.Statement.Selects) == 0 {
		tx.Statement.AddClause(clause.Select{Expression: clause.Expr{SQL: "count(*)"}})
	} else if !strings.HasPrefix(strings.TrimSpace(strings.ToLower(tx.Statement.Selects[0])), "count(") {
		expr := clause.Expr{SQL: "count(*)"}

		if len(tx.Statement.Selects) == 1 {
			dbName := tx.Statement.Selects[0]
			fields := strings.FieldsFunc(dbName, utils.IsInvalidDBNameChar)
			if len(fields) == 1 || (len(fields) == 3 && (strings.ToUpper(fields[1]) == "AS" || fields[1] == ".")) {
				if tx.Statement.Parse(tx.Statement.Model) == nil {
					if f := tx.Statement.Schema.LookUpField(dbName); f != nil {
						dbName = f.DBName
					}
				}

				if tx.Statement.Distinct {
					expr = clause.Expr{SQL: "COUNT(DISTINCT(?))", Vars: []interface{}{clause.Column{Name: dbName}}}
				} else if dbName != "*" {
					expr = clause.Expr{SQL: "COUNT(?)", Vars: []interface{}{clause.Column{Name: dbName}}}
				}
			}
		}

		tx.Statement.AddClause(clause.Select{Expression: expr})
	}

	if orderByClause, ok := db.Statement.Clauses["ORDER BY"]; ok {
		if _, ok := db.Statement.Clauses["GROUP BY"]; !ok {
			delete(tx.Statement.Clauses, "ORDER BY")
			defer func() {
				tx.Statement.Clauses["ORDER BY"] = orderByClause
			}()
		}
	}

	tx.Statement.Dest = count
	tx = tx.callbacks.Query().Execute(tx)

	if _, ok := db.Statement.Clauses["GROUP BY"]; ok || tx.RowsAffected != 1 {
		*count = tx.RowsAffected
	}

	return
}

// Row executes the row-query callback and returns a *sql.Row for manual scanning.
// Call Scan to retrieve values and observe database errors, including sql.ErrNoRows.
// It does not run the normal model query hooks or preloading. DryRun cannot produce
// a row; construction failures may also leave a nil result. Use Rows when an explicit
// error return and row iteration are required.
func (db *DB) Row() *sql.Row {
	tx := db.getInstance().Set("rows", false)
	tx = tx.callbacks.Row().Execute(tx)
	row, ok := tx.Statement.Dest.(*sql.Row)
	if !ok && tx.DryRun {
		db.Logger.Error(tx.Statement.Context, ErrDryRunModeUnsupported.Error())
	}
	return row
}

// Rows executes the row-query callback and returns rows for manual iteration.
// Check the error before use, defer rows.Close(), and check rows.Err() after the loop.
// Use ScanRows to map a positioned row into a model. This path does not run normal
// query hooks or preloading. DryRun returns ErrDryRunModeUnsupported.
func (db *DB) Rows() (*sql.Rows, error) {
	tx := db.getInstance().Set("rows", true)
	tx = tx.callbacks.Row().Execute(tx)
	rows, ok := tx.Statement.Dest.(*sql.Rows)
	if !ok && tx.DryRun && tx.Error == nil {
		tx.Error = ErrDryRunModeUnsupported
	}
	return rows, tx.Error
}

// Scan executes the query and maps its selected columns into dest, which may be a
// model, projection struct, slice, or supported scalar pointer. Use Table or Model
// when the destination type does not identify the source table, or build SQL with Raw.
// Scan closes its rows internally. Check Error and RowsAffected; an empty result is
// not automatically ErrRecordNotFound. It uses row callbacks, not normal query
// preloading.
func (db *DB) Scan(dest interface{}) (tx *DB) {
	config := *db.Config
	currentLogger, newLogger := config.Logger, logger.Recorder.New()
	config.Logger = newLogger

	tx = db.getInstance()
	tx.Config = &config

	if rows, err := tx.Rows(); err == nil {
		defer func() {
			if err := rows.Close(); err != nil {
				_ = tx.AddError(err)
			}
		}()

		if rows.Next() {
			tx.ScanRows(rows, dest)
		} else {
			tx.RowsAffected = 0
			tx.AddError(rows.Err())
		}
	}

	currentLogger.Trace(tx.Statement.Context, newLogger.BeginAt, func() (string, int64) {
		return newLogger.SQL, tx.RowsAffected
	}, tx.Error)
	tx.Logger = currentLogger
	return
}

// Pluck reads one column into a destination slice pointer using the current model or
// table.
// Use Distinct for unique values; use Select and Scan for multiple columns.
// The result reports Error and RowsAffected. Supply a trusted column name or
// expression.
//
//	var names []string
//	result := db.Model(&User{}).Where("active = ?", true).Pluck("name", &names)
func (db *DB) Pluck(column string, dest interface{}) (tx *DB) {
	tx = db.getInstance()
	if tx.Statement.Model != nil {
		if tx.Statement.Parse(tx.Statement.Model) == nil {
			if f := tx.Statement.Schema.LookUpField(column); f != nil {
				column = f.DBName
			}
		}
	}

	if len(tx.Statement.Selects) != 1 {
		fields := strings.FieldsFunc(column, utils.IsInvalidDBNameChar)
		tx.Statement.AddClauseIfNotExists(clause.Select{
			Distinct: tx.Statement.Distinct,
			Columns:  []clause.Column{{Name: column, Raw: len(fields) != 1}},
		})
	}
	tx.Statement.Dest = dest
	return tx.callbacks.Query().Execute(tx)
}

// ScanRows maps rows, already positioned by a successful Next call, into dest.
// For a struct destination it consumes the current row; slice scanning can consume
// remaining rows. The caller retains responsibility for Close and iteration errors.
// It performs schema-based conversion without executing SQL or model query hooks.
//
//	for rows.Next() {
//		var user User
//		if err := db.ScanRows(rows, &user); err != nil { /* handle err */ }
//	}
func (db *DB) ScanRows(rows *sql.Rows, dest interface{}) error {
	tx := db.getInstance()
	if err := tx.Statement.Parse(dest); !errors.Is(err, schema.ErrUnsupportedDataType) {
		tx.AddError(err)
	}
	tx.Statement.Dest = dest
	tx.Statement.ReflectValue = reflect.ValueOf(dest)
	for tx.Statement.ReflectValue.Kind() == reflect.Ptr {
		elem := tx.Statement.ReflectValue.Elem()
		if !elem.IsValid() {
			elem = reflect.New(tx.Statement.ReflectValue.Type().Elem())
			tx.Statement.ReflectValue.Set(elem)
		}
		tx.Statement.ReflectValue = elem
	}
	Scan(rows, tx, ScanInitialized)
	return tx.Error
}

// Connection acquires one *sql.Conn for the duration of fc and returns it to the pool
// after fc returns or panics. Use the callback's tx for every operation that must use
// that connection. It does not begin a transaction or make the operations atomic;
// use Transaction for transactional work. The returned error is the acquisition or
// callback error. Do not retain tx beyond the callback.
//
// Connection acquires a connection from the underlying pool even when invoked on an
// existing transaction handle; the callback does not participate in that transaction.
// Continue using the existing transaction handle directly when its transactional scope
// must be preserved.
func (db *DB) Connection(fc func(tx *DB) error) (err error) {
	if db.Error != nil {
		return db.Error
	}

	tx := db.getInstance()
	sqlDB, err := tx.DB()
	if err != nil {
		return
	}

	conn, err := sqlDB.Conn(tx.Statement.Context)
	if err != nil {
		return
	}

	defer conn.Close()
	tx.Statement.ConnPool = conn
	return fc(tx)
}

// Transaction executes fc using a transaction handle. Use the supplied tx for all
// operations and return their errors: nil commits, an error rolls back, and a panic
// triggers rollback before propagating. The return value reports begin, callback,
// or commit failure; a failed commit must not be treated as success.
//
// Inside an existing transaction, it uses a savepoint and rolls back to it on error
// or panic unless DisableNestedTransaction is enabled. Dialect support is required
// for savepoints; disabling them removes isolation of the inner callback's writes.
// Only the first opts value is used for a newly started transaction.
//
//	return db.Transaction(func(tx *gorm.DB) error {
//		return tx.Create(&user).Error
//	})
//
// See https://gorm.io/docs/transactions.html.
func (db *DB) Transaction(fc func(tx *DB) error, opts ...*sql.TxOptions) (err error) {
	panicked := true

	if committer, ok := db.Statement.ConnPool.(TxCommitter); ok && committer != nil {
		// nested transaction
		if !db.DisableNestedTransaction {
			spID := new(maphash.Hash).Sum64()
			err = db.SavePoint(fmt.Sprintf("sp%d", spID)).Error
			if err != nil {
				return
			}
			defer func() {
				// Make sure to rollback when panic, Block error or Commit error
				if panicked || err != nil {
					db.RollbackTo(fmt.Sprintf("sp%d", spID))
				}
			}()
		}
		err = fc(db.Session(&Session{NewDB: db.clone == 1}))
	} else {
		tx := db.Begin(opts...)
		if tx.Error != nil {
			return tx.Error
		}

		defer func() {
			// Make sure to rollback when panic, Block error or Commit error
			if panicked || err != nil {
				tx.Rollback()
			}
		}()

		if err = fc(tx); err == nil {
			panicked = false
			return tx.Commit().Error
		}
	}

	panicked = false
	return
}

// Begin starts a transaction on the active connection pool and returns its handle.
// Check the returned Error before issuing work, use that handle for every statement,
// and finish with Commit or Rollback. Only the first opts value is used.
// The statement context controls the transaction; DefaultTransactionTimeout is a
// fallback only when no deadline exists. Use Transaction for managed rollback on
// errors and panics, and for nested-savepoint behavior.
func (db *DB) Begin(opts ...*sql.TxOptions) *DB {
	var (
		// clone statement
		tx  = db.getInstance().Session(&Session{Context: db.Statement.Context, NewDB: db.clone == 1})
		opt *sql.TxOptions
		err error
	)

	if len(opts) > 0 {
		opt = opts[0]
	}

	ctx := tx.Statement.Context
	if db.DefaultTransactionTimeout > 0 {
		if _, ok := ctx.Deadline(); !ok {
			ctx, _ = context.WithTimeout(ctx, db.DefaultTransactionTimeout)
		}
	}

	switch beginner := tx.Statement.ConnPool.(type) {
	case TxBeginner:
		tx.Statement.ConnPool, err = beginner.BeginTx(ctx, opt)
	case ConnPoolBeginner:
		tx.Statement.ConnPool, err = beginner.BeginTx(ctx, opt)
	default:
		err = ErrInvalidTransaction
	}

	if err != nil {
		tx.AddError(err)
	}

	return tx
}

// Commit commits the current transaction and records any error on the same handle.
// It returns ErrInvalidTransaction through Error when the active connection does not
// support transaction completion. Check Error: executing writes successfully does
// not prove that committing them succeeded. The transaction cannot be reused afterward.
func (db *DB) Commit() *DB {
	if committer, ok := db.Statement.ConnPool.(TxCommitter); ok && committer != nil && !reflect.ValueOf(committer).IsNil() {
		db.AddError(committer.Commit())
	} else {
		db.AddError(ErrInvalidTransaction)
	}
	return db
}

// Rollback aborts the current transaction and records any error on the same handle.
// It returns ErrInvalidTransaction through Error when the active connection is not
// a transaction. The handle retains earlier errors and is not reusable after rollback.
// For a nested Transaction callback, use its returned error to trigger savepoint
// rollback.
func (db *DB) Rollback() *DB {
	if committer, ok := db.Statement.ConnPool.(TxCommitter); ok && committer != nil {
		if !reflect.ValueOf(committer).IsNil() {
			db.AddError(committer.Rollback())
		}
	} else {
		db.AddError(ErrInvalidTransaction)
	}
	return db
}

// SavePoint asks the dialector to create a named savepoint in the active transaction.
// Check the returned Error; unsupported dialectors return ErrUnsupportedDriver.
// Use a trusted application-generated name because quoting is dialect-specific.
// A prepared-statement transaction temporarily uses its underlying transaction for
// this operation. SavePoint does not begin or commit a transaction.
func (db *DB) SavePoint(name string) *DB {
	if savePointer, ok := db.Dialector.(SavePointerDialectorInterface); ok {
		// close prepared statement, because SavePoint not support prepared statement.
		// e.g. mysql8.0 doc: https://dev.mysql.com/doc/refman/8.0/en/sql-prepared-statements.html
		var (
			preparedStmtTx   *PreparedStmtTX
			isPreparedStmtTx bool
		)
		// close prepared statement, because SavePoint not support prepared statement.
		if preparedStmtTx, isPreparedStmtTx = db.Statement.ConnPool.(*PreparedStmtTX); isPreparedStmtTx {
			db.Statement.ConnPool = preparedStmtTx.Tx
		}
		db.AddError(savePointer.SavePoint(db, name))
		// restore prepared statement
		if isPreparedStmtTx {
			db.Statement.ConnPool = preparedStmtTx
		}
	} else {
		db.AddError(ErrUnsupportedDriver)
	}
	return db
}

// RollbackTo asks the dialector to undo changes after a named savepoint while keeping
// the enclosing transaction active. Check Error; unsupported dialectors return
// ErrUnsupportedDriver. The savepoint must already exist on this transaction, and
// name must be trusted application-controlled text.
func (db *DB) RollbackTo(name string) *DB {
	if savePointer, ok := db.Dialector.(SavePointerDialectorInterface); ok {
		// close prepared statement, because RollbackTo not support prepared statement.
		// e.g. mysql8.0 doc: https://dev.mysql.com/doc/refman/8.0/en/sql-prepared-statements.html
		var (
			preparedStmtTx   *PreparedStmtTX
			isPreparedStmtTx bool
		)
		// close prepared statement, because SavePoint not support prepared statement.
		if preparedStmtTx, isPreparedStmtTx = db.Statement.ConnPool.(*PreparedStmtTX); isPreparedStmtTx {
			db.Statement.ConnPool = preparedStmtTx.Tx
		}
		db.AddError(savePointer.RollbackTo(db, name))
		// restore prepared statement
		if isPreparedStmtTx {
			db.Statement.ConnPool = preparedStmtTx
		}
	} else {
		db.AddError(ErrUnsupportedDriver)
	}
	return db
}

// Exec executes a trusted SQL statement with positional or named bound values.
// Use it for statements that do not need scanned result rows; use Raw with Scan or
// Rows for result sets. Inspect Error and RowsAffected on the returned handle.
// Raw execution does not automatically add model scopes, soft-delete predicates,
// write hooks, or the missing-WHERE guard. Bind untrusted values through placeholders.
//
//	result := db.Exec("UPDATE users SET active = ? WHERE id = ?", false, id)
func (db *DB) Exec(sql string, values ...interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.SQL = strings.Builder{}

	if strings.Contains(sql, "@") {
		clause.NamedExpr{SQL: sql, Vars: values}.Build(tx.Statement)
	} else {
		clause.Expr{SQL: sql, Vars: values}.Build(tx.Statement)
	}

	return tx.callbacks.Raw().Execute(tx)
}
