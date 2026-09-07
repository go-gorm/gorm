package migrator

import (
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

// An unsupported model type makes schema parsing fail, which leaves
// Statement.Schema nil. ReorderModels must not dereference it.
func TestReorderModelsWithUnsupportedType(t *testing.T) {
	db, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open dummy db: %v", err)
	}

	m := Migrator{Config: Config{DB: db}}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ReorderModels panicked on an unsupported model type: %v", r)
		}
	}()

	m.ReorderModels([]interface{}{123}, true)
}
