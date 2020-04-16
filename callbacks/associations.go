package callbacks

import (
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/schema"
	"github.com/jinzhu/gorm/utils"
)

func SaveBeforeAssociations(db *gorm.DB) {
	if db.Statement.Schema != nil {
		for _, rel := range db.Statement.Schema.Relationships.BelongsTo {
			creatable, updatable, saveRef := saveAssociationCheck(db, rel.Field)

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
