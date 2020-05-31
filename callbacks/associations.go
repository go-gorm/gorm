package callbacks

import (
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
	"github.com/jinzhu/gorm/schema"
	"github.com/jinzhu/gorm/utils"
)

func SaveBeforeAssociations(db *gorm.DB) {
	if db.Error == nil && db.Statement.Schema != nil {
		selectColumns, restricted := SelectAndOmitColumns(db.Statement, true, false)

		// Save Belongs To associations
		for _, rel := range db.Statement.Schema.Relationships.BelongsTo {
			if !saveAssociationCheck(db, rel, selectColumns, restricted) {
				continue
			}

			setupReferences := func(obj reflect.Value, elem reflect.Value) {
				for _, ref := range rel.References {
					if !ref.OwnPrimaryKey {
						pv, _ := ref.PrimaryKey.ValueOf(elem)
						ref.ForeignKey.Set(obj, pv)
					}
				}
			}

			switch db.Statement.ReflectValue.Kind() {
			case reflect.Slice, reflect.Array:
				var (
					objs      []reflect.Value
					fieldType = rel.Field.FieldType
					isPtr     = fieldType.Kind() == reflect.Ptr
				)

				if !isPtr {
					fieldType = reflect.PtrTo(fieldType)
				}

				elems := reflect.MakeSlice(reflect.SliceOf(fieldType), 0, 0)
				for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
					obj := db.Statement.ReflectValue.Index(i)
					if _, zero := rel.Field.ValueOf(obj); !zero { // check belongs to relation value
						rv := rel.Field.ReflectValueOf(obj) // relation reflect value
						if _, isZero := rel.FieldSchema.PrioritizedPrimaryField.ValueOf(rv); isZero {
							objs = append(objs, obj)
							if isPtr {
								elems = reflect.Append(elems, rv)
							} else {
								elems = reflect.Append(elems, rv.Addr())
							}
						} else {
							setupReferences(obj, rv)
						}
					}
				}

				if elems.Len() > 0 {
					if db.AddError(db.Session(&gorm.Session{}).Create(elems.Interface()).Error) == nil {
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

					if _, isZero := rel.FieldSchema.PrioritizedPrimaryField.ValueOf(rv); isZero {
						db.Session(&gorm.Session{}).Create(rv.Interface())
					}
					setupReferences(db.Statement.ReflectValue, rv)
				}
			}
		}
	}
}

func SaveAfterAssociations(db *gorm.DB) {
	if db.Error == nil && db.Statement.Schema != nil {
		selectColumns, restricted := SelectAndOmitColumns(db.Statement, true, false)

		// Save Has One associations
		for _, rel := range db.Statement.Schema.Relationships.HasOne {
			if !saveAssociationCheck(db, rel, selectColumns, restricted) {
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

				elems := reflect.MakeSlice(reflect.SliceOf(fieldType), 0, 0)

				for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
					obj := db.Statement.ReflectValue.Index(i)

					if _, zero := rel.Field.ValueOf(obj); !zero {
						rv := rel.Field.ReflectValueOf(obj)
						if rv.Kind() != reflect.Ptr {
							rv = rv.Addr()
						}

						for _, ref := range rel.References {
							if ref.OwnPrimaryKey {
								fv, _ := ref.PrimaryKey.ValueOf(obj)
								ref.ForeignKey.Set(rv, fv)
							} else if ref.PrimaryValue != "" {
								ref.ForeignKey.Set(rv, ref.PrimaryValue)
							}
						}

						if _, isZero := rel.FieldSchema.PrioritizedPrimaryField.ValueOf(rv); isZero {
							elems = reflect.Append(elems, rv)
						} else {
							db.Session(&gorm.Session{}).Save(rv.Addr().Interface())
						}
					}
				}

				if elems.Len() > 0 {
					db.Session(&gorm.Session{}).Create(elems.Interface())
				}
			case reflect.Struct:
				if _, zero := rel.Field.ValueOf(db.Statement.ReflectValue); !zero {
					f := rel.Field.ReflectValueOf(db.Statement.ReflectValue)
					if f.Kind() != reflect.Ptr {
						f = f.Addr()
					}

					for _, ref := range rel.References {
						if ref.OwnPrimaryKey {
							fv, _ := ref.PrimaryKey.ValueOf(db.Statement.ReflectValue)
							ref.ForeignKey.Set(f, fv)
						} else if ref.PrimaryValue != "" {
							ref.ForeignKey.Set(f, ref.PrimaryValue)
						}
					}

					if _, isZero := rel.FieldSchema.PrioritizedPrimaryField.ValueOf(f); isZero {
						db.Session(&gorm.Session{}).Create(f.Interface())
					} else {
						db.Session(&gorm.Session{}).Save(f.Interface())
					}
				}
			}
		}

		// Save Has Many associations
		for _, rel := range db.Statement.Schema.Relationships.HasMany {
			if !saveAssociationCheck(db, rel, selectColumns, restricted) {
				continue
			}

			fieldType := rel.Field.IndirectFieldType.Elem()
			isPtr := fieldType.Kind() == reflect.Ptr
			if !isPtr {
				fieldType = reflect.PtrTo(fieldType)
			}
			elems := reflect.MakeSlice(reflect.SliceOf(fieldType), 0, 0)
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

						if _, isZero := rel.FieldSchema.PrioritizedPrimaryField.ValueOf(elem); isZero {
							if isPtr {
								elems = reflect.Append(elems, elem)
							} else {
								elems = reflect.Append(elems, elem.Addr())
							}
						} else {
							db.Session(&gorm.Session{}).Save(elem.Addr().Interface())
						}
					}
				}
			}

			switch db.Statement.ReflectValue.Kind() {
			case reflect.Slice, reflect.Array:
				for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
					appendToElems(db.Statement.ReflectValue.Index(i))
				}
			case reflect.Struct:
				appendToElems(db.Statement.ReflectValue)
			}

			if elems.Len() > 0 {
				db.Session(&gorm.Session{}).Create(elems.Interface())
			}
		}

		// Save Many2Many associations
		for _, rel := range db.Statement.Schema.Relationships.Many2Many {
			if !saveAssociationCheck(db, rel, selectColumns, restricted) {
				continue
			}

			fieldType := rel.Field.IndirectFieldType.Elem()
			isPtr := fieldType.Kind() == reflect.Ptr
			if !isPtr {
				fieldType = reflect.PtrTo(fieldType)
			}
			elems := reflect.MakeSlice(reflect.SliceOf(fieldType), 0, 0)
			joins := reflect.MakeSlice(reflect.SliceOf(reflect.PtrTo(rel.JoinTable.ModelType)), 0, 0)
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

						if _, isZero := rel.FieldSchema.PrioritizedPrimaryField.ValueOf(elem); isZero {
							objs = append(objs, v)
							if isPtr {
								elems = reflect.Append(elems, elem)
							} else {
								elems = reflect.Append(elems, elem.Addr())
							}
						} else {
							appendToJoins(v, elem)
						}
					}
				}
			}

			switch db.Statement.ReflectValue.Kind() {
			case reflect.Slice, reflect.Array:
				for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
					appendToElems(db.Statement.ReflectValue.Index(i))
				}
			case reflect.Struct:
				appendToElems(db.Statement.ReflectValue)
			}

			if elems.Len() > 0 {
				db.Session(&gorm.Session{}).Create(elems.Interface())

				for i := 0; i < elems.Len(); i++ {
					appendToJoins(objs[i], elems.Index(i))
				}
			}

			if joins.Len() > 0 {
				db.Session(&gorm.Session{}).Clauses(clause.OnConflict{DoNothing: true}).Create(joins.Interface())
			}
		}
	}
}

func saveAssociationCheck(db *gorm.DB, rel *schema.Relationship, selectColumns map[string]bool, restricted bool) bool {
	savable := true
	if value, ok := db.Get("gorm:save_association"); ok {
		savable = utils.CheckTruth(value)
	}

	if savable {
		if v, ok := selectColumns[rel.Name]; (ok && v) || (!ok && !restricted) {
			return true
		}
	}

	return false
}
