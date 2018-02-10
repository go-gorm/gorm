package gorm

import "database/sql"

// Gormw is an interface which DB implements
type Gormw interface {
	Close() error
	DB() *sql.DB
	New() Gormw
	NewScope(value interface{}) *Scope
	CommonDB() SQLCommon
	Callback() *Callback
	SetLogger(l Logger)
	LogMode(enable bool) Gormw
	SingularTable(enable bool)
	Where(query interface{}, args ...interface{}) Gormw
	Or(query interface{}, args ...interface{}) Gormw
	Not(query interface{}, args ...interface{}) Gormw
	Limit(value int) Gormw
	Offset(value int) Gormw
	Order(value string, reorder ...bool) Gormw
	Select(query interface{}, args ...interface{}) Gormw
	Omit(columns ...string) Gormw
	Group(query string) Gormw
	Having(query string, values ...interface{}) Gormw
	Joins(query string, args ...interface{}) Gormw
	Scopes(funcs ...func(*DB) *DB) Gormw
	Unscoped() Gormw
	Attrs(attrs ...interface{}) Gormw
	Assign(attrs ...interface{}) Gormw
	First(out interface{}, where ...interface{}) Gormw
	Last(out interface{}, where ...interface{}) Gormw
	Find(out interface{}, where ...interface{}) Gormw
	Scan(dest interface{}) Gormw
	Row() *sql.Row
	Rows() (*sql.Rows, error)
	ScanRows(rows *sql.Rows, result interface{}) error
	Pluck(column string, value interface{}) Gormw
	Count(value interface{}) Gormw
	Related(value interface{}, foreignKeys ...string) Gormw
	FirstOrInit(out interface{}, where ...interface{}) Gormw
	FirstOrCreate(out interface{}, where ...interface{}) Gormw
	Update(attrs ...interface{}) Gormw
	Updates(values interface{}, ignoreProtectedAttrs ...bool) Gormw
	UpdateColumn(attrs ...interface{}) Gormw
	UpdateColumns(values interface{}) Gormw
	Save(value interface{}) Gormw
	Create(value interface{}) Gormw
	Delete(value interface{}, where ...interface{}) Gormw
	Raw(sql string, values ...interface{}) Gormw
	Exec(sql string, values ...interface{}) Gormw
	Model(value interface{}) Gormw
	Table(name string) Gormw
	Debug() Gormw
	Begin() Gormw
	Commit() Gormw
	Rollback() Gormw
	NewRecord(value interface{}) bool
	RecordNotFound() bool
	CreateTable(values ...interface{}) Gormw
	DropTable(values ...interface{}) Gormw
	DropTableIfExists(values ...interface{}) Gormw
	HasTable(value interface{}) bool
	AutoMigrate(values ...interface{}) Gormw
	ModifyColumn(column string, typ string) Gormw
	DropColumn(column string) Gormw
	AddIndex(indexName string, column ...string) Gormw
	AddUniqueIndex(indexName string, column ...string) Gormw
	RemoveIndex(indexName string) Gormw
	AddForeignKey(field string, dest string, onDelete string, onUpdate string) Gormw
	Association(column string) *Association
	Preload(column string, conditions ...interface{}) Gormw
	Set(name string, value interface{}) Gormw
	InstantSet(name string, value interface{}) Gormw
	Get(name string) (value interface{}, ok bool)
	SetJoinTableHandler(source interface{}, column string, handler JoinTableHandlerInterface)
	AddError(err error) error
	GetErrors() (errors []error)

	// extra
	Error() error
	RowsAffected() int64
}

type gormw struct {
	w *DB
}

// Openw is a drop-in replacement for Open()
func Openw(dialect string, args ...interface{}) (db Gormw, err error) {
	gormdb, err := Open(dialect, args...)
	return Wrap(gormdb), err
}

// Wrap wraps gorm.DB in an interface
func Wrap(db *DB) Gormw {
	return &gormw{db}
}

func (it *gormw) Close() error {
	return it.w.Close()
}

func (it *gormw) DB() *sql.DB {
	return it.w.DB()
}

func (it *gormw) New() Gormw {
	return Wrap(it.w.New())
}

func (it *gormw) NewScope(value interface{}) *Scope {
	return it.w.NewScope(value)
}

func (it *gormw) CommonDB() SQLCommon {
	return it.w.CommonDB()
}

func (it *gormw) Callback() *Callback {
	return it.w.Callback()
}

func (it *gormw) SetLogger(log Logger) {
	it.w.SetLogger(log)
}

func (it *gormw) LogMode(enable bool) Gormw {
	return Wrap(it.w.LogMode(enable))
}

func (it *gormw) SingularTable(enable bool) {
	it.w.SingularTable(enable)
}

func (it *gormw) Where(query interface{}, args ...interface{}) Gormw {
	return Wrap(it.w.Where(query, args...))
}

func (it *gormw) Or(query interface{}, args ...interface{}) Gormw {
	return Wrap(it.w.Or(query, args...))
}

func (it *gormw) Not(query interface{}, args ...interface{}) Gormw {
	return Wrap(it.w.Not(query, args...))
}

func (it *gormw) Limit(value int) Gormw {
	return Wrap(it.w.Limit(value))
}

func (it *gormw) Offset(value int) Gormw {
	return Wrap(it.w.Offset(value))
}

func (it *gormw) Order(value string, reorder ...bool) Gormw {
	return Wrap(it.w.Order(value, reorder...))
}

func (it *gormw) Select(query interface{}, args ...interface{}) Gormw {
	return Wrap(it.w.Select(query, args...))
}

func (it *gormw) Omit(columns ...string) Gormw {
	return Wrap(it.w.Omit(columns...))
}

func (it *gormw) Group(query string) Gormw {
	return Wrap(it.w.Group(query))
}

func (it *gormw) Having(query string, values ...interface{}) Gormw {
	return Wrap(it.w.Having(query, values...))
}

func (it *gormw) Joins(query string, args ...interface{}) Gormw {
	return Wrap(it.w.Joins(query, args...))
}

func (it *gormw) Scopes(funcs ...func(*DB) *DB) Gormw {
	return Wrap(it.w.Scopes(funcs...))
}

func (it *gormw) Unscoped() Gormw {
	return Wrap(it.w.Unscoped())
}

func (it *gormw) Attrs(attrs ...interface{}) Gormw {
	return Wrap(it.w.Attrs(attrs...))
}

func (it *gormw) Assign(attrs ...interface{}) Gormw {
	return Wrap(it.w.Assign(attrs...))
}

func (it *gormw) First(out interface{}, where ...interface{}) Gormw {
	return Wrap(it.w.First(out, where...))
}

func (it *gormw) Last(out interface{}, where ...interface{}) Gormw {
	return Wrap(it.w.Last(out, where...))
}

func (it *gormw) Find(out interface{}, where ...interface{}) Gormw {
	return Wrap(it.w.Find(out, where...))
}

func (it *gormw) Scan(dest interface{}) Gormw {
	return Wrap(it.w.Scan(dest))
}

func (it *gormw) Row() *sql.Row {
	return it.w.Row()
}

func (it *gormw) Rows() (*sql.Rows, error) {
	return it.w.Rows()
}

func (it *gormw) ScanRows(rows *sql.Rows, result interface{}) error {
	return it.w.ScanRows(rows, result)
}

func (it *gormw) Pluck(column string, value interface{}) Gormw {
	return Wrap(it.w.Pluck(column, value))
}

func (it *gormw) Count(value interface{}) Gormw {
	return Wrap(it.w.Count(value))
}

func (it *gormw) Related(value interface{}, foreignKeys ...string) Gormw {
	return Wrap(it.w.Related(value, foreignKeys...))
}

func (it *gormw) FirstOrInit(out interface{}, where ...interface{}) Gormw {
	return Wrap(it.w.FirstOrInit(out, where...))
}

func (it *gormw) FirstOrCreate(out interface{}, where ...interface{}) Gormw {
	return Wrap(it.w.FirstOrCreate(out, where...))
}

func (it *gormw) Update(attrs ...interface{}) Gormw {
	return Wrap(it.w.Update(attrs...))
}

func (it *gormw) Updates(values interface{}, ignoreProtectedAttrs ...bool) Gormw {
	return Wrap(it.w.Updates(values, ignoreProtectedAttrs...))
}

func (it *gormw) UpdateColumn(attrs ...interface{}) Gormw {
	return Wrap(it.w.UpdateColumn(attrs...))
}

func (it *gormw) UpdateColumns(values interface{}) Gormw {
	return Wrap(it.w.UpdateColumns(values))
}

func (it *gormw) Save(value interface{}) Gormw {
	return Wrap(it.w.Save(value))
}

func (it *gormw) Create(value interface{}) Gormw {
	return Wrap(it.w.Create(value))
}

func (it *gormw) Delete(value interface{}, where ...interface{}) Gormw {
	return Wrap(it.w.Delete(value, where...))
}

func (it *gormw) Raw(sql string, values ...interface{}) Gormw {
	return Wrap(it.w.Raw(sql, values...))
}

func (it *gormw) Exec(sql string, values ...interface{}) Gormw {
	return Wrap(it.w.Exec(sql, values...))
}

func (it *gormw) Model(value interface{}) Gormw {
	return Wrap(it.w.Model(value))
}

func (it *gormw) Table(name string) Gormw {
	return Wrap(it.w.Table(name))
}

func (it *gormw) Debug() Gormw {
	return Wrap(it.w.Debug())
}

func (it *gormw) Begin() Gormw {
	return Wrap(it.w.Begin())
}

func (it *gormw) Commit() Gormw {
	return Wrap(it.w.Commit())
}

func (it *gormw) Rollback() Gormw {
	return Wrap(it.w.Rollback())
}

func (it *gormw) NewRecord(value interface{}) bool {
	return it.w.NewRecord(value)
}

func (it *gormw) RecordNotFound() bool {
	return it.w.RecordNotFound()
}

func (it *gormw) CreateTable(values ...interface{}) Gormw {
	return Wrap(it.w.CreateTable(values...))
}

func (it *gormw) DropTable(values ...interface{}) Gormw {
	return Wrap(it.w.DropTable(values...))
}

func (it *gormw) DropTableIfExists(values ...interface{}) Gormw {
	return Wrap(it.w.DropTableIfExists(values...))
}

func (it *gormw) HasTable(value interface{}) bool {
	return it.w.HasTable(value)
}

func (it *gormw) AutoMigrate(values ...interface{}) Gormw {
	return Wrap(it.w.AutoMigrate(values...))
}

func (it *gormw) ModifyColumn(column string, typ string) Gormw {
	return Wrap(it.w.ModifyColumn(column, typ))
}

func (it *gormw) DropColumn(column string) Gormw {
	return Wrap(it.w.DropColumn(column))
}

func (it *gormw) AddIndex(indexName string, columns ...string) Gormw {
	return Wrap(it.w.AddIndex(indexName, columns...))
}

func (it *gormw) AddUniqueIndex(indexName string, columns ...string) Gormw {
	return Wrap(it.w.AddUniqueIndex(indexName, columns...))
}

func (it *gormw) RemoveIndex(indexName string) Gormw {
	return Wrap(it.w.RemoveIndex(indexName))
}

func (it *gormw) Association(column string) *Association {
	return it.w.Association(column)
}

func (it *gormw) Preload(column string, conditions ...interface{}) Gormw {
	return Wrap(it.w.Preload(column, conditions...))
}

func (it *gormw) Set(name string, value interface{}) Gormw {
	return Wrap(it.w.Set(name, value))
}

func (it *gormw) InstantSet(name string, value interface{}) Gormw {
	return Wrap(it.w.InstantSet(name, value))
}

func (it *gormw) Get(name string) (interface{}, bool) {
	return it.w.Get(name)
}

func (it *gormw) SetJoinTableHandler(source interface{}, column string, handler JoinTableHandlerInterface) {
	it.w.SetJoinTableHandler(source, column, handler)
}

func (it *gormw) AddForeignKey(field string, dest string, onDelete string, onUpdate string) Gormw {
	return Wrap(it.w.AddForeignKey(field, dest, onDelete, onUpdate))
}

func (it *gormw) AddError(err error) error {
	return it.w.AddError(err)
}

func (it *gormw) GetErrors() (errors []error) {
	return it.w.GetErrors()
}

func (it *gormw) RowsAffected() int64 {
	return it.w.RowsAffected
}

func (it *gormw) Error() error {
	return it.w.Error
}
