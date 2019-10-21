// +build go1.13

package gorm_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jinzhu/gorm"
)

func TestNotFound(t *testing.T) {
	var errs = []error{
		gorm.ErrRecordNotFound,
		fmt.Errorf("get user fail: %w", gorm.ErrRecordNotFound),
		gorm.Errors{gorm.ErrRecordNotFound, gorm.ErrRecordNotFound},
		gorm.Errors{fmt.Errorf("get user fail: %w", gorm.ErrRecordNotFound), gorm.ErrRecordNotFound},
		gorm.Errors{gorm.ErrRecordNotFound, fmt.Errorf("get user fail: %w", gorm.ErrRecordNotFound)},
		gorm.Errors{fmt.Errorf("get user fail: %w", gorm.ErrRecordNotFound), fmt.Errorf("get user fail: %w", gorm.ErrRecordNotFound)},
		gorm.Errors{gorm.Errors{fmt.Errorf("get user fail: %w", gorm.ErrRecordNotFound), gorm.ErrRecordNotFound}, gorm.ErrRecordNotFound},

		fmt.Errorf("get user fail: %w", gorm.Errors{gorm.ErrRecordNotFound}),
		fmt.Errorf("get user fail: %w", gorm.Errors{gorm.ErrRecordNotFound, fmt.Errorf("get user fail: %w", gorm.ErrRecordNotFound)}),
		fmt.Errorf("get user fail: %w", gorm.Errors{fmt.Errorf("get user fail: %w", gorm.ErrRecordNotFound), gorm.ErrRecordNotFound}),
		fmt.Errorf("get user fail: %w", gorm.Errors{fmt.Errorf("get user fail: %w", gorm.ErrRecordNotFound), fmt.Errorf("get user fail: %w", gorm.ErrRecordNotFound)}),
	}

	for idx, err := range errs {
		if !gorm.IsRecordNotFoundError(err) {
			t.Errorf("%s(%d) should be ErrRecordNotFound", err, idx)
		}
	}

	errs = []error{
		errors.New("err"),
		fmt.Errorf("get user fail: %s", gorm.ErrRecordNotFound),
		fmt.Errorf("get user fail: %v", gorm.ErrRecordNotFound),
		fmt.Errorf("get user fail: %+v", gorm.ErrRecordNotFound),
	}

	for idx, err := range errs {
		if gorm.IsRecordNotFoundError(err) {
			t.Errorf("%s(%d) should not be ErrRecordNotFound", err, idx)
		}
	}
}
