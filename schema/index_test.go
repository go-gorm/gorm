package schema_test

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils/tests"
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
	Name7        string `gorm:"index:type"`
	Name8        string `gorm:"index:,length:10;index:,collate:utf8"`

	CompName1 string `gorm:"index:,unique,composite:idx_compname_1,option:NULLS NOT DISTINCT;not null"`
	CompName2 string `gorm:"index:,composite:idx_compname_1"`

	// Composite Index: Flattened structure.
	Data0A string `gorm:"index:,composite:comp_id0"`
	Data0B string `gorm:"index:,composite:comp_id0"`

	// Composite Index: Nested structure.
	Data1A string `gorm:"index:,composite:comp_id1"`
	CompIdxLevel1C

	// Composite Index: Unique and priority.
	Data2A string `gorm:"index:,unique,composite:comp_id2,priority:2"`
	CompIdxLevel2C
}

type CompIdxLevel1C struct {
	CompIdxLevel1B
	Data1C string `gorm:"index:,composite:comp_id1"`
}

type CompIdxLevel1B struct {
	Data1B string `gorm:"index:,composite:comp_id1"`
}

type CompIdxLevel2C struct {
	CompIdxLevel2B
	Data2C string `gorm:"index:,unique,composite:comp_id2,priority:1"`
}

type CompIdxLevel2B struct {
	Data2B string `gorm:"index:,unique,composite:comp_id2,priority:3"`
}

func TestParseIndex(t *testing.T) {
	user, err := schema.Parse(&UserIndex{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse user index, got error %v", err)
	}

	results := []*schema.Index{
		{
			Name:   "idx_user_indices_name",
			Fields: []schema.IndexOption{{Field: &schema.Field{Name: "Name"}}},
		},
		{
			Name:   "idx_name",
			Class:  "UNIQUE",
			Fields: []schema.IndexOption{{Field: &schema.Field{Name: "Name2", UniqueIndex: "idx_name"}}},
		},
		{
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
		{
			Name:   "idx_user_indices_name4",
			Class:  "UNIQUE",
			Fields: []schema.IndexOption{{Field: &schema.Field{Name: "Name4", UniqueIndex: "idx_user_indices_name4"}}},
		},
		{
			Name:    "idx_user_indices_name5",
			Class:   "FULLTEXT",
			Comment: "hello , world",
			Where:   "age > 10",
			Fields:  []schema.IndexOption{{Field: &schema.Field{Name: "Name5"}}},
		},
		{
			Name:    "profile",
			Comment: "hello , world",
			Where:   "age > 10",
			Option:  "WITH PARSER parser_name",
			Fields: []schema.IndexOption{{Field: &schema.Field{Name: "Name6"}}, {
				Field:      &schema.Field{Name: "Age"},
				Expression: "ABS(age)",
			}},
		},
		{
			Name:   "idx_id",
			Fields: []schema.IndexOption{{Field: &schema.Field{Name: "MemberNumber"}}, {Field: &schema.Field{Name: "OID", UniqueIndex: "idx_oid"}}},
		},
		{
			Name:   "idx_oid",
			Class:  "UNIQUE",
			Fields: []schema.IndexOption{{Field: &schema.Field{Name: "OID", UniqueIndex: "idx_oid"}}},
		},
		{
			Name:   "type",
			Type:   "",
			Fields: []schema.IndexOption{{Field: &schema.Field{Name: "Name7"}}},
		},
		{
			Name: "idx_user_indices_name8",
			Type: "",
			Fields: []schema.IndexOption{
				{Field: &schema.Field{Name: "Name8"}, Length: 10},
				// Note: Duplicate Columns
				{Field: &schema.Field{Name: "Name8"}, Collate: "utf8"},
			},
		},
		{
			Class:  "UNIQUE",
			Name:   "idx_user_indices_idx_compname_1",
			Option: "NULLS NOT DISTINCT",
			Fields: []schema.IndexOption{
				{Field: &schema.Field{Name: "CompName1", NotNull: true}},
				{Field: &schema.Field{Name: "CompName2"}},
			},
		},
		{
			Name: "idx_user_indices_comp_id0",
			Type: "",
			Fields: []schema.IndexOption{{
				Field: &schema.Field{Name: "Data0A"},
			}, {
				Field: &schema.Field{Name: "Data0B"},
			}},
		},
		{
			Name: "idx_user_indices_comp_id1",
			Fields: []schema.IndexOption{{
				Field: &schema.Field{Name: "Data1A"},
			}, {
				Field: &schema.Field{Name: "Data1B"},
			}, {
				Field: &schema.Field{Name: "Data1C"},
			}},
		},
		{
			Name:  "idx_user_indices_comp_id2",
			Class: "UNIQUE",
			Fields: []schema.IndexOption{{
				Field: &schema.Field{Name: "Data2C"},
			}, {
				Field: &schema.Field{Name: "Data2A"},
			}, {
				Field: &schema.Field{Name: "Data2B"},
			}},
		},
	}

	CheckIndices(t, results, user.ParseIndexes())
}

func TestParseIndexWithUniqueIndexAndUnique(t *testing.T) {
	type IndexTest struct {
		FieldA string `gorm:"unique;index"` // unique and index
		FieldB string `gorm:"unique"`       // unique

		FieldC string `gorm:"index:,unique"`     // uniqueIndex
		FieldD string `gorm:"uniqueIndex;index"` // uniqueIndex and index

		FieldE1 string `gorm:"uniqueIndex:uniq_field_e1_e2"` // mul uniqueIndex
		FieldE2 string `gorm:"uniqueIndex:uniq_field_e1_e2"`

		FieldF1 string `gorm:"uniqueIndex:uniq_field_f1_f2;index"` // mul uniqueIndex and index
		FieldF2 string `gorm:"uniqueIndex:uniq_field_f1_f2;"`

		FieldG string `gorm:"unique;uniqueIndex"` // unique and uniqueIndex

		FieldH1 string `gorm:"unique;uniqueIndex:uniq_field_h1_h2"` // unique and mul uniqueIndex
		FieldH2 string `gorm:"uniqueIndex:uniq_field_h1_h2"`        // unique and mul uniqueIndex
	}
	indexSchema, err := schema.Parse(&IndexTest{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse user index, got error %v", err)
	}
	indices := indexSchema.ParseIndexes()
	expectedIndices := []*schema.Index{
		{
			Name:   "idx_index_tests_field_a",
			Fields: []schema.IndexOption{{Field: &schema.Field{Name: "FieldA", Unique: true}}},
		},
		{
			Name:   "idx_index_tests_field_c",
			Class:  "UNIQUE",
			Fields: []schema.IndexOption{{Field: &schema.Field{Name: "FieldC", UniqueIndex: "idx_index_tests_field_c"}}},
		},
		{
			Name:  "idx_index_tests_field_d",
			Class: "UNIQUE",
			Fields: []schema.IndexOption{
				{Field: &schema.Field{Name: "FieldD"}},
				// Note: Duplicate Columns
				{Field: &schema.Field{Name: "FieldD"}},
			},
		},
		{
			Name:  "uniq_field_e1_e2",
			Class: "UNIQUE",
			Fields: []schema.IndexOption{
				{Field: &schema.Field{Name: "FieldE1"}},
				{Field: &schema.Field{Name: "FieldE2"}},
			},
		},
		{
			Name:  "uniq_field_f1_f2",
			Class: "UNIQUE",
			Fields: []schema.IndexOption{
				{Field: &schema.Field{Name: "FieldF1"}},
				{Field: &schema.Field{Name: "FieldF2"}},
			},
		},
		{
			Name:   "idx_index_tests_field_f1",
			Fields: []schema.IndexOption{{Field: &schema.Field{Name: "FieldF1"}}},
		},
		{
			Name:   "idx_index_tests_field_g",
			Class:  "UNIQUE",
			Fields: []schema.IndexOption{{Field: &schema.Field{Name: "FieldG", Unique: true, UniqueIndex: "idx_index_tests_field_g"}}},
		},
		{
			Name:  "uniq_field_h1_h2",
			Class: "UNIQUE",
			Fields: []schema.IndexOption{
				{Field: &schema.Field{Name: "FieldH1", Unique: true}},
				{Field: &schema.Field{Name: "FieldH2"}},
			},
		},
	}
	CheckIndices(t, expectedIndices, indices)
}

func CheckIndices(t *testing.T, expected, actual []*schema.Index) {
	if len(expected) != len(actual) {
		t.Errorf("expected %d indices, but got %d", len(expected), len(actual))
		return
	}

	for i, ei := range expected {
		t.Run(ei.Name, func(t *testing.T) {
			ai := actual[i]
			tests.AssertObjEqual(t, ai, ei, "Name", "Class", "Type", "Where", "Comment", "Option")

			if len(ei.Fields) != len(ai.Fields) {
				t.Errorf("expected index %q field length is %d but actual %d", ei.Name, len(ei.Fields), len(ai.Fields))
				return
			}
			for i, ef := range ei.Fields {
				af := ai.Fields[i]
				tests.AssertObjEqual(t, af, ef, "Name", "Unique", "UniqueIndex", "Expression", "Sort", "Collate", "Length", "NotNull")
			}
		})
	}
}
