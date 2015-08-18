package gorm

import (
	"errors"
	"strings"
)

var (
	RecordNotFound       = errors.New("record not found")
	InvalidSql           = errors.New("invalid sql")
	NoNewAttrs           = errors.New("no new attributes")
	NoValidTransaction   = errors.New("no valid transaction")
	CantStartTransaction = errors.New("can't start transaction")
)

type errorsInterface interface {
	GetErrors() []error
}

type Errors struct {
	errors []error
}

func (errs Errors) GetErrors() []error {
	return errs.errors
}

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

func (errs Errors) Error() string {
	var errors = []string{}
	for _, e := range errs.errors {
		errors = append(errors, e.Error())
	}
	return strings.Join(errors, "; ")
}
