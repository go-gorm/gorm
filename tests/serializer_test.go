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
	Name            []byte                 `gorm:"json"`
	Roles           Roles                  `gorm:"serializer:json"`
	Contracts       map[string]interface{} `gorm:"serializer:json"`
	JobInfo         Job                    `gorm:"type:bytes;serializer:gob"`
	CreatedTime     int64                  `gorm:"serializer:unixtime;type:time"` // store time in db, use int as field type
	UpdatedTime     *int64                 `gorm:"serializer:unixtime;type:time"` // store time in db, use int as field type
	EncryptedString EncryptedString
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

func TestSerializer(t *testing.T) {
	DB.Migrator().DropTable(&SerializerStruct{})
	if err := DB.Migrator().AutoMigrate(&SerializerStruct{}); err != nil {
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
	}

	if err := DB.Create(&data).Error; err != nil {
		t.Fatalf("failed to create data, got error %v", err)
	}

	var result SerializerStruct
	if err := DB.First(&result, data.ID).Error; err != nil {
		t.Fatalf("failed to query data, got error %v", err)
	}

	AssertEqual(t, result, data)

}

func TestSerializerAssignFirstOrCreate(t *testing.T) {
	DB.Migrator().DropTable(&SerializerStruct{})
	if err := DB.Migrator().AutoMigrate(&SerializerStruct{}); err != nil {
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

	//update record
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
