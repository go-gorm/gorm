package schema_test

import (
	"reflect"
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

type UserIndex struct {
	Name         string `gorm:"index"`
	Name2        string `gorm:"index:idx_name,unique"`
	Name3        string `gorm:"index:,sort:desc,collate:utf8,type:btree,length:10,where:name3 != 'jinzhu'"`
	Name4        string `gorm:"uniqueIndex"`
	Name5        int64  `gorm:"index:,class:FULLTEXT,comment:hello \\, world,where:age > 10"`
	Name6        int64  `gorm:"index:profile,comment:hello \\, world,where:age > 10"`
	Age          int64  `gorm:"index:profile,expression:ABS(age),option:WITH PARSER parser_name"`
	OID          int64  `gorm:"index:idx_id;index:idx_oid,unique"`
	MemberNumber string `gorm:"index:idx_id,priority:1"`
}

func TestParseIndex(t *testing.T) {
	user, err := schema.Parse(&UserIndex{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse user index, got error %v", err)
	}

	results := map[string]schema.Index{
		"idx_user_indices_name": {
			Name:   "idx_user_indices_name",
			Fields: []schema.IndexOption{{Field: &schema.Field{Name: "Name"}}},
		},
		"idx_name": {
			Name:   "idx_name",
			Class:  "UNIQUE",
			Fields: []schema.IndexOption{{Field: &schema.Field{Name: "Name2"}}},
		},
		"idx_user_indices_name3": {
			Name:  "idx_user_indices_name3",
			Type:  "btree",
			Where: "name3 != 'jinzhu'",
			Fields: []schema.IndexOption{{
				Field:   &schema.Field{Name: "Name3"},
				Sort:    "desc",
				Collate: "utf8",
				Length:  10,
			}},
		},
		"idx_user_indices_name4": {
			Name:   "idx_user_indices_name4",
			Class:  "UNIQUE",
			Fields: []schema.IndexOption{{Field: &schema.Field{Name: "Name4"}}},
		},
		"idx_user_indices_name5": {
			Name:    "idx_user_indices_name5",
			Class:   "FULLTEXT",
			Comment: "hello , world",
			Where:   "age > 10",
			Fields:  []schema.IndexOption{{Field: &schema.Field{Name: "Name5"}}},
		},
		"profile": {
			Name:    "profile",
			Comment: "hello , world",
			Where:   "age > 10",
			Option:  "WITH PARSER parser_name",
			Fields: []schema.IndexOption{{Field: &schema.Field{Name: "Name6"}}, {
				Field:      &schema.Field{Name: "Age"},
				Expression: "ABS(age)",
			}},
		},
		"idx_id": {
			Name:   "idx_id",
			Fields: []schema.IndexOption{{Field: &schema.Field{Name: "MemberNumber"}}, {Field: &schema.Field{Name: "OID"}}},
		},
		"idx_oid": {
			Name:   "idx_oid",
			Class:  "UNIQUE",
			Fields: []schema.IndexOption{{Field: &schema.Field{Name: "OID"}}},
		},
	}

	indices := user.ParseIndexes()

	for k, result := range results {
		v, ok := indices[k]
		if !ok {
			t.Fatalf("Failed to found index %v from parsed indices %+v", k, indices)
		}

		for _, name := range []string{"Name", "Class", "Type", "Where", "Comment", "Option"} {
			if reflect.ValueOf(result).FieldByName(name).Interface() != reflect.ValueOf(v).FieldByName(name).Interface() {
				t.Errorf(
					"index %v %v should equal, expects %v, got %v",
					k, name, reflect.ValueOf(result).FieldByName(name).Interface(), reflect.ValueOf(v).FieldByName(name).Interface(),
				)
			}
		}

		for idx, ef := range result.Fields {
			rf := v.Fields[idx]
			if rf.Field.Name != ef.Field.Name {
				t.Fatalf("index field should equal, expects %v, got %v", rf.Field.Name, ef.Field.Name)
			}

			for _, name := range []string{"Expression", "Sort", "Collate", "Length"} {
				if reflect.ValueOf(ef).FieldByName(name).Interface() != reflect.ValueOf(rf).FieldByName(name).Interface() {
					t.Errorf(
						"index %v field #%v's %v should equal, expects %v, got %v", k, idx+1, name,
						reflect.ValueOf(ef).FieldByName(name).Interface(), reflect.ValueOf(rf).FieldByName(name).Interface(),
					)
				}
			}
		}
	}
}
