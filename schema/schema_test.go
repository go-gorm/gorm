package schema_test

import (
	"reflect"
	"sync"
	"testing"

	"github.com/jinzhu/gorm/schema"
	"github.com/jinzhu/gorm/tests"
)

func TestParseSchema(t *testing.T) {
	cacheMap := sync.Map{}
	user, err := schema.Parse(&tests.User{}, &cacheMap, schema.NamingStrategy{})

	if err != nil {
		t.Fatalf("failed to parse user, got error %v", err)
	}

	checkSchemaFields(t, user)
}

func checkSchemaFields(t *testing.T, s *schema.Schema) {
	fields := []schema.Field{
		schema.Field{
			Name: "ID", DBName: "id", BindNames: []string{"Model", "ID"}, DataType: schema.Uint,
			PrimaryKey: true, Tag: `gorm:"primarykey"`, TagSettings: map[string]string{"PRIMARYKEY": "PRIMARYKEY"},
		},
		schema.Field{Name: "CreatedAt", DBName: "created_at", BindNames: []string{"Model", "CreatedAt"}, DataType: schema.Time},
		schema.Field{Name: "UpdatedAt", DBName: "updated_at", BindNames: []string{"Model", "UpdatedAt"}, DataType: schema.Time},
		schema.Field{Name: "DeletedAt", DBName: "deleted_at", BindNames: []string{"Model", "DeletedAt"}, Tag: `gorm:"index"`, DataType: schema.Time},
		schema.Field{Name: "Name", DBName: "name", BindNames: []string{"Name"}, DataType: schema.String},
		schema.Field{Name: "Age", DBName: "age", BindNames: []string{"Age"}, DataType: schema.Uint},
		schema.Field{Name: "Birthday", DBName: "birthday", BindNames: []string{"Birthday"}, DataType: schema.Time},
		schema.Field{Name: "CompanyID", DBName: "company_id", BindNames: []string{"CompanyID"}, DataType: schema.Int},
		schema.Field{Name: "ManagerID", DBName: "manager_id", BindNames: []string{"ManagerID"}, DataType: schema.Uint},
	}

	for _, f := range fields {
		f.Creatable = true
		f.Updatable = true
		if f.TagSettings == nil {
			if f.Tag != "" {
				f.TagSettings = schema.ParseTagSetting(f.Tag)
			} else {
				f.TagSettings = map[string]string{}
			}
		}

		if foundField, ok := s.FieldsByName[f.Name]; !ok {
			t.Errorf("schema %v failed to look up field with name %v", s, f.Name)
		} else {
			checkSchemaField(t, foundField, f)

			if field, ok := s.FieldsByDBName[f.DBName]; !ok || foundField != field {
				t.Errorf("schema %v failed to look up field with dbname %v", s, f.DBName)
			}

			for _, name := range []string{f.DBName, f.Name} {
				if field := s.LookUpField(name); field == nil || foundField != field {
					t.Errorf("schema %v failed to look up field with dbname %v", s, f.DBName)
				}
			}
		}
	}
}

func checkSchemaField(t *testing.T, parsedField *schema.Field, field schema.Field) {
	equalFieldNames := []string{"Name", "DBName", "BindNames", "DataType", "DBDataType", "PrimaryKey", "AutoIncrement", "Creatable", "Updatable", "HasDefaultValue", "DefaultValue", "NotNull", "Unique", "Comment", "Size", "Precision", "Tag", "TagSettings"}

	for _, name := range equalFieldNames {
		got := reflect.ValueOf(parsedField).Elem().FieldByName(name).Interface()
		expects := reflect.ValueOf(field).FieldByName(name).Interface()
		if !reflect.DeepEqual(got, expects) {
			t.Errorf("%v is not equal, expects: %v, got %v", name, expects, got)
		}
	}
}
