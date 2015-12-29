package gorm_test

import (
	"testing"
	"time"
)

type CustomizeColumn struct {
	ID   int64     `gorm:"column:mapped_id; primary_key:yes"`
	Name string    `gorm:"column:mapped_name"`
	Date time.Time `gorm:"column:mapped_time"`
}

// Make sure an ignored field does not interfere with another field's custom
// column name that matches the ignored field.
type CustomColumnAndIgnoredFieldClash struct {
	Body    string `sql:"-"`
	RawBody string `gorm:"column:body"`
}

func TestCustomizeColumn(t *testing.T) {
	col := "mapped_name"
	DB.DropTable(&CustomizeColumn{})
	DB.AutoMigrate(&CustomizeColumn{})

	scope := DB.NewScope(&CustomizeColumn{})
	if !scope.Dialect().HasColumn(scope, scope.TableName(), col) {
		t.Errorf("CustomizeColumn should have column %s", col)
	}

	col = "mapped_id"
	if scope.PrimaryKey() != col {
		t.Errorf("CustomizeColumn should have primary key %s, but got %q", col, scope.PrimaryKey())
	}
	
	tableName := "customize_columns"
	idInsRes := DB.EnableIdentityInsert(&DB, tableName)
	if idInsRes.Error != nil {
		t.Errorf("Error while setting IDENTITY_INSERT ON for table:%v :%v", tableName, idInsRes.Error)
	}

	expected := "foo"
	cc := CustomizeColumn{ID: 666, Name: expected, Date: time.Now()}

    res := DB.Create(&cc)
	if res.Error != nil {
		t.Errorf("Error while creating CustomizeColumn:%v", res.Error)
	} 
	
	if count := res.RowsAffected; count != 1 {
		t.Errorf("There should be one record be affected when create record. count:%v", count)
	}

	var cc1 CustomizeColumn
	DB.First(&cc1, 666)

	if cc1.Name != expected {
		t.Errorf("Failed to query CustomizeColumn")
	}

	cc.Name = "bar"
	DB.Save(&cc)

	var cc2 CustomizeColumn
	DB.First(&cc2, 666)
	if cc2.Name != "bar" {
		t.Errorf("Failed to query CustomizeColumn")
	}
}

func TestCustomColumnAndIgnoredFieldClash(t *testing.T) {
	DB.DropTable(&CustomColumnAndIgnoredFieldClash{})
	if err := DB.AutoMigrate(&CustomColumnAndIgnoredFieldClash{}).Error; err != nil {
		t.Errorf("Should not raise error: %s", err)
	}
}
