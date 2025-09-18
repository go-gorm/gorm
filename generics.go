package gorm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
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
	ExecInterface[T]
	// chain methods available at start; return ChainInterface
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

	Table(name string, args ...interface{}) CreateInterface[T]
	Create(ctx context.Context, r *T) error
	CreateInBatches(ctx context.Context, r *[]T, batchSize int) error
	Set(assignments ...clause.Assigner) SetCreateOrUpdateInterface[T]
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
	Set(assignments ...clause.Assigner) SetUpdateOnlyInterface[T]

	Build(builder clause.Builder)

	Table(name string, args ...interface{}) ChainInterface[T]
	Delete(ctx context.Context) (rowsAffected int, err error)
	Update(ctx context.Context, name string, value any) (rowsAffected int, err error)
	Updates(ctx context.Context, t T) (rowsAffected int, err error)
	Count(ctx context.Context, column string) (result int64, err error)
}

// SetUpdateOnlyInterface is returned by Set after chaining; only Update is allowed
type SetUpdateOnlyInterface[T any] interface {
	Update(ctx context.Context) (rowsAffected int, err error)
}

// SetCreateOrUpdateInterface is returned by Set at start; Create or Update are allowed
type SetCreateOrUpdateInterface[T any] interface {
	Create(ctx context.Context) error
	Update(ctx context.Context) (rowsAffected int, err error)
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
			var r T
			return db.Model(r).Raw(sql, values...)
		}),
	}}
}

func (c *g[T]) Exec(ctx context.Context, sql string, values ...interface{}) error {
	var r T
	return c.apply(ctx).Model(r).Exec(sql, values...).Error
}

type createG[T any] struct {
	chainG[T]
}

func (c createG[T]) Table(name string, args ...interface{}) CreateInterface[T] {
	return createG[T]{c.with(func(db *DB) *DB {
		return db.Table(name, args...)
	})}
}

func (c createG[T]) Set(assignments ...clause.Assigner) SetCreateOrUpdateInterface[T] {
	return c.processSet(assignments...)
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

func (c chainG[T]) Table(name string, args ...interface{}) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		return db.Table(name, args...)
	})
}

func (c chainG[T]) Scopes(scopes ...func(db *Statement)) ChainInterface[T] {
	return c.with(func(db *DB) *DB {
		for _, fc := range scopes {
			fc(db.Statement)
		}
		return db
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

func (c chainG[T]) Set(assignments ...clause.Assigner) SetUpdateOnlyInterface[T] {
	return c.processSet(assignments...)
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
					relationships := &db.Statement.Schema.Relationships
					for _, field := range preloadFields {
						var ok bool
						relation, ok = relationships.Relations[field]
						if ok {
							relationships = &relation.FieldSchema.Relationships
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
	err := g.g.apply(ctx).Model(r).Find(result).Error
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

func (c chainG[T]) processSet(items ...clause.Assigner) setCreateOrUpdateG[T] {
	var (
		assigns  []clause.Assignment
		assocOps []clause.Association
	)

	for _, item := range items {
		// Check if it's an AssociationAssigner
		if assocAssigner, ok := item.(clause.AssociationAssigner); ok {
			assocOps = append(assocOps, assocAssigner.AssociationAssignments()...)
		} else {
			assigns = append(assigns, item.Assignments()...)
		}
	}

	return setCreateOrUpdateG[T]{
		c:        c,
		assigns:  assigns,
		assocOps: assocOps,
	}
}

// setCreateOrUpdateG[T] is a struct that holds operations to be executed in a batch.
// It supports regular assignments and association operations.
type setCreateOrUpdateG[T any] struct {
	c        chainG[T]
	assigns  []clause.Assignment
	assocOps []clause.Association
}

func (s setCreateOrUpdateG[T]) Update(ctx context.Context) (rowsAffected int, err error) {
	// Execute association operations
	for _, assocOp := range s.assocOps {
		if err := s.executeAssociationOperation(ctx, assocOp); err != nil {
			return 0, err
		}
	}

	// Execute assignment operations
	if len(s.assigns) > 0 {
		var r T
		res := s.c.g.apply(ctx).Model(r).Clauses(clause.Set(s.assigns)).Updates(map[string]interface{}{})
		return int(res.RowsAffected), res.Error
	}

	return 0, nil
}

func (s setCreateOrUpdateG[T]) Create(ctx context.Context) error {
	// Execute association operations
	for _, assocOp := range s.assocOps {
		if err := s.executeAssociationOperation(ctx, assocOp); err != nil {
			return err
		}
	}

	// Execute assignment operations
	if len(s.assigns) > 0 {
		data := make(map[string]interface{}, len(s.assigns))
		for _, a := range s.assigns {
			data[a.Column.Name] = a.Value
		}
		var r T
		return s.c.g.apply(ctx).Model(r).Create(data).Error
	}

	return nil
}

// executeAssociationOperation executes an association operation
func (s setCreateOrUpdateG[T]) executeAssociationOperation(ctx context.Context, op clause.Association) error {
	var r T
	base := s.c.g.apply(ctx).Model(r)

	switch op.Type {
	case clause.OpCreate:
		return s.handleAssociationCreate(ctx, base, op)
	case clause.OpUnlink, clause.OpDelete, clause.OpUpdate:
		return s.handleAssociation(ctx, base, op)
	default:
		return fmt.Errorf("unknown association operation type: %v", op.Type)
	}
}

func (s setCreateOrUpdateG[T]) handleAssociationCreate(ctx context.Context, base *DB, op clause.Association) error {
	if len(op.Set) > 0 {
		return s.handleAssociationForOwners(base, ctx, func(owner T, assoc *Association) error {
			data := make(map[string]interface{}, len(op.Set))
			for _, a := range op.Set {
				data[a.Column.Name] = a.Value
			}
			return assoc.Append(data)
		}, op.Association)
	}

	return s.handleAssociationForOwners(base, ctx, func(owner T, assoc *Association) error {
		return assoc.Append(op.Values...)
	}, op.Association)
}

// handleAssociationForOwners is a helper function that handles associations for all owners
func (s setCreateOrUpdateG[T]) handleAssociationForOwners(base *DB, ctx context.Context, handler func(owner T, association *Association) error, associationName string) error {
	var owners []T
	if err := base.Find(&owners).Error; err != nil {
		return err
	}

	for _, owner := range owners {
		assoc := base.Session(&Session{NewDB: true, Context: ctx}).Model(&owner).Association(associationName)
		if assoc.Error != nil {
			return assoc.Error
		}

		if err := handler(owner, assoc); err != nil {
			return err
		}
	}
	return nil
}

func (s setCreateOrUpdateG[T]) handleAssociation(ctx context.Context, base *DB, op clause.Association) error {
	assoc := base.Association(op.Association)
	if assoc.Error != nil {
		return assoc.Error
	}

	var (
		rel            = assoc.Relationship
		assocModel     = reflect.New(rel.FieldSchema.ModelType).Interface()
		fkNil          = map[string]any{}
		setMap         = make(map[string]any, len(op.Set))
		ownerPKNames   []string
		ownerFKNames   []string
		primaryColumns []any
		foreignColumns []any
	)

	for _, a := range op.Set {
		setMap[a.Column.Name] = a.Value
	}

	for _, ref := range rel.References {
		fkNil[ref.ForeignKey.DBName] = nil

		if ref.OwnPrimaryKey && ref.PrimaryKey != nil {
			ownerPKNames = append(ownerPKNames, ref.PrimaryKey.DBName)
			primaryColumns = append(primaryColumns, clause.Column{Name: ref.PrimaryKey.DBName})
			foreignColumns = append(foreignColumns, clause.Column{Name: ref.ForeignKey.DBName})
		} else if !ref.OwnPrimaryKey && ref.PrimaryKey != nil {
			ownerFKNames = append(ownerFKNames, ref.ForeignKey.DBName)
			primaryColumns = append(primaryColumns, clause.Column{Name: ref.PrimaryKey.DBName})
		}
	}

	assocDB := s.c.g.db.Session(&Session{NewDB: true, Context: ctx}).Model(assocModel).Where(op.Conditions)

	switch rel.Type {
	case schema.HasOne, schema.HasMany:
		assocDB = assocDB.Where("? IN (?)", foreignColumns, base.Select(ownerPKNames))
		switch op.Type {
		case clause.OpUnlink:
			return assocDB.Updates(fkNil).Error
		case clause.OpDelete:
			return assocDB.Delete(assocModel).Error
		case clause.OpUpdate:
			return assocDB.Updates(setMap).Error
		}
	case schema.BelongsTo:
		switch op.Type {
		case clause.OpDelete:
			return base.Transaction(func(tx *DB) error {
				assocDB.Statement.ConnPool = tx.Statement.ConnPool
				base.Statement.ConnPool = tx.Statement.ConnPool

				if err := assocDB.Where("? IN (?)", primaryColumns, base.Select(ownerFKNames)).Delete(assocModel).Error; err != nil {
					return err
				}
				return base.Updates(fkNil).Error
			})
		case clause.OpUnlink:
			return base.Updates(fkNil).Error
		case clause.OpUpdate:
			return assocDB.Where("? IN (?)", primaryColumns, base.Select(ownerFKNames)).Updates(setMap).Error
		}
	case schema.Many2Many:
		joinModel := reflect.New(rel.JoinTable.ModelType).Interface()
		joinDB := base.Session(&Session{NewDB: true, Context: ctx}).Model(joinModel)

		// EXISTS owners: owners.pk = join.owner_fk for all owner refs
		ownersExists := base.Session(&Session{NewDB: true, Context: ctx}).Table(rel.Schema.Table).Select("1")
		for _, ref := range rel.References {
			if ref.OwnPrimaryKey && ref.PrimaryKey != nil {
				ownersExists = ownersExists.Where(clause.Eq{
					Column: clause.Column{Table: rel.Schema.Table, Name: ref.PrimaryKey.DBName},
					Value:  clause.Column{Table: rel.JoinTable.Table, Name: ref.ForeignKey.DBName},
				})
			}
		}

		// EXISTS related: related.pk = join.rel_fk for all related refs, plus optional conditions
		relatedExists := base.Session(&Session{NewDB: true, Context: ctx}).Table(rel.FieldSchema.Table).Select("1")
		for _, ref := range rel.References {
			if !ref.OwnPrimaryKey && ref.PrimaryKey != nil {
				relatedExists = relatedExists.Where(clause.Eq{
					Column: clause.Column{Table: rel.FieldSchema.Table, Name: ref.PrimaryKey.DBName},
					Value:  clause.Column{Table: rel.JoinTable.Table, Name: ref.ForeignKey.DBName},
				})
			}
		}
		relatedExists = relatedExists.Where(op.Conditions)

		switch op.Type {
		case clause.OpUnlink, clause.OpDelete:
			joinDB = joinDB.Where("EXISTS (?)", ownersExists)
			if len(op.Conditions) > 0 {
				joinDB = joinDB.Where("EXISTS (?)", relatedExists)
			}
			return joinDB.Delete(nil).Error
		case clause.OpUpdate:
			// Update related table rows that have join rows matching owners
			relatedDB := base.Session(&Session{NewDB: true, Context: ctx}).Table(rel.FieldSchema.Table).Where(op.Conditions)

			// correlated join subquery: join.rel_fk = related.pk AND EXISTS owners
			joinSub := base.Session(&Session{NewDB: true, Context: ctx}).Table(rel.JoinTable.Table).Select("1")
			for _, ref := range rel.References {
				if !ref.OwnPrimaryKey && ref.PrimaryKey != nil {
					joinSub = joinSub.Where(clause.Eq{
						Column: clause.Column{Table: rel.JoinTable.Table, Name: ref.ForeignKey.DBName},
						Value:  clause.Column{Table: rel.FieldSchema.Table, Name: ref.PrimaryKey.DBName},
					})
				}
			}
			joinSub = joinSub.Where("EXISTS (?)", ownersExists)
			return relatedDB.Where("EXISTS (?)", joinSub).Updates(setMap).Error
		}
	}
	return errors.New("unsupported relationship")
}
