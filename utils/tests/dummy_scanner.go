package tests

import (
	"database/sql/driver"
	"fmt"
)

type DummyString struct {
	value string
}

func NewDummyString(s string) DummyString {
	return DummyString{
		value: s,
	}
}

func (d *DummyString) Scan(value interface{}) error {
	switch v := value.(type) {
	case string:
		d.value = v
	default:
		d.value = fmt.Sprintf("%v", value)
	}

	return nil
}

func (d DummyString) Value() (driver.Value, error) {
	return d.value, nil
}

func (d DummyString) String() string {
	return d.value
}
