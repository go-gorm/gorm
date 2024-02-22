package callbacks

import (
	"reflect"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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
