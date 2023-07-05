//go:build cgo && !pure
// +build cgo,!pure

package tests_test

import (
	"os"
	"path/filepath"
)

var (
	sqliteDSN	= filepath.Join(os.TempDir(), "gorm.db")
)
