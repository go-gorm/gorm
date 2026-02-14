package tests_test

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	. "gorm.io/gorm/utils/tests"
)

type SerializerStruct struct {
	gorm.Model
	Name                   []byte                 `gorm:"json"`
	Roles                  Roles                  `gorm:"serializer:json"`
	Roles2                 *Roles                 `gorm:"serializer:json"`
	Roles3                 *Roles                 `gorm:"serializer:json;not null"`
	Contracts              map[string]interface{} `gorm:"serializer:json"`
	JobInfo                Job                    `gorm:"type:bytes;serializer:gob"`
	CreatedTime            int64                  `gorm:"serializer:unixtime;type:datetime"` // store time in db, use int as field type
	UpdatedTime            *int64                 `gorm:"serializer:unixtime;type:datetime"` // store time in db, use int as field type
	CustomSerializerString string                 `gorm:"serializer:custom"`
	EncryptedString        EncryptedString
}

type SerializerPostgresStruct struct {
	gorm.Model
	Name                   []byte                 `gorm:"json"`
	Roles                  Roles                  `gorm:"serializer:json"`
	Roles2                 *Roles                 `gorm:"serializer:json"`
	Roles3                 *Roles                 `gorm:"serializer:json;not null"`
	Contracts              map[string]interface{} `gorm:"serializer:json"`
	JobInfo                Job                    `gorm:"type:bytes;serializer:gob"`
	CreatedTime            int64                  `gorm:"serializer:unixtime;type:timestamptz"` // store time in db, use int as field type
	UpdatedTime            *int64                 `gorm:"serializer:unixtime;type:timestamptz"` // store time in db, use int as field type
	CustomSerializerString string                 `gorm:"serializer:custom"`
	EncryptedString        EncryptedString
}

func (*SerializerPostgresStruct) TableName() string { return "serializer_structs" }

func adaptorSerializerModel(s *SerializerStruct) interface{} {
	if DB.Dialector.Name() == "postgres" || DB.Dialector.Name() == "gaussdb" {
		sps := SerializerPostgresStruct(*s)
		return &sps
	}
	return s
}

type Roles []string

type Job struct {
	Title    string
	Number   int
	Location string
	IsIntern bool
}

type EncryptedString string

func (es *EncryptedString) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) (err error) {
	switch value := dbValue.(type) {
	case []byte:
		*es = EncryptedString(bytes.TrimPrefix(value, []byte("hello")))
	case string:
		*es = EncryptedString(strings.TrimPrefix(value, "hello"))
	default:
		return fmt.Errorf("unsupported data %#v", dbValue)
	}
	return nil
}

func (es EncryptedString) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	return "hello" + string(es), nil
}

type CustomSerializer struct {
	prefix []byte
}

func NewCustomSerializer(prefix string) *CustomSerializer {
	return &CustomSerializer{prefix: []byte(prefix)}
}

func (c *CustomSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) (err error) {
	switch value := dbValue.(type) {
	case []byte:
		err = field.Set(ctx, dst, bytes.TrimPrefix(value, c.prefix))
	case string:
		err = field.Set(ctx, dst, strings.TrimPrefix(value, string(c.prefix)))
	default:
		err = fmt.Errorf("unsupported data %#v", dbValue)
	}
	return err
}

func (c *CustomSerializer) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	return fmt.Sprintf("%s%s", c.prefix, fieldValue), nil
}

func TestSerializer(t *testing.T) {
	schema.RegisterSerializer("custom", NewCustomSerializer("hello"))
	DB.Migrator().DropTable(adaptorSerializerModel(&SerializerStruct{}))
	if err := DB.Migrator().AutoMigrate(adaptorSerializerModel(&SerializerStruct{})); err != nil {
		t.Fatalf("no error should happen when migrate scanner, valuer struct, got error %v", err)
	}

	createdAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Unix()

	data := SerializerStruct{
		Name:            []byte("jinzhu"),
		Roles:           []string{"r1", "r2"},
		Contracts:       map[string]interface{}{"name": "jinzhu", "age": 10},
		EncryptedString: EncryptedString("pass"),
		CreatedTime:     createdAt.Unix(),
		UpdatedTime:     &updatedAt,
		JobInfo: Job{
			Title:    "programmer",
			Number:   9920,
			Location: "Kenmawr",
			IsIntern: false,
		},
		CustomSerializerString: "world",
	}

	if err := DB.Create(&data).Error; err != nil {
		t.Fatalf("failed to create data, got error %v", err)
	}

	var result SerializerStruct
	if err := DB.Where("roles2 IS NULL AND roles3 = ?", "").First(&result, data.ID).Error; err != nil {
		t.Fatalf("failed to query data, got error %v", err)
	}

	AssertEqual(t, result, data)

	if err := DB.Model(&result).Update("roles", "").Error; err != nil {
		t.Fatalf("failed to update data's roles, got error %v", err)
	}

	if err := DB.First(&result, data.ID).Error; err != nil {
		t.Fatalf("failed to query data, got error %v", err)
	}
}

func TestSerializerZeroValue(t *testing.T) {
	schema.RegisterSerializer("custom", NewCustomSerializer("hello"))
	DB.Migrator().DropTable(adaptorSerializerModel(&SerializerStruct{}))
	if err := DB.Migrator().AutoMigrate(adaptorSerializerModel(&SerializerStruct{})); err != nil {
		t.Fatalf("no error should happen when migrate scanner, valuer struct, got error %v", err)
	}

	data := SerializerStruct{}

	if err := DB.Create(&data).Error; err != nil {
		t.Fatalf("failed to create data, got error %v", err)
	}

	var result SerializerStruct
	if err := DB.First(&result, data.ID).Error; err != nil {
		t.Fatalf("failed to query data, got error %v", err)
	}

	AssertEqual(t, result, data)

	if err := DB.Model(&result).Update("roles", "").Error; err != nil {
		t.Fatalf("failed to update data's roles, got error %v", err)
	}

	if err := DB.First(&result, data.ID).Error; err != nil {
		t.Fatalf("failed to query data, got error %v", err)
	}
}

func TestSerializerAssignFirstOrCreate(t *testing.T) {
	schema.RegisterSerializer("custom", NewCustomSerializer("hello"))
	DB.Migrator().DropTable(adaptorSerializerModel(&SerializerStruct{}))
	if err := DB.Migrator().AutoMigrate(adaptorSerializerModel(&SerializerStruct{})); err != nil {
		t.Fatalf("no error should happen when migrate scanner, valuer struct, got error %v", err)
	}

	createdAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	data := SerializerStruct{
		Name:            []byte("ag9920"),
		Roles:           []string{"r1", "r2"},
		Contracts:       map[string]interface{}{"name": "jing1", "age": 11},
		EncryptedString: EncryptedString("pass"),
		CreatedTime:     createdAt.Unix(),
		JobInfo: Job{
			Title:    "programmer",
			Number:   9920,
			Location: "Shadyside",
			IsIntern: false,
		},
		CustomSerializerString: "world",
	}

	// first time insert record
	out := SerializerStruct{}
	if err := DB.Assign(data).FirstOrCreate(&out).Error; err != nil {
		t.Fatalf("failed to FirstOrCreate Assigned data, got error %v", err)
	}

	var result SerializerStruct
	if err := DB.First(&result, out.ID).Error; err != nil {
		t.Fatalf("failed to query data, got error %v", err)
	}
	AssertEqual(t, result, out)

	// update record
	data.Roles = append(data.Roles, "r3")
	data.JobInfo.Location = "Gates Hillman Complex"
	if err := DB.Assign(data).FirstOrCreate(&out).Error; err != nil {
		t.Fatalf("failed to FirstOrCreate Assigned data, got error %v", err)
	}
	if err := DB.First(&result, out.ID).Error; err != nil {
		t.Fatalf("failed to query data, got error %v", err)
	}

	AssertEqual(t, result.Roles, data.Roles)
	AssertEqual(t, result.JobInfo.Location, data.JobInfo.Location)
}

// Test for: panic when serializer field with any type is nil
func TestSerializerWithAnyType(t *testing.T) {
	type ProductWithAny struct {
		gorm.Model
		Name string
		Data any `gorm:"serializer:json"`
	}

	DB.Migrator().DropTable(&ProductWithAny{})
	if err := DB.AutoMigrate(&ProductWithAny{}); err != nil {
		t.Fatalf("failed to migrate ProductWithAny, got error %v", err)
	}

	// Test creating record with nil any field
	product := ProductWithAny{Name: "Product 1"}
	if err := DB.Create(&product).Error; err != nil {
		t.Fatalf("failed to create product with nil any field, got error %v", err)
	}

	// Test updating/saving record with nil any field (should not panic)
	product.Name = "Product 1 (Updated)"
	if err := DB.Save(&product).Error; err != nil {
		t.Fatalf("failed to save product with nil any field, got error %v", err)
	}

	// Verify the record was saved correctly
	var result ProductWithAny
	if err := DB.First(&result, product.ID).Error; err != nil {
		t.Fatalf("failed to query product, got error %v", err)
	}

	if result.Name != "Product 1 (Updated)" {
		t.Errorf("expected name to be 'Product 1 (Updated)', got %s", result.Name)
	}

	if result.Data != nil {
		t.Errorf("expected Data to be nil, got %v", result.Data)
	}

	// Test with non-nil value
	dataValue := map[string]interface{}{"key": "value"}
	product2 := ProductWithAny{Name: "Product 2", Data: dataValue}
	if err := DB.Create(&product2).Error; err != nil {
		t.Fatalf("failed to create product with non-nil any field, got error %v", err)
	}

	var result2 ProductWithAny
	if err := DB.First(&result2, product2.ID).Error; err != nil {
		t.Fatalf("failed to query product2, got error %v", err)
	}

	if result2.Data == nil {
		t.Error("expected Data to be non-nil")
	}
}
