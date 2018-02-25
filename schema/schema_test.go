package schema

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
)

type MyStruct struct {
	ID   int
	Name string
}

type BelongTo struct {
}

func TestParseSchema(t *testing.T) {
	Schema := ParseSchema(&MyStruct{})
	spew.Dump(Schema)
}
