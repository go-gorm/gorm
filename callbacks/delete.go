package callbacks

import (
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
)

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

func parseNestedDelete(schema *schema.Schema, selects []string) map[string][]string {
	result := make(map[string][]string)

	for _, selectItem := range selects {
		switch {
		case selectItem == clause.Associations:
			for name := range schema.Relationships.Relations {
				result[name] = nil
			}
		case strings.Contains(selectItem, "."):
			parts := strings.Split(selectItem, ".")
			if len(parts) > 0 {
				firstRel := parts[0]
				if _, ok := schema.Relationships.Relations[firstRel]; ok {
					if len(parts) > 1 {
						result[firstRel] = append(result[firstRel], strings.Join(parts[1:], "."))
					} else {
						result[firstRel] = nil
					}
				}
			}
		default:
			if _, ok := schema.Relationships.Relations[selectItem]; ok {
				result[selectItem] = nil
			}
		}
	}

	return result
}

func deleteNestedAssociations(db *gorm.DB, rel *schema.Relationship, nestedPaths []string) error {
	switch rel.Type {
	case schema.HasOne, schema.HasMany:
		return deleteHasOneOrManyNestedAssociations(db, rel, nestedPaths)
	case schema.Many2Many:
		return deleteMany2ManyNestedAssociations(db, rel, nestedPaths)
	case schema.BelongsTo:
		return deleteBelongsToNestedAssociations(db, rel, nestedPaths)
	}
	return nil
}

func deleteWithNestedSelect(db *gorm.DB, value interface{}, nestedPaths []string) error {
	tx := db.Session(&gorm.Session{NewDB: true})
	for _, path := range nestedPaths {
		tx = tx.Select(path)
	}
	return tx.Delete(value).Error
}

func DeleteBeforeAssociations(db *gorm.DB) {
	if db.Error == nil && db.Statement.Schema != nil {
		if len(db.Statement.Selects) > 0 {
			hasRelationshipSelects := false
			for _, selectItem := range db.Statement.Selects {
				if selectItem == clause.Associations {
					hasRelationshipSelects = true
					break
				}
				if _, ok := db.Statement.Schema.Relationships.Relations[selectItem]; ok {
					hasRelationshipSelects = true
					break
				}
				if strings.Contains(selectItem, ".") {
					parts := strings.Split(selectItem, ".")
					if len(parts) > 0 {
						if _, ok := db.Statement.Schema.Relationships.Relations[parts[0]]; ok {
							hasRelationshipSelects = true
							break
						}
					}
				}
			}

			if hasRelationshipSelects {
				hasClauseAssociations := false
				var otherSelects []string
				for _, s := range db.Statement.Selects {
					if s == clause.Associations {
						hasClauseAssociations = true
					} else {
						otherSelects = append(otherSelects, s)
					}
				}

				if hasClauseAssociations {
					selectColumns, restricted := db.Statement.SelectAndOmitColumns(true, false)

					if restricted {
						explicitRelations := make(map[string]bool)
						for _, s := range otherSelects {
							if strings.Contains(s, ".") {
								parts := strings.Split(s, ".")
								if len(parts) > 0 {
									explicitRelations[parts[0]] = true
								}
							} else {
								explicitRelations[s] = true
							}
						}

						for column, v := range selectColumns {
							if !v {
								continue
							}

							if explicitRelations[column] {
								continue
							}

							rel, ok := db.Statement.Schema.Relationships.Relations[column]
							if !ok {
								continue
							}

							if err := deleteAssociation(db, rel); err != nil {
								_ = db.AddError(err)
								return
							}
						}
					}
				}

				if len(otherSelects) > 0 {
					for _, selectItem := range otherSelects {
						if selectItem != clause.Associations {
							parts := strings.Split(selectItem, ".")
							if len(parts) > 0 {
								firstRel := parts[0]
								if _, ok := db.Statement.Schema.Relationships.Relations[firstRel]; !ok {
									if field := db.Statement.Schema.LookUpField(firstRel); field != nil {
										_ = db.AddError(fmt.Errorf("field %s is not a valid relationship", firstRel))
										return
									} else {
										_ = db.AddError(fmt.Errorf("%s is not a valid relationship", firstRel))
										return
									}
								}
							}
						}
					}

					if db.Statement.ReflectValue.Kind() == reflect.Struct || db.Statement.ReflectValue.Kind() == reflect.Slice {
						needsLoad := false

						if db.Statement.ReflectValue.Kind() == reflect.Struct {
							isZero := true
							for _, field := range db.Statement.Schema.PrimaryFields {
								value, _ := field.ValueOf(db.Statement.Context, db.Statement.ReflectValue)
								if !reflect.ValueOf(value).IsZero() {
									isZero = false
									break
								}
							}
							needsLoad = isZero
						} else if db.Statement.ReflectValue.Kind() == reflect.Slice {
							needsLoad = db.Statement.ReflectValue.Len() == 0
						}

						if needsLoad {
							loadDB := db.Session(&gorm.Session{NewDB: true}).Model(db.Statement.Dest)
							if db.Statement.Unscoped {
								loadDB = loadDB.Unscoped()
							}

							if whereClause, ok := db.Statement.Clauses["WHERE"]; ok {
								if where, ok := whereClause.Expression.(clause.Where); ok {
									loadDB.Statement.AddClause(where)
								}
							}

							if err := loadDB.First(db.Statement.Dest).Error; err != nil {
								return
							}
							db.Statement.ReflectValue = reflect.ValueOf(db.Statement.Dest).Elem()
						}
					}

					nestedDeletes := parseNestedDelete(db.Statement.Schema, otherSelects)

					for relName, nestedPaths := range nestedDeletes {
						rel := db.Statement.Schema.Relationships.Relations[relName]
						if rel == nil {
							continue
						}

						if err := deleteNestedAssociations(db, rel, nestedPaths); err != nil {
							_ = db.AddError(err)
							return
						}
					}
				}
				return
			}
		}

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

			if err := deleteAssociation(db, rel); err != nil {
				_ = db.AddError(err)
				return
			}
		}
	}
}

func deleteAssociation(db *gorm.DB, rel *schema.Relationship) error {
	switch rel.Type {
	case schema.HasOne, schema.HasMany:
		return deleteHasOneOrManyAssociation(db, rel)
	case schema.Many2Many:
		return deleteMany2ManyAssociation(db, rel)
	case schema.BelongsTo:
		// For clause.Associations, BelongsTo should not be deleted
		// as it would violate foreign key constraints
		return nil
	}
	return nil
}

func deleteHasOneOrManyAssociation(db *gorm.DB, rel *schema.Relationship) error {
	queryConds := rel.ToQueryConditions(db.Statement.Context, db.Statement.ReflectValue)
	modelValue := reflect.New(rel.FieldSchema.ModelType).Interface()
	tx := db.Session(&gorm.Session{NewDB: true}).Model(modelValue)

	if db.Statement.Unscoped {
		tx = tx.Unscoped()
	}

	if hasEmptyConditions(queryConds) {
		return nil
	}

	return tx.Clauses(clause.Where{Exprs: queryConds}).Delete(modelValue).Error
}

func deleteMany2ManyAssociation(db *gorm.DB, rel *schema.Relationship) error {
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

	if len(values) > 0 {
		queryConds = append(queryConds, clause.IN{Column: column, Values: values})
	}

	if len(queryConds) > 0 {
		return tx.Clauses(clause.Where{Exprs: queryConds}).Delete(modelValue).Error
	}
	return nil
}

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
				gorm.Scan(rows, db, mode)

				if db.Statement.Result != nil {
					db.Statement.Result.RowsAffected = db.RowsAffected
				}
				db.AddError(rows.Close())
			}
		}
	}
}

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

func hasEmptyConditions(queryConds []clause.Expression) bool {
	for _, cond := range queryConds {
		if c, ok := cond.(clause.IN); ok && len(c.Values) == 0 {
			return true
		}
	}
	return false
}

func deleteHasOneOrManyNestedAssociations(db *gorm.DB, rel *schema.Relationship, nestedPaths []string) error {
	queryConds := rel.ToQueryConditions(db.Statement.Context, db.Statement.ReflectValue)
	modelValue := reflect.New(rel.FieldSchema.ModelType).Interface()
	tx := db.Session(&gorm.Session{NewDB: true}).Model(modelValue)

	if db.Statement.Unscoped {
		tx = tx.Unscoped()
	}

	if hasEmptyConditions(queryConds) {
		return nil
	}

	if len(nestedPaths) > 0 {
		return deleteWithNestedPaths(tx, rel, queryConds, nestedPaths)
	}

	result := tx.Clauses(clause.Where{Exprs: queryConds}).Delete(modelValue)
	return result.Error
}

func deleteWithNestedPaths(tx *gorm.DB, rel *schema.Relationship, queryConds []clause.Expression, nestedPaths []string) error {
	var records reflect.Value
	searchTx := tx.Unscoped()

	if rel.Type == schema.HasOne {
		records = reflect.New(rel.FieldSchema.ModelType)
		if err := searchTx.Clauses(clause.Where{Exprs: queryConds}).First(records.Interface()).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}
		if records.Elem().IsZero() {
			return nil
		}
	} else {
		records = reflect.New(reflect.SliceOf(rel.FieldSchema.ModelType))
		if err := searchTx.Clauses(clause.Where{Exprs: queryConds}).Find(records.Interface()).Error; err != nil {
			return err
		}
		if records.Elem().Len() == 0 {
			return nil
		}
	}

	return deleteWithNestedSelect(tx, records.Interface(), nestedPaths)
}

func deleteMany2ManyNestedAssociations(db *gorm.DB, rel *schema.Relationship, nestedPaths []string) error {
	associatedRecords, err := findMany2ManyAssociatedRecords(db, rel)
	if err != nil {
		return err
	}

	if associatedRecords.IsValid() && associatedRecords.Elem().Len() > 0 {
		if len(nestedPaths) > 0 {
			if err := deleteWithNestedSelect(db.Session(&gorm.Session{NewDB: true}), associatedRecords.Interface(), nestedPaths); err != nil {
				return err
			}
		} else {
			if err := db.Session(&gorm.Session{NewDB: true}).Delete(associatedRecords.Interface()).Error; err != nil {
				return err
			}
		}
	}

	return deleteMany2ManyJoinTable(db, rel)
}

func findMany2ManyAssociatedRecords(db *gorm.DB, rel *schema.Relationship) (reflect.Value, error) {
	joinTable := rel.JoinTable.Table
	selectQuery := db.Session(&gorm.Session{NewDB: true})

	if db.Statement.Unscoped {
		selectQuery = selectQuery.Unscoped()
	}

	var joinConditions []string
	var queryArgs []interface{}

	for _, ref := range rel.References {
		if ref.OwnPrimaryKey {
			if db.Statement.ReflectValue.Kind() == reflect.Slice {
				continue
			}
			value, _ := ref.PrimaryKey.ValueOf(db.Statement.Context, db.Statement.ReflectValue)
			joinConditions = append(joinConditions, joinTable+"."+ref.ForeignKey.DBName+" = ?")
			queryArgs = append(queryArgs, value)
		}
	}

	if len(joinConditions) == 0 {
		return reflect.Value{}, nil
	}

	if len(rel.References) == 0 || len(rel.FieldSchema.PrimaryFieldDBNames) == 0 {
		return reflect.Value{}, fmt.Errorf("missing references or primary field names for relationship")
	}

	associatedRecords := reflect.New(reflect.SliceOf(rel.FieldSchema.ModelType))
	query := selectQuery.Table(rel.FieldSchema.Table).
		Joins("INNER JOIN "+joinTable+" ON "+rel.FieldSchema.Table+"."+rel.FieldSchema.PrimaryFieldDBNames[0]+" = "+joinTable+"."+rel.References[len(rel.References)-1].ForeignKey.DBName).
		Where(strings.Join(joinConditions, " AND "), queryArgs...)

	err := query.Find(associatedRecords.Interface()).Error
	return associatedRecords, err
}

func deleteMany2ManyJoinTable(db *gorm.DB, rel *schema.Relationship) error {
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

	if len(values) > 0 {
		queryConds = append(queryConds, clause.IN{Column: column, Values: values})
	}

	if len(queryConds) > 0 {
		return tx.Clauses(clause.Where{Exprs: queryConds}).Delete(modelValue).Error
	}
	return nil
}

func deleteBelongsToNestedAssociations(db *gorm.DB, rel *schema.Relationship, nestedPaths []string) error {
	if len(nestedPaths) == 0 {
		return nil
	}

	queryConds := rel.ToQueryConditions(db.Statement.Context, db.Statement.ReflectValue)
	modelValue := reflect.New(rel.FieldSchema.ModelType).Interface()
	tx := db.Session(&gorm.Session{NewDB: true}).Model(modelValue)

	if err := tx.Clauses(clause.Where{Exprs: queryConds}).First(modelValue).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	return deleteWithNestedSelect(tx, modelValue, nestedPaths)
}
