package gorm_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/jinzhu/gorm"
)

type (
	Widget interface {
		GetType() string
	}

	WidgetUser struct {
		User
		WidgetType string
		Widget     Widget `gorm:"use_encoder;column:widget;type:jsonb"`
	}

	SimpleWidget struct {
		Type   string `json:"type"`
		Width  int64  `json:"width"`
		Height int64  `json:"height"`
	}

	ComplexWidget struct {
		SimpleWidget
		Color string `json:"color"`
	}
)

func (m *SimpleWidget) GetType() string {
	return "simple"
}

func (m *ComplexWidget) GetType() string {
	return "complex"
}

func (m *WidgetUser) EncodeField(scope *gorm.Scope, column string) (interface{}, error) {
	switch column {
	case "widget":
		val, err := json.Marshal(m.Widget)
		if err != nil {
			return nil, err
		}
		return string(val), nil
	}

	return nil, nil
}

func (m *WidgetUser) DecodeField(scope *gorm.Scope, column string, value interface{}) error {
	switch column {
	case "widget":
		b, ok := value.([]byte)
		if !ok {
			return errors.New("Invalid type for Widget")
		}
		switch m.WidgetType {
		case "simple":
			var result SimpleWidget
			if err := json.Unmarshal(b, &result); err != nil {
				return err
			}
			m.Widget = &result
		case "complex":
			var result ComplexWidget
			if err := json.Unmarshal(b, &result); err != nil {
				return err
			}
			m.Widget = &result
		default:
			return errors.New("unsupported Widget type")
		}
	}
	return nil
}

func TestEncoder(t *testing.T) {
	DB.AutoMigrate(&WidgetUser{})

	user := &WidgetUser{
		User: User{
			Id:   1,
			Name: "bob",
		},
		WidgetType: "simple",
		Widget:     &SimpleWidget{Type: "simple", Width: 12, Height: 10},
	}

	if err := DB.Save(user).Error; err != nil {
		t.Errorf("failed to save WidgetUser %v", err)
	}

	user1 := WidgetUser{}

	if err := DB.First(&user1, "id=?", 1).Error; err != nil {
		t.Errorf("failed to retrieve WidgetUser %v", err)
	}

	if user1.Widget.GetType() != "simple" {
		t.Errorf("user widget invalid")
	}

	if w, ok := user1.Widget.(*SimpleWidget); !ok {
		t.Errorf("user widget is not valid")
	} else {
		if w.Width != 12 || w.Height != 10 {
			t.Errorf("user widget is not valid")
		}
	}
}
