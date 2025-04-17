package gorm

import (
	"context"
	"database/sql"

	"gorm.io/gorm/clause"
)

type Interface[T any] interface {
	Raw(sql string, values ...interface{}) ExecInterface[T]
	Exec(ctx context.Context, sql string, values ...interface{}) error
	CreateInterface[T]
}

type CreateInterface[T any] interface {
	ChainInterface[T]
	Table(name string, args ...interface{}) CreateInterface[T]
	Create(ctx context.Context, r *T) error
	CreateInBatches(ctx context.Context, r *[]T, batchSize int) error
}

type ChainInterface[T any] interface {
	ExecInterface[T]
	Scopes(scopes ...func(db *Statement)) ChainInterface[T]
	Where(query interface{}, args ...interface{}) ChainInterface[T]
	Not(query interface{}, args ...interface{}) ChainInterface[T]
	Or(query interface{}, args ...interface{}) ChainInterface[T]
	Limit(offset int) ChainInterface[T]
	Offset(offset int) ChainInterface[T]
	Joins(query string, args ...interface{}) ChainInterface[T]
	InnerJoins(query string, args ...interface{}) ChainInterface[T]
	Select(query string, args ...interface{}) ChainInterface[T]
	Omit(columns ...string) ChainInterface[T]
	MapColumns(m map[string]string) ChainInterface[T]
	Distinct(args ...interface{}) ChainInterface[T]
	Group(name string) ChainInterface[T]
	Having(query interface{}, args ...interface{}) ChainInterface[T]
	Order(value interface{}) ChainInterface[T]
	Preload(query string, args ...interface{}) ChainInterface[T]

	Delete(ctx context.Context) (rowsAffected int, err error)
	Update(ctx context.Context, name string, value any) (rowsAffected int, err error)
	Updates(ctx context.Context, t T) (rowsAffected int, err error)
	Count(ctx context.Context, column string) (result int64, err error)
}

type ExecInterface[T any] interface {
	Scan(ctx context.Context, r interface{}) error
	First(context.Context) (T, error)
	Last(ctx context.Context) (T, error)
	Find(ctx context.Context) ([]T, error)
	FindInBatches(ctx context.Context, batchSize int, fc func(data []T, batch int) error) error
	Row(ctx context.Context) *sql.Row
	Rows(ctx context.Context) (*sql.Rows, error)
}

func G[T any](db *DB, opts ...clause.Expression) Interface[T] {
	v := &g[T]{
		db:   db.Session(&Session{NewDB: true}).Clauses(opts...),
		opts: opts,
	}

	v.createG = &createG[T]{
		chainG: chainG[T]{
			execG: execG[T]{g: v},
		},
	}
	return v
}

type g[T any] struct {
	*createG[T]
	db   *DB
	opts []clause.Expression
}

func (g *g[T]) Raw(sql string, values ...interface{}) ExecInterface[T] {
	g.db = g.db.Raw(sql, values...)
	return &g.execG
}

func (g *g[T]) Exec(ctx context.Context, sql string, values ...interface{}) error {
	return g.db.WithContext(ctx).Exec(sql, values...).Error
}

type createG[T any] struct {
	chainG[T]
}

func (g *createG[T]) Table(name string, args ...interface{}) CreateInterface[T] {
	g.g.db = g.g.db.Table(name, args...)
	return g
}

func (g *createG[T]) Create(ctx context.Context, r *T) error {
	return g.g.db.WithContext(ctx).Create(r).Error
}

func (g *createG[T]) CreateInBatches(ctx context.Context, r *[]T, batchSize int) error {
	return g.g.db.WithContext(ctx).CreateInBatches(r, batchSize).Error
}

type chainG[T any] struct {
	execG[T]
}

func (g *chainG[T]) Scopes(scopes ...func(db *Statement)) ChainInterface[T] {
	for _, fc := range scopes {
		fc(g.g.db.Statement)
	}
	return g
}

func (g *chainG[T]) Where(query interface{}, args ...interface{}) ChainInterface[T] {
	g.g.db = g.g.db.Where(query, args...)
	return g
}

func (g *chainG[T]) Not(query interface{}, args ...interface{}) ChainInterface[T] {
	g.g.db = g.g.db.Not(query, args...)
	return g
}

func (g *chainG[T]) Or(query interface{}, args ...interface{}) ChainInterface[T] {
	g.g.db = g.g.db.Or(query, args...)
	return g
}

func (g *chainG[T]) Limit(offset int) ChainInterface[T] {
	g.g.db = g.g.db.Limit(offset)
	return g
}

func (g *chainG[T]) Offset(offset int) ChainInterface[T] {
	g.g.db = g.g.db.Offset(offset)
	return g
}

func (g *chainG[T]) Joins(query string, args ...interface{}) ChainInterface[T] {
	g.g.db = g.g.db.Joins(query, args...)
	return g
}

func (g *chainG[T]) InnerJoins(query string, args ...interface{}) ChainInterface[T] {
	g.g.db = g.g.db.InnerJoins(query, args...)
	return g
}

func (g *chainG[T]) Select(query string, args ...interface{}) ChainInterface[T] {
	g.g.db = g.g.db.Select(query, args...)
	return g
}

func (g *chainG[T]) Omit(columns ...string) ChainInterface[T] {
	g.g.db = g.g.db.Omit(columns...)
	return g
}

func (g *chainG[T]) MapColumns(m map[string]string) ChainInterface[T] {
	g.g.db = g.g.db.MapColumns(m)
	return g
}

func (g *chainG[T]) Distinct(args ...interface{}) ChainInterface[T] {
	g.g.db = g.g.db.Distinct(args...)
	return g
}

func (g *chainG[T]) Group(name string) ChainInterface[T] {
	g.g.db = g.g.db.Group(name)
	return g
}

func (g *chainG[T]) Having(query interface{}, args ...interface{}) ChainInterface[T] {
	g.g.db = g.g.db.Having(query, args...)
	return g
}

func (g *chainG[T]) Order(value interface{}) ChainInterface[T] {
	g.g.db = g.g.db.Order(value)
	return g
}

func (g *chainG[T]) Preload(query string, args ...interface{}) ChainInterface[T] {
	g.g.db = g.g.db.Preload(query, args...)
	return g
}

func (g *chainG[T]) Delete(ctx context.Context) (rowsAffected int, err error) {
	r := new(T)
	res := g.g.db.WithContext(ctx).Delete(r)
	return int(res.RowsAffected), res.Error
}

func (g *chainG[T]) Update(ctx context.Context, name string, value any) (rowsAffected int, err error) {
	var r T
	res := g.g.db.WithContext(ctx).Model(r).Update(name, value)
	return int(res.RowsAffected), res.Error
}

func (g *chainG[T]) Updates(ctx context.Context, t T) (rowsAffected int, err error) {
	res := g.g.db.WithContext(ctx).Updates(t)
	return int(res.RowsAffected), res.Error
}

func (g *chainG[T]) Count(ctx context.Context, column string) (result int64, err error) {
	var r T
	err = g.g.db.WithContext(ctx).Model(r).Select(column).Count(&result).Error
	return
}

type execG[T any] struct {
	g *g[T]
}

func (g *execG[T]) First(ctx context.Context) (T, error) {
	var r T
	err := g.g.db.WithContext(ctx).First(&r).Error
	return r, err
}

func (g *execG[T]) Scan(ctx context.Context, result interface{}) error {
	var r T
	err := g.g.db.WithContext(ctx).Model(r).Find(&result).Error
	return err
}

func (g *execG[T]) Last(ctx context.Context) (T, error) {
	var r T
	err := g.g.db.WithContext(ctx).Last(&r).Error
	return r, err
}

func (g *execG[T]) Find(ctx context.Context) ([]T, error) {
	var r []T
	err := g.g.db.WithContext(ctx).Find(&r).Error
	return r, err
}

func (g *execG[T]) FindInBatches(ctx context.Context, batchSize int, fc func(data []T, batch int) error) error {
	var data []T
	return g.g.db.WithContext(ctx).FindInBatches(data, batchSize, func(tx *DB, batch int) error {
		return fc(data, batch)
	}).Error
}

func (g *execG[T]) Row(ctx context.Context) *sql.Row {
	return g.g.db.WithContext(ctx).Row()
}

func (g *execG[T]) Rows(ctx context.Context) (*sql.Rows, error) {
	return g.g.db.WithContext(ctx).Rows()
}
