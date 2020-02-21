package schema_test

import (
	"reflect"
	"sync"
	"testing"

	"github.com/jinzhu/gorm/schema"
)

type UserIndex struct {
	Name  string `gorm:"index"`
	Name2 string `gorm:"index:idx_name,unique"`
	Name3 string `gorm:"index:,sort:desc,collate:utf8,type:btree,length:10,where:name3 != 'jinzhu'"`
	Name4 string `gorm:"unique_index"`
	Name5 int64  `gorm:"index:,class:FULLTEXT,comment:hello \\, world,where:age > 10"`
	Name6 int64  `gorm:"index:profile,comment:hello \\, world,where:age > 10"`
	Age   int64  `gorm:"index:profile,expression:(age+10)"`
}

func TestParseIndex(t *testing.T) {
	user, err := schema.Parse(&UserIndex{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse user index index, got error %v", err)
	}

	results := map[string]schema.Index{
		"idx_user_indices_name": {
			Name:   "idx_user_indices_name",
			Fields: []schema.IndexOption{{}},
		},
		"idx_name": {
			Name:   "idx_name",
			Class:  "UNIQUE",
			Fields: []schema.IndexOption{{}},
		},
		"idx_user_indices_name3": {
			Name: "idx_user_indices_name3",
			Fields: []schema.IndexOption{{
				Sort:    "desc",
				Collate: "utf8",
				Length:  10,
				Type:    "btree",
				Where:   "name3 != 'jinzhu'",
			}},
		},
		"idx_user_indices_name4": {
			Name:   "idx_user_indices_name4",
			Class:  "UNIQUE",
			Fields: []schema.IndexOption{{}},
		},
		"idx_user_indices_name5": {
			Name:  "idx_user_indices_name5",
			Class: "FULLTEXT",
			Fields: []schema.IndexOption{{
				Comment: "hello , world",
				Where:   "age > 10",
			}},
		},
		"profile": {
			Name: "profile",
			Fields: []schema.IndexOption{{
				Comment: "hello , world",
				Where:   "age > 10",
			}, {
				Expression: "(age+10)",
			}},
		},
	}

	indices := user.ParseIndexes()

	for k, result := range results {
		v, ok := indices[k]
		if !ok {
			t.Errorf("Failed to found index %v from parsed indices %+v", k, indices)
		}

		if result.Name != v.Name {
			t.Errorf("index %v name should equal, expects %v, got %v", k, result.Name, v.Name)
		}

		if result.Class != v.Class {
			t.Errorf("index %v Class should equal, expects %v, got %v", k, result.Class, v.Class)
		}

		for idx, ef := range result.Fields {
			rf := v.Fields[idx]
			for _, name := range []string{"Expression", "Sort", "Collate", "Length", "Type", "Where"} {
				if reflect.ValueOf(ef).FieldByName(name).Interface() != reflect.ValueOf(rf).FieldByName(name).Interface() {
					t.Errorf("index %v field #%v's %v should equal, expects %v, got %v", k, idx+1, name, reflect.ValueOf(ef).FieldByName(name).Interface(), reflect.ValueOf(rf).FieldByName(name).Interface())
				}
			}
		}
	}
}
