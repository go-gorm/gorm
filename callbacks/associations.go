package callbacks

import (
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/schema"
	"github.com/jinzhu/gorm/utils"
)

func SaveBeforeAssociations(db *gorm.DB) {
	if db.Statement.Schema != nil {
		selectColumns, restricted := SelectAndOmitColumns(db.Statement, true, false)

		// Save Belongs To associations
		for _, rel := range db.Statement.Schema.Relationships.BelongsTo {
			if !saveAssociationCheck(db, rel, selectColumns, restricted) {
				continue
			}

			switch db.Statement.ReflectValue.Kind() {
			case reflect.Slice:
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
							for _, ref := range rel.References {
								if !ref.OwnPrimaryKey {
									pv, _ := ref.PrimaryKey.ValueOf(rv)
									ref.ForeignKey.Set(objs[i], pv)
								}
							}
						}
					}
				}

				if elems.Len() > 0 {
					if db.AddError(db.Session(&gorm.Session{}).Create(elems.Interface()).Error) == nil {
						for i := 0; i < elems.Len(); i++ {
							for _, ref := range rel.References {
								if !ref.OwnPrimaryKey {
									pv, _ := ref.PrimaryKey.ValueOf(elems.Index(i))
									ref.ForeignKey.Set(objs[i], pv)
								}
							}
						}
					}
				}
			case reflect.Struct:
				if _, zero := rel.Field.ValueOf(db.Statement.ReflectValue); !zero {
					rv := rel.Field.ReflectValueOf(db.Statement.ReflectValue) // relation reflect value
					if _, isZero := rel.FieldSchema.PrioritizedPrimaryField.ValueOf(rv); isZero {
						if rv.Kind() == reflect.Ptr {
							db.Session(&gorm.Session{}).Create(rv.Interface())
						} else {
							db.Session(&gorm.Session{}).Create(rv.Addr().Interface())
						}

						for _, ref := range rel.References {
							if !ref.OwnPrimaryKey {
								pv, _ := ref.PrimaryKey.ValueOf(rv)
								ref.ForeignKey.Set(db.Statement.ReflectValue, pv)
							}
						}
					}
				}
			}
		}
	}
}

func SaveAfterAssociations(db *gorm.DB) {
	if db.Statement.Schema != nil {
		selectColumns, restricted := SelectAndOmitColumns(db.Statement, true, false)

		// Save Has One associations
		for _, rel := range db.Statement.Schema.Relationships.HasOne {
			if !saveAssociationCheck(db, rel, selectColumns, restricted) {
				continue
			}

			switch db.Statement.ReflectValue.Kind() {
			case reflect.Slice:
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
					if rv, zero := rel.Field.ValueOf(obj); !zero {
						rv := reflect.ValueOf(rv)
						for _, ref := range rel.References {
							if ref.OwnPrimaryKey {
								fv, _ := ref.PrimaryKey.ValueOf(obj)
								ref.ForeignKey.Set(rv, fv)
							} else if ref.PrimaryValue != "" {
								ref.ForeignKey.Set(rv, ref.PrimaryValue)
							}
						}

						if _, isZero := rel.FieldSchema.PrioritizedPrimaryField.ValueOf(rv); isZero {
							if isPtr {
								elems = reflect.Append(elems, rv)
							} else {
								elems = reflect.Append(elems, rv.Addr())
							}
						}
					}
				}

				if elems.Len() > 0 {
					db.Session(&gorm.Session{}).Create(elems.Interface())
				}
			case reflect.Struct:
				if _, zero := rel.Field.ValueOf(db.Statement.ReflectValue); !zero {
					f := rel.Field.ReflectValueOf(db.Statement.ReflectValue)

					for _, ref := range rel.References {
						if ref.OwnPrimaryKey {
							fv, _ := ref.PrimaryKey.ValueOf(db.Statement.ReflectValue)
							ref.ForeignKey.Set(f, fv)
						} else if ref.PrimaryValue != "" {
							ref.ForeignKey.Set(f, ref.PrimaryValue)
						}
					}

					if _, isZero := rel.FieldSchema.PrioritizedPrimaryField.ValueOf(f); isZero {
						if f.Kind() == reflect.Ptr {
							db.Session(&gorm.Session{}).Create(f.Interface())
						} else {
							db.Session(&gorm.Session{}).Create(f.Addr().Interface())
						}
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
			isPtr := true
			if fieldType.Kind() != reflect.Ptr {
				isPtr = false
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
						}
					}
				}
			}

			switch db.Statement.ReflectValue.Kind() {
			case reflect.Slice:
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
			isPtr := true
			if fieldType.Kind() != reflect.Ptr {
				isPtr = false
				fieldType = reflect.PtrTo(fieldType)
			}
			elems := reflect.MakeSlice(reflect.SliceOf(fieldType), 0, 0)

			switch db.Statement.ReflectValue.Kind() {
			case reflect.Slice:
				for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
					db.Statement.ReflectValue.Index(i)
				}
			case reflect.Struct:
				if _, zero := rel.Field.ValueOf(db.Statement.ReflectValue); !zero {
					f := reflect.Indirect(rel.Field.ReflectValueOf(db.Statement.ReflectValue))

					for i := 0; i < f.Len(); i++ {
						elem := f.Index(i)
						for _, ref := range rel.References {
							if ref.OwnPrimaryKey {
								fv, _ := ref.PrimaryKey.ValueOf(db.Statement.ReflectValue)
								ref.ForeignKey.Set(elem, fv)
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
						}
					}
				}
			}

			if elems.Len() > 0 {
				db.Session(&gorm.Session{}).Create(elems.Interface())
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
