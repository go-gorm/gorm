package tests_test

import (
	"testing"
	"time"

	. "github.com/jinzhu/gorm/tests"
)

func TestCustomizeColumn(t *testing.T) {
	type CustomizeColumn struct {
		ID   int64      `gorm:"column:mapped_id; primary_key:yes"`
		Name string     `gorm:"column:mapped_name"`
		Date *time.Time `gorm:"column:mapped_time"`
	}

	DB.Migrator().DropTable(&CustomizeColumn{})
	DB.AutoMigrate(&CustomizeColumn{})

	expected := "foo"
	now := time.Now()
	cc := CustomizeColumn{ID: 666, Name: expected, Date: &now}

	if count := DB.Create(&cc).RowsAffected; count != 1 {
		t.Error("There should be one record be affected when create record")
	}

	var cc1 CustomizeColumn
	DB.First(&cc1, "mapped_name = ?", "foo")

	if cc1.Name != expected {
		t.Errorf("Failed to query CustomizeColumn")
	}

	cc.Name = "bar"
	DB.Save(&cc)

	var cc2 CustomizeColumn
	DB.First(&cc2, "mapped_id = ?", 666)
	if cc2.Name != "bar" {
		t.Errorf("Failed to query CustomizeColumn")
	}
}

func TestCustomColumnAndIgnoredFieldClash(t *testing.T) {
	// Make sure an ignored field does not interfere with another field's custom
	// column name that matches the ignored field.
	type CustomColumnAndIgnoredFieldClash struct {
		Body    string `gorm:"-"`
		RawBody string `gorm:"column:body"`
	}

	DB.Migrator().DropTable(&CustomColumnAndIgnoredFieldClash{})

	if err := DB.AutoMigrate(&CustomColumnAndIgnoredFieldClash{}); err != nil {
		t.Errorf("Should not raise error: %v", err)
	}
}
