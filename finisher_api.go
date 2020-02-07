package gorm

import (
	"database/sql"

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
	return
}

// First find first record that match given conditions, order by primary key
func (db *DB) First(out interface{}, where ...interface{}) (tx *DB) {
	tx = db.getInstance().Limit(1).Order(clause.OrderByColumn{
		Column: clause.Column{Table: clause.CurrentTable, Name: clause.PrimaryKey},
		Desc:   true,
	})
	tx.Statement.Dest = out
	tx.callbacks.Query().Execute(tx)
	return
}

// Take return a record that match given conditions, the order will depend on the database implementation
func (db *DB) Take(out interface{}, where ...interface{}) (tx *DB) {
	tx = db.getInstance()
	return
}

// Last find last record that match given conditions, order by primary key
func (db *DB) Last(out interface{}, where ...interface{}) (tx *DB) {
	tx = db.getInstance()
	return
}

// Find find records that match given conditions
func (db *DB) Find(out interface{}, where ...interface{}) (tx *DB) {
	tx = db.getInstance()
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
	return
}

// Updates update attributes with callbacks, refer: https://jinzhu.github.io/gorm/crud.html#update
func (db *DB) Updates(values interface{}) (tx *DB) {
	tx = db.getInstance()
	return
}

func (db *DB) UpdateColumn(column string, value interface{}) (tx *DB) {
	tx = db.getInstance()
	return
}

func (db *DB) UpdateColumns(values interface{}) (tx *DB) {
	tx = db.getInstance()
	return
}

// Delete delete value match given conditions, if the value has primary key, then will including the primary key as condition
func (db *DB) Delete(value interface{}, where ...interface{}) (tx *DB) {
	tx = db.getInstance()
	return
}

func (db *DB) Related(value interface{}, foreignKeys ...string) (tx *DB) {
	tx = db.getInstance()
	return
}

//Preloads only preloads relations, don`t touch out
func (db *DB) Preloads(out interface{}) (tx *DB) {
	tx = db.getInstance()
	return
}

func (db *DB) Association(column string) *Association {
	return nil
}

func (db *DB) Count(value interface{}) (tx *DB) {
	tx = db.getInstance()
	return
}

func (db *DB) Row() *sql.Row {
	return nil
}

func (db *DB) Rows() (*sql.Rows, error) {
	return nil, nil
}

// Scan scan value to a struct
func (db *DB) Scan(dest interface{}) (tx *DB) {
	tx = db.getInstance()
	return
}

func (db *DB) ScanRows(rows *sql.Rows, result interface{}) error {
	return nil
}

func (db *DB) Transaction(fc func(tx *DB) error, opts ...*sql.TxOptions) (err error) {
	panicked := true
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

	panicked = false
	return
}

func (db *DB) Begin(opts ...*sql.TxOptions) (tx *DB) {
	tx = db.getInstance()
	return
}

func (db *DB) Commit() (tx *DB) {
	tx = db.getInstance()
	return
}

func (db *DB) Rollback() (tx *DB) {
	tx = db.getInstance()
	return
}

func (db *DB) Exec(sql string, values ...interface{}) (tx *DB) {
	tx = db.getInstance()
	return
}
