package tests_test

import (
	"database/sql"
	"gorm.io/gorm"
	"testing"

	. "gorm.io/gorm/utils/tests"
)

func TestSqlNullableFields(t *testing.T) {
	type record struct {
		gorm.Model
		DateTime sql.NullTime
		Count    *sql.NullInt64
	}

	if err := DB.AutoMigrate(record{}); err != nil {
		t.Fatal("failed to migrate record")
	}
	defer func() {
		_ = DB.Migrator().DropTable(record{})
	}()

	r := record{
		DateTime: sql.NullTime{
			Time:  *Now(),
			Valid: true,
		},
		Count: &sql.NullInt64{
			Int64: 1,
			Valid: true,
		},
	}
	if err := DB.Create(&r).Error; err != nil {
		t.Fatalf("failed to create record %s", err.Error())
	}

	if err := DB.Model(r).
		Update("dateTime", nil).
		Update("count", nil).
		Where("id", r.ID).Error; err != nil {
		t.Fatalf("failed to update record %s", err.Error())
	}

	if err := DB.Where("id", r.ID).First(&r).Error; err != nil {
		t.Fatalf("failed to find record: %s", err.Error())
	}

	if r.Count.Valid {
		t.Fatal("record count should be invalid")
	}
	if r.DateTime.Valid {
		t.Fatal("record datetime should be invalid")
	}
}
