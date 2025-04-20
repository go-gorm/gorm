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
	Take(context.Context) (T, error)
	Find(ctx context.Context) ([]T, error)
	FindInBatches(ctx context.Context, batchSize int, fc func(data []T, batch int) error) error
	Row(ctx context.Context) *sql.Row
	Rows(ctx context.Context) (*sql.Rows, error)
}

type op func(*DB) *DB

func G[T any](db *DB, opts ...clause.Expression) Interface[T] {
	v := &g[T]{
		db:  db.Session(&Session{NewDB: true}),
		ops: make([]op, 0, 5),
	}

	if len(opts) > 0 {
		v.ops = append(v.ops, func(db *DB) *DB {
			return db.Clauses(opts...)
		})
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
	db  *DB
	ops []op
}

func (g *g[T]) apply(ctx context.Context) *DB {
	db := g.db.Session(&Session{NewDB: true, Context: ctx}).getInstance()
	for _, op := range g.ops {
		db = op(db)
	}
	return db
}

func (c *g[T]) Raw(sql string, values ...interface{}) ExecInterface[T] {
	return execG[T]{g: &g[T]{
		db: c.db,
		ops: append(c.ops, func(db *DB) *DB {
			return db.Raw(sql, values...)
		}),
	}}
}

func (c *g[T]) Exec(ctx context.Context, sql string, values ...interface{}) error {
	return c.apply(ctx).Exec(sql, values...).Error
}

type createG[T any] struct {
	chainG[T]
}

func (c createG[T]) Table(name string, args ...interface{}) CreateInterface[T] {
	return createG[T]{c.with(func(db *DB) *DB {
		return db.Table(name, args...)
	})}
}

func (c createG[T]) Create(ctx context.Context, r *T) error {
	return c.g.apply(ctx).Create(r).Error
}

func (c createG[T]) CreateInBatches(ctx context.Context, r *[]T, batchSize int) error {
	return c.g.apply(ctx).CreateInBatches(r, batchSize).Error
}

type chainG[T any] struct {
	execG[T]
}

func (c chainG[T]) getInstance() *DB {
	var r T
	return c.g.apply(context.Background()).Model(r).getInstance()
}

func (c chainG[T]) with(op op) chainG[T] {
	return chainG[T]{
		execG: execG[T]{g: &g[T]{
			db:  c.g.db,
			ops: append(c.g.ops, op),
		}},
	}
}

func (c chainG[T]) Scopes(scopes ...func(db *Statement)) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		for _, fc := range scopes {
			fc(db.Statement)
		}
		return db
	})
}

func (c chainG[T]) Table(name string, args ...interface{}) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Table(name, args...)
	})
}

func (c chainG[T]) Where(query interface{}, args ...interface{}) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Where(query, args...)
	})
}

func (c chainG[T]) Not(query interface{}, args ...interface{}) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Not(query, args...)
	})
}

func (c chainG[T]) Or(query interface{}, args ...interface{}) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Or(query, args...)
	})
}

func (c chainG[T]) Limit(offset int) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Limit(offset)
	})
}

func (c chainG[T]) Offset(offset int) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Offset(offset)
	})
}

func (c chainG[T]) Joins(query string, args ...interface{}) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Joins(query, args...)
	})
}

func (c chainG[T]) InnerJoins(query string, args ...interface{}) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.InnerJoins(query, args...)
	})
}

func (c chainG[T]) Select(query string, args ...interface{}) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Select(query, args...)
	})
}

func (c chainG[T]) Omit(columns ...string) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Omit(columns...)
	})
}

func (c chainG[T]) MapColumns(m map[string]string) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.MapColumns(m)
	})
}

func (c chainG[T]) Distinct(args ...interface{}) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Distinct(args...)
	})
}

func (c chainG[T]) Group(name string) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Group(name)
	})
}

func (c chainG[T]) Having(query interface{}, args ...interface{}) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Having(query, args...)
	})
}

func (c chainG[T]) Order(value interface{}) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Order(value)
	})
}

func (c chainG[T]) Preload(query string, args ...interface{}) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Preload(query, args...)
	})
}

func (c chainG[T]) Delete(ctx context.Context) (rowsAffected int, err error) {
	r := new(T)
	res := c.g.apply(ctx).Delete(r)
	return int(res.RowsAffected), res.Error
}

func (c chainG[T]) Update(ctx context.Context, name string, value any) (rowsAffected int, err error) {
	var r T
	res := c.g.apply(ctx).Model(r).Update(name, value)
	return int(res.RowsAffected), res.Error
}

func (c chainG[T]) Updates(ctx context.Context, t T) (rowsAffected int, err error) {
	res := c.g.apply(ctx).Updates(t)
	return int(res.RowsAffected), res.Error
}

func (c chainG[T]) Count(ctx context.Context, column string) (result int64, err error) {
	var r T
	err = c.g.apply(ctx).Model(r).Select(column).Count(&result).Error
	return
}

type execG[T any] struct {
	g *g[T]
}

func (g execG[T]) First(ctx context.Context) (T, error) {
	var r T
	err := g.g.apply(ctx).First(&r).Error
	return r, err
}

func (g execG[T]) Scan(ctx context.Context, result interface{}) error {
	var r T
	err := g.g.apply(ctx).Model(r).Find(&result).Error
	return err
}

func (g execG[T]) Last(ctx context.Context) (T, error) {
	var r T
	err := g.g.apply(ctx).Last(&r).Error
	return r, err
}

func (g execG[T]) Take(ctx context.Context) (T, error) {
	var r T
	err := g.g.apply(ctx).Take(&r).Error
	return r, err
}

func (g execG[T]) Find(ctx context.Context) ([]T, error) {
	var r []T
	err := g.g.apply(ctx).Find(&r).Error
	return r, err
}

func (g execG[T]) FindInBatches(ctx context.Context, batchSize int, fc func(data []T, batch int) error) error {
	var data []T
	return g.g.apply(ctx).FindInBatches(&data, batchSize, func(tx *DB, batch int) error {
		return fc(data, batch)
	}).Error
}

func (g execG[T]) Row(ctx context.Context) *sql.Row {
	return g.g.apply(ctx).Row()
}

func (g execG[T]) Rows(ctx context.Context) (*sql.Rows, error) {
	return g.g.apply(ctx).Rows()
}
