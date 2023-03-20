package callbacks

import (
	"reflect"
	"testing"
)

type unsupportedMockStruct struct {
	ExportedField        string
	unexportedField      string
	ExportedSliceField   []string
	unexportedSliceField []string
	ExportedMapField     map[string]string
	unexportedMapField   map[string]string
}

type supportedMockStruct struct {
	ExportedField      string
	ExportedSliceField []string
	ExportedMapField   map[string]string
}

func TestDeepCopy(t *testing.T) {
	t.Run("struct", func(t *testing.T) {
		t.Run("supported", func(t *testing.T) {
			srcStruct := supportedMockStruct{
				ExportedField:      "exported field",
				ExportedSliceField: []string{"1st elem of an exported slice field", "2nd elem of an exported slice field"},
				ExportedMapField: map[string]string{
					"key1": "exported map elem",
					"key2": "exported map elem",
				},
			}
			dstStruct := supportedMockStruct{}

			if err := deepCopy(srcStruct, &dstStruct); err != nil {
				t.Errorf("deepCopy returned an unexpected error %+v", err)
			}

			if !reflect.DeepEqual(srcStruct, dstStruct) {
				t.Errorf("deepCopy failed to copy structure: got %+v, want %+v", dstStruct, srcStruct)
			}
		})
		t.Run("unsupported", func(t *testing.T) {
			srcStruct := unsupportedMockStruct{
				ExportedField:        "exported field",
				unexportedField:      "unexported field",
				ExportedSliceField:   []string{"1st elem of an exported slice field", "2nd elem of an exported slice field"},
				unexportedSliceField: []string{"1st elem of an unexported slice field", "2nd elem of an unexported slice field"},
				ExportedMapField: map[string]string{
					"key1": "exported map elem",
					"key2": "exported map elem",
				},
				unexportedMapField: map[string]string{
					"key1": "unexported map elem",
					"key2": "unexported map elem",
				},
			}
			dstStruct := unsupportedMockStruct{}

			if err := deepCopy(srcStruct, &dstStruct); err == nil {
				t.Error("deepCopy was expected to fail copying an structure with unexported fields")
			}
		})
	})

	t.Run("map", func(t *testing.T) {
		t.Run("map[string]string", func(t *testing.T) {
			srcMap := map[string]string{
				"key1": "value1",
				"key2": "value2",
			}
			dstMap := make(map[string]string)

			if err := deepCopy(srcMap, &dstMap); err != nil {
				t.Errorf("deepCopy returned an unexpected error %+v", err)
			}

			if !reflect.DeepEqual(srcMap, dstMap) {
				t.Errorf("deepCopy failed to copy map: got %+v, want %+v", dstMap, srcMap)
			}
		})

		t.Run("map[string]struct", func(t *testing.T) {
			srcMap := map[string]supportedMockStruct{
				"key1": {
					ExportedField:      "exported field",
					ExportedSliceField: []string{"1st elem of an exported slice field", "2nd elem of an exported slice field"},
					ExportedMapField: map[string]string{
						"key1": "exported map elem",
						"key2": "exported map elem",
					},
				},
				"key2": {
					ExportedField:      "exported field",
					ExportedSliceField: []string{"1st elem of an exported slice field", "2nd elem of an exported slice field"},
					ExportedMapField: map[string]string{
						"key1": "exported map elem",
						"key2": "exported map elem",
					},
				},
			}
			dstMap := make(map[string]supportedMockStruct)

			if err := deepCopy(srcMap, &dstMap); err != nil {
				t.Errorf("deepCopy returned an unexpected error %+v", err)
			}

			if !reflect.DeepEqual(srcMap, dstMap) {
				t.Errorf("deepCopy failed to copy map: got %+v, want %+v", dstMap, srcMap)
			}
		})
	})

	t.Run("slice", func(t *testing.T) {
		t.Run("[]string", func(t *testing.T) {
			srcSlice := []string{"A", "B", "C"}
			dstSlice := make([]string, len(srcSlice))

			if err := deepCopy(srcSlice, &dstSlice); err != nil {
				t.Errorf("deepCopy returned an unexpected error %+v", err)
			}

			if !reflect.DeepEqual(srcSlice, dstSlice) {
				t.Errorf("deepCopy failed to copy slice: got %+v, want %+v", dstSlice, srcSlice)
			}
		})
		t.Run("[]struct", func(t *testing.T) {
			srcSlice := []supportedMockStruct{
				{
					ExportedField:      "exported field",
					ExportedSliceField: []string{"1st elem of an exported slice field", "2nd elem of an exported slice field"},
					ExportedMapField: map[string]string{
						"key1": "exported map elem",
						"key2": "exported map elem",
					},
				}, {
					ExportedField:      "exported field",
					ExportedSliceField: []string{"1st elem of an exported slice field", "2nd elem of an exported slice field"},
					ExportedMapField: map[string]string{
						"key1": "exported map elem",
						"key2": "exported map elem",
					},
				}, {
					ExportedField:      "exported field",
					ExportedSliceField: []string{"1st elem of an exported slice field", "2nd elem of an exported slice field"},
					ExportedMapField: map[string]string{
						"key1": "exported map elem",
						"key2": "exported map elem",
					},
				},
			}
			dstSlice := make([]supportedMockStruct, len(srcSlice))

			if err := deepCopy(srcSlice, &dstSlice); err != nil {
				t.Errorf("deepCopy returned an unexpected error %+v", err)
			}

			if !reflect.DeepEqual(srcSlice, dstSlice) {
				t.Errorf("deepCopy failed to copy slice: got %+v, want %+v", dstSlice, srcSlice)
			}
		})
	})

	t.Run("pointer", func(t *testing.T) {
		srcStruct := &supportedMockStruct{
			ExportedField:      "exported field",
			ExportedSliceField: []string{"1st elem of an exported slice field", "2nd elem of an exported slice field"},
			ExportedMapField: map[string]string{
				"key1": "exported map elem",
				"key2": "exported map elem",
			},
		}
		dstStruct := &supportedMockStruct{}

		if err := deepCopy(srcStruct, dstStruct); err != nil {
			t.Errorf("deepCopy returned an unexpected error %+v", err)
		}

		if !reflect.DeepEqual(srcStruct, dstStruct) {
			t.Errorf("deepCopy failed to copy structure: got %+v, want %+v", dstStruct, srcStruct)
		}
	})

	t.Run("mismatched", func(t *testing.T) {
		src := "a string"
		dst := 123

		if err := deepCopy(src, &dst); err == nil {
			t.Error("deepCopy did not return an error when provided mismatched types")
		}
	})
}
