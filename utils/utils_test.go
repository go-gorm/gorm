package utils

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

func TestIsValidDBNameChar(t *testing.T) {
	for _, db := range []string{"db", "dbName", "db_name", "db1", "1dbname", "db$name"} {
		if fields := strings.FieldsFunc(db, IsValidDBNameChar); len(fields) != 1 {
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

func TestMaxInt(t *testing.T) {

	type testVal struct {
		n1, n2 int
	}

	integerSet := []int{100, 10, 0, -10, -100} // test set in desc order
	samples := []testVal{}

	for _, i := range integerSet {
		for _, j := range integerSet {
			samples = append(samples, testVal{n1: i, n2: j})
		}
	}

	for _, sample := range samples {
		t.Run("", func(t *testing.T) {
			result := MaxInt(sample.n1, sample.n2)
			if !(result >= sample.n1 && result >= sample.n2) {
				t.Fatalf("For n1=%d and n2=%d, result is %d;", sample.n1, sample.n2, result)
			}
		})
	}
}

func TestMinInt(t *testing.T) {

	type testVal struct {
		n1, n2 int
	}

	integerSet := []int{100, 10, 0, -10, -100} // test set in desc order
	samples := []testVal{}

	for _, i := range integerSet {
		for _, j := range integerSet {
			samples = append(samples, testVal{n1: i, n2: j})
		}
	}

	for _, sample := range samples {
		t.Run("", func(t *testing.T) {
			result := MinInt(sample.n1, sample.n2)
			if !(result <= sample.n1 && result <= sample.n2) {
				t.Fatalf("For n1=%d and n2=%d, result is %d;", sample.n1, sample.n2, result)
			}
		})
	}
}

func TestRTrimSlice(t *testing.T) {
	samples := []struct {
		input    []int
		trimLen  int
		expected []int
	}{
		{[]int{1, 2, 3, 4, 5}, 3, []int{1, 2, 3}},
		{[]int{1, 2, 3, 4, 5}, 0, []int{}},
		{[]int{1, 2, 3, 4, 5}, 5, []int{1, 2, 3, 4, 5}},
		{[]int{1, 2, 3, 4, 5}, 10, []int{1, 2, 3, 4, 5}}, // trimLen greater than slice length
		{[]int{1, 2, 3, 4, 5}, -1, []int{}},              // negative trimLen
		{[]int{}, 3, []int{}},                            // empty slice
		{[]int{1, 2, 3}, 1, []int{1}},                    // trim to a single element
	}

	for _, sample := range samples {
		t.Run("", func(t *testing.T) {
			result := RTrimSlice(sample.input, sample.trimLen)
			if !AssertEqual(result, sample.expected) {
				t.Errorf("Triming %v by length %d gives %v but want %v", sample.input, sample.trimLen, result, sample.expected)
			}
		})
	}
}
