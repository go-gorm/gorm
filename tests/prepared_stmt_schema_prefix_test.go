package tests_test

import (
	"fmt"
	"testing"

	"gorm.io/gorm"
)

// prefixUser is a minimal model with auto-timestamp fields (CreatedAt,
// UpdatedAt) and no relationships, used to exercise the Create prepared
// statement path described in https://github.com/go-gorm/gorm/issues/7826.
type prefixUser struct {
	gorm.Model
	Name string
}

func (prefixUser) TableName() string { return "prefix_users" }

// schemaTablePrefixPlugin simulates the schema-per-tenant plugin described in
// issue #7826: a Before("*") callback mutates Statement.Table based on a
// session setting ("schema_key").
type schemaTablePrefixPlugin struct{}

func (schemaTablePrefixPlugin) Name() string { return "schema_table_prefix" }

func (schemaTablePrefixPlugin) Initialize(db *gorm.DB) error {
	applyPrefix := func(d *gorm.DB) {
		if schema, ok := d.Get("schema_key"); ok {
			if s, ok := schema.(string); ok && s != "" {
				d.Statement.Table = s + "_" + d.Statement.Table
			}
		}
	}

	db.Callback().Create().Before("*").Register("schema_table_prefix:create", applyPrefix)
	db.Callback().Query().Before("*").Register("schema_table_prefix:query", applyPrefix)
	return nil
}

// TestPrepareStmtWithSchemaTablePrefixPlugin reproduces the scenario from issue
// #7826: PrepareStmt:true combined with a Before("*") callback that mutates
// Statement.Table must not panic when creating rows with auto-timestamp fields,
// even when the same shared prepared-statement cache serves multiple tenants.
func TestPrepareStmtWithSchemaTablePrefixPlugin(t *testing.T) {
	db, err := OpenTestConnection(&gorm.Config{PrepareStmt: true})
	if err != nil {
		t.Fatalf("failed to open test connection: %v", err)
	}

	if err := db.Use(schemaTablePrefixPlugin{}); err != nil {
		t.Fatalf("failed to use plugin: %v", err)
	}

	// Create the base table and the per-tenant copies the plugin will target so
	// the full INSERT/RETURNING path is exercised without a SQL error.
	if err := DB.AutoMigrate(&prefixUser{}); err != nil {
		t.Fatalf("failed to auto migrate prefixUser: %v", err)
	}
	for _, tenant := range []string{"tenant_a", "tenant_b"} {
		prefixed := tenant + "_prefix_users"
		if err := DB.Migrator().DropTable(prefixed); err != nil {
			t.Fatalf("failed to drop %s: %v", prefixed, err)
		}
		if err := DB.Table(prefixed).Migrator().CreateTable(&prefixUser{}); err != nil {
			t.Fatalf("failed to create %s: %v", prefixed, err)
		}
	}

	// A panic here is the bug described in the issue.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Create panicked: %v", r)
		}
	}()

	// Interleave creates across two tenants to exercise the shared
	// prepared-statement cache with different mutated table names.
	for i := 0; i < 10; i++ {
		tenant := "tenant_a"
		if i%2 == 1 {
			tenant = "tenant_b"
		}
		user := prefixUser{Name: fmt.Sprintf("prepare_stmt_prefix_%d", i)}

		tx := db.Session(&gorm.Session{}).Set("schema_key", tenant)
		if err := tx.Create(&user).Error; err != nil {
			t.Fatalf("Create #%d (%s) failed: %v", i, tenant, err)
		}

		if user.ID == 0 {
			t.Fatalf("Create #%d (%s) did not set primary key", i, tenant)
		}
		if user.CreatedAt.IsZero() {
			t.Fatalf("Create #%d (%s) did not set CreatedAt", i, tenant)
		}
		if user.UpdatedAt.IsZero() {
			t.Fatalf("Create #%d (%s) did not set UpdatedAt", i, tenant)
		}

		// Verify the row landed in the tenant-prefixed table.
		var count int64
		if err := DB.Table(tenant+"_prefix_users").Where("id = ?", user.ID).Count(&count).Error; err != nil {
			t.Fatalf("Count #%d (%s) failed: %v", i, tenant, err)
		}
		if count != 1 {
			t.Fatalf("Create #%d (%s) expected 1 row in %s_prefix_users, got %d", i, tenant, tenant, count)
		}
	}
}
