package gorm

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type result struct {
	Result       sql.Result
	RowsAffected int64
}

func (info *result) ModifyStatement(stmt *Statement) {
	stmt.Result = info
}

// Build implements clause.Expression interface
func (result) Build(clause.Builder) {
}

func WithResult() *result {
	return &result{}
}

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
	Joins(query clause.JoinTarget, on func(db JoinBuilder, joinTable clause.Table, curTable clause.Table) error) ChainInterface[T]
	Preload(association string, query func(db PreloadBuilder) error) ChainInterface[T]
	Select(query string, args ...interface{}) ChainInterface[T]
	Omit(columns ...string) ChainInterface[T]
	MapColumns(m map[string]string) ChainInterface[T]
	Distinct(args ...interface{}) ChainInterface[T]
	Group(name string) ChainInterface[T]
	Having(query interface{}, args ...interface{}) ChainInterface[T]
	Order(value interface{}) ChainInterface[T]

	Build(builder clause.Builder)

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

type JoinBuilder interface {
	Select(...string) JoinBuilder
	Omit(...string) JoinBuilder
	Where(query interface{}, args ...interface{}) JoinBuilder
	Not(query interface{}, args ...interface{}) JoinBuilder
	Or(query interface{}, args ...interface{}) JoinBuilder
}

type PreloadBuilder interface {
	Select(...string) PreloadBuilder
	Omit(...string) PreloadBuilder
	Where(query interface{}, args ...interface{}) PreloadBuilder
	Not(query interface{}, args ...interface{}) PreloadBuilder
	Or(query interface{}, args ...interface{}) PreloadBuilder
	Limit(offset int) PreloadBuilder
	Offset(offset int) PreloadBuilder
	Order(value interface{}) PreloadBuilder
	LimitPerRecord(num int) PreloadBuilder
}

type op func(*DB) *DB

func G[T any](db *DB, opts ...clause.Expression) Interface[T] {
	v := &g[T]{
		db:  db,
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
	db := g.db
	if !db.DryRun {
		db = db.Session(&Session{NewDB: true, Context: ctx}).getInstance()
	}

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

func (c chainG[T]) with(v op) chainG[T] {
	return chainG[T]{
		execG: execG[T]{g: &g[T]{
			db:  c.g.db,
			ops: append(append([]op(nil), c.g.ops...), v),
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

type joinBuilder struct {
	db *DB
}

func (q *joinBuilder) Where(query interface{}, args ...interface{}) JoinBuilder {
	q.db.Where(query, args...)
	return q
}

func (q *joinBuilder) Or(query interface{}, args ...interface{}) JoinBuilder {
	q.db.Where(query, args...)
	return q
}

func (q *joinBuilder) Not(query interface{}, args ...interface{}) JoinBuilder {
	q.db.Where(query, args...)
	return q
}

func (q *joinBuilder) Select(columns ...string) JoinBuilder {
	q.db.Select(columns)
	return q
}

func (q *joinBuilder) Omit(columns ...string) JoinBuilder {
	q.db.Omit(columns...)
	return q
}

type preloadBuilder struct {
	limitPerRecord int
	db             *DB
}

func (q *preloadBuilder) Where(query interface{}, args ...interface{}) PreloadBuilder {
	q.db.Where(query, args...)
	return q
}

func (q *preloadBuilder) Or(query interface{}, args ...interface{}) PreloadBuilder {
	q.db.Where(query, args...)
	return q
}

func (q *preloadBuilder) Not(query interface{}, args ...interface{}) PreloadBuilder {
	q.db.Where(query, args...)
	return q
}

func (q *preloadBuilder) Select(columns ...string) PreloadBuilder {
	q.db.Select(columns)
	return q
}

func (q *preloadBuilder) Omit(columns ...string) PreloadBuilder {
	q.db.Omit(columns...)
	return q
}

func (q *preloadBuilder) Limit(limit int) PreloadBuilder {
	q.db.Limit(limit)
	return q
}

func (q *preloadBuilder) Offset(offset int) PreloadBuilder {
	q.db.Offset(offset)
	return q
}

func (q *preloadBuilder) Order(value interface{}) PreloadBuilder {
	q.db.Order(value)
	return q
}

func (q *preloadBuilder) LimitPerRecord(num int) PreloadBuilder {
	q.limitPerRecord = num
	return q
}

func (c chainG[T]) Joins(jt clause.JoinTarget, on func(db JoinBuilder, joinTable clause.Table, curTable clause.Table) error) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		if jt.Table == "" {
			jt.Table = clause.JoinTable(strings.Split(jt.Association, ".")...).Name
		}

		q := joinBuilder{db: db.Session(&Session{NewDB: true, Initialized: true}).Table(jt.Table)}
		if on != nil {
			if err := on(&q, clause.Table{Name: jt.Table}, clause.Table{Name: clause.CurrentTable}); err != nil {
				db.AddError(err)
			}
		}

		j := join{
			Name:     jt.Association,
			Alias:    jt.Table,
			Selects:  q.db.Statement.Selects,
			Omits:    q.db.Statement.Omits,
			JoinType: jt.Type,
		}

		if where, ok := q.db.Statement.Clauses["WHERE"].Expression.(clause.Where); ok {
			j.On = &where
		}

		if jt.Subquery != nil {
			joinType := j.JoinType
			if joinType == "" {
				joinType = clause.LeftJoin
			}

			if db, ok := jt.Subquery.(interface{ getInstance() *DB }); ok {
				stmt := db.getInstance().Statement
				if len(j.Selects) == 0 {
					j.Selects = stmt.Selects
				}
				if len(j.Omits) == 0 {
					j.Omits = stmt.Omits
				}
			}

			expr := clause.NamedExpr{SQL: fmt.Sprintf("%s JOIN (?) AS ?", joinType), Vars: []interface{}{jt.Subquery, clause.Table{Name: j.Alias}}}

			if j.On != nil {
				expr.SQL += " ON ?"
				expr.Vars = append(expr.Vars, clause.AndConditions{Exprs: j.On.Exprs})
			}

			j.Expression = expr
		}

		db.Statement.Joins = append(db.Statement.Joins, j)
		sort.Slice(db.Statement.Joins, func(i, j int) bool {
			return db.Statement.Joins[i].Name < db.Statement.Joins[j].Name
		})
		return db
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

func (c chainG[T]) Preload(association string, query func(db PreloadBuilder) error) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Preload(association, func(tx *DB) *DB {
			q := preloadBuilder{db: tx.getInstance()}
			if query != nil {
				if err := query(&q); err != nil {
					db.AddError(err)
				}
			}

			relation, ok := db.Statement.Schema.Relationships.Relations[association]
			if !ok {
				if preloadFields := strings.Split(association, "."); len(preloadFields) > 1 {
					relationships := db.Statement.Schema.Relationships
					for _, field := range preloadFields {
						var ok bool
						relation, ok = relationships.Relations[field]
						if ok {
							relationships = relation.FieldSchema.Relationships
						} else {
							db.AddError(fmt.Errorf("relation %s not found", association))
							return nil
						}
					}
				} else {
					db.AddError(fmt.Errorf("relation %s not found", association))
					return nil
				}
			}

			if q.limitPerRecord > 0 {
				if relation.JoinTable != nil {
					tx.AddError(fmt.Errorf("many2many relation %s don't support LimitPerRecord", association))
					return tx
				}

				refColumns := []clause.Column{}
				for _, rel := range relation.References {
					if rel.OwnPrimaryKey {
						refColumns = append(refColumns, clause.Column{Name: rel.ForeignKey.DBName})
					}
				}

				if len(refColumns) != 0 {
					selectExpr := clause.CommaExpression{}
					for _, column := range q.db.Statement.Selects {
						selectExpr.Exprs = append(selectExpr.Exprs, clause.Expr{SQL: "?", Vars: []interface{}{clause.Column{Name: column}}})
					}

					if len(selectExpr.Exprs) == 0 {
						selectExpr.Exprs = []clause.Expression{clause.Expr{SQL: "*", Vars: []interface{}{}}}
					}

					partitionBy := clause.CommaExpression{}
					for _, column := range refColumns {
						partitionBy.Exprs = append(partitionBy.Exprs, clause.Expr{SQL: "?", Vars: []interface{}{clause.Column{Name: column.Name}}})
					}

					rnnColumn := clause.Column{Name: "gorm_preload_rnn"}
					sql := "ROW_NUMBER() OVER (PARTITION BY ? ?)"
					vars := []interface{}{partitionBy}
					if orderBy, ok := q.db.Statement.Clauses["ORDER BY"]; ok {
						vars = append(vars, orderBy)
					} else {
						vars = append(vars, clause.Clause{Name: "ORDER BY", Expression: clause.OrderBy{
							Columns: []clause.OrderByColumn{{Column: clause.PrimaryColumn, Desc: true}},
						}})
					}
					vars = append(vars, rnnColumn)

					selectExpr.Exprs = append(selectExpr.Exprs, clause.Expr{SQL: sql + " AS ?", Vars: vars})

					q.db.Clauses(clause.Select{Expression: selectExpr})

					return q.db.Session(&Session{NewDB: true}).Unscoped().Table("(?) t", q.db).Where("? <= ?", rnnColumn, q.limitPerRecord)
				}
			}

			return q.db
		})
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

func (c chainG[T]) Build(builder clause.Builder) {
	subdb := c.getInstance()
	subdb.Logger = logger.Discard
	subdb.DryRun = true

	if stmt, ok := builder.(*Statement); ok {
		if subdb.Statement.SQL.Len() > 0 {
			var (
				vars = subdb.Statement.Vars
				sql  = subdb.Statement.SQL.String()
			)

			subdb.Statement.Vars = make([]interface{}, 0, len(vars))
			for _, vv := range vars {
				subdb.Statement.Vars = append(subdb.Statement.Vars, vv)
				bindvar := strings.Builder{}
				subdb.BindVarTo(&bindvar, subdb.Statement, vv)
				sql = strings.Replace(sql, bindvar.String(), "?", 1)
			}

			subdb.Statement.SQL.Reset()
			subdb.Statement.Vars = stmt.Vars
			if strings.Contains(sql, "@") {
				clause.NamedExpr{SQL: sql, Vars: vars}.Build(subdb.Statement)
			} else {
				clause.Expr{SQL: sql, Vars: vars}.Build(subdb.Statement)
			}
		} else {
			subdb.Statement.Vars = append(stmt.Vars, subdb.Statement.Vars...)
			subdb.callbacks.Query().Execute(subdb)
		}

		builder.WriteString(subdb.Statement.SQL.String())
		stmt.Vars = subdb.Statement.Vars
	}
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
