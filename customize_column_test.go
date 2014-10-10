package gorm_test

import (
	"testing"
	"time"
)

type CustomizeColumn struct {
	Id   int64     `gorm:"column:mapped_id; primary_key:yes"`
	Name string    `gorm:"column:mapped_name"`
	Date time.Time `gorm:"column:mapped_time"`
}

// Make sure an ignored field does not interfere with another field's custom
// column name that matches the ignored field.
type CustomColumnAndIgnoredFieldClash struct {
	Body        string `sql:"-"`
	RawBody     string `gorm:"column:body"`
}

func TestCustomizeColumn(t *testing.T) {
	col := "mapped_name"
	DB.DropTable(&CustomizeColumn{})
	DB.AutoMigrate(&CustomizeColumn{})

	scope := DB.Model("").NewScope(&CustomizeColumn{})
	if !scope.Dialect().HasColumn(scope, scope.TableName(), col) {
		t.Errorf("CustomizeColumn should have column %s", col)
	}

	col = "mapped_id"
	if scope.PrimaryKey() != col {
		t.Errorf("CustomizeColumn should have primary key %s, but got %q", col, scope.PrimaryKey())
	}

	expected := "foo"
	cc := CustomizeColumn{Id: 666, Name: expected, Date: time.Now()}

	if count := DB.Save(&cc).RowsAffected; count != 1 {
		t.Error("There should be one record be affected when create record")
	}

	var ccs []CustomizeColumn
	DB.Find(&ccs)

	if len(ccs) > 0 && ccs[0].Name != expected && ccs[0].Id != 666 {
		t.Errorf("Failed to query CustomizeColumn")
	}
}

func TestCustomColumnAndIgnoredFieldClash(t *testing.T) {
	DB.DropTable(&CustomColumnAndIgnoredFieldClash{})
	DB.AutoMigrate(&CustomColumnAndIgnoredFieldClash{})
}
