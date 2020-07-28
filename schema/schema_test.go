package schema_test

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils/tests"
)

func TestParseSchema(t *testing.T) {
	user, err := schema.Parse(&tests.User{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse user, got error %v", err)
	}

	checkUserSchema(t, user)
}

func TestParseSchemaWithPointerFields(t *testing.T) {
	user, err := schema.Parse(&User{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse pointer user, got error %v", err)
	}

	checkUserSchema(t, user)
}

func checkUserSchema(t *testing.T, user *schema.Schema) {
	// check schema
	checkSchema(t, user, schema.Schema{Name: "User", Table: "users"}, []string{"ID"})

	// check fields
	fields := []schema.Field{
		{Name: "ID", DBName: "id", BindNames: []string{"Model", "ID"}, DataType: schema.Uint, PrimaryKey: true, Tag: `gorm:"primarykey" json:"id"`, TagSettings: map[string]string{"PRIMARYKEY": "PRIMARYKEY"}, Size: 64, HasDefaultValue: true, AutoIncrement: true},
		{Name: "CreatedAt", DBName: "created_at", BindNames: []string{"Model", "CreatedAt"}, DataType: schema.Time, Tag: `json:"created_at"`},
		{Name: "UpdatedAt", DBName: "updated_at", BindNames: []string{"Model", "UpdatedAt"}, DataType: schema.Time, Tag: `json:"updated_at"`},
		{Name: "DeletedAt", DBName: "deleted_at", BindNames: []string{"Model", "DeletedAt"}, Tag: `gorm:"index" json:"deleted_at"`, DataType: schema.Time},
		{Name: "Name", DBName: "name", BindNames: []string{"Name"}, DataType: schema.String},
		{Name: "Age", DBName: "age", BindNames: []string{"Age"}, DataType: schema.Uint, Size: 64},
		{Name: "Birthday", DBName: "birthday", BindNames: []string{"Birthday"}, DataType: schema.Time},
		{Name: "CompanyID", DBName: "company_id", BindNames: []string{"CompanyID"}, DataType: schema.Int, Size: 64},
		{Name: "ManagerID", DBName: "manager_id", BindNames: []string{"ManagerID"}, DataType: schema.Uint, Size: 64},
		{Name: "Active", DBName: "active", BindNames: []string{"Active"}, DataType: schema.Bool},
	}

	for _, f := range fields {
		checkSchemaField(t, user, &f, func(f *schema.Field) {
			f.Creatable = true
			f.Updatable = true
			f.Readable = true
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
					Tag: `gorm:"primarykey" json:"id"`, Creatable: true, Updatable: true, Readable: true, PrimaryKey: true, Size: 64,
				},
				{
					Name: "LanguageCode", DBName: "language_code", BindNames: []string{"LanguageCode"}, DataType: schema.String,
					Tag: `gorm:"primarykey"`, Creatable: true, Updatable: true, Readable: true, PrimaryKey: true,
				},
			}},
			References: []Reference{{"ID", "User", "UserID", "UserSpeak", "", true}, {"Code", "Language", "LanguageCode", "UserSpeak", "", false}},
		},
		{
			Name: "Friends", Type: schema.Many2Many, Schema: "User", FieldSchema: "User",
			JoinTable: JoinTable{Name: "user_friends", Table: "user_friends", Fields: []schema.Field{
				{
					Name: "UserID", DBName: "user_id", BindNames: []string{"UserID"}, DataType: schema.Uint,
					Tag: `gorm:"primarykey" json:"id"`, Creatable: true, Updatable: true, Readable: true, PrimaryKey: true, Size: 64,
				},
				{
					Name: "FriendID", DBName: "friend_id", BindNames: []string{"FriendID"}, DataType: schema.Uint,
					Tag: `gorm:"primarykey" json:"id"`, Creatable: true, Updatable: true, Readable: true, PrimaryKey: true, Size: 64,
				},
			}},
			References: []Reference{{"ID", "User", "UserID", "user_friends", "", true}, {"ID", "User", "FriendID", "user_friends", "", false}},
		},
	}

	for _, relation := range relations {
		checkSchemaRelation(t, user, relation)
	}
}

func TestParseSchemaWithAdvancedDataType(t *testing.T) {
	user, err := schema.Parse(&AdvancedDataTypeUser{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse pointer user, got error %v", err)
	}

	// check schema
	checkSchema(t, user, schema.Schema{Name: "AdvancedDataTypeUser", Table: "advanced_data_type_users"}, []string{"ID"})

	// check fields
	fields := []schema.Field{
		{Name: "ID", DBName: "id", BindNames: []string{"ID"}, DataType: schema.Int, PrimaryKey: true, Size: 64, HasDefaultValue: true, AutoIncrement: true},
		{Name: "Name", DBName: "name", BindNames: []string{"Name"}, DataType: schema.String},
		{Name: "Birthday", DBName: "birthday", BindNames: []string{"Birthday"}, DataType: schema.Time},
		{Name: "RegisteredAt", DBName: "registered_at", BindNames: []string{"RegisteredAt"}, DataType: schema.Time},
		{Name: "DeletedAt", DBName: "deleted_at", BindNames: []string{"DeletedAt"}, DataType: schema.Time},
		{Name: "Active", DBName: "active", BindNames: []string{"Active"}, DataType: schema.Bool},
		{Name: "Admin", DBName: "admin", BindNames: []string{"Admin"}, DataType: schema.Bool},
	}

	for _, f := range fields {
		checkSchemaField(t, user, &f, func(f *schema.Field) {
			f.Creatable = true
			f.Updatable = true
			f.Readable = true
		})
	}
}

type CustomizeTable struct {
}

func (CustomizeTable) TableName() string {
	return "customize"
}

func TestCustomizeTableName(t *testing.T) {
	customize, err := schema.Parse(&CustomizeTable{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse pointer user, got error %v", err)
	}

	if customize.Table != "customize" {
		t.Errorf("Failed to customize table with TableName method")
	}
}
