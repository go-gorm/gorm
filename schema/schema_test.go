package schema

import (
	"database/sql"
	"testing"
	"time"
)

type MyStruct struct {
	ID            int
	Int           uint
	IntPointer    *uint
	String        string
	StringPointer *string
	Time          time.Time
	TimePointer   *time.Time
	NullInt64     sql.NullInt64
}

type BelongsTo struct {
	ID   int
	Name string
}

type HasOne struct {
	ID         int
	MyStructID uint
}

type HasMany struct {
	ID         int
	MyStructID uint
	Name       string
}

type Many2Many struct {
	ID   int
	Name string
}

func TestParseSchema(t *testing.T) {
	ParseSchema(&MyStruct{})
}

func TestParseBelongsToRel(t *testing.T) {
	type MyStruct struct {
		ID        int
		Name      string
		BelongsTo BelongsTo
	}

	ParseSchema(&MyStruct{})
}

func TestParseHasOneRel(t *testing.T) {
	type MyStruct struct {
		ID     int
		Name   string
		HasOne HasOne
	}

	ParseSchema(&MyStruct{})
}

func TestParseHasManyRel(t *testing.T) {
	type MyStruct struct {
		ID      int
		Name    string
		HasMany []HasMany
	}

	ParseSchema(&MyStruct{})
}

func TestParseManyToManyRel(t *testing.T) {
	type MyStruct struct {
		ID      int
		Name    string
		HasMany []HasMany
	}

	ParseSchema(&MyStruct{})
}

func TestEmbeddedStruct(t *testing.T) {
}

func TestCustomizePrimaryKey(t *testing.T) {
}

func TestCompositePrimaryKeys(t *testing.T) {
}
