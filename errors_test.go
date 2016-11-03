package gorm_test

import (
	"errors"
	"testing"

	"github.com/jinzhu/gorm"
)

func TestErrorsCanBeUsedOutsideGorm(t *testing.T) {
	errs := []error{errors.New("First"), errors.New("Second")}

	gErrs := gorm.Errors(errs)
	gErrs = gErrs.Add(errors.New("Third"))
	gErrs = gErrs.Add(gErrs)

	if gErrs.Error() != "First; Second; Third" {
		t.Fatalf("Gave wrong error, got %s", gErrs.Error())
	}
}
