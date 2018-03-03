package sqlbuilder

import "bytes"

// Builder sql builder
type Builder struct {
	SQL  bytes.Buffer
	Args []interface{}
}
