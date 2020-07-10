package gorm

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/utils"
)

// Create insert the value into database
func (db *DB) Create(value interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Dest = value
	tx.callbacks.Create().Execute(tx)
	return
}

// Save update value in database, if the value doesn't have primary key, will insert it
func (db *DB) Save(value interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Dest = value

	reflectValue := reflect.Indirect(reflect.ValueOf(value))
	switch reflectValue.Kind() {
	case reflect.Slice, reflect.Array:
		tx.Statement.UpdatingColumn = true
		tx.callbacks.Create().Execute(tx)
	case reflect.Struct:
		if err := tx.Statement.Parse(value); err == nil && tx.Statement.Schema != nil {
			where := clause.Where{Exprs: make([]clause.Expression, len(tx.Statement.Schema.PrimaryFields))}
			for idx, pf := range tx.Statement.Schema.PrimaryFields {
				if pv, isZero := pf.ValueOf(reflectValue); isZero {
					tx.callbacks.Create().Execute(tx)
					return
				} else {
					where.Exprs[idx] = clause.Eq{Column: pf.DBName, Value: pv}
				}
			}

			tx.Statement.AddClause(where)
		}

		fallthrough
	default:
		if len(tx.Statement.Selects) == 0 {
			tx.Statement.Selects = append(tx.Statement.Selects, "*")
		}

		tx.callbacks.Update().Execute(tx)
	}

	return
}

// First find first record that match given conditions, order by primary key
func (db *DB) First(dest interface{}, conds ...interface{}) (tx *DB) {
	tx = db.Limit(1).Order(clause.OrderByColumn{
		Column: clause.Column{Table: clause.CurrentTable, Name: clause.PrimaryKey},
	})
	if len(conds) > 0 {
		tx.Statement.AddClause(clause.Where{Exprs: tx.Statement.BuildCondition(conds[0], conds[1:]...)})
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
		tx.Statement.AddClause(clause.Where{Exprs: tx.Statement.BuildCondition(conds[0], conds[1:]...)})
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
		tx.Statement.AddClause(clause.Where{Exprs: tx.Statement.BuildCondition(conds[0], conds[1:]...)})
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
		tx.Statement.AddClause(clause.Where{Exprs: tx.Statement.BuildCondition(conds[0], conds[1:]...)})
	}
	tx.Statement.Dest = dest
	tx.callbacks.Query().Execute(tx)
	return
}

// FindInBatches find records in batches
func (db *DB) FindInBatches(dest interface{}, batchSize int, fc func(tx *DB, batch int) error) (tx *DB) {
	tx = db.Session(&Session{WithConditions: true})
	rowsAffected := int64(0)
	batch := 0

	for {
		result := tx.Limit(batchSize).Offset(batch * batchSize).Find(dest)
		rowsAffected += result.RowsAffected
		batch++

		if result.Error == nil && result.RowsAffected != 0 {
			tx.AddError(fc(result, batch))
		}

		if tx.Error != nil || int(result.RowsAffected) < batchSize {
			break
		}
	}

	tx.RowsAffected = rowsAffected
	return
}

func (tx *DB) assignExprsToValue(exprs []clause.Expression) {
	for _, expr := range exprs {
		if eq, ok := expr.(clause.Eq); ok {
			switch column := eq.Column.(type) {
			case string:
				if field := tx.Statement.Schema.LookUpField(column); field != nil {
					field.Set(tx.Statement.ReflectValue, eq.Value)
				}
			case clause.Column:
				if field := tx.Statement.Schema.LookUpField(column.Name); field != nil {
					field.Set(tx.Statement.ReflectValue, eq.Value)
				}
			default:
			}
		}
	}
}

func (db *DB) FirstOrInit(dest interface{}, conds ...interface{}) (tx *DB) {
	if tx = db.First(dest, conds...); errors.Is(tx.Error, ErrRecordNotFound) {
		if c, ok := tx.Statement.Clauses["WHERE"]; ok {
			if where, ok := c.Expression.(clause.Where); ok {
				tx.assignExprsToValue(where.Exprs)
			}
		}

		// initialize with attrs, conds
		if len(tx.Statement.attrs) > 0 {
			exprs := tx.Statement.BuildCondition(tx.Statement.attrs[0], tx.Statement.attrs[1:]...)
			tx.assignExprsToValue(exprs)
		}
		tx.Error = nil
	}

	// initialize with attrs, conds
	if len(tx.Statement.assigns) > 0 {
		exprs := tx.Statement.BuildCondition(tx.Statement.assigns[0], tx.Statement.assigns[1:]...)
		tx.assignExprsToValue(exprs)
	}
	return
}

func (db *DB) FirstOrCreate(dest interface{}, conds ...interface{}) (tx *DB) {
	if tx = db.First(dest, conds...); errors.Is(tx.Error, ErrRecordNotFound) {
		tx.Error = nil

		if c, ok := tx.Statement.Clauses["WHERE"]; ok {
			if where, ok := c.Expression.(clause.Where); ok {
				tx.assignExprsToValue(where.Exprs)
			}
		}

		// initialize with attrs, conds
		if len(tx.Statement.attrs) > 0 {
			exprs := tx.Statement.BuildCondition(tx.Statement.attrs[0], tx.Statement.attrs[1:]...)
			tx.assignExprsToValue(exprs)
		}

		// initialize with attrs, conds
		if len(tx.Statement.assigns) > 0 {
			exprs := tx.Statement.BuildCondition(tx.Statement.assigns[0], tx.Statement.assigns[1:]...)
			tx.assignExprsToValue(exprs)
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
	tx.Statement.UpdatingColumn = true
	tx.callbacks.Update().Execute(tx)
	return
}

func (db *DB) UpdateColumns(values interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Dest = values
	tx.Statement.UpdatingColumn = true
	tx.callbacks.Update().Execute(tx)
	return
}

// Delete delete value match given conditions, if the value has primary key, then will including the primary key as condition
func (db *DB) Delete(value interface{}, conds ...interface{}) (tx *DB) {
	tx = db.getInstance()
	if len(conds) > 0 {
		tx.Statement.AddClause(clause.Where{Exprs: tx.Statement.BuildCondition(conds[0], conds[1:]...)})
	}
	tx.Statement.Dest = value
	tx.callbacks.Delete().Execute(tx)
	return
}

func (db *DB) Count(count *int64) (tx *DB) {
	tx = db.getInstance()
	if tx.Statement.Model == nil {
		tx.Statement.Model = tx.Statement.Dest
	}

	if len(tx.Statement.Selects) == 0 {
		tx.Statement.AddClause(clause.Select{Expression: clause.Expr{SQL: "count(1)"}})
		defer tx.Statement.AddClause(clause.Select{})
	} else if !strings.Contains(strings.ToLower(tx.Statement.Selects[0]), "count(") {
		expr := clause.Expr{SQL: "count(1)"}

		if len(tx.Statement.Selects) == 1 {
			if tx.Statement.Parse(tx.Statement.Model) == nil {
				if f := tx.Statement.Schema.LookUpField(tx.Statement.Selects[0]); f != nil {
					expr = clause.Expr{SQL: "COUNT(DISTINCT(?))", Vars: []interface{}{clause.Column{Name: f.DBName}}}
				}
			}
		}

		tx.Statement.AddClause(clause.Select{Expression: expr})
		defer tx.Statement.AddClause(clause.Select{})
	}

	tx.Statement.Dest = count
	tx.callbacks.Query().Execute(tx)
	if tx.RowsAffected != 1 {
		*count = tx.RowsAffected
	}
	return
}

func (db *DB) Row() *sql.Row {
	tx := db.getInstance()
	tx.callbacks.Row().Execute(tx)
	return tx.Statement.Dest.(*sql.Row)
}

func (db *DB) Rows() (*sql.Rows, error) {
	tx := db.Set("rows", true)
	tx.callbacks.Row().Execute(tx)
	return tx.Statement.Dest.(*sql.Rows), tx.Error
}

// Scan scan value to a struct
func (db *DB) Scan(dest interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Dest = dest
	tx.callbacks.Query().Execute(tx)
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
		tx.AddError(ErrorModelValueRequired)
	}

	fields := strings.FieldsFunc(column, utils.IsChar)
	tx.Statement.AddClauseIfNotExists(clause.Select{
		Distinct: tx.Statement.Distinct,
		Columns:  []clause.Column{{Name: column, Raw: len(fields) != 1}},
	})
	tx.Statement.Dest = dest
	tx.callbacks.Query().Execute(tx)
	return
}

func (db *DB) ScanRows(rows *sql.Rows, dest interface{}) error {
	tx := db.getInstance()
	tx.Error = tx.Statement.Parse(dest)
	tx.Statement.Dest = dest
	tx.Statement.ReflectValue = reflect.Indirect(reflect.ValueOf(dest))
	Scan(rows, tx, true)
	return tx.Error
}

// Transaction start a transaction as a block, return error will rollback, otherwise to commit.
func (db *DB) Transaction(fc func(tx *DB) error, opts ...*sql.TxOptions) (err error) {
	panicked := true

	if committer, ok := db.Statement.ConnPool.(TxCommitter); ok && committer != nil {
		// nested transaction
		db.SavePoint(fmt.Sprintf("sp%p", fc))
		defer func() {
			// Make sure to rollback when panic, Block error or Commit error
			if panicked || err != nil {
				db.RollbackTo(fmt.Sprintf("sp%p", fc))
			}
		}()

		err = fc(db.Session(&Session{WithConditions: true}))
	} else {
		tx := db.Begin(opts...)

		defer func() {
			// Make sure to rollback when panic, Block error or Commit error
			if panicked || err != nil {
				tx.Rollback()
			}
		}()

		err = fc(tx)

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
		tx  = db.Session(&Session{WithConditions: true, Context: db.Statement.Context})
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
	if committer, ok := db.Statement.ConnPool.(TxCommitter); ok && committer != nil {
		db.AddError(committer.Commit())
	} else {
		db.AddError(ErrInvalidTransaction)
	}
	return db
}

// Rollback rollback a transaction
func (db *DB) Rollback() *DB {
	if committer, ok := db.Statement.ConnPool.(TxCommitter); ok && committer != nil {
		db.AddError(committer.Rollback())
	} else {
		db.AddError(ErrInvalidTransaction)
	}
	return db
}

func (db *DB) SavePoint(name string) *DB {
	if savePointer, ok := db.Dialector.(SavePointerDialectorInterface); ok {
		savePointer.SavePoint(db, name)
	} else {
		db.AddError(ErrUnsupportedDriver)
	}
	return db
}

func (db *DB) RollbackTo(name string) *DB {
	if savePointer, ok := db.Dialector.(SavePointerDialectorInterface); ok {
		savePointer.RollbackTo(db, name)
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
