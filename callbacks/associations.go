package callbacks

import (
	"reflect"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

func SaveBeforeAssociations(create bool) func(db *gorm.DB) {
	return func(db *gorm.DB) {
		if db.Error == nil && db.Statement.Schema != nil {
			selectColumns, restricted := db.Statement.SelectAndOmitColumns(create, !create)

			// Save Belongs To associations
			for _, rel := range db.Statement.Schema.Relationships.BelongsTo {
				if v, ok := selectColumns[rel.Name]; (ok && !v) || (!ok && restricted) {
					continue
				}

				setupReferences := func(obj reflect.Value, elem reflect.Value) {
					for _, ref := range rel.References {
						if !ref.OwnPrimaryKey {
							pv, _ := ref.PrimaryKey.ValueOf(elem)
							db.AddError(ref.ForeignKey.Set(obj, pv))

							if dest, ok := db.Statement.Dest.(map[string]interface{}); ok {
								dest[ref.ForeignKey.DBName] = pv
								if _, ok := dest[rel.Name]; ok {
									dest[rel.Name] = elem.Interface()
								}
							}
						}
					}
				}

				switch db.Statement.ReflectValue.Kind() {
				case reflect.Slice, reflect.Array:
					var (
						objs      = make([]reflect.Value, 0, db.Statement.ReflectValue.Len())
						fieldType = rel.Field.FieldType
						isPtr     = fieldType.Kind() == reflect.Ptr
					)

					if !isPtr {
						fieldType = reflect.PtrTo(fieldType)
					}

					elems := reflect.MakeSlice(reflect.SliceOf(fieldType), 0, 10)
					for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
						obj := db.Statement.ReflectValue.Index(i)

						if reflect.Indirect(obj).Kind() == reflect.Struct {
							if _, zero := rel.Field.ValueOf(obj); !zero { // check belongs to relation value
								rv := rel.Field.ReflectValueOf(obj) // relation reflect value
								objs = append(objs, obj)
								if isPtr {
									elems = reflect.Append(elems, rv)
								} else {
									elems = reflect.Append(elems, rv.Addr())
								}
							}
						} else {
							break
						}
					}

					if elems.Len() > 0 {
						if saveAssociations(db, rel, elems.Interface(), selectColumns, restricted, nil) == nil {
							for i := 0; i < elems.Len(); i++ {
								setupReferences(objs[i], elems.Index(i))
							}
						}
					}
				case reflect.Struct:
					if _, zero := rel.Field.ValueOf(db.Statement.ReflectValue); !zero {
						rv := rel.Field.ReflectValueOf(db.Statement.ReflectValue) // relation reflect value
						if rv.Kind() != reflect.Ptr {
							rv = rv.Addr()
						}

						if saveAssociations(db, rel, rv.Interface(), selectColumns, restricted, nil) == nil {
							setupReferences(db.Statement.ReflectValue, rv)
						}
					}
				}
			}
		}
	}
}

func SaveAfterAssociations(create bool) func(db *gorm.DB) {
	return func(db *gorm.DB) {
		if db.Error == nil && db.Statement.Schema != nil {
			selectColumns, restricted := db.Statement.SelectAndOmitColumns(create, !create)

			// Save Has One associations
			for _, rel := range db.Statement.Schema.Relationships.HasOne {
				if v, ok := selectColumns[rel.Name]; (ok && !v) || (!ok && restricted) {
					continue
				}

				switch db.Statement.ReflectValue.Kind() {
				case reflect.Slice, reflect.Array:
					var (
						fieldType = rel.Field.FieldType
						isPtr     = fieldType.Kind() == reflect.Ptr
					)

					if !isPtr {
						fieldType = reflect.PtrTo(fieldType)
					}

					elems := reflect.MakeSlice(reflect.SliceOf(fieldType), 0, 10)

					for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
						obj := db.Statement.ReflectValue.Index(i)

						if reflect.Indirect(obj).Kind() == reflect.Struct {
							if _, zero := rel.Field.ValueOf(obj); !zero {
								rv := rel.Field.ReflectValueOf(obj)
								if rv.Kind() != reflect.Ptr {
									rv = rv.Addr()
								}

								for _, ref := range rel.References {
									if ref.OwnPrimaryKey {
										fv, _ := ref.PrimaryKey.ValueOf(obj)
										db.AddError(ref.ForeignKey.Set(rv, fv))
									} else if ref.PrimaryValue != "" {
										db.AddError(ref.ForeignKey.Set(rv, ref.PrimaryValue))
									}
								}

								elems = reflect.Append(elems, rv)
							}
						}
					}

					if elems.Len() > 0 {
						assignmentColumns := make([]string, 0, len(rel.References))
						for _, ref := range rel.References {
							assignmentColumns = append(assignmentColumns, ref.ForeignKey.DBName)
						}

						saveAssociations(db, rel, elems.Interface(), selectColumns, restricted, assignmentColumns)
					}
				case reflect.Struct:
					if _, zero := rel.Field.ValueOf(db.Statement.ReflectValue); !zero {
						f := rel.Field.ReflectValueOf(db.Statement.ReflectValue)
						if f.Kind() != reflect.Ptr {
							f = f.Addr()
						}

						assignmentColumns := make([]string, 0, len(rel.References))
						for _, ref := range rel.References {
							if ref.OwnPrimaryKey {
								fv, _ := ref.PrimaryKey.ValueOf(db.Statement.ReflectValue)
								ref.ForeignKey.Set(f, fv)
							} else if ref.PrimaryValue != "" {
								ref.ForeignKey.Set(f, ref.PrimaryValue)
							}
							assignmentColumns = append(assignmentColumns, ref.ForeignKey.DBName)
						}

						saveAssociations(db, rel, f.Interface(), selectColumns, restricted, assignmentColumns)
					}
				}
			}

			// Save Has Many associations
			for _, rel := range db.Statement.Schema.Relationships.HasMany {
				if v, ok := selectColumns[rel.Name]; (ok && !v) || (!ok && restricted) {
					continue
				}

				fieldType := rel.Field.IndirectFieldType.Elem()
				isPtr := fieldType.Kind() == reflect.Ptr
				if !isPtr {
					fieldType = reflect.PtrTo(fieldType)
				}
				elems := reflect.MakeSlice(reflect.SliceOf(fieldType), 0, 10)
				appendToElems := func(v reflect.Value) {
					if _, zero := rel.Field.ValueOf(v); !zero {
						f := reflect.Indirect(rel.Field.ReflectValueOf(v))

						for i := 0; i < f.Len(); i++ {
							elem := f.Index(i)
							for _, ref := range rel.References {
								if ref.OwnPrimaryKey {
									pv, _ := ref.PrimaryKey.ValueOf(v)
									ref.ForeignKey.Set(elem, pv)
								} else if ref.PrimaryValue != "" {
									ref.ForeignKey.Set(elem, ref.PrimaryValue)
								}
							}

							if isPtr {
								elems = reflect.Append(elems, elem)
							} else {
								elems = reflect.Append(elems, elem.Addr())
							}
						}
					}
				}

				switch db.Statement.ReflectValue.Kind() {
				case reflect.Slice, reflect.Array:
					for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
						obj := db.Statement.ReflectValue.Index(i)
						if reflect.Indirect(obj).Kind() == reflect.Struct {
							appendToElems(obj)
						}
					}
				case reflect.Struct:
					appendToElems(db.Statement.ReflectValue)
				}

				if elems.Len() > 0 {
					assignmentColumns := make([]string, 0, len(rel.References))
					for _, ref := range rel.References {
						assignmentColumns = append(assignmentColumns, ref.ForeignKey.DBName)
					}

					saveAssociations(db, rel, elems.Interface(), selectColumns, restricted, assignmentColumns)
				}
			}

			// Save Many2Many associations
			for _, rel := range db.Statement.Schema.Relationships.Many2Many {
				if v, ok := selectColumns[rel.Name]; (ok && !v) || (!ok && restricted) {
					continue
				}

				fieldType := rel.Field.IndirectFieldType.Elem()
				isPtr := fieldType.Kind() == reflect.Ptr
				if !isPtr {
					fieldType = reflect.PtrTo(fieldType)
				}
				elems := reflect.MakeSlice(reflect.SliceOf(fieldType), 0, 10)
				joins := reflect.MakeSlice(reflect.SliceOf(reflect.PtrTo(rel.JoinTable.ModelType)), 0, 10)
				objs := []reflect.Value{}

				appendToJoins := func(obj reflect.Value, elem reflect.Value) {
					joinValue := reflect.New(rel.JoinTable.ModelType)
					for _, ref := range rel.References {
						if ref.OwnPrimaryKey {
							fv, _ := ref.PrimaryKey.ValueOf(obj)
							ref.ForeignKey.Set(joinValue, fv)
						} else if ref.PrimaryValue != "" {
							ref.ForeignKey.Set(joinValue, ref.PrimaryValue)
						} else {
							fv, _ := ref.PrimaryKey.ValueOf(elem)
							ref.ForeignKey.Set(joinValue, fv)
						}
					}
					joins = reflect.Append(joins, joinValue)
				}

				appendToElems := func(v reflect.Value) {
					if _, zero := rel.Field.ValueOf(v); !zero {
						f := reflect.Indirect(rel.Field.ReflectValueOf(v))

						for i := 0; i < f.Len(); i++ {
							elem := f.Index(i)

							objs = append(objs, v)
							if isPtr {
								elems = reflect.Append(elems, elem)
							} else {
								elems = reflect.Append(elems, elem.Addr())
							}
						}
					}
				}

				switch db.Statement.ReflectValue.Kind() {
				case reflect.Slice, reflect.Array:
					for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
						obj := db.Statement.ReflectValue.Index(i)
						if reflect.Indirect(obj).Kind() == reflect.Struct {
							appendToElems(obj)
						}
					}
				case reflect.Struct:
					appendToElems(db.Statement.ReflectValue)
				}

				// optimize elems of reflect value length
				if elemLen := elems.Len(); elemLen > 0 {
					if v, ok := selectColumns[rel.Name+".*"]; !ok || v {
						saveAssociations(db, rel, elems.Interface(), selectColumns, restricted, nil)
					}

					for i := 0; i < elemLen; i++ {
						appendToJoins(objs[i], elems.Index(i))
					}
				}

				if joins.Len() > 0 {
					db.AddError(db.Session(&gorm.Session{NewDB: true}).Clauses(clause.OnConflict{DoNothing: true}).Session(&gorm.Session{
						SkipHooks:                db.Statement.SkipHooks,
						DisableNestedTransaction: true,
					}).Create(joins.Interface()).Error)
				}
			}
		}
	}
}

func onConflictOption(stmt *gorm.Statement, s *schema.Schema, selectColumns map[string]bool, restricted bool, defaultUpdatingColumns []string) (onConflict clause.OnConflict) {
	if len(defaultUpdatingColumns) > 0 || stmt.DB.FullSaveAssociations {
		onConflict.Columns = make([]clause.Column, 0, len(s.PrimaryFieldDBNames))
		for _, dbName := range s.PrimaryFieldDBNames {
			onConflict.Columns = append(onConflict.Columns, clause.Column{Name: dbName})
		}

		onConflict.UpdateAll = stmt.DB.FullSaveAssociations
		if !onConflict.UpdateAll {
			onConflict.DoUpdates = clause.AssignmentColumns(defaultUpdatingColumns)
		}
	} else {
		onConflict.DoNothing = true
	}

	return
}

func saveAssociations(db *gorm.DB, rel *schema.Relationship, values interface{}, selectColumns map[string]bool, restricted bool, defaultUpdatingColumns []string) error {
	var (
		selects, omits []string
		onConflict     = onConflictOption(db.Statement, rel.FieldSchema, selectColumns, restricted, defaultUpdatingColumns)
		refName        = rel.Name + "."
	)

	for name, ok := range selectColumns {
		columnName := ""
		if strings.HasPrefix(name, refName) {
			columnName = strings.TrimPrefix(name, refName)
		}

		if columnName != "" {
			if ok {
				selects = append(selects, columnName)
			} else {
				omits = append(omits, columnName)
			}
		}
	}

	tx := db.Session(&gorm.Session{NewDB: true}).Clauses(onConflict).Session(&gorm.Session{
		FullSaveAssociations:     db.FullSaveAssociations,
		SkipHooks:                db.Statement.SkipHooks,
		DisableNestedTransaction: true,
	})

	db.Statement.Settings.Range(func(k, v interface{}) bool {
		tx.Statement.Settings.Store(k, v)
		return true
	})

	if tx.Statement.FullSaveAssociations {
		tx = tx.Set("gorm:update_track_time", true)
	}

	if len(selects) > 0 {
		tx = tx.Select(selects)
	} else if restricted && len(omits) == 0 {
		tx = tx.Omit(clause.Associations)
	}

	if len(omits) > 0 {
		tx = tx.Omit(omits...)
	}

	return db.AddError(tx.Create(values).Error)
}
