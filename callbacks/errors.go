package callbacks

import (
	"fmt"
	"gorm.io/gorm/schema"
)

type _SetFieldValueError struct {
	Field *schema.Field
	Err   error
}

func (e _SetFieldValueError) Error() string {
	return fmt.Sprintf("error when set value for field %s: %v", e.Field.Name, e.Err)
}

func (e _SetFieldValueError) Unwrap() error {
	return e.Err
}

func newSetFieldValueError(field *schema.Field, e error) error {
	if e == nil {
		return nil
	}
	//goland:noinspection GoTypeAssertionOnErrors
	if we, ok := e.(*_SetFieldValueError); ok && we.Field == field {
		return e
	}
	if field == nil {
		panic("field is nil")
	}
	return &_SetFieldValueError{
		Field: field,
		Err:   e,
	}
}
