package tests_test

import (
	"testing"

	"gorm.io/gorm"
)

type Man struct {
	ID     int
	Age    int
	Name   string
	Detail string
}

// Panic-safe BeforeUpdate hook that checks for Changed("age")
func (m *Man) BeforeUpdate(tx *gorm.DB) (err error) {
	if !tx.Statement.Changed("age") {
		return nil
	}
	return nil
}

func TestSubModel(t *testing.T) {
	man := Man{Age: 18, Name: "random-name"}
	if err := DB.Create(&man).Error; err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := DB.Model(&man).Where("id = ?", man.ID).Updates(struct {
		Age int
	}{Age: 20}).Error; err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result = struct {
		ID  int
		Age int
	}{}
	if err := DB.Model(&man).Where("id = ?", man.ID).Find(&result).Error; err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != man.ID || result.Age != 20 {
		t.Fatalf("expected ID %d and Age 20, got ID %d and age", result.ID, result.Age)
	}
}
