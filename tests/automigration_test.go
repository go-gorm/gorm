package tests_test

import (
	"testing"
	"time"
)

func TestSmartAutoMigrateColumnNullable(t *testing.T) {
	fullSupported := map[string]bool{"mysql": true, "postgres": true}[DB.Dialector.Name()]

	type UserMigrateColumn struct {
		ID       uint
		Name     string
		Salary   float64
		Bonus    float64 `gorm:"not null"`
		Stock    float64
		Birthday time.Time `gorm:"precision:4"`
	}

	DB.Migrator().DropTable(&UserMigrateColumn{})

	DB.AutoMigrate(&UserMigrateColumn{})

	type UserMigrateColumn2 struct {
		ID                  uint
		Name                string  `gorm:"size:128"`
		Salary              float64 `gorm:"precision:2"`
		Bonus               float64
		Stock               float64   `gorm:"not null"`
		Birthday            time.Time `gorm:"precision:2"`
		NameIgnoreMigration string    `gorm:"size:100"`
	}

	if err := DB.Table("user_migrate_columns").AutoMigrate(&UserMigrateColumn2{}); err != nil {
		t.Fatalf("failed to auto migrate, got error: %v", err)
	}

	columnTypes, err := DB.Table("user_migrate_columns").Migrator().ColumnTypes(&UserMigrateColumn{})
	if err != nil {
		t.Fatalf("failed to get column types, got error: %v", err)
	}

	for _, columnType := range columnTypes {
		switch columnType.Name() {
		case "name":
			if length, _ := columnType.Length(); (fullSupported || length != 0) && length != 128 {
				t.Fatalf("name's length should be 128, but got %v", length)
			}
		case "salary":
			if precision, o, _ := columnType.DecimalSize(); (fullSupported || precision != 0) && precision != 2 {
				t.Fatalf("salary's precision should be 2, but got %v %v", precision, o)
			}
		case "bonus":
			// allow to change non-nullable to nullable
			if nullable, _ := columnType.Nullable(); !nullable {
				t.Fatalf("bonus's nullable should be true, bug got %t", nullable)
			}
		case "stock":
			// do not allow to change nullable to non-nullable
			if nullable, _ := columnType.Nullable(); !nullable {
				t.Fatalf("stock's nullable should be true, bug got %t", nullable)
			}
		case "birthday":
			if precision, _, _ := columnType.DecimalSize(); (fullSupported || precision != 0) && precision != 2 {
				t.Fatalf("birthday's precision should be 2, but got %v", precision)
			}
		}
	}
}
