package utils

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestIsInvalidDBNameChar(t *testing.T) {
	for _, db := range []string{"db", "dbName", "db_name", "db1", "1dbname", "db$name"} {
		if fields := strings.FieldsFunc(db, IsInvalidDBNameChar); len(fields) != 1 {
			t.Fatalf("failed to parse db name %v", db)
		}
	}
}

func TestCheckTruth(t *testing.T) {
	checkTruthTests := []struct {
		v   string
		out bool
	}{
		{"123", true},
		{"true", true},
		{"", false},
		{"false", false},
		{"False", false},
		{"FALSE", false},
		{"\u0046alse", false},
	}

	for _, test := range checkTruthTests {
		t.Run(test.v, func(t *testing.T) {
			if out := CheckTruth(test.v); out != test.out {
				t.Errorf("CheckTruth(%s) want: %t, got: %t", test.v, test.out, out)
			}
		})
	}
}

func TestToStringKey(t *testing.T) {
	cases := []struct {
		values []interface{}
		key    string
	}{
		{[]interface{}{"a"}, "a"},
		{[]interface{}{1, 2, 3}, "1_2_3"},
		{[]interface{}{1, nil, 3}, "1_nil_3"},
		{[]interface{}{[]interface{}{1, 2, 3}}, "[1 2 3]"},
		{[]interface{}{[]interface{}{"1", "2", "3"}}, "[1 2 3]"},
		{[]interface{}{[]interface{}{"1", nil, "3"}}, "[1 <nil> 3]"},
	}
	for _, c := range cases {
		if key := ToStringKey(c.values...); key != c.key {
			t.Errorf("%v: expected %v, got %v", c.values, c.key, key)
		}
	}
}

func TestContains(t *testing.T) {
	containsTests := []struct {
		name  string
		elems []string
		elem  string
		out   bool
	}{
		{"exists", []string{"1", "2", "3"}, "1", true},
		{"not exists", []string{"1", "2", "3"}, "4", false},
	}
	for _, test := range containsTests {
		t.Run(test.name, func(t *testing.T) {
			if out := Contains(test.elems, test.elem); test.out != out {
				t.Errorf("Contains(%v, %s) want: %t, got: %t", test.elems, test.elem, test.out, out)
			}
		})
	}
}

type ModifyAt sql.NullTime

// Value return a Unix time.
func (n ModifyAt) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Time.Unix(), nil
}

func TestAssertEqual(t *testing.T) {
	now := time.Now()
	assertEqualTests := []struct {
		name     string
		src, dst interface{}
		out      bool
	}{
		{"error equal", errors.New("1"), errors.New("1"), true},
		{"error not equal", errors.New("1"), errors.New("2"), false},
		{"driver.Valuer equal", ModifyAt{Time: now, Valid: true}, ModifyAt{Time: now, Valid: true}, true},
		{"driver.Valuer not equal", ModifyAt{Time: now, Valid: true}, ModifyAt{Time: now.Add(time.Second), Valid: true}, false},
		{"driver.Valuer equal (ptr to nil ptr)", (*ModifyAt)(nil), &ModifyAt{}, false},
	}
	for _, test := range assertEqualTests {
		t.Run(test.name, func(t *testing.T) {
			if out := AssertEqual(test.src, test.dst); test.out != out {
				t.Errorf("AssertEqual(%v, %v) want: %t, got: %t", test.src, test.dst, test.out, out)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		out  string
	}{
		{"int", math.MaxInt64, "9223372036854775807"},
		{"int8", int8(math.MaxInt8), "127"},
		{"int16", int16(math.MaxInt16), "32767"},
		{"int32", int32(math.MaxInt32), "2147483647"},
		{"int64", int64(math.MaxInt64), "9223372036854775807"},
		{"uint", uint(math.MaxUint64), "18446744073709551615"},
		{"uint8", uint8(math.MaxUint8), "255"},
		{"uint16", uint16(math.MaxUint16), "65535"},
		{"uint32", uint32(math.MaxUint32), "4294967295"},
		{"uint64", uint64(math.MaxUint64), "18446744073709551615"},
		{"string", "abc", "abc"},
		{"other", true, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if out := ToString(test.in); test.out != out {
				t.Fatalf("ToString(%v) want: %s, got: %s", test.in, test.out, out)
			}
		})
	}
}

func TestRTrimSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		trimLen  int
		expected []int
	}{
		{
			name:     "Trim two elements from end",
			input:    []int{1, 2, 3, 4, 5},
			trimLen:  2,
			expected: []int{1, 2, 3},
		},
		{
			name:     "Trim entire slice",
			input:    []int{1, 2, 3},
			trimLen:  3,
			expected: []int{},
		},
		{
			name:     "Trim length greater than slice length",
			input:    []int{1, 2, 3},
			trimLen:  5,
			expected: []int{},
		},
		{
			name:     "Zero trim length",
			input:    []int{1, 2, 3},
			trimLen:  0,
			expected: []int{1, 2, 3},
		},
		{
			name:     "Trim one element from end",
			input:    []int{1, 2, 3},
			trimLen:  1,
			expected: []int{1, 2},
		},
		{
			name:     "Empty slice",
			input:    []int{},
			trimLen:  2,
			expected: []int{},
		},
		{
			name:     "Negative trim length (should be treated as zero)",
			input:    []int{1, 2, 3},
			trimLen:  -1,
			expected: []int{1, 2, 3},
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			result := RTrimSlice(testcase.input, testcase.trimLen)
			if !AssertEqual(result, testcase.expected) {
				t.Errorf("RTrimSlice(%v, %d) = %v; want %v", testcase.input, testcase.trimLen, result, testcase.expected)
			}
		})
	}
}

//go:noinline
func fileWithLineNumWrappedInnerForTest() string {
	return FileWithLineNum()
}

//go:noinline
func fileWithLineNumWrappedOuterForTest() (got, want string) {
	_, file, line, _ := runtime.Caller(0)
	got = filepath.ToSlash(fileWithLineNumWrappedInnerForTest())
	want = fmt.Sprintf("%s:%d", filepath.ToSlash(file), line+1)
	return got, want
}

func TestFileWithLineNumWrappedCallRegression(t *testing.T) {
	got, want := fileWithLineNumWrappedOuterForTest()
	if got != want {
		t.Fatalf("FileWithLineNum wrapped caller mismatch, got %s, want %s", got, want)
	}
}

// toStringKeyReference is the original reflection-based ToStringKey, kept to
// verify the fast paths produce identical keys.
func toStringKeyReference(values ...interface{}) string {
	results := make([]string, len(values))

	for idx, value := range values {
		if valuer, ok := value.(driver.Valuer); ok {
			value, _ = valuer.Value()
		}

		switch v := value.(type) {
		case string:
			results[idx] = v
		case []byte:
			results[idx] = string(v)
		case uint:
			results[idx] = strconv.FormatUint(uint64(v), 10)
		default:
			results[idx] = "nil"
			vv := reflect.ValueOf(v)
			if vv.IsValid() && !vv.IsZero() {
				results[idx] = fmt.Sprint(reflect.Indirect(vv).Interface())
			}
		}
	}

	return strings.Join(results, "_")
}

func TestToStringKeyMatchesReference(t *testing.T) {
	i64, zero := int64(42), 0
	values := []interface{}{
		nil, "", "a", []byte("b"),
		[]byte{},
		0, 1, -1, math.MaxInt64, math.MinInt64,
		int8(0), int8(-8), int16(16), int32(-32), int64(0), int64(64),
		uint(0), uint(7), uint8(0), uint8(8), uint16(16), uint32(32), uint64(0), uint64(math.MaxUint64),
		1.5, float32(2.5), true, false,
		&i64, &zero, (*int)(nil),
		sql.NullInt64{Int64: 5, Valid: true},
		sql.NullInt64{},
		sql.NullString{String: "s", Valid: true},
		time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
		[]interface{}{1, "2", nil},
	}

	for _, v := range values {
		if got, want := ToStringKey(v), toStringKeyReference(v); got != want {
			t.Errorf("ToStringKey(%#v) = %q, want %q", v, got, want)
		}
	}
	for i := range values {
		pair := []interface{}{values[i], values[(i+1)%len(values)]}
		if got, want := ToStringKey(pair...), toStringKeyReference(pair...); got != want {
			t.Errorf("ToStringKey(%#v) = %q, want %q", pair, got, want)
		}
	}
	if got := ToStringKey(); got != "" {
		t.Errorf("ToStringKey() = %q, want empty", got)
	}
}

func BenchmarkToStringKey(b *testing.B) {
	cases := []struct {
		name   string
		values []interface{}
	}{
		{"int64", []interface{}{int64(123456)}},
		{"int", []interface{}{123456}},
		{"uint", []interface{}{uint(123456)}},
		{"string", []interface{}{"abc-123"}},
		{"composite_int64", []interface{}{int64(1), int64(2)}},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				ToStringKey(c.values...)
			}
		})
	}
}
