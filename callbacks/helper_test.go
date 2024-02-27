package callbacks

import (
	"reflect"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestLoadOrStoreVisitMap(t *testing.T) {
	var vm visitMap
	var loaded bool
	type testM struct {
		Name string
	}

	t1 := testM{Name: "t1"}
	t2 := testM{Name: "t2"}
	t3 := testM{Name: "t3"}

	vm = make(visitMap)
	if loaded = loadOrStoreVisitMap(&vm, reflect.ValueOf(&t1)); loaded {
		t.Fatalf("loaded should be false")
	}

	if loaded = loadOrStoreVisitMap(&vm, reflect.ValueOf(&t1)); !loaded {
		t.Fatalf("loaded should be true")
	}

	// t1 already exist but t2 not
	if loaded = loadOrStoreVisitMap(&vm, reflect.ValueOf([]*testM{&t1, &t2, &t3})); loaded {
		t.Fatalf("loaded should be false")
	}

	if loaded = loadOrStoreVisitMap(&vm, reflect.ValueOf([]*testM{&t2, &t3})); !loaded {
		t.Fatalf("loaded should be true")
	}
}

func TestConvertMapToValuesForCreate(t *testing.T) {
	testCase := []struct {
		name   string
		input  map[string]interface{}
		expect clause.Values
	}{
		{
			name: "Test convert string value",
			input: map[string]interface{}{
				"name": "my name",
			},
			expect: clause.Values{
				Columns: []clause.Column{{Name: "name"}},
				Values:  [][]interface{}{{"my name"}},
			},
		},
		{
			name: "Test convert int value",
			input: map[string]interface{}{
				"age": 18,
			},
			expect: clause.Values{
				Columns: []clause.Column{{Name: "age"}},
				Values:  [][]interface{}{{18}},
			},
		},
		{
			name: "Test convert float value",
			input: map[string]interface{}{
				"score": 99.5,
			},
			expect: clause.Values{
				Columns: []clause.Column{{Name: "score"}},
				Values:  [][]interface{}{{99.5}},
			},
		},
		{
			name: "Test convert bool value",
			input: map[string]interface{}{
				"active": true,
			},
			expect: clause.Values{
				Columns: []clause.Column{{Name: "active"}},
				Values:  [][]interface{}{{true}},
			},
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			actual := ConvertMapToValuesForCreate(&gorm.Statement{}, tc.input)
			if !reflect.DeepEqual(actual, tc.expect) {
				t.Errorf("expect %v got %v", tc.expect, actual)
			}
		})
	}
}

func TestConvertSliceOfMapToValuesForCreate(t *testing.T) {
	testCase := []struct {
		name   string
		input  []map[string]interface{}
		expect clause.Values
	}{
		{
			name: "Test convert slice of string value",
			input: []map[string]interface{}{
				{"name": "my name"},
			},
			expect: clause.Values{
				Columns: []clause.Column{{Name: "name"}},
				Values:  [][]interface{}{{"my name"}},
			},
		},
		{
			name: "Test convert slice of int value",
			input: []map[string]interface{}{
				{"age": 18},
			},
			expect: clause.Values{
				Columns: []clause.Column{{Name: "age"}},
				Values:  [][]interface{}{{18}},
			},
		},
		{
			name: "Test convert slice of float value",
			input: []map[string]interface{}{
				{"score": 99.5},
			},
			expect: clause.Values{
				Columns: []clause.Column{{Name: "score"}},
				Values:  [][]interface{}{{99.5}},
			},
		},
		{
			name: "Test convert slice of bool value",
			input: []map[string]interface{}{
				{"active": true},
			},
			expect: clause.Values{
				Columns: []clause.Column{{Name: "active"}},
				Values:  [][]interface{}{{true}},
			},
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			actual := ConvertSliceOfMapToValuesForCreate(&gorm.Statement{}, tc.input)

			if !reflect.DeepEqual(actual, tc.expect) {
				t.Errorf("expected %v but got %v", tc.expect, actual)
			}
		})
	}

}
