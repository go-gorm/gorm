// +build go1.13

package gorm_test

import (
	"fmt"
	"testing"

	"github.com/jinzhu/gorm"
)

func TestNotFound(t *testing.T) {
	var user User
	err := DB.Where("name = ?", "not found").First(&user).Error
	if err != gorm.ErrRecordNotFound {
		t.Error("should not found")
	}

	err = fmt.Errorf("get user fail: %w", err)
	if !gorm.IsRecordNotFoundError(err) {
		t.Errorf("%s should IsRecordNotFoundError", err)
	}
}
