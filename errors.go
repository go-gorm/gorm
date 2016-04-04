package gorm

import (
	"errors"
	"strings"
)

var (
	// ErrRecordNotFound record not found error, happens when haven't find any matched data when looking up with a struct
	ErrRecordNotFound = errors.New("record not found")
	// ErrInvalidSQL invalid SQL error, happens when you passed invalid SQL
	ErrInvalidSQL = errors.New("invalid SQL")
	// ErrInvalidTransaction invalid transaction when you are trying to `Commit` or `Rollback`
	ErrInvalidTransaction = errors.New("no valid transaction")
	// ErrCantStartTransaction can't start transaction when you are trying to start one with `Begin`
	ErrCantStartTransaction = errors.New("can't start transaction")
	// ErrUnaddressable unaddressable value
	ErrUnaddressable = errors.New("using unaddressable value")
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
