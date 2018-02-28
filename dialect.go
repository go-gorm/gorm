package gorm

// Dialect GORM dialect interface
type Dialect interface {
	Insert(*DB) error
	Query(*DB) error
	Update(*DB) error
	Delete(*DB) error

	Quote(string) string
}
