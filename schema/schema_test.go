package schema_test

import (
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

	// check schema
	checkSchema(t, user, schema.Schema{Name: "User", Table: "users"}, []string{"ID"})

	// check fields
	fields := []schema.Field{
		{Name: "ID", DBName: "id", BindNames: []string{"Model", "ID"}, DataType: schema.Uint, PrimaryKey: true, Tag: `gorm:"primarykey"`, TagSettings: map[string]string{"PRIMARYKEY": "PRIMARYKEY"}},
		{Name: "CreatedAt", DBName: "created_at", BindNames: []string{"Model", "CreatedAt"}, DataType: schema.Time},
		{Name: "UpdatedAt", DBName: "updated_at", BindNames: []string{"Model", "UpdatedAt"}, DataType: schema.Time},
		{Name: "DeletedAt", DBName: "deleted_at", BindNames: []string{"Model", "DeletedAt"}, Tag: `gorm:"index"`, DataType: schema.Time},
		{Name: "Name", DBName: "name", BindNames: []string{"Name"}, DataType: schema.String},
		{Name: "Age", DBName: "age", BindNames: []string{"Age"}, DataType: schema.Uint},
		{Name: "Birthday", DBName: "birthday", BindNames: []string{"Birthday"}, DataType: schema.Time},
		{Name: "CompanyID", DBName: "company_id", BindNames: []string{"CompanyID"}, DataType: schema.Int},
		{Name: "ManagerID", DBName: "manager_id", BindNames: []string{"ManagerID"}, DataType: schema.Uint},
	}

	for _, f := range fields {
		checkSchemaField(t, user, &f, func(f *schema.Field) {
			f.Creatable = true
			f.Updatable = true
		})
	}

	// check relations
	relations := []Relation{
		{Name: "Pets", Type: schema.HasMany, Schema: "User", FieldSchema: "Pet", References: []Reference{{"ID", "User", "UserID", "Pet", true}}},
	}
	for _, relation := range relations {
		checkSchemaRelation(t, user, relation)
	}
}
