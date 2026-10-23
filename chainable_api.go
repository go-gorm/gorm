package gorm

import (
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/utils"
)

// Model selects the model whose schema and table are used by subsequent operations.
// A model with a non-zero primary key can also add a primary-key condition to
// updates and deletes. Model(&User{}) selects a table, not all-row authorization:
// an Update or Delete still needs a condition unless AllowGlobalUpdate is enabled.
// Model only changes statement state and does not execute SQL.
//
//	db.Model(&User{}).Where("active = ?", true).Update("name", "alice")
func (db *DB) Model(value interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Model = value
	return
}

// Clauses adds SQL clauses or statement modifiers without executing SQL.
// A clause.Interface merges by clause name, a StatementModifier changes statement
// state, and other expressions become WHERE conditions. Dialect support determines
// which clauses can be executed (for example locking or conflict handling).
//
//	db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, id)
//
// See https://gorm.io/docs/sql_builder.html#Clauses.
func (db *DB) Clauses(conds ...clause.Expression) (tx *DB) {
	tx = db.getInstance()
	var whereConds []interface{}

	for _, cond := range conds {
		if c, ok := cond.(clause.Interface); ok {
			tx.Statement.AddClause(c)
		} else if optimizer, ok := cond.(StatementModifier); ok {
			optimizer.ModifyStatement(tx.Statement)
		} else {
			whereConds = append(whereConds, cond)
		}
	}

	if len(whereConds) > 0 {
		tx.Statement.AddClause(clause.Where{Exprs: tx.Statement.BuildCondition(whereConds[0], whereConds[1:]...)})
	}
	return
}

// tableRegexp recognizes aliases in table expressions so model columns can be
// qualified with the alias. It is not a SQL parser or an input-validation boundary.
var tableRegexp = regexp.MustCompile(`(?i)(?:.+? AS (\w+)\s*(?:$|,)|^\w+\s+(\w+)$)`)

// Table overrides the table used by subsequent operations without supplying a model
// schema.
// A simple name is quoted; names containing spaces, backticks, or arguments are
// SQL expressions, so use only trusted table names and templates. An empty name
// clears the override. Use Model when field mapping or association metadata is needed.
//
//	db.Table("archived_users").Where("id = ?", id).Take(&user)
func (db *DB) Table(name string, args ...interface{}) (tx *DB) {
	tx = db.getInstance()
	if strings.Contains(name, " ") || strings.Contains(name, "`") || len(args) > 0 {
		tx.Statement.TableExpr = &clause.Expr{SQL: name, Vars: args}
		if results := tableRegexp.FindStringSubmatch(name); len(results) == 3 {
			if results[1] != "" {
				tx.Statement.Table = results[1]
			} else {
				tx.Statement.Table = results[2]
			}
		}
	} else if tables := strings.Split(name, "."); len(tables) == 2 {
		tx.Statement.TableExpr = &clause.Expr{SQL: tx.Statement.Quote(name)}
		tx.Statement.Table = tables[1]
	} else if name != "" {
		tx.Statement.TableExpr = &clause.Expr{SQL: tx.Statement.Quote(name)}
		tx.Statement.Table = name
	} else {
		tx.Statement.TableExpr = nil
		tx.Statement.Table = ""
	}
	return
}

// Distinct enables DISTINCT for subsequent queries. When args are supplied it also
// sets the selection using Select; otherwise it keeps the current selection.
// Distinct("name").Pluck("name", &names) retrieves unique names, and
// Distinct("name").Count(&count) counts distinct values of that column.
func (db *DB) Distinct(args ...interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Distinct = true
	if len(args) > 0 {
		tx = tx.Select(args[0], args[1:]...)
	}
	return
}

// Select chooses fields for queries, creates, or updates without executing SQL.
// It accepts field names, a []string, or a trusted SQL expression with bound args.
// For struct Updates, explicitly selecting a field includes its zero value;
// Select("*") selects all permitted fields. Omit and field permissions still apply.
//
//	db.Model(&user).Select("active").Updates(User{Active: false})
//
// SQL expressions and identifiers are not sanitized; bind data values using args.
func (db *DB) Select(query interface{}, args ...interface{}) (tx *DB) {
	tx = db.getInstance()

	switch v := query.(type) {
	case []string:
		tx.Statement.Selects = v

		for _, arg := range args {
			switch arg := arg.(type) {
			case string:
				tx.Statement.Selects = append(tx.Statement.Selects, arg)
			case []string:
				tx.Statement.Selects = append(tx.Statement.Selects, arg...)
			default:
				tx.AddError(fmt.Errorf("unsupported select args %v %v", query, args))
				return
			}
		}

		if clause, ok := tx.Statement.Clauses["SELECT"]; ok {
			clause.Expression = nil
			tx.Statement.Clauses["SELECT"] = clause
		}
	case string:
		if strings.Count(v, "?") >= len(args) && len(args) > 0 {
			tx.Statement.AddClause(clause.Select{
				Distinct:   db.Statement.Distinct,
				Expression: clause.Expr{SQL: v, Vars: args},
			})
		} else if strings.Count(v, "@") > 0 && len(args) > 0 {
			tx.Statement.AddClause(clause.Select{
				Distinct:   db.Statement.Distinct,
				Expression: clause.NamedExpr{SQL: v, Vars: args},
			})
		} else {
			tx.Statement.Selects = []string{v}

			for _, arg := range args {
				switch arg := arg.(type) {
				case string:
					tx.Statement.Selects = append(tx.Statement.Selects, arg)
				case []string:
					tx.Statement.Selects = append(tx.Statement.Selects, arg...)
				default:
					tx.Statement.AddClause(clause.Select{
						Distinct:   db.Statement.Distinct,
						Expression: clause.Expr{SQL: v, Vars: args},
					})
					return
				}
			}

			if clause, ok := tx.Statement.Clauses["SELECT"]; ok {
				clause.Expression = nil
				tx.Statement.Clauses["SELECT"] = clause
			}
		}
	default:
		tx.AddError(fmt.Errorf("unsupported select args %v %v", query, args))
	}

	return
}

// Omit excludes fields from subsequent queries or writes. It replaces the current
// omit list and accepts separate names or a single comma-separated list.
// Omit(clause.Associations) skips association saving; Omit("Languages.*") skips
// upserting associated languages while still allowing join-table references.
// See https://gorm.io/docs/associations.html#Skip-Auto-Create-Update.
func (db *DB) Omit(columns ...string) (tx *DB) {
	tx = db.getInstance()

	if len(columns) == 1 && strings.ContainsRune(columns[0], ',') {
		tx.Statement.Omits = strings.FieldsFunc(columns[0], utils.IsInvalidDBNameChar)
	} else {
		tx.Statement.Omits = columns
	}
	return
}

// MapColumns maps database result-column names to model field names during scanning.
// It does not rename SQL columns or alter the schema. The map is retained by the
// statement, so do not mutate it while operations using the mapping are running.
func (db *DB) MapColumns(m map[string]string) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.ColumnMapping = m
	return
}

// Where adds predicates to the current statement, normally combined with AND.
// query may be a trusted SQL template with bound args, a struct, a map, or a clause
// expression. Struct conditions omit zero values by default; use a map, or name the
// zero-valued fields as extra arguments to the same Where call
// (db.Where(&User{Name: "jinzhu"}, "Age")), when false, 0, or an empty string must
// participate in a condition.
//
//	db.Where("name = ?", name).Where(map[string]interface{}{"active": false}).Find(&users)
//
// Values belong in placeholders; query text and identifiers must be trusted.
// Where only builds conditions. See https://gorm.io/docs/query.html#Conditions.
func (db *DB) Where(query interface{}, args ...interface{}) (tx *DB) {
	tx = db.getInstance()
	if conds := tx.Statement.BuildCondition(query, args...); len(conds) > 0 {
		tx.Statement.AddClause(clause.Where{Exprs: conds})
	}
	return
}

// Not adds a negated condition using the same inputs and zero-value rules as Where.
// Expressions supporting negation can render their inverse operator; other
// expressions are wrapped in NOT. It does not execute a query.
//
//	db.Not(map[string]interface{}{"active": false}).Find(&users)
func (db *DB) Not(query interface{}, args ...interface{}) (tx *DB) {
	tx = db.getInstance()
	if conds := tx.Statement.BuildCondition(query, args...); len(conds) > 0 {
		tx.Statement.AddClause(clause.Where{Exprs: []clause.Expression{clause.Not(conds...)}})
	}
	return
}

// Or adds a disjunction to the current WHERE expression using Where's input rules.
// Use nested DB conditions to express explicit grouping when mixing AND and OR;
// SQL precedence can otherwise broaden a predicate unexpectedly.
//
//	db.Where(db.Where("role = ?", "admin").Or("role = ?", "owner")).Find(&users)
func (db *DB) Or(query interface{}, args ...interface{}) (tx *DB) {
	tx = db.getInstance()
	if conds := tx.Statement.BuildCondition(query, args...); len(conds) > 0 {
		tx.Statement.AddClause(clause.Where{Exprs: []clause.Expression{clause.Or(clause.And(conds...))}})
	}
	return
}

// Joins adds a LEFT JOIN for a named association, or a trusted raw JOIN expression.
// A *DB argument for an association contributes selected fields and ON conditions.
// Join preloading is intended for one-to-one associations such as belongs-to and
// has-one; use Preload for collection associations. Bind raw JOIN values with args.
//
//	db.Joins("Company").Find(&users)
//	db.Joins("JOIN emails ON emails.user_id = users.id AND emails.address = ?", address).Find(&users)
func (db *DB) Joins(query string, args ...interface{}) (tx *DB) {
	return joins(db, clause.LeftJoin, query, args...)
}

// InnerJoins adds an INNER JOIN for a named association using Joins' argument rules.
// Unlike a LEFT JOIN, it excludes parent rows without a matching related row.
// Raw join text is used as supplied and must be trusted.
func (db *DB) InnerJoins(query string, args ...interface{}) (tx *DB) {
	return joins(db, clause.InnerJoin, query, args...)
}

// joins records a join for the query callback. A single *DB argument supplies
// association field selection and ON predicates; other arguments remain expression
// values. SQL generation is deferred until the statement is executed.
func joins(db *DB, joinType clause.JoinType, query string, args ...interface{}) (tx *DB) {
	tx = db.getInstance()

	if len(args) == 1 {
		if db, ok := args[0].(*DB); ok {
			j := join{
				Name: query, Conds: args, Selects: db.Statement.Selects,
				Omits: db.Statement.Omits, JoinType: joinType,
			}
			if where, ok := db.Statement.Clauses["WHERE"].Expression.(clause.Where); ok {
				j.On = &where
			}
			tx.Statement.Joins = append(tx.Statement.Joins, j)
			return
		}
	}

	tx.Statement.Joins = append(tx.Statement.Joins, join{Name: query, Conds: args, JoinType: joinType})
	return
}

// Group adds a GROUP BY column or trusted SQL expression to subsequent queries.
// Pair it with Select for aggregate projections and Having for group predicates.
// Only simple column names are quoted; do not pass unvalidated user expressions.
func (db *DB) Group(name string) (tx *DB) {
	tx = db.getInstance()

	fields := strings.FieldsFunc(name, utils.IsInvalidDBNameChar)
	tx.Statement.AddClause(clause.GroupBy{
		Columns: []clause.Column{{Name: name, Raw: len(fields) != 1}},
	})
	return
}

// Having adds predicates evaluated after GROUP BY, using Where's condition formats.
// It appends to existing group predicates without executing SQL. Bind aggregate
// comparison values, for example Having("SUM(amount) > ?", minimum).
func (db *DB) Having(query interface{}, args ...interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.AddClause(clause.GroupBy{
		Having: tx.Statement.BuildCondition(query, args...),
	})
	return
}

// Order appends sorting for subsequent queries. It accepts a trusted SQL string,
// clause.OrderByColumn, or clause.OrderBy. String input is raw SQL; validate dynamic
// column names and directions or construct a clause.Column from an allowlist.
// Use OrderByColumn.Reorder to replace earlier order terms. Stable pagination
// requires deterministic ordering, usually including a unique key.
func (db *DB) Order(value interface{}) (tx *DB) {
	tx = db.getInstance()

	switch v := value.(type) {
	case clause.OrderBy:
		tx.Statement.AddClause(v)
	case clause.OrderByColumn:
		tx.Statement.AddClause(clause.OrderBy{
			Columns: []clause.OrderByColumn{v},
		})
	case string:
		if v != "" {
			tx.Statement.AddClause(clause.OrderBy{
				Columns: []clause.OrderByColumn{{
					Column: clause.Column{Name: v, Raw: true},
				}},
			})
		}
	}
	return
}

// Limit records the maximum number of rows to retrieve; Limit(-1) cancels a
// previous limit. A fresh Limit(0) requests no rows, but when merging into an
// existing non-nil limit, zero preserves that earlier value. Use a new session
// without conditions when a fresh zero limit is needed. Add deterministic Order
// when selecting a reproducible subset of rows.
func (db *DB) Limit(limit int) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.AddClause(clause.Limit{Limit: &limit})
	return
}

// Offset sets the number of rows to skip; Offset(-1) cancels a previous offset.
// It only modifies statement state. Use deterministic ordering for pagination;
// concurrent inserts and deletes can still shift offset-based pages.
func (db *DB) Offset(offset int) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.AddClause(clause.Limit{Offset: offset})
	return
}

// Scopes queues reusable statement transformations in argument order.
// Each function receives the current handle and must return the handle to continue
// with. Execution is deferred until callbacks run; scopes added by a scope are
// processed in a later pass. Return conditional changes rather than executing a
// finisher inside a scope when composing a query.
//
//	db.Scopes(func(tx *gorm.DB) *gorm.DB {
//		return tx.Where("tenant_id = ?", tenantID)
//	}).Find(&users)
func (db *DB) Scopes(funcs ...func(*DB) *DB) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.scopes = append(tx.Statement.scopes, funcs...)
	return tx
}

// executeScopes consumes the current scope queue and applies each transformation
// in order. Clearing the queue first leaves newly queued scopes for the callback
// processor's next pass rather than executing the old scopes repeatedly.
func (db *DB) executeScopes() (tx *DB) {
	scopes := db.Statement.scopes
	db.Statement.scopes = nil
	for _, scope := range scopes {
		db = scope(db)
	}
	return db
}

// Preload schedules an association to be fetched in separate queries after the parent
// query.
// query is a model association path such as "Orders.Items", not a table name.
// args may be conditions or a func(*DB) *DB that customizes the association query.
// Association predicates filter related rows, not the parent rows. Repeating a path
// replaces its arguments. clause.Associations loads direct associations, not all
// nested levels. See https://gorm.io/docs/preload.html.
func (db *DB) Preload(query string, args ...interface{}) (tx *DB) {
	tx = db.getInstance()
	if tx.Statement.Preloads == nil {
		tx.Statement.Preloads = map[string][]interface{}{}
	}
	tx.Statement.Preloads[query] = args
	return
}

// Attrs supplies initialization values for FirstOrInit or FirstOrCreate only when
// no matching row is found. These values are not search predicates, and Attrs alone
// does not write anything. Use a map to explicitly initialize zero-valued fields;
// struct attributes omit zero values. A later Attrs call replaces earlier values.
func (db *DB) Attrs(attrs ...interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.attrs = attrs
	return
}

// Assign supplies values for FirstOrInit or FirstOrCreate whether or not a row is
// found.
// FirstOrInit changes only the in-memory destination; FirstOrCreate also updates an
// existing database row. These values are not search predicates. Use a map to assign
// zero values explicitly; struct attributes omit zero values.
func (db *DB) Assign(attrs ...interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.assigns = attrs
	return
}

// Unscoped disables GORM's soft-delete filtering for models with a soft-delete field
// such as DeletedAt. Queries can then return deleted rows, and Delete physically
// deletes matching rows instead of setting the deletion timestamp.
// It does not remove user WHERE clauses or bypass the global-update guard.
//
//	db.Unscoped().Where("id = ?", id).Delete(&User{})
func (db *DB) Unscoped() (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Unscoped = true
	return
}

// Raw builds a SQL statement with positional or named bound values; it does not execute
// it.
// Finish with Scan, Row, or Rows to read results, or use Exec directly for writes.
// Raw SQL is used as supplied: ordinary model scopes, soft-delete filters, and
// chainable WHERE clauses are not automatically appended to it.
//
//	db.Raw("SELECT id, name FROM users WHERE id = ?", id).Scan(&user)
//
// The SQL template must be trusted. Never concatenate user values into sql.
func (db *DB) Raw(sql string, values ...interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.SQL = strings.Builder{}

	if strings.Contains(sql, "@") {
		clause.NamedExpr{SQL: sql, Vars: values}.Build(tx.Statement)
	} else {
		clause.Expr{SQL: sql, Vars: values}.Build(tx.Statement)
	}
	return
}
