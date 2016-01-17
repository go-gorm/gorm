package gorm

import (
	"database/sql"
)

type sqlCommon interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

type sqlDb interface {
	Begin() (*sql.Tx, error)
}

type sqlTx interface {
	Commit() error
	Rollback() error
}

type Database interface {
	Close() error
	DB() *sql.DB
	New() Database
	NewScope(value interface{}) *Scope
	CommonDB() sqlCommon
	Callback() *callback
	SetLogger(l logger)
	LogMode(enable bool) Database
	SingularTable(enable bool)

	Where(query interface{}, args ...interface{}) Database
	Or(query interface{}, args ...interface{}) Database
	Not(query interface{}, args ...interface{}) Database
	Limit(value interface{}) Database
	Offset(value interface{}) Database
	Order(value string, reorder ...bool) Database
	Select(query interface{}, args ...interface{}) Database
	Omit(columns ...string) Database
	Group(query string) Database
	Having(query string, values ...interface{}) Database
	Joins(query string) Database

	//Scopes(funcs ...func(Database) Database) Database
	Scopes(funcs ...func(*DB) *DB) *DB
	Unscoped() Database

	Attrs(attrs ...interface{}) Database
	Assign(attrs ...interface{}) Database
	First(out interface{}, where ...interface{}) Database
	Last(out interface{}, where ...interface{}) Database
	Find(out interface{}, where ...interface{}) Database
	Scan(dest interface{}) Database
	Row() *sql.Row
	Rows() (*sql.Rows, error)
	Pluck(column string, value interface{}) Database
	Count(value interface{}) Database

	Related(value interface{}, foreignKeys ...string) Database

	FirstOrInit(out interface{}, where ...interface{}) Database
	FirstOrCreate(out interface{}, where ...interface{}) Database
	Update(attrs ...interface{}) Database
	Updates(values interface{}, ignoreProtectedAttrs ...bool) Database
	UpdateColumn(attrs ...interface{}) Database
	UpdateColumns(values interface{}) Database
	Save(value interface{}) Database
	Create(value interface{}) Database
	Delete(value interface{}, where ...interface{}) Database

	Raw(sql string, values ...interface{}) Database
	Exec(sql string, values ...interface{}) Database
	Model(value interface{}) Database
	Table(name string) Database
	Debug() Database

	Begin() Database
	Commit() Database
	Rollback() Database

	NewRecord(value interface{}) bool
	RecordNotFound() bool

	CreateTable(values ...interface{}) Database
	DropTable(values ...interface{}) Database
	DropTableIfExists(values ...interface{}) Database
	HasTable(value interface{}) bool
	AutoMigrate(values ...interface{}) Database
	ModifyColumn(column string, typ string) Database
	DropColumn(column string) Database
	AddIndex(indexName string, column ...string) Database
	AddUniqueIndex(indexName string, column ...string) Database
	RemoveIndex(indexName string) Database
	CurrentDatabase() string
	AddForeignKey(field string, dest string, onDelete string, onUpdate string) Database

	Association(column string) *Association
	Preload(column string, conditions ...interface{}) Database
	Set(name string, value interface{}) Database
	InstantSet(name string, value interface{}) Database
	Get(name string) (value interface{}, ok bool)
	SetJoinTableHandler(source interface{}, column string, handler JoinTableHandlerInterface)

	AddError(err error) error
	GetError() error
	GetErrors() (errors []error)

	GetRowsAffected() int64
	SetRowsAffected(num int64)
}

