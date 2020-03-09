package gorm

import (
	"database/sql"
	"reflect"
	"strings"

	"github.com/jinzhu/gorm/clause"
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

	if err := tx.Statement.Parse(value); err != nil && tx.Statement.Schema != nil {
		where := clause.Where{Exprs: make([]clause.Expression, len(tx.Statement.Schema.PrimaryFields))}
		reflectValue := reflect.ValueOf(value)
		for idx, pf := range tx.Statement.Schema.PrimaryFields {
			if pv, isZero := pf.ValueOf(reflectValue); isZero {
				tx.callbacks.Create().Execute(tx)
				where.Exprs[idx] = clause.Eq{Column: pf.DBName, Value: pv}
				return
			}
		}

		tx.Statement.AddClause(where)
	}

	if len(tx.Statement.Selects) == 0 {
		tx.Statement.Selects = []string{"*"}
	}
	tx.callbacks.Update().Execute(tx)
	return
}

// First find first record that match given conditions, order by primary key
func (db *DB) First(out interface{}, conds ...interface{}) (tx *DB) {
	tx = db.getInstance().Limit(1).Order(clause.OrderByColumn{
		Column: clause.Column{Table: clause.CurrentTable, Name: clause.PrimaryKey},
	})
	if len(conds) > 0 {
		tx.Statement.AddClause(clause.Where{Exprs: tx.Statement.BuildCondtion(conds[0], conds[1:]...)})
	}
	tx.Statement.RaiseErrorOnNotFound = true
	tx.Statement.Dest = out
	tx.callbacks.Query().Execute(tx)
	return
}

// Take return a record that match given conditions, the order will depend on the database implementation
func (db *DB) Take(out interface{}, conds ...interface{}) (tx *DB) {
	tx = db.getInstance().Limit(1)
	if len(conds) > 0 {
		tx.Statement.AddClause(clause.Where{Exprs: tx.Statement.BuildCondtion(conds[0], conds[1:]...)})
	}
	tx.Statement.RaiseErrorOnNotFound = true
	tx.Statement.Dest = out
	tx.callbacks.Query().Execute(tx)
	return
}

// Last find last record that match given conditions, order by primary key
func (db *DB) Last(out interface{}, conds ...interface{}) (tx *DB) {
	tx = db.getInstance().Limit(1).Order(clause.OrderByColumn{
		Column: clause.Column{Table: clause.CurrentTable, Name: clause.PrimaryKey},
		Desc:   true,
	})
	if len(conds) > 0 {
		tx.Statement.AddClause(clause.Where{Exprs: tx.Statement.BuildCondtion(conds[0], conds[1:]...)})
	}
	tx.Statement.RaiseErrorOnNotFound = true
	tx.Statement.Dest = out
	tx.callbacks.Query().Execute(tx)
	return
}

// Find find records that match given conditions
func (db *DB) Find(out interface{}, conds ...interface{}) (tx *DB) {
	tx = db.getInstance()
	if len(conds) > 0 {
		tx.Statement.AddClause(clause.Where{Exprs: tx.Statement.BuildCondtion(conds[0], conds[1:]...)})
	}
	tx.Statement.Dest = out
	tx.callbacks.Query().Execute(tx)
	return
}

func (db *DB) FirstOrInit(out interface{}, where ...interface{}) (tx *DB) {
	tx = db.getInstance()
	return
}

func (db *DB) FirstOrCreate(out interface{}, where ...interface{}) (tx *DB) {
	tx = db.getInstance()
	return
}

// Update update attributes with callbacks, refer: https://jinzhu.github.io/gorm/crud.html#update
func (db *DB) Update(column string, value interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Dest = map[string]interface{}{column: value}
	tx.callbacks.Update().Execute(tx)
	return
}

// Updates update attributes with callbacks, refer: https://jinzhu.github.io/gorm/crud.html#update
func (db *DB) Updates(values interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Dest = values
	tx.callbacks.Update().Execute(tx)
	return
}

func (db *DB) UpdateColumn(column string, value interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Dest = map[string]interface{}{column: value}
	tx.callbacks.Update().Execute(tx)
	return
}

func (db *DB) UpdateColumns(values interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Dest = values
	tx.callbacks.Update().Execute(tx)
	return
}

// Delete delete value match given conditions, if the value has primary key, then will including the primary key as condition
func (db *DB) Delete(value interface{}, conds ...interface{}) (tx *DB) {
	tx = db.getInstance()
	if len(conds) > 0 {
		tx.Statement.AddClause(clause.Where{Exprs: tx.Statement.BuildCondtion(conds[0], conds[1:]...)})
	}
	tx.Statement.Dest = value
	tx.callbacks.Delete().Execute(tx)
	return
}

func (db *DB) Count(value interface{}) (tx *DB) {
	tx = db.getInstance()
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
	return
}

func (db *DB) ScanRows(rows *sql.Rows, result interface{}) error {
	return nil
}

// Transaction start a transaction as a block, return error will rollback, otherwise to commit.
func (db *DB) Transaction(fc func(tx *DB) error, opts ...*sql.TxOptions) (err error) {
	panicked := true
	tx := db.Begin(opts...)
	defer func() {
		// Make sure to rollback when panic, Block error or Commit error
		if panicked || err != nil {
			tx.Rollback()
		}
	}()

	err = fc(tx.Session(&Session{}))

	if err == nil {
		err = tx.Commit().Error
	}

	panicked = false
	return
}

// Begin begins a transaction
func (db *DB) Begin(opts ...*sql.TxOptions) (tx *DB) {
	tx = db.getInstance()
	if beginner, ok := tx.Statement.ConnPool.(TxBeginner); ok {
		var opt *sql.TxOptions
		var err error
		if len(opts) > 0 {
			opt = opts[0]
		}

		if tx.Statement.ConnPool, err = beginner.BeginTx(db.Statement.Context, opt); err != nil {
			tx.AddError(err)
		}
	} else {
		tx.AddError(ErrInvalidTransaction)
	}
	return
}

// Commit commit a transaction
func (db *DB) Commit() *DB {
	if comminter, ok := db.Statement.ConnPool.(TxCommiter); ok && comminter != nil {
		db.AddError(comminter.Commit())
	} else {
		db.AddError(ErrInvalidTransaction)
	}
	return db
}

// Rollback rollback a transaction
func (db *DB) Rollback() *DB {
	if comminter, ok := db.Statement.ConnPool.(TxCommiter); ok && comminter != nil {
		db.AddError(comminter.Rollback())
	} else {
		db.AddError(ErrInvalidTransaction)
	}
	return db
}

// Exec execute raw sql
func (db *DB) Exec(sql string, values ...interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.SQL = strings.Builder{}
	clause.Expr{SQL: sql, Vars: values}.Build(tx.Statement)
	tx.callbacks.Raw().Execute(tx)
	return
}
