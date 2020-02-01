package schema_test

import (
	"reflect"
	"testing"

	"github.com/jinzhu/gorm/schema"
)

func checkSchema(t *testing.T, s *schema.Schema, v schema.Schema, primaryFields []string) {
	equalFieldNames := []string{"Name", "Table"}

	for _, name := range equalFieldNames {
		got := reflect.ValueOf(s).Elem().FieldByName(name).Interface()
		expects := reflect.ValueOf(v).FieldByName(name).Interface()
		if !reflect.DeepEqual(got, expects) {
			t.Errorf("schema %v %v is not equal, expects: %v, got %v", s, name, expects, got)
		}
	}

	for idx, field := range primaryFields {
		var found bool
		for _, f := range s.PrimaryFields {
			if f.Name == field {
				found = true
			}
		}

		if idx == 0 {
			if field != s.PrioritizedPrimaryField.Name {
				t.Errorf("schema %v prioritized primary field should be %v, but got %v", s, field, s.PrioritizedPrimaryField.Name)
			}
		}

		if !found {
			t.Errorf("schema %v failed to found priamry key: %v", s, field)
		}
	}
}

func checkSchemaField(t *testing.T, s *schema.Schema, f *schema.Field, fc func(*schema.Field)) {
	if fc != nil {
		fc(f)
	}

	if f.TagSettings == nil {
		if f.Tag != "" {
			f.TagSettings = schema.ParseTagSetting(f.Tag)
		} else {
			f.TagSettings = map[string]string{}
		}
	}

	if parsedField, ok := s.FieldsByName[f.Name]; !ok {
		t.Errorf("schema %v failed to look up field with name %v", s, f.Name)
	} else {
		equalFieldNames := []string{"Name", "DBName", "BindNames", "DataType", "DBDataType", "PrimaryKey", "AutoIncrement", "Creatable", "Updatable", "HasDefaultValue", "DefaultValue", "NotNull", "Unique", "Comment", "Size", "Precision", "Tag", "TagSettings"}

		for _, name := range equalFieldNames {
			got := reflect.ValueOf(parsedField).Elem().FieldByName(name).Interface()
			expects := reflect.ValueOf(f).Elem().FieldByName(name).Interface()
			if !reflect.DeepEqual(got, expects) {
				t.Errorf("%v is not equal, expects: %v, got %v", name, expects, got)
			}
		}

		if field, ok := s.FieldsByDBName[f.DBName]; !ok || parsedField != field {
			t.Errorf("schema %v failed to look up field with dbname %v", s, f.DBName)
		}

		for _, name := range []string{f.DBName, f.Name} {
			if field := s.LookUpField(name); field == nil || parsedField != field {
				t.Errorf("schema %v failed to look up field with dbname %v", s, f.DBName)
			}
		}

		if f.PrimaryKey {
			var found bool
			for _, primaryField := range s.PrimaryFields {
				if primaryField == parsedField {
					found = true
				}
			}

			if !found {
				t.Errorf("schema %v doesn't include field %v", s, f.Name)
			}
		}
	}
}

type Relation struct {
	Name            string
	Type            schema.RelationshipType
	Polymorphic     schema.Polymorphic
	Schema          string
	FieldSchema     string
	JoinTable       string
	JoinTableFields []schema.Field
	References      []Reference
}

type Reference struct {
	PrimaryKey    string
	PrimarySchema string
	ForeignKey    string
	ForeignSchema string
	OwnPriamryKey bool
}

func checkSchemaRelation(t *testing.T, s *schema.Schema, relation Relation) {
	if r, ok := s.Relationships.Relations[relation.Name]; ok {
		if r.Name != relation.Name {
			t.Errorf("schema %v relation name expects %v, but got %v", s, relation.Name, r.Name)
		}

		if r.Type != relation.Type {
			t.Errorf("schema %v relation name expects %v, but got %v", s, relation.Type, r.Type)
		}
	} else {
		t.Errorf("schema %v failed to find relations by name %v", s, relation.Name)
	}
}
