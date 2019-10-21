// +build go1.13

package gorm_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jinzhu/gorm"
)

func TestNotFound(t *testing.T) {
	type testcase struct {
		err                 error
		isErrRecordNotFound bool
	}
	var wrapErrRecordNotFound = fmt.Errorf("get user fail: %w", gorm.ErrRecordNotFound)
	var cases = []testcase{
		{gorm.ErrRecordNotFound, true},
		{gorm.Errors{gorm.ErrRecordNotFound}, true},
		{gorm.Errors{gorm.ErrRecordNotFound, gorm.ErrRecordNotFound}, true},

		{wrapErrRecordNotFound, true},
		{gorm.Errors{wrapErrRecordNotFound}, true},
		{gorm.Errors{gorm.ErrRecordNotFound, wrapErrRecordNotFound}, true},
		{gorm.Errors{wrapErrRecordNotFound, gorm.ErrRecordNotFound}, true},
		{gorm.Errors{wrapErrRecordNotFound, wrapErrRecordNotFound}, true},

		{fmt.Errorf("get user fail: %w", gorm.ErrRecordNotFound), true},
		{fmt.Errorf("get user fail: %w", wrapErrRecordNotFound), true},
		{fmt.Errorf("get user fail: %w", fmt.Errorf("get user fail: %w", wrapErrRecordNotFound)), true},

		{fmt.Errorf("get user fail: %w", gorm.Errors{gorm.ErrRecordNotFound}), true},
		{fmt.Errorf("get user fail: %w", gorm.Errors{wrapErrRecordNotFound}), true},
		{fmt.Errorf("get user fail: %w", gorm.Errors{gorm.ErrRecordNotFound, gorm.ErrRecordNotFound}), true},
		{fmt.Errorf("get user fail: %w", gorm.Errors{wrapErrRecordNotFound, wrapErrRecordNotFound}), true},
		{fmt.Errorf("get user fail: %w", gorm.Errors{wrapErrRecordNotFound, gorm.ErrRecordNotFound}), true},
		{fmt.Errorf("get user fail: %w", gorm.Errors{gorm.ErrRecordNotFound, wrapErrRecordNotFound}), true},
		{fmt.Errorf("get user fail: %w", fmt.Errorf("get user fail: %w", gorm.Errors{gorm.ErrRecordNotFound, wrapErrRecordNotFound})), true},

		{errors.New("err"), false},
		{fmt.Errorf("get user fail: %s", gorm.ErrRecordNotFound), false},
		{fmt.Errorf("get user fail: %v", gorm.ErrRecordNotFound), false},
		{fmt.Errorf("get user fail: %+v", gorm.ErrRecordNotFound), false},
	}
	for idx, err := range cases {
		isRecordNotFoundError := gorm.IsRecordNotFoundError(err.err)
		if isRecordNotFoundError != err.isErrRecordNotFound {
			t.Errorf("err: %s(%d) should be ErrRecordNotFound: %v, but got: %v", err.err, idx, err.isErrRecordNotFound, isRecordNotFoundError)
		}
	}
}
