package schema

import "testing"

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

func TestBelongsToRel(t *testing.T) {
	type MyStruct struct {
		ID        int
		Name      string
		BelongsTo BelongsTo
	}

	Parse(&MyStruct{})
}

func TestHasOneRel(t *testing.T) {
	type MyStruct struct {
		ID     int
		Name   string
		HasOne HasOne
	}

	Parse(&MyStruct{})
}

func TestHasManyRel(t *testing.T) {
	type MyStruct struct {
		ID      int
		Name    string
		HasMany []HasMany
	}

	Parse(&MyStruct{})
}

func TestManyToManyRel(t *testing.T) {
	type MyStruct struct {
		ID      int
		Name    string
		HasMany []HasMany
	}

	Parse(&MyStruct{})
}

