package callbacks

import (
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/schema"
	"github.com/jinzhu/gorm/utils"
)

func SaveBeforeAssociations(db *gorm.DB) {
	if db.Statement.Schema != nil {
		// Save Belongs To associations
		for _, rel := range db.Statement.Schema.Relationships.BelongsTo {
			creatable, updatable, saveRef := saveAssociationCheck(db, rel.Field)
			if !(creatable || updatable) {
				continue
			}

			switch db.Statement.ReflectValue.Kind() {
			case reflect.Slice:
			case reflect.Struct:
				if _, zero := rel.Field.ValueOf(db.Statement.ReflectValue); !zero {
					f := rel.Field.ReflectValueOf(db.Statement.ReflectValue)
					_, isZero := rel.FieldSchema.PrioritizedPrimaryField.ValueOf(f)

					if isZero && creatable {
						if f.Kind() == reflect.Ptr {
							db.Session(&gorm.Session{}).Create(f.Interface())
						} else {
							db.Session(&gorm.Session{}).Create(f.Addr().Interface())
						}
					} else if !isZero && updatable {
						if f.Kind() == reflect.Ptr {
							db.Session(&gorm.Session{}).Save(f.Interface())
						} else {
							db.Session(&gorm.Session{}).Save(f.Addr().Interface())
						}
					} else {
						continue
					}

					if saveRef {
						for _, ref := range rel.References {
							if !ref.OwnPrimaryKey {
								fv, _ := ref.PrimaryKey.ValueOf(f)
								ref.ForeignKey.Set(db.Statement.ReflectValue, fv)
							}
						}
					}
				}
			}
		}
	}
}

func SaveAfterAssociations(db *gorm.DB) {
	// Save Has One associations
	for _, rel := range db.Statement.Schema.Relationships.HasOne {
		creatable, updatable, saveRef := saveAssociationCheck(db, rel.Field)
		if !(creatable || updatable) {
			continue
		}

		switch db.Statement.ReflectValue.Kind() {
		case reflect.Slice:
		case reflect.Struct:
			if _, zero := rel.Field.ValueOf(db.Statement.ReflectValue); !zero {
				f := rel.Field.ReflectValueOf(db.Statement.ReflectValue)

				if saveRef {
					for _, ref := range rel.References {
						if ref.OwnPrimaryKey {
							fv, _ := ref.PrimaryKey.ValueOf(db.Statement.ReflectValue)
							ref.ForeignKey.Set(f, fv)
						} else if ref.PrimaryValue != "" {
							ref.ForeignKey.Set(f, ref.PrimaryValue)
						}
					}
				}

				_, isZero := rel.FieldSchema.PrioritizedPrimaryField.ValueOf(f)

				if isZero && creatable {
					if f.Kind() == reflect.Ptr {
						db.Session(&gorm.Session{}).Create(f.Interface())
					} else {
						db.Session(&gorm.Session{}).Create(f.Addr().Interface())
					}
				} else if !isZero && updatable {
					if f.Kind() == reflect.Ptr {
						db.Session(&gorm.Session{}).Save(f.Interface())
					} else {
						db.Session(&gorm.Session{}).Save(f.Addr().Interface())
					}
				} else {
					continue
				}
			}
		}
	}

	// Save Has Many associations
	for _, rel := range db.Statement.Schema.Relationships.HasMany {
		creatable, updatable, _ := saveAssociationCheck(db, rel.Field)
		if !(creatable || updatable) {
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
					_, isZero := rel.FieldSchema.PrioritizedPrimaryField.ValueOf(elem)
					for _, ref := range rel.References {
						if ref.OwnPrimaryKey {
							fv, _ := ref.PrimaryKey.ValueOf(db.Statement.ReflectValue)
							ref.ForeignKey.Set(elem, fv)
						} else if ref.PrimaryValue != "" {
							ref.ForeignKey.Set(elem, ref.PrimaryValue)
						}
					}

					if isZero && creatable {
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

func saveAssociationCheck(db *gorm.DB, field *schema.Field) (bool, bool, bool) {
	creatable := field.Creatable
	updatable := field.Updatable
	saveRef := true

	if value, ok := db.Get("gorm:association_autocreate"); creatable && ok {
		creatable = utils.CheckTruth(value)
	}

	if value, ok := db.Get("gorm:association_autoupdate"); updatable && ok {
		updatable = utils.CheckTruth(value)
	}

	if value, ok := db.Get("gorm:association_save_reference"); ok {
		saveRef = utils.CheckTruth(value)
	}

	return creatable, updatable, saveRef
}
