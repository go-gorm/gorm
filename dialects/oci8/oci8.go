// package oci8 implements a gorm dialect for oracle
package oci8

import (
	"github.com/jinzhu/gorm"
	// go-oci8 imported for the driver
	_ "github.com/mattn/go-oci8"
)

const dialectName = "oci8"

var _ gorm.Dialect = (*oci8)(nil)
var _ gorm.OraDialect = (*oci8)(nil)

type oci8 struct {
	db gorm.SQLCommon
	gorm.DefaultForeignKeyNamer
	gorm.OraCommon
}

func init() {
	gorm.RegisterDialect(dialectName, &oci8{})
}

// GetName returns the dialect name
func (oci8) GetName() string {
	return dialectName
}
