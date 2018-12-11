package gorm

// Encoder is a value encoding interface for complex field types
type Encoder interface {
	EncodeField(column string) (interface{}, error)
	DecodeField(column string, value interface{}) error
}

// decoder defers decoding until necessary
type decoder struct {
	Encoder
	column string
	value  interface{}
}

func newDecoder(encoder Encoder, scope *Scope, column string) *decoder {
	return &decoder{
		encoder,
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
	return d.DecodeField(d.column, d.value)
}
