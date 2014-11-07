package gorm_test

import "testing"

type Example struct {
	Id uint64 `gorm:"column:id; primary_key:yes"`
}

func (e *Example) TableName() string {
	return "exampling"
}

//Test to reproduce invalid/non-existant TableName behaviour
func TestInsertWithTableName(t *testing.T) {
	DB.Exec("drop table examples;")
	DB.Exec("drop table exampling;")

	exampling := Example{}

	if err := DB.CreateTable(exampling).Error; err != nil {
		t.Error(err)
	}

	if err := DB.Save(exampling).Error; err != nil {
		t.Error(err)
	}

	if err := DB.Exec("SELECT count(*) from exampling;").Error; err != nil {
		t.Error(err)
	}
}

func TestInsertWithTableNameExplicit(t *testing.T) {
	DB.Exec("drop table examples;")
	DB.Exec("drop table exampling;")

	exampling := Example{}

	if err := DB.Table("exampling").CreateTable(exampling).Error; err != nil {
		t.Error(err)
	}

	if err := DB.Table("exampling").Save(exampling).Error; err != nil {
		t.Error(err)
	}

	if err := DB.Exec("SELECT count(*) from exampling;").Error; err != nil {
		t.Error(err)
	}
}
