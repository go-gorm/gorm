package schema_test

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestIndexFunctionTag(t *testing.T) {
	// Define a test struct with FUNCTION tag
	type User struct {
		ID    uint
		Name  string  `gorm:"index:,function:upper"`
		Age   int     `gorm:"index:,function:coalesce:-1"`
		Score float64 `gorm:"index:,function:round:2"`
		Nick  string  `gorm:"index:,function:concat:prefix:_suffix"`
		Email string  `gorm:"index:,function:myfunc:param1::param3"`
	}

	// Parse the schema with a valid cacheStore
	cacheStore := &sync.Map{}
	userSchema, err := schema.Parse(&User{}, cacheStore, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	// Get the indexes
	indexes := userSchema.ParseIndexes()

	// Check if indexes are generated correctly
	if len(indexes) != 5 {
		t.Fatalf("Expected 5 indexes, got %d", len(indexes))
	}

	// Test cases
	testCases := []struct {
		name     string
		indexIdx int
		expected string
	}{
		{"single parameter function - upper(name)", 0, "upper(name)"},
		{"two parameter function - coalesce(age, -1)", 1, "coalesce(age, -1)"},
		{"two parameter function - round(score, 2)", 2, "round(score, 2)"},
		{"multiple parameter function - concat(nick, prefix, _suffix)", 3, "concat(nick, prefix, _suffix)"},
		{"function with empty parameters - myfunc(email, param1, param3)", 4, "myfunc(email, param1, param3)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			index := indexes[tc.indexIdx]
			if len(index.Fields) != 1 {
				t.Fatalf("Expected 1 field in index, got %d", len(index.Fields))
			}
			field := index.Fields[0]
			if field.Expression != tc.expected {
				t.Errorf("Expected expression '%s', got '%s'", tc.expected, field.Expression)
			}
		})
	}
}

func TestIndexFunctionTagWithExpression(t *testing.T) {
	// Define a test struct with both FUNCTION and EXPRESSION tags
	// EXPRESSION should take precedence over FUNCTION
	type User struct {
		Name string `gorm:"index:,function:upper,expression:custom_expr(name)"`
	}

	cacheStore := &sync.Map{}
	userSchema, err := schema.Parse(&User{}, cacheStore, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	indexes := userSchema.ParseIndexes()

	if len(indexes) != 1 {
		t.Fatalf("Expected 1 index, got %d", len(indexes))
	}

	nameField := indexes[0].Fields[0]
	// EXPRESSION should be used when both FUNCTION and EXPRESSION are specified
	if nameField.Expression != "custom_expr(name)" {
		t.Errorf("Expected expression 'custom_expr(name)' (from EXPRESSION tag), got '%s'", nameField.Expression)
	}
}
