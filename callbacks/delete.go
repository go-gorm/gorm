package callbacks

import (
	"reflect"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
)

// BeforeDelete invokes the model's BeforeDelete hook when schema metadata enables it,
// db has no error, and SkipHooks is false. Hook errors are recorded on db before
// association or owner deletion.
func BeforeDelete(db *gorm.DB) {
	if db.Error == nil && db.Statement.Schema != nil && !db.Statement.SkipHooks && db.Statement.Schema.BeforeDelete {
		callMethod(db, func(value interface{}, tx *gorm.DB) bool {
			if i, ok := value.(BeforeDeleteInterface); ok {
				db.AddError(i.BeforeDelete(tx))
				return true
			}

			return false
		})
	}
}

// DeleteBeforeAssociations deletes explicitly selected has-one/has-many associations
// and many-to-many join rows before deleting the owner. It derives predicates from
// nonzero owner identities and respects nested association selections. Has-one/has-many
// deletion explicitly propagates Unscoped; the fresh many-to-many join statement
// inherits it only when PropagateUnscoped is enabled, which matters for soft-deletable
// join models. Ordinary Delete does not recursively delete every association, and
// many-to-many target rows are not deleted by this callback. Related deletions share
// the operation's connection or transaction.
func DeleteBeforeAssociations(db *gorm.DB) {
	if db.Error == nil && db.Statement.Schema != nil {
		selectColumns, restricted := db.Statement.SelectAndOmitColumns(true, false)
		if !restricted {
			return
		}

		for column, v := range selectColumns {
			if !v {
				continue
			}

			rel, ok := db.Statement.Schema.Relationships.Relations[column]
			if !ok {
				continue
			}

			switch rel.Type {
			case schema.HasOne, schema.HasMany:
				queryConds := rel.ToQueryConditions(db.Statement.Context, db.Statement.ReflectValue)
				modelValue := reflect.New(rel.FieldSchema.ModelType).Interface()
				tx := db.Session(&gorm.Session{NewDB: true}).Model(modelValue)
				withoutConditions := false
				if db.Statement.Unscoped {
					tx = tx.Unscoped()
				}

				if len(db.Statement.Selects) > 0 {
					selects := make([]string, 0, len(db.Statement.Selects))
					for _, s := range db.Statement.Selects {
						if s == clause.Associations {
							selects = append(selects, s)
						} else if columnPrefix := column + "."; strings.HasPrefix(s, columnPrefix) {
							selects = append(selects, strings.TrimPrefix(s, columnPrefix))
						}
					}

					if len(selects) > 0 {
						tx = tx.Select(selects)
					}
				}

				for _, cond := range queryConds {
					if c, ok := cond.(clause.IN); ok && len(c.Values) == 0 {
						withoutConditions = true
						break
					}
				}

				if !withoutConditions && db.AddError(tx.Clauses(clause.Where{Exprs: queryConds}).Delete(modelValue).Error) != nil {
					return
				}
			case schema.Many2Many:
				var (
					queryConds     = make([]clause.Expression, 0, len(rel.References))
					foreignFields  = make([]*schema.Field, 0, len(rel.References))
					relForeignKeys = make([]string, 0, len(rel.References))
					modelValue     = reflect.New(rel.JoinTable.ModelType).Interface()
					table          = rel.JoinTable.Table
					tx             = db.Session(&gorm.Session{NewDB: true}).Model(modelValue).Table(table)
				)

				for _, ref := range rel.References {
					if ref.OwnPrimaryKey {
						foreignFields = append(foreignFields, ref.PrimaryKey)
						relForeignKeys = append(relForeignKeys, ref.ForeignKey.DBName)
					} else if ref.PrimaryValue != "" {
						queryConds = append(queryConds, clause.Eq{
							Column: clause.Column{Table: rel.JoinTable.Table, Name: ref.ForeignKey.DBName},
							Value:  ref.PrimaryValue,
						})
					}
				}

				_, foreignValues := schema.GetIdentityFieldValuesMap(db.Statement.Context, db.Statement.ReflectValue, foreignFields)
				column, values := schema.ToQueryValues(table, relForeignKeys, foreignValues)
				queryConds = append(queryConds, clause.IN{Column: column, Values: values})

				if db.AddError(tx.Clauses(clause.Where{Exprs: queryConds}).Delete(modelValue).Error) != nil {
					return
				}
			}
		}

	}
}

// Delete returns the dialector-configured deletion callback. It adds schema delete
// clauses; the built-in soft-delete modifier decides whether to replace DELETE with
// UPDATE based on Unscoped. For
// ordinary deletion it adds primary-key predicates from Dest or Model and builds
// DELETE/FROM. The missing-WHERE guard prevents an unqualified operation unless
// AllowGlobalUpdate is enabled.
//
// DryRun builds without executing. Supported RETURNING scans deleted rows and closes
// the result set; other operations use ExecContext and report RowsAffected. Hooks,
// selected association deletion, and transaction completion belong to the surrounding
// callback chain.
func Delete(config *Config) func(db *gorm.DB) {
	supportReturning := utils.Contains(config.DeleteClauses, "RETURNING")

	return func(db *gorm.DB) {
		if db.Error != nil {
			return
		}

		if db.Statement.Schema != nil {
			for _, c := range db.Statement.Schema.DeleteClauses {
				db.Statement.AddClause(c)
			}
		}

		if db.Statement.SQL.Len() == 0 {
			db.Statement.SQL.Grow(100)
			db.Statement.AddClauseIfNotExists(clause.Delete{})

			if db.Statement.Schema != nil {
				_, queryValues := schema.GetIdentityFieldValuesMap(db.Statement.Context, db.Statement.ReflectValue, db.Statement.Schema.PrimaryFields)
				column, values := schema.ToQueryValues(db.Statement.Table, db.Statement.Schema.PrimaryFieldDBNames, queryValues)

				if len(values) > 0 {
					db.Statement.AddClause(clause.Where{Exprs: []clause.Expression{clause.IN{Column: column, Values: values}}})
				}

				if db.Statement.ReflectValue.CanAddr() && db.Statement.Dest != db.Statement.Model && db.Statement.Model != nil {
					_, queryValues = schema.GetIdentityFieldValuesMap(db.Statement.Context, reflect.ValueOf(db.Statement.Model), db.Statement.Schema.PrimaryFields)
					column, values = schema.ToQueryValues(db.Statement.Table, db.Statement.Schema.PrimaryFieldDBNames, queryValues)

					if len(values) > 0 {
						db.Statement.AddClause(clause.Where{Exprs: []clause.Expression{clause.IN{Column: column, Values: values}}})
					}
				}
			}

			db.Statement.AddClauseIfNotExists(clause.From{})

			db.Statement.Build(db.Statement.BuildClauses...)
		}

		checkMissingWhereConditions(db)

		if !db.DryRun && db.Error == nil {
			ok, mode := hasReturning(db, supportReturning)
			if !ok {
				result, err := db.Statement.ConnPool.ExecContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)

				if db.AddError(err) == nil {
					db.RowsAffected, _ = result.RowsAffected()

					if db.Statement.Result != nil {
						db.Statement.Result.Result = result
						db.Statement.Result.RowsAffected = db.RowsAffected
					}
				}

				return
			}

			if rows, err := db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...); db.AddError(err) == nil {
				defer func() {
					if err := rows.Close(); err != nil {
						_ = db.AddError(err)
					}
				}()

				gorm.Scan(rows, db, mode)

				if db.Statement.Result != nil {
					db.Statement.Result.RowsAffected = db.RowsAffected
				}
			}
		}
	}
}

// AfterDelete invokes the model's AfterDelete hook when db has no error, schema
// metadata enables it, and SkipHooks is false. It applies to both soft and physical
// deletion and runs before the owned default transaction commits.
func AfterDelete(db *gorm.DB) {
	if db.Error == nil && db.Statement.Schema != nil && !db.Statement.SkipHooks && db.Statement.Schema.AfterDelete {
		callMethod(db, func(value interface{}, tx *gorm.DB) bool {
			if i, ok := value.(AfterDeleteInterface); ok {
				db.AddError(i.AfterDelete(tx))
				return true
			}
			return false
		})
	}
}
