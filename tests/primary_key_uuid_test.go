package tests_test

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm/schema"
)

func TestStringPrimaryKeyDefault(t *testing.T) {
	type Product struct {
		ID   uuid.UUID
		Code string
		Name string
	}
	type ProductWithNamedPrimaryKey struct {
		ProductID uuid.UUID `gorm:"primaryKey"`
		Code      string
		Name      string
	}

	product, err := schema.Parse(&Product{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse product struct with composite primary key, got error %v", err)
	}

	isInDefault := false
	for _, field := range product.FieldsWithDefaultDBValue {
		if field.Name == "ID" {
			isInDefault = true
			break
		}
	}
	if !isInDefault {
		t.Errorf("ID should be fields with default")
	}

	productWithNamedPrimaryKey, err := schema.Parse(&ProductWithNamedPrimaryKey{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse product struct with composite primary key, got error %v", err)
	}

	isInDefault = false
	for _, field := range productWithNamedPrimaryKey.FieldsWithDefaultDBValue {
		if field.Name == "ProductID" {
			isInDefault = true
			break
		}
	}
	if !isInDefault {
		t.Errorf("ProductID should be fields with default")
	}
}
