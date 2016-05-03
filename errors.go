package gorm

import (
	"errors"
	"strings"
)

type errorsInterface interface {
	GetErrors() []error
}

// Errors contains all happened errors
type Errors struct {
	errors []error
}

// GetErrors get all happened errors
func (errs Errors) GetErrors() []error {
	return errs.errors
}

type ErrRecordNotFound struct{ error }

type ErrInvalidSQL struct{ error }

type ErrInvalidTransaction struct{ error }

type ErrCantStartTransaction struct{ error }

type ErrUnaddressable struct{ error }

func NewErrRecordNotFound() error {
	return ErrRecordNotFound{errors.New("record not found")}
}

func NewErrInvalidSQL() error {
	return ErrInvalidSQL{errors.New("invalid SQL")}
}

func NewErrInvalidTransaction() error {
	return ErrCantStartTransaction{errors.New("no valid transaction")}
}

func NewErrCantStartTransaction() error {
	return ErrCantStartTransaction{errors.New("can't start transaction")}
}

func NewErrUnaddressable() error {
	return ErrUnaddressable{errors.New("using unaddressable value")}
}

// Add add an error
func (errs *Errors) Add(err error) {
	if errors, ok := err.(errorsInterface); ok {
		for _, err := range errors.GetErrors() {
			errs.Add(err)
		}
	} else {
		for _, e := range errs.errors {
			if err == e {
				return
			}
		}
		errs.errors = append(errs.errors, err)
	}
}

// Error format happened errors
func (errs Errors) Error() string {
	var errors = []string{}
	for _, e := range errs.errors {
		errors = append(errors, e.Error())
	}
	return strings.Join(errors, "; ")
}
