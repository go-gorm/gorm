package gorm_test

import (
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/jinzhu/gorm"
)

type CalculateField struct {
	gorm.Model
	Name     string
	Children []CalculateFieldChild
	Category CalculateFieldCategory
	EmbeddedField
}

type EmbeddedField struct {
	EmbeddedName string `sql:"NOT NULL;DEFAULT:'hello'"`
}

type CalculateFieldChild struct {
	gorm.Model
	CalculateFieldID uint
	Name             string
}

type CalculateFieldCategory struct {
	gorm.Model
	CalculateFieldID uint
	Name             string
}

func TestCalculateField(t *testing.T) {
	var field CalculateField
	var scope = DB.NewScope(&field)
	if field, ok := scope.FieldByName("Children"); !ok || field.Relationship == nil {
		t.Errorf("Should calculate fields correctly for the first time")
	}

	if field, ok := scope.FieldByName("Category"); !ok || field.Relationship == nil {
		t.Errorf("Should calculate fields correctly for the first time")
	}

	if field, ok := scope.FieldByName("embedded_name"); !ok {
		t.Errorf("should find embedded field")
	} else if _, ok := field.TagSettingsGet("NOT NULL"); !ok {
		t.Errorf("should find embedded field's tag settings")
	}
}

type UUID [16]byte

type NullUUID struct {
	UUID
	Valid bool
}

func FromString(input string) (u UUID) {
	src := []byte(input)
	return FromBytes(src)
}

func FromBytes(src []byte) (u UUID) {
	dst := u[:]
	hex.Decode(dst[0:4], src[0:8])
	hex.Decode(dst[4:6], src[9:13])
	hex.Decode(dst[6:8], src[14:18])
	hex.Decode(dst[8:10], src[19:23])
	hex.Decode(dst[10:], src[24:])
	return
}

func (u UUID) String() string {
	buf := make([]byte, 36)
	src := u[:]
	hex.Encode(buf[0:8], src[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], src[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], src[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], src[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:], src[10:])
	return string(buf)
}

func (u UUID) Value() (driver.Value, error) {
	return u.String(), nil
}

func (u *UUID) Scan(src interface{}) error {
	switch src := src.(type) {
	case UUID: // support gorm convert from UUID to NullUUID
		*u = src
		return nil
	case []byte:
		*u = FromBytes(src)
		return nil
	case string:
		*u = FromString(src)
		return nil
	}
	return fmt.Errorf("uuid: cannot convert %T to UUID", src)
}

func (u *NullUUID) Scan(src interface{}) error {
	u.Valid = true
	return u.UUID.Scan(src)
}

func TestFieldSet(t *testing.T) {
	type TestFieldSetNullUUID struct {
		NullUUID NullUUID
	}
	scope := DB.NewScope(&TestFieldSetNullUUID{})
	field := scope.Fields()[0]
	err := field.Set(FromString("3034d44a-da03-11e8-b366-4a00070b9f00"))
	if err != nil {
		t.Fatal(err)
	}
	if id, ok := field.Field.Addr().Interface().(*NullUUID); !ok {
		t.Fatal()
	} else if !id.Valid || id.UUID.String() != "3034d44a-da03-11e8-b366-4a00070b9f00" {
		t.Fatal(id)
	}
}
