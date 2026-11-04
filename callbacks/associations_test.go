package callbacks

import (
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func TestOnConflictOption_NoPrimaryKeys_DoNothing(t *testing.T) {
	// Model without any primary key fields
	type NoPKModel struct {
		Name  string
		Value string
	}

	s, err := schema.Parse(&NoPKModel{}, schemaCache, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	if len(s.PrimaryFieldDBNames) != 0 {
		t.Fatalf("expected no primary key fields, got %v", s.PrimaryFieldDBNames)
	}

	stmt := &gorm.Statement{
		DB: &gorm.DB{
			Config: &gorm.Config{
				NowFunc: func() time.Time { return time.Time{} },
			},
			Statement: &gorm.Statement{
				Settings: sync.Map{},
				Schema:   s,
			},
		},
		Schema: s,
	}

	// When defaultUpdatingColumns is non-empty, onConflictOption enters the
	// branch that builds ON CONFLICT columns from primary keys. With zero
	// primary keys, it should fall back to DoNothing.
	onConflict := onConflictOption(stmt, s, []string{"name"})
	if !onConflict.DoNothing {
		t.Errorf("expected DoNothing to be true when schema has no primary keys, got false")
	}
	if len(onConflict.Columns) != 0 {
		t.Errorf("expected no conflict columns, got %v", onConflict.Columns)
	}
}

func TestOnConflictOption_WithPrimaryKeys_DoUpdate(t *testing.T) {
	type PKModel struct {
		ID   int `gorm:"primaryKey"`
		Name string
	}

	s, err := schema.Parse(&PKModel{}, schemaCache, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	stmt := &gorm.Statement{
		DB: &gorm.DB{
			Config: &gorm.Config{
				NowFunc: func() time.Time { return time.Time{} },
			},
			Statement: &gorm.Statement{
				Settings: sync.Map{},
				Schema:   s,
			},
		},
		Schema: s,
	}

	onConflict := onConflictOption(stmt, s, []string{"name"})
	if onConflict.DoNothing {
		t.Errorf("expected DoNothing to be false when schema has primary keys")
	}
	if len(onConflict.Columns) != 1 || onConflict.Columns[0].Name != "id" {
		t.Errorf("expected conflict column [id], got %v", onConflict.Columns)
	}
}
