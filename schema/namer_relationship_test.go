package schema_test

import (
	"strings"
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

type upperColumnNamer struct{ schema.NamingStrategy }

func (n upperColumnNamer) ColumnName(table, column string) string {
	return strings.ToUpper(n.NamingStrategy.ColumnName(table, column))
}

func TestNamerRelationshipInference(t *testing.T) {
	type Company struct {
		ID int `gorm:"primaryKey;column:company_id"`
	}
	type User struct {
		ID        int
		CompanyID int
		Company   Company `gorm:"foreignKey:CompanyID;references:ID"`
	}
	type ForeignKeyOnly struct {
		ID        int
		CompanyID int
		Company   Company `gorm:"foreignKey:CompanyID"`
	}
	type DBNameTag struct {
		ID        int
		CompanyID int
		Company   Company `gorm:"foreignKey:company_id;references:id"`
	}
	type OrdinaryCompany struct{ ID int }
	type OrdinaryUser struct {
		ID        int
		CompanyID int
		Company   OrdinaryCompany
	}
	type ExactCollision struct {
		ID        int
		CompanyID int
		Company   OrdinaryCompany `gorm:"foreignKey:ID;references:ID"`
	}
	type DisambiguatedCompany struct {
		ID   int `gorm:"primaryKey;column:company_id"`
		Code string
	}
	type Disambiguated struct {
		ID        int
		CompanyID string
		Company   DisambiguatedCompany `gorm:"foreignKey:CompanyID;references:Code"`
	}
	type ForcedBelongs struct {
		ID        int
		CompanyID int
		Company   Company `gorm:"foreignKey:CompanyID;references:ID;belongsTo:Company"`
	}
	type ConventionalCompany struct {
		ID int `gorm:"primaryKey;column:user_id"`
	}
	// The local type name User is important to conventional has-one inference.
	conventional := func() interface{} {
		type User struct {
			ID        int
			CompanyID int
			Company   ConventionalCompany
		}
		return &User{}
	}
	conventionalNonID := func() interface{} {
		type Company struct {
			Code int `gorm:"primaryKey;column:user_code"`
		}
		type User struct {
			Code        int `gorm:"primaryKey"`
			CompanyCode int
			Company     Company
		}
		return &User{}
	}
	type NamerOnlyUser struct {
		ID        int
		CompanyID int
		Company   OrdinaryCompany `gorm:"foreignKey:companyId;references:ID"`
	}
	cases := []struct {
		name    string
		model   interface{}
		want    schema.RelationshipType
		foreign string
	}{
		{"foreign-key-and-references", &User{}, schema.BelongsTo, "CompanyID"},
		{"foreign-key-only", &ForeignKeyOnly{}, schema.BelongsTo, "CompanyID"},
		{"database-name-tags", &DBNameTag{}, schema.HasOne, "ID"},
		{"ordinary-conventional", &OrdinaryUser{}, schema.BelongsTo, "CompanyID"},
		{"exact-name-collision-predates-pr", &ExactCollision{}, schema.HasOne, "ID"},
		{"references-disambiguates", &Disambiguated{}, schema.BelongsTo, "CompanyID"},
		{"forced-belongs-to", &ForcedBelongs{}, schema.BelongsTo, "CompanyID"},
		{"conventional-column-collision", conventional(), schema.BelongsTo, "CompanyID"},
		{"conventional-non-ID-column-collision", conventionalNonID(), schema.BelongsTo, "CompanyCode"},
		{"namer-only-tags", &NamerOnlyUser{}, schema.BelongsTo, "CompanyID"},
	}
	namers := []struct {
		name  string
		namer schema.Namer
	}{
		{"default", schema.NamingStrategy{}},
		{"table-prefix", schema.NamingStrategy{TablePrefix: "tenant_", SingularTable: true}},
		{"no-lower-case", schema.NamingStrategy{NoLowerCase: true}},
		{"replacer", schema.NamingStrategy{NameReplacer: strings.NewReplacer("Company", "Employer")}},
		{"uppercase", upperColumnNamer{}},
	}
	for _, n := range namers {
		for _, c := range cases {
			t.Run(n.name+"/"+c.name, func(t *testing.T) {
				checkNamerRelationship(t, c.name, n.name, c.model, n.namer, c.want, c.foreign)
			})
		}
	}
}

func TestNamerLookupOracleCompatibility(t *testing.T) {
	type Record struct {
		ID    int
		Value string `gorm:"column:VALUE"`
	}
	s, err := schema.Parse(&Record{}, &sync.Map{}, upperColumnNamer{})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Value", "VALUE", "value"} {
		if s.LookUpField(name) != s.FieldsByName["Value"] {
			t.Fatalf("failed to resolve column %q", name)
		}
	}
	if s.LookUpField("missing") != nil {
		t.Fatal("unexpected missing field")
	}
}

func checkNamerRelationship(t *testing.T, caseName, namerName string, model interface{}, namer schema.Namer, want schema.RelationshipType, foreign string) {
	t.Helper()
	wantErr := namerRelationshipWantError(caseName, namerName)
	s, err := schema.Parse(model, &sync.Map{}, namer)
	if wantErr {
		if err == nil {
			t.Fatal("expected invalid relationship to remain invalid")
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	rel := s.Relationships.Relations["Company"]
	if rel == nil || len(rel.References) != 1 {
		t.Fatalf("unexpected relation: %#v", rel)
	}
	r := rel.References[0]
	t.Logf("type=%s primary=%s.%s foreign=%s.%s own=%t", rel.Type, r.PrimaryKey.Schema.Name, r.PrimaryKey.Name, r.ForeignKey.Schema.Name, r.ForeignKey.Name, r.OwnPrimaryKey)
	want, foreign = namerRelationshipExpectation(caseName, namerName, want, foreign)
	if rel.Type != want || r.ForeignKey.Name != foreign {
		t.Fatalf("got %s foreign %s; want %s foreign %s", rel.Type, r.ForeignKey.Name, want, foreign)
	}
	checkNamerReferenceOwnership(t, caseName, s, rel, want)
}

func checkNamerReferenceOwnership(t *testing.T, caseName string, s *schema.Schema, rel *schema.Relationship, want schema.RelationshipType) {
	t.Helper()
	r := rel.References[0]
	primary := "ID"
	if caseName == "references-disambiguates" || caseName == "conventional-non-ID-column-collision" {
		primary = "Code"
	}
	primarySchema, foreignSchema := rel.FieldSchema, s
	own := want == schema.HasOne
	if own {
		primarySchema, foreignSchema = s, rel.FieldSchema
	}
	if r.PrimaryKey.Name != primary || r.PrimaryKey.Schema != primarySchema || r.ForeignKey.Schema != foreignSchema || r.OwnPrimaryKey != own {
		t.Fatalf("incorrect reference ownership: %#v", r)
	}
}

func namerRelationshipWantError(caseName, namerName string) bool {
	return (caseName == "database-name-tags" && namerName == "no-lower-case") ||
		(caseName == "namer-only-tags" && (namerName == "no-lower-case" || namerName == "replacer"))
}

func namerRelationshipExpectation(caseName, namerName string, want schema.RelationshipType, foreign string) (schema.RelationshipType, string) {
	if caseName == "conventional-column-collision" && (namerName == "default" || namerName == "table-prefix" || namerName == "replacer") {
		return schema.HasOne, "ID"
	}
	return want, foreign
}
