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
		{Name: "ManagerID", DBName: "manager_id", BindNames: []string{"ManagerID"}, DataType: schema.Int},
		{Name: "Active", DBName: "active", BindNames: []string{"Active"}, DataType: schema.Bool},
	}

	for _, f := range fields {
		checkSchemaField(t, user, &f, func(f *schema.Field) {
			f.Creatable = true
			f.Updatable = true
		})
	}

	// check relations
	relations := []Relation{
		{
			Name: "Account", Type: schema.HasOne, Schema: "User", FieldSchema: "Account",
			References: []Reference{{"ID", "User", "UserID", "Account", "", true}},
		},
		{
			Name: "Pets", Type: schema.HasMany, Schema: "User", FieldSchema: "Pet",
			References: []Reference{{"ID", "User", "UserID", "Pet", "", true}},
		},
		{
			Name: "Toys", Type: schema.HasMany, Schema: "User", FieldSchema: "Toy",
			Polymorphic: Polymorphic{ID: "OwnerID", Type: "OwnerType", Value: "users"},
			References:  []Reference{{"ID", "User", "OwnerID", "Toy", "", true}, {"", "", "OwnerType", "Toy", "users", false}},
		},
		{
			Name: "Company", Type: schema.BelongsTo, Schema: "User", FieldSchema: "Company",
			References: []Reference{{"ID", "Company", "CompanyID", "User", "", false}},
		},
		{
			Name: "Manager", Type: schema.BelongsTo, Schema: "User", FieldSchema: "User",
			References: []Reference{{"ID", "User", "ManagerID", "User", "", false}},
		},
		{
			Name: "Team", Type: schema.HasMany, Schema: "User", FieldSchema: "User",
			References: []Reference{{"ID", "User", "ManagerID", "User", "", true}},
		},
		{
			Name: "Languages", Type: schema.Many2Many, Schema: "User", FieldSchema: "Language",
			JoinTable: JoinTable{Name: "UserSpeak", Table: "user_speaks", Fields: []schema.Field{
				{
					Name: "UserID", DBName: "user_id", BindNames: []string{"UserID"}, DataType: schema.Uint,
					Tag: `gorm:"primarykey"`, Creatable: true, Updatable: true, PrimaryKey: true,
				},
				{
					Name: "LanguageCode", DBName: "language_code", BindNames: []string{"LanguageCode"}, DataType: schema.String,
					Tag: `gorm:"primarykey"`, Creatable: true, Updatable: true, PrimaryKey: true,
				},
			}},
			References: []Reference{{"ID", "User", "UserID", "UserSpeak", "", true}, {"Code", "Language", "LanguageCode", "UserSpeak", "", false}},
		},
		{
			Name: "Friends", Type: schema.Many2Many, Schema: "User", FieldSchema: "User",
			JoinTable: JoinTable{Name: "user_friends", Table: "user_friends", Fields: []schema.Field{
				{
					Name: "UserID", DBName: "user_id", BindNames: []string{"UserID"}, DataType: schema.Uint,
					Tag: `gorm:"primarykey"`, Creatable: true, Updatable: true, PrimaryKey: true,
				},
				{
					Name: "FriendID", DBName: "friend_id", BindNames: []string{"FriendID"}, DataType: schema.Uint,
					Tag: `gorm:"primarykey"`, Creatable: true, Updatable: true, PrimaryKey: true,
				},
			}},
			References: []Reference{{"ID", "User", "UserID", "user_friends", "", true}, {"ID", "User", "FriendID", "user_friends", "", false}},
		},
	}

	for _, relation := range relations {
		checkSchemaRelation(t, user, relation)
	}
}
