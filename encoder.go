package gorm

// Encoder is a value encoding interface for complex field types
type Encoder interface {
	EncodeField(*Scope, string) (interface{}, error)
	DecodeField(scope *Scope, column string, value interface{}) error
}

// decoder defers decoding until necessary
type decoder struct {
	Encoder
	scope  *Scope
	column string
	value  interface{}
}

func newDecoder(encoder Encoder, scope *Scope, column string) *decoder {
	return &decoder{
		encoder,
		scope,
		column,
		nil,
	}
}

// Scan implements the sql.Scanner interface
func (d *decoder) Scan(src interface{}) error {
	d.value = src
	return nil
}

// Decode handles the decoding at a later time
func (d *decoder) Decode() error {
	return d.DecodeField(d.scope, d.column, d.value)
}
