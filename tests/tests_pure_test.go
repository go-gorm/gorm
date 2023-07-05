//go:build !cgo || (cgo && pure)
// +build !cgo cgo,pure

package tests_test

import (
	"os"
	"path/filepath"
)

var (
	sqliteDSN	= filepath.Join(os.TempDir(), "gorm.db?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
)
