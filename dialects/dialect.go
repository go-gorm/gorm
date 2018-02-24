package dialects

import (
	"github.com/jinzhu/gorm/builder"
)

// Dialect GORM dialect interface
type Dialect interface {
	Insert(*builder.Statement) error
	Query(*builder.Statement) error
	Update(*builder.Statement) error
	Delete(*builder.Statement) error
}
