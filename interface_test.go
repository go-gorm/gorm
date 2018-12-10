package gorm_test

import (
	"reflect"
	"testing"

	"github.com/kr/pretty"
)

type (
	UserInterface interface {
		UserName() string
		UserType() string
	}

	UserCommon struct {
		Name string
		Type string
	}

	BasicUser struct {
		User
	}

	AdminUser struct {
		BasicUser
	}

	GroupUser struct {
		GroupID int64
		User    UserInterface
	}

	Group struct {
		Users []GroupUser
	}
)

func (m *BasicUser) UserName() string {
	return m.Name
}

func (m *BasicUser) Type() string {
	return "basic"
}

func (m *AdminUser) Type() string {
	return "admin"
}

// ScanType returns the scan type for the field
func (m *GroupUser) ScanType(field string) reflect.Type {
	switch field {
	case "User":
		// The geometry data should be encoded as a []byte first
		return reflect.TypeOf(User{})
	default:
		return reflect.TypeOf(nil)
	}
}

// ScanField handle exporting scanned fields
func (m *GroupUser) ScanField(field string, data interface{}) error {
	switch field {
	case "User":
		m.User = data.(UserInterface)
	}

	return nil
}

var tt *testing.T

func TestInterface(t *testing.T) {
	tt = t
	DB.AutoMigrate(&UserCommon{})

	user1 := UserCommon{Name: "RowUser1", type: "basic"}

	DB.Save(&user1)

	t.Log("loading the users")
	users := make([]*UserWrapper, 0)

	if DB.Table("users").Find(&users).Error != nil {
		t.Errorf("No errors should happen if set table for find")
	}
	t.Logf(pretty.Sprint(users))
}
