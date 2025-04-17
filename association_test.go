package gorm

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type User struct {
	ID   uint
	Name string
}

type TestReplaceCompany struct {
	ID    uint `gorm:"primaryKey"`
	Name  string
	Users []TestReplaceUser `gorm:"foreignKey:CompanyID"`
}

type TestReplaceUser struct {
	ID        uint `gorm:"primaryKey"`
	Name      string
	CompanyID uint
}

type TestM2MUser struct {
	ID        uint `gorm:"primaryKey"`
	Name      string
	Languages []TestM2MLanguage `gorm:"many2many:user_languages;"`
}

type TestM2MLanguage struct {
	ID    uint `gorm:"primaryKey"`
	Name  string
	Users []TestM2MUser `gorm:"many2many:user_languages;"`
}

func TestAssociation_RelationshipExists(t *testing.T) {
	db := &DB{
		Config: &Config{
			cacheStore:     &sync.Map{},
			NamingStrategy: schema.NamingStrategy{},
		},
	}
	rel := &schema.Relationship{}
	stmt := &Statement{
		DB:    db,
		Model: &User{},
		Table: "users",
	}
	db.Statement = stmt

	_ = db.Statement.Parse(db.Statement.Model)
	db.Statement.Schema.Relationships = schema.Relationships{
		Relations: map[string]*schema.Relationship{
			"User": rel,
		},
	}

	assoc := db.Association("User")
	if assoc.Error != nil {
		t.Errorf("expected no error, got %v", assoc.Error)
	}
	if assoc.Relationship != rel {
		t.Errorf("expected relationship to be set")
	}
}

func TestAssociation_RelationshipNotExists(t *testing.T) {
	db := &DB{
		Config: &Config{
			cacheStore:     &sync.Map{},
			NamingStrategy: schema.NamingStrategy{},
		},
	}
	stmt := &Statement{
		DB:    db,
		Model: &User{},
		Table: "users",
	}
	db.Statement = stmt

	_ = db.Statement.Parse(db.Statement.Model)
	db.Statement.Schema.Relationships = schema.Relationships{
		Relations: map[string]*schema.Relationship{},
	}

	assoc := db.Association("NotExist")
	if assoc.Error == nil {
		t.Errorf("expected error for unsupported relation")
	}
	if assoc.Relationship != nil {
		t.Errorf("expected relationship to be nil")
	}
}

func TestAssociation_ParseError(t *testing.T) {
	db := &DB{
		Config: &Config{
			cacheStore:     &sync.Map{},
			NamingStrategy: schema.NamingStrategy{},
		},
	}
	stmt := &Statement{
		DB:    db,
		Model: nil,
		Table: "users",
	}
	db.Statement = stmt

	assoc := db.Association("Any")
	if assoc.Error == nil {
		t.Errorf("expected parse error, got nil")
	}
}

func TestAssociation_Unscoped(t *testing.T) {
	db := &DB{}
	rel := &schema.Relationship{}
	assoc := &Association{
		DB:           db,
		Relationship: rel,
		Error:        nil,
		Unscope:      false,
	}
	unscoped := assoc.Unscoped()
	if !unscoped.Unscope {
		t.Errorf("expected Unscope to be true")
	}
	if unscoped.DB != db {
		t.Errorf("expected DB to be the same")
	}
	if unscoped.Relationship != rel {
		t.Errorf("expected Relationship to be the same")
	}
	if unscoped.Error != nil {
		t.Errorf("expected Error to be nil")
	}
}

func TestAssociation_Find_ErrorPropagation(t *testing.T) {
	assoc := &Association{
		Error: errAssert,
	}
	var out []User
	err := assoc.Find(&out)
	if err != errAssert {
		t.Errorf("expected error to propagate, got %v", err)
	}
}

func TestAssociation_Find_CallsBuildConditionAndFind(t *testing.T) {
	db := &DB{
		Config: &Config{
			cacheStore:     &sync.Map{},
			NamingStrategy: schema.NamingStrategy{},
			callbacks: &callbacks{
				processors: map[string]*processor{
					"query": {}, // Use {} instead of &processor{} for brevity if it works
				},
			},
		},
	}
	stmt := &Statement{
		DB:      db,
		Model:   &User{},
		Table:   "users",
		Clauses: map[string]clause.Clause{},
	}
	db.Statement = stmt
	_ = db.Statement.Parse(db.Statement.Model)

	// Create a fully-populated dummy relationship with FieldSchema set
	fieldSchema := db.Statement.Schema
	rel := &schema.Relationship{
		Schema: fieldSchema,
		Field: &schema.Field{
			Name:   "User",
			Schema: fieldSchema,
		},
		Type: schema.HasMany,
		References: []*schema.Reference{
			{
				PrimaryKey:   &schema.Field{Name: "ID", Schema: db.Statement.Schema},
				ForeignKey:   &schema.Field{Name: "UserID", Schema: db.Statement.Schema},
				PrimaryValue: "1",
			},
		},
		FieldSchema: fieldSchema,
	}
	db.Statement.Schema.Relationships = schema.Relationships{
		Relations: map[string]*schema.Relationship{
			"User": rel,
		},
	}

	assoc := db.Association("User")

	var out []User
	err := assoc.Find(&out)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestAssociation_Append_HasOneOrBelongsTo(t *testing.T) {
	db := &DB{
		Config: &Config{
			cacheStore:     &sync.Map{},
			NamingStrategy: schema.NamingStrategy{},
			callbacks: &callbacks{
				processors: map[string]*processor{
					"query":  {},
					"update": {},
				},
			},
		},
	}
	stmt := &Statement{
		DB:      db,
		Model:   &User{},
		Table:   "users",
		Clauses: map[string]clause.Clause{},
		Context: context.Background(),
	}
	db.Statement = stmt
	_ = db.Statement.Parse(db.Statement.Model)

	fieldSchema := db.Statement.Schema
	field := &schema.Field{
		Name:      "User",
		Schema:    fieldSchema,
		FieldType: reflect.TypeOf(&User{}),
		Set: func(ctx context.Context, value reflect.Value, v interface{}) error {
			return nil
		},
		ValueOf: func(ctx context.Context, value reflect.Value) (interface{}, bool) {
			return value.Interface(), value.IsZero()
		},
	}
	fieldSchema.PrimaryFields = []*schema.Field{
		{
			Name:      "ID",
			Schema:    fieldSchema,
			FieldType: reflect.TypeOf(uint(0)),
			ValueOf: func(ctx context.Context, value reflect.Value) (interface{}, bool) {
				return uint(0), true
			},
		},
	}
	rel := &schema.Relationship{
		Schema: fieldSchema,
		Field:  field,
		Type:   schema.HasOne,
		References: []*schema.Reference{
			{
				PrimaryKey:   fieldSchema.PrimaryFields[0],
				ForeignKey:   &schema.Field{Name: "UserID", Schema: fieldSchema},
				PrimaryValue: "1",
			},
		},
		FieldSchema: fieldSchema,
	}
	db.Statement.Schema.Relationships = schema.Relationships{
		Relations: map[string]*schema.Relationship{
			"User": rel,
		},
	}

	assoc := db.Association("User")
	assoc.Relationship.Type = schema.HasOne
	assoc.Relationship.Field = field
	_ = assoc.Append(&User{})
	if assoc.Error != nil {
		t.Errorf("expected no error, got %v", assoc.Error)
	}

	assoc.Relationship.Type = schema.BelongsTo
	assoc.Relationship.Field = field
	_ = assoc.Append(&User{})
	if assoc.Error != nil {
		t.Errorf("expected no error, got %v", assoc.Error)
	}
}

// Helper function to setup DB and Statement for association tests
func setupAssociationTestDB(model interface{}, config *Config) (*DB, *Statement) {
	db := &DB{Config: config}
	stmt := &Statement{
		DB:      db,
		Model:   model,
		Clauses: map[string]clause.Clause{},
		Context: context.Background(),
	}
	if model != nil {
		stmt.ReflectValue = reflect.ValueOf(model)
		if err := stmt.Parse(model); err != nil {
			panic(fmt.Sprintf("failed to parse model in setup: %v", err))
		}
		stmt.Table = stmt.Schema.Table
	}
	db.Statement = stmt
	return db, stmt
}

// Helper to validate a relationship exists and has the correct properties
func validateRelationship(t *testing.T, s *schema.Schema, relName string, relType schema.RelationshipType, fieldSchemaType reflect.Type, fieldName string) *schema.Relationship {
	t.Helper()
	rel, ok := s.Relationships.Relations[relName]
	if !ok {
		t.Fatalf("Relationship '%s' not found in schema", relName)
	}
	if rel.Type != relType {
		t.Fatalf("Relationship '%s' is not %s type, got %v", relName, relType, rel.Type)
	}
	if rel.FieldSchema == nil || rel.FieldSchema.ModelType != fieldSchemaType {
		t.Fatalf("Relationship '%s' FieldSchema is incorrect, expected %v, got %v", relName, fieldSchemaType, rel.FieldSchema.ModelType)
	}
	if rel.Field == nil || rel.Field.Name != fieldName {
		t.Fatalf("Relationship '%s' Field is incorrect, expected name '%s'", relName, fieldName)
	}
	if rel.Field.Set == nil {
		t.Fatalf("Relationship field '%s' has a nil Set function after parse", fieldName)
	}
	return rel
}

func setPrimaryValueInReferences(t *testing.T, rel *schema.Relationship, stmt *Statement) {
	t.Helper()
	pkValue, isZero := rel.Schema.PrimaryFields[0].ValueOf(stmt.Context, stmt.ReflectValue)
	if isZero {
		t.Fatal("Primary key value is zero for the model instance")
	}
	foundRef := false
	for _, ref := range rel.References {
		if ref.PrimaryKey.Name == rel.Schema.PrimaryFields[0].Name && ref.OwnPrimaryKey {
			ref.PrimaryValue = fmt.Sprintf("%v", pkValue)
			foundRef = true
			break
		}
	}
	if !foundRef {
		t.Fatalf("Could not set primary value for relationship '%s'", rel.Name)
	}
}

func assertReplaceHasManyResult(t *testing.T, company *TestReplaceCompany, expectedLen int, expectedNames ...string) {
	t.Helper()
	if company.Users == nil || len(company.Users) != expectedLen {
		t.Errorf("Expected company Users field to be updated to length %d, got: %v (len %d)", expectedLen, company.Users, len(company.Users))
		return
	}
	if len(expectedNames) != expectedLen {
		t.Errorf("Assertion setup error: expected %d names, got %d", expectedLen, len(expectedNames))
		return
	}
	for i := 0; i < expectedLen; i++ {
		if company.Users[i].Name != expectedNames[i] {
			t.Errorf("Expected company Users field content mismatch at index %d. Expected '%s', got: '%s'. Full slice: %v", i, expectedNames[i], company.Users[i].Name, company.Users)
			return
		}
	}
}

func TestAssociation_Replace_HasMany_Unscoped(t *testing.T) {
	config := &Config{
		cacheStore:     &sync.Map{},
		NamingStrategy: schema.NamingStrategy{},
		callbacks: &callbacks{
			processors: map[string]*processor{"update": {}, "delete": {}},
		},
	}

	companySchema, err := schema.Parse(&TestReplaceCompany{}, config.cacheStore, config.NamingStrategy)
	if err != nil {
		t.Fatalf("Failed to parse TestReplaceCompany schema: %v", err)
	}
	_, err = schema.Parse(&TestReplaceUser{}, config.cacheStore, config.NamingStrategy)
	if err != nil {
		t.Fatalf("Failed to parse TestReplaceUser schema: %v", err)
	}

	company := &TestReplaceCompany{ID: 1, Name: "TestCorp"}
	db, stmt := setupAssociationTestDB(company, config)
	stmt.Schema = companySchema

	rel := validateRelationship(t, companySchema, "Users", schema.HasMany, reflect.TypeOf(TestReplaceUser{}), "Users")

	setPrimaryValueInReferences(t, rel, stmt)

	assoc := db.Association("Users")
	if assoc.Error != nil {
		t.Fatalf("Failed to get association 'Users': %v", assoc.Error)
	}
	if assoc.Relationship != rel {
		t.Fatalf("Association relationship is incorrect or nil")
	}

	assoc.Unscope = true
	newUsers := []*TestReplaceUser{{ID: 10, Name: "Alice"}, {ID: 11, Name: "Bob"}}
	err = assoc.Replace(newUsers)

	if err != nil {
		t.Errorf("Replace failed with validation/setup error: %v", err)
	} else {
		assertReplaceHasManyResult(t, company, 2, "Alice", "Bob")
	}
}

func TestAssociation_Delete_Many2Many(t *testing.T) {
	config := &Config{
		cacheStore:     &sync.Map{},
		NamingStrategy: schema.NamingStrategy{},
		callbacks:      &callbacks{processors: map[string]*processor{"delete": {}}}, // Use {}
	}

	// Parse schemas
	langSchema, err := schema.Parse(&TestM2MLanguage{}, config.cacheStore, config.NamingStrategy)
	if err != nil {
		t.Fatalf("Failed to parse TestM2MLanguage schema: %v", err)
	}
	_, err = schema.Parse(&TestM2MUser{}, config.cacheStore, config.NamingStrategy)
	if err != nil {
		t.Fatalf("Failed to parse TestM2MUser schema: %v", err)
	}

	alice := TestM2MUser{ID: 10, Name: "Alice"}
	bob := TestM2MUser{ID: 11, Name: "Bob"}
	english := &TestM2MLanguage{
		ID:    1,
		Name:  "English",
		Users: []TestM2MUser{alice, bob},
	}

	db, _ := setupAssociationTestDB(english, config)
	db.Statement.Schema = langSchema

	rel := validateRelationship(t, langSchema, "Users", schema.Many2Many, reflect.TypeOf(TestM2MUser{}), "Users")
	if rel.JoinTable == nil {
		t.Fatal("Relationship 'Users' JoinTable is nil")
	}

	assoc := db.Association("Users")
	if assoc.Error != nil {
		t.Fatalf("Failed to get association 'Users': %v", assoc.Error)
	}
	if assoc.Relationship != rel {
		t.Fatalf("Association relationship is incorrect or nil")
	}

	userToDeleteAlice := TestM2MUser{ID: 10}
	err = assoc.Delete(&userToDeleteAlice)

	if err != nil {
		t.Errorf("Delete (Alice) failed with error: %v", err)
	} else {
		if len(english.Users) != 1 {
			t.Errorf("Expected english.Users length to be 1 after deleting Alice, got %d", len(english.Users))
		} else if english.Users[0].ID != bob.ID || english.Users[0].Name != bob.Name {
			t.Errorf("Expected remaining user in english.Users to be Bob (%+v), got: %+v", bob, english.Users[0])
		}
	}
}

var errAssert = errors.New("assert error")
