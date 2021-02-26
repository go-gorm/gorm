package gorm

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
)

// Create insert the value into database
func (db *DB) Create(value interface{}) (tx *DB) {
	if db.CreateBatchSize > 0 {
		return db.CreateInBatches(value, db.CreateBatchSize)
	}

	tx = db.getInstance()
	tx.Statement.Dest = value
	tx.callbacks.Create().Execute(tx)
	return
}

// CreateInBatches insert the value in batches into database
func (db *DB) CreateInBatches(value interface{}, batchSize int) (tx *DB) {
	reflectValue := reflect.Indirect(reflect.ValueOf(value))

	switch reflectValue.Kind() {
	case reflect.Slice, reflect.Array:
		var rowsAffected int64
		tx = db.getInstance()

		callFc := func(tx *DB) error {
			for i := 0; i < reflectValue.Len(); i += batchSize {
				ends := i + batchSize
				if ends > reflectValue.Len() {
					ends = reflectValue.Len()
				}

				subtx := tx.getInstance()
				subtx.Statement.Dest = reflectValue.Slice(i, ends).Interface()
				subtx.callbacks.Create().Execute(subtx)
				if subtx.Error != nil {
					return subtx.Error
				}
				rowsAffected += subtx.RowsAffected
			}
			return nil
		}

		if tx.SkipDefaultTransaction {
			tx.AddError(callFc(tx.Session(&Session{})))
		} else {
			tx.AddError(tx.Transaction(callFc))
		}

		tx.RowsAffected = rowsAffected
	default:
		tx = db.getInstance()
		tx.Statement.Dest = value
		tx.callbacks.Create().Execute(tx)
	}
	return
}

// Save update value in database, if the value doesn't have primary key, will insert it
func (db *DB) Save(value interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Dest = value

	reflectValue := reflect.Indirect(reflect.ValueOf(value))
	switch reflectValue.Kind() {
	case reflect.Slice, reflect.Array:
		if _, ok := tx.Statement.Clauses["ON CONFLICT"]; !ok {
			tx = tx.Clauses(clause.OnConflict{UpdateAll: true})
		}
		tx.callbacks.Create().Execute(tx.InstanceSet("gorm:update_track_time", true))
	case reflect.Struct:
		if err := tx.Statement.Parse(value); err == nil && tx.Statement.Schema != nil {
			for _, pf := range tx.Statement.Schema.PrimaryFields {
				if _, isZero := pf.ValueOf(reflectValue); isZero {
					tx.callbacks.Create().Execute(tx)
					return
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

		tx.callbacks.Update().Execute(tx)

		if tx.Error == nil && tx.RowsAffected == 0 && !tx.DryRun && !selectedUpdate {
			result := reflect.New(tx.Statement.Schema.ModelType).Interface()
			if err := tx.Session(&Session{}).First(result).Error; errors.Is(err, ErrRecordNotFound) {
				return tx.Create(value)
			}
		}
	}

	return
}

// First find first record that match given conditions, order by primary key
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
	tx.callbacks.Query().Execute(tx)
	return
}

// Take return a record that match given conditions, the order will depend on the database implementation
func (db *DB) Take(dest interface{}, conds ...interface{}) (tx *DB) {
	tx = db.Limit(1)
	if len(conds) > 0 {
		if exprs := tx.Statement.BuildCondition(conds[0], conds[1:]...); len(exprs) > 0 {
			tx.Statement.AddClause(clause.Where{Exprs: exprs})
		}
	}
	tx.Statement.RaiseErrorOnNotFound = true
	tx.Statement.Dest = dest
	tx.callbacks.Query().Execute(tx)
	return
}

// Last find last record that match given conditions, order by primary key
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
	tx.callbacks.Query().Execute(tx)
	return
}

// Find find records that match given conditions
func (db *DB) Find(dest interface{}, conds ...interface{}) (tx *DB) {
	tx = db.getInstance()
	if len(conds) > 0 {
		if exprs := tx.Statement.BuildCondition(conds[0], conds[1:]...); len(exprs) > 0 {
			tx.Statement.AddClause(clause.Where{Exprs: exprs})
		}
	}
	tx.Statement.Dest = dest
	tx.callbacks.Query().Execute(tx)
	return
}

// FindInBatches find records in batches
func (db *DB) FindInBatches(dest interface{}, batchSize int, fc func(tx *DB, batch int) error) *DB {
	var (
		tx = db.Order(clause.OrderByColumn{
			Column: clause.Column{Table: clause.CurrentTable, Name: clause.PrimaryKey},
		}).Session(&Session{})
		queryDB      = tx
		rowsAffected int64
		batch        int
	)

	for {
		result := queryDB.Limit(batchSize).Find(dest)
		rowsAffected += result.RowsAffected
		batch++

		if result.Error == nil && result.RowsAffected != 0 {
			tx.AddError(fc(result, batch))
		}

		if tx.Error != nil || int(result.RowsAffected) < batchSize {
			break
		} else {
			resultsValue := reflect.Indirect(reflect.ValueOf(dest))
			if result.Statement.Schema.PrioritizedPrimaryField == nil {
				tx.AddError(ErrPrimaryKeyRequired)
				break
			} else {
				primaryValue, _ := result.Statement.Schema.PrioritizedPrimaryField.ValueOf(resultsValue.Index(resultsValue.Len() - 1))
				queryDB = tx.Clauses(clause.Gt{Column: clause.Column{Table: clause.CurrentTable, Name: clause.PrimaryKey}, Value: primaryValue})
			}
		}
	}

	tx.RowsAffected = rowsAffected
	return tx
}

func (tx *DB) assignInterfacesToValue(values ...interface{}) {
	for _, value := range values {
		switch v := value.(type) {
		case []clause.Expression:
			for _, expr := range v {
				if eq, ok := expr.(clause.Eq); ok {
					switch column := eq.Column.(type) {
					case string:
						if field := tx.Statement.Schema.LookUpField(column); field != nil {
							tx.AddError(field.Set(tx.Statement.ReflectValue, eq.Value))
						}
					case clause.Column:
						if field := tx.Statement.Schema.LookUpField(column.Name); field != nil {
							tx.AddError(field.Set(tx.Statement.ReflectValue, eq.Value))
						}
					}
				} else if andCond, ok := expr.(clause.AndConditions); ok {
					tx.assignInterfacesToValue(andCond.Exprs)
				}
			}
		case clause.Expression, map[string]string, map[interface{}]interface{}, map[string]interface{}:
			if exprs := tx.Statement.BuildCondition(value); len(exprs) > 0 {
				tx.assignInterfacesToValue(exprs)
			}
		default:
			if s, err := schema.Parse(value, tx.cacheStore, tx.NamingStrategy); err == nil {
				reflectValue := reflect.Indirect(reflect.ValueOf(value))
				switch reflectValue.Kind() {
				case reflect.Struct:
					for _, f := range s.Fields {
						if f.Readable {
							if v, isZero := f.ValueOf(reflectValue); !isZero {
								if field := tx.Statement.Schema.LookUpField(f.Name); field != nil {
									tx.AddError(field.Set(tx.Statement.ReflectValue, v))
								}
							}
						}
					}
				}
			} else if len(values) > 0 {
				if exprs := tx.Statement.BuildCondition(values[0], values[1:]...); len(exprs) > 0 {
					tx.assignInterfacesToValue(exprs)
				}
				return
			}
		}
	}
}

func (db *DB) FirstOrInit(dest interface{}, conds ...interface{}) (tx *DB) {
	queryTx := db.Limit(1).Order(clause.OrderByColumn{
		Column: clause.Column{Table: clause.CurrentTable, Name: clause.PrimaryKey},
	})

	if tx = queryTx.Find(dest, conds...); queryTx.RowsAffected == 0 {
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

func (db *DB) FirstOrCreate(dest interface{}, conds ...interface{}) (tx *DB) {
	queryTx := db.Limit(1).Order(clause.OrderByColumn{
		Column: clause.Column{Table: clause.CurrentTable, Name: clause.PrimaryKey},
	})

	if tx = queryTx.Find(dest, conds...); queryTx.RowsAffected == 0 {
		if c, ok := tx.Statement.Clauses["WHERE"]; ok {
			if where, ok := c.Expression.(clause.Where); ok {
				tx.assignInterfacesToValue(where.Exprs)
			}
		}

		// initialize with attrs, conds
		if len(tx.Statement.attrs) > 0 {
			tx.assignInterfacesToValue(tx.Statement.attrs...)
		}

		// initialize with attrs, conds
		if len(tx.Statement.assigns) > 0 {
			tx.assignInterfacesToValue(tx.Statement.assigns...)
		}

		return tx.Create(dest)
	} else if len(db.Statement.assigns) > 0 {
		exprs := tx.Statement.BuildCondition(tx.Statement.assigns[0], tx.Statement.assigns[1:]...)
		assigns := map[string]interface{}{}
		for _, expr := range exprs {
			if eq, ok := expr.(clause.Eq); ok {
				switch column := eq.Column.(type) {
				case string:
					assigns[column] = eq.Value
				case clause.Column:
					assigns[column.Name] = eq.Value
				default:
				}
			}
		}

		return tx.Model(dest).Updates(assigns)
	}

	return db
}

// Update update attributes with callbacks, refer: https://gorm.io/docs/update.html#Update-Changed-Fields
func (db *DB) Update(column string, value interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Dest = map[string]interface{}{column: value}
	tx.callbacks.Update().Execute(tx)
	return
}

// Updates update attributes with callbacks, refer: https://gorm.io/docs/update.html#Update-Changed-Fields
func (db *DB) Updates(values interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Dest = values
	tx.callbacks.Update().Execute(tx)
	return
}

func (db *DB) UpdateColumn(column string, value interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Dest = map[string]interface{}{column: value}
	tx.Statement.SkipHooks = true
	tx.callbacks.Update().Execute(tx)
	return
}

func (db *DB) UpdateColumns(values interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Dest = values
	tx.Statement.SkipHooks = true
	tx.callbacks.Update().Execute(tx)
	return
}

// Delete delete value match given conditions, if the value has primary key, then will including the primary key as condition
func (db *DB) Delete(value interface{}, conds ...interface{}) (tx *DB) {
	tx = db.getInstance()
	if len(conds) > 0 {
		if exprs := tx.Statement.BuildCondition(conds[0], conds[1:]...); len(exprs) > 0 {
			tx.Statement.AddClause(clause.Where{Exprs: exprs})
		}
	}
	tx.Statement.Dest = value
	tx.callbacks.Delete().Execute(tx)
	return
}

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
			db.Statement.Clauses["SELECT"] = selectClause
		}()
	} else {
		defer delete(tx.Statement.Clauses, "SELECT")
	}

	if len(tx.Statement.Selects) == 0 {
		tx.Statement.AddClause(clause.Select{Expression: clause.Expr{SQL: "count(1)"}})
	} else if !strings.HasPrefix(strings.TrimSpace(strings.ToLower(tx.Statement.Selects[0])), "count(") {
		expr := clause.Expr{SQL: "count(1)"}

		if len(tx.Statement.Selects) == 1 {
			dbName := tx.Statement.Selects[0]
			fields := strings.FieldsFunc(dbName, utils.IsValidDBNameChar)
			if len(fields) == 1 || (len(fields) == 3 && strings.ToUpper(fields[1]) == "AS") {
				if tx.Statement.Parse(tx.Statement.Model) == nil {
					if f := tx.Statement.Schema.LookUpField(dbName); f != nil {
						dbName = f.DBName
					}
				}

				if tx.Statement.Distinct {
					expr = clause.Expr{SQL: "COUNT(DISTINCT(?))", Vars: []interface{}{clause.Column{Name: dbName}}}
				} else {
					expr = clause.Expr{SQL: "COUNT(?)", Vars: []interface{}{clause.Column{Name: dbName}}}
				}
			}
		}

		tx.Statement.AddClause(clause.Select{Expression: expr})
	}

	if orderByClause, ok := db.Statement.Clauses["ORDER BY"]; ok {
		if _, ok := db.Statement.Clauses["GROUP BY"]; !ok {
			delete(db.Statement.Clauses, "ORDER BY")
			defer func() {
				db.Statement.Clauses["ORDER BY"] = orderByClause
			}()
		}
	}

	tx.Statement.Dest = count
	tx.callbacks.Query().Execute(tx)
	if tx.RowsAffected != 1 {
		*count = tx.RowsAffected
	}
	return
}

func (db *DB) Row() *sql.Row {
	tx := db.getInstance().InstanceSet("rows", false)
	tx.callbacks.Row().Execute(tx)
	row, ok := tx.Statement.Dest.(*sql.Row)
	if !ok && tx.DryRun {
		db.Logger.Error(tx.Statement.Context, ErrDryRunModeUnsupported.Error())
	}
	return row
}

func (db *DB) Rows() (*sql.Rows, error) {
	tx := db.getInstance().InstanceSet("rows", true)
	tx.callbacks.Row().Execute(tx)
	rows, ok := tx.Statement.Dest.(*sql.Rows)
	if !ok && tx.DryRun && tx.Error == nil {
		tx.Error = ErrDryRunModeUnsupported
	}
	return rows, tx.Error
}

// Scan scan value to a struct
func (db *DB) Scan(dest interface{}) (tx *DB) {
	config := *db.Config
	currentLogger, newLogger := config.Logger, logger.Recorder.New()
	config.Logger = newLogger

	tx = db.getInstance()
	tx.Config = &config

	if rows, err := tx.Rows(); err != nil {
		tx.AddError(err)
	} else {
		defer rows.Close()
		if rows.Next() {
			tx.ScanRows(rows, dest)
		} else {
			tx.RowsAffected = 0
		}
	}

	currentLogger.Trace(tx.Statement.Context, newLogger.BeginAt, func() (string, int64) {
		return newLogger.SQL, tx.RowsAffected
	}, tx.Error)
	tx.Logger = currentLogger
	return
}

// Pluck used to query single column from a model as a map
//     var ages []int64
//     db.Find(&users).Pluck("age", &ages)
func (db *DB) Pluck(column string, dest interface{}) (tx *DB) {
	tx = db.getInstance()
	if tx.Statement.Model != nil {
		if tx.Statement.Parse(tx.Statement.Model) == nil {
			if f := tx.Statement.Schema.LookUpField(column); f != nil {
				column = f.DBName
			}
		}
	} else if tx.Statement.Table == "" {
		tx.AddError(ErrModelValueRequired)
	}

	if len(tx.Statement.Selects) != 1 {
		fields := strings.FieldsFunc(column, utils.IsValidDBNameChar)
		tx.Statement.AddClauseIfNotExists(clause.Select{
			Distinct: tx.Statement.Distinct,
			Columns:  []clause.Column{{Name: column, Raw: len(fields) != 1}},
		})
	}
	tx.Statement.Dest = dest
	tx.callbacks.Query().Execute(tx)
	return
}

func (db *DB) ScanRows(rows *sql.Rows, dest interface{}) error {
	tx := db.getInstance()
	if err := tx.Statement.Parse(dest); !errors.Is(err, schema.ErrUnsupportedDataType) {
		tx.AddError(err)
	}
	tx.Statement.Dest = dest
	tx.Statement.ReflectValue = reflect.ValueOf(dest)
	for tx.Statement.ReflectValue.Kind() == reflect.Ptr {
		tx.Statement.ReflectValue = tx.Statement.ReflectValue.Elem()
	}
	Scan(rows, tx, true)
	return tx.Error
}

// Transaction start a transaction as a block, return error will rollback, otherwise to commit.
func (db *DB) Transaction(fc func(tx *DB) error, opts ...*sql.TxOptions) (err error) {
	panicked := true

	if committer, ok := db.Statement.ConnPool.(TxCommitter); ok && committer != nil {
		// nested transaction
		if !db.DisableNestedTransaction {
			err = db.SavePoint(fmt.Sprintf("sp%p", fc)).Error
			defer func() {
				// Make sure to rollback when panic, Block error or Commit error
				if panicked || err != nil {
					db.RollbackTo(fmt.Sprintf("sp%p", fc))
				}
			}()
		}

		if err == nil {
			err = fc(db.Session(&Session{}))
		}
	} else {
		tx := db.Begin(opts...)

		defer func() {
			// Make sure to rollback when panic, Block error or Commit error
			if panicked || err != nil {
				tx.Rollback()
			}
		}()

		if err = tx.Error; err == nil {
			err = fc(tx)
		}

		if err == nil {
			err = tx.Commit().Error
		}
	}

	panicked = false
	return
}

// Begin begins a transaction
func (db *DB) Begin(opts ...*sql.TxOptions) *DB {
	var (
		// clone statement
		tx  = db.getInstance().Session(&Session{Context: db.Statement.Context})
		opt *sql.TxOptions
		err error
	)

	if len(opts) > 0 {
		opt = opts[0]
	}

	if beginner, ok := tx.Statement.ConnPool.(TxBeginner); ok {
		tx.Statement.ConnPool, err = beginner.BeginTx(tx.Statement.Context, opt)
	} else if beginner, ok := tx.Statement.ConnPool.(ConnPoolBeginner); ok {
		tx.Statement.ConnPool, err = beginner.BeginTx(tx.Statement.Context, opt)
	} else {
		err = ErrInvalidTransaction
	}

	if err != nil {
		tx.AddError(err)
	}

	return tx
}

// Commit commit a transaction
func (db *DB) Commit() *DB {
	if committer, ok := db.Statement.ConnPool.(TxCommitter); ok && committer != nil && !reflect.ValueOf(committer).IsNil() {
		db.AddError(committer.Commit())
	} else {
		db.AddError(ErrInvalidTransaction)
	}
	return db
}

// Rollback rollback a transaction
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

func (db *DB) SavePoint(name string) *DB {
	if savePointer, ok := db.Dialector.(SavePointerDialectorInterface); ok {
		db.AddError(savePointer.SavePoint(db, name))
	} else {
		db.AddError(ErrUnsupportedDriver)
	}
	return db
}

func (db *DB) RollbackTo(name string) *DB {
	if savePointer, ok := db.Dialector.(SavePointerDialectorInterface); ok {
		db.AddError(savePointer.RollbackTo(db, name))
	} else {
		db.AddError(ErrUnsupportedDriver)
	}
	return db
}

// Exec execute raw sql
func (db *DB) Exec(sql string, values ...interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.SQL = strings.Builder{}

	if strings.Contains(sql, "@") {
		clause.NamedExpr{SQL: sql, Vars: values}.Build(tx.Statement)
	} else {
		clause.Expr{SQL: sql, Vars: values}.Build(tx.Statement)
	}

	tx.callbacks.Raw().Execute(tx)
	return
}
