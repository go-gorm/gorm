package logger_test

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/jinzhu/now"
	"gorm.io/gorm/logger"
)

type JSON json.RawMessage

func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.RawMessage(j).MarshalJSON()
}

type ExampleStruct struct {
	Name string
	Val  string
}

func (s ExampleStruct) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func format(v []byte, escaper string) string {
	return escaper + strings.ReplaceAll(string(v), escaper, escaper+escaper) + escaper
}

type (
	namedFloat32    float32
	namedFloat64    float64
	stringerFloat32 float32
	stringerFloat64 float64
)

func (stringerFloat32) String() string { return "not a SQL number" }
func (stringerFloat64) String() string { return "not a SQL number" }

func TestExplainSQLNamedFloatPrecision(t *testing.T) {
	for _, value := range []interface{}{
		namedFloat32(0.0000001), namedFloat64(0.0000001),
		stringerFloat32(0.0000001), stringerFloat64(0.0000001),
	} {
		if got := logger.ExplainSQL("SELECT ?", nil, "'", value); got != "SELECT 0.0000001" {
			t.Errorf("%T: got %q, want SELECT 0.0000001", value, got)
		}
	}
	for _, value := range []interface{}{
		namedFloat32(1.23), namedFloat64(1.23),
		stringerFloat32(1.23), stringerFloat64(1.23),
	} {
		if got := logger.ExplainSQL("SELECT ?", nil, "'", value); got != "SELECT 1.23" {
			t.Errorf("%T: got %q, want SELECT 1.23", value, got)
		}
	}
}

func TestExplainSQL(t *testing.T) {
	type role string
	type password []byte
	type intType int
	type floatType float64
	var (
		tt                 = now.MustParse("2020-02-23 11:10:10")
		myrole             = role("admin")
		pwd                = password("pass")
		jsVal              = []byte(`{"Name":"test","Val":"test"}`)
		js                 = JSON(jsVal)
		esVal              = []byte(`{"Name":"test","Val":"test"}`)
		es                 = ExampleStruct{Name: "test", Val: "test"}
		intVal   intType   = 1
		floatVal floatType = 1.23
	)

	results := []struct {
		SQL           string
		NumericRegexp *regexp.Regexp
		Vars          []interface{}
		Result        string
	}{
		{
			SQL:           "create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			NumericRegexp: nil,
			Vars:          []interface{}{"jinzhu", 1, 999.99, true, []byte("12345"), tt, &tt, nil, "w@g.\"com", myrole, pwd},
			Result:        `create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass) values ("jinzhu", 1, 999.99, true, "12345", "2020-02-23 11:10:10", "2020-02-23 11:10:10", NULL, "w@g.""com", "admin", "pass")`,
		},
		{
			SQL:           "create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			NumericRegexp: nil,
			Vars:          []interface{}{"jinzhu?", 1, 999.99, true, []byte("12345"), tt, &tt, nil, "w@g.\"com", myrole, pwd},
			Result:        `create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass) values ("jinzhu?", 1, 999.99, true, "12345", "2020-02-23 11:10:10", "2020-02-23 11:10:10", NULL, "w@g.""com", "admin", "pass")`,
		},
		{
			SQL:           "create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass) values (@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9, @p10, @p11)",
			NumericRegexp: regexp.MustCompile(`@p(\d+)`),
			Vars:          []interface{}{"jinzhu", 1, 999.99, true, []byte("12345"), tt, &tt, nil, "w@g.com", myrole, pwd},
			Result:        `create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass) values ("jinzhu", 1, 999.99, true, "12345", "2020-02-23 11:10:10", "2020-02-23 11:10:10", NULL, "w@g.com", "admin", "pass")`,
		},
		{
			SQL:           "create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass) values ($3, $4, $1, $2, $7, $8, $5, $6, $9, $10, $11)",
			NumericRegexp: regexp.MustCompile(`\$(\d+)`),
			Vars:          []interface{}{999.99, true, "jinzhu", 1, &tt, nil, []byte("12345"), tt, "w@g.com", myrole, pwd},
			Result:        `create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass) values ("jinzhu", 1, 999.99, true, "12345", "2020-02-23 11:10:10", "2020-02-23 11:10:10", NULL, "w@g.com", "admin", "pass")`,
		},
		{
			SQL:           "create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass) values (@p1, @p11, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9, @p10)",
			NumericRegexp: regexp.MustCompile(`@p(\d+)`),
			Vars:          []interface{}{"jinzhu", 999.99, true, []byte("12345"), tt, &tt, nil, "w@g.com", myrole, pwd, 1},
			Result:        `create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass) values ("jinzhu", 1, 999.99, true, "12345", "2020-02-23 11:10:10", "2020-02-23 11:10:10", NULL, "w@g.com", "admin", "pass")`,
		},
		{
			SQL:           "create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass, json_struct, example_struct) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			NumericRegexp: nil,
			Vars:          []interface{}{"jinzhu", 1, 999.99, true, []byte("12345"), tt, &tt, nil, "w@g.\"com", myrole, pwd, js, es},
			Result:        fmt.Sprintf(`create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass, json_struct, example_struct) values ("jinzhu", 1, 999.99, true, "12345", "2020-02-23 11:10:10", "2020-02-23 11:10:10", NULL, "w@g.""com", "admin", "pass", %v, %v)`, format(jsVal, `"`), format(esVal, `"`)),
		},
		{
			SQL:           "create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass, json_struct, example_struct) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			NumericRegexp: nil,
			Vars:          []interface{}{"jinzhu", 1, 999.99, true, []byte("12345"), tt, &tt, nil, "w@g.\"com", myrole, pwd, &js, &es},
			Result:        fmt.Sprintf(`create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass, json_struct, example_struct) values ("jinzhu", 1, 999.99, true, "12345", "2020-02-23 11:10:10", "2020-02-23 11:10:10", NULL, "w@g.""com", "admin", "pass", %v, %v)`, format(jsVal, `"`), format(esVal, `"`)),
		},
		{
			SQL:           "create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass, json_struct, example_struct) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			NumericRegexp: nil,
			Vars:          []interface{}{"jinzhu", 1, 0.1753607109, true, []byte("12345"), tt, &tt, nil, "w@g.\"com", myrole, pwd, &js, &es},
			Result:        fmt.Sprintf(`create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass, json_struct, example_struct) values ("jinzhu", 1, 0.1753607109, true, "12345", "2020-02-23 11:10:10", "2020-02-23 11:10:10", NULL, "w@g.""com", "admin", "pass", %v, %v)`, format(jsVal, `"`), format(esVal, `"`)),
		},
		{
			SQL:           "create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass, json_struct, example_struct) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			NumericRegexp: nil,
			Vars:          []interface{}{"jinzhu", 1, float32(999.99), true, []byte("12345"), tt, &tt, nil, "w@g.\"com", myrole, pwd, &js, &es},
			Result:        fmt.Sprintf(`create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass, json_struct, example_struct) values ("jinzhu", 1, 999.99, true, "12345", "2020-02-23 11:10:10", "2020-02-23 11:10:10", NULL, "w@g.""com", "admin", "pass", %v, %v)`, format(jsVal, `"`), format(esVal, `"`)),
		},
		{
			SQL:           "create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass, int_val) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			NumericRegexp: nil,
			Vars:          []interface{}{"jinzhu?", 1, 999.99, true, []byte("12345"), tt, &tt, nil, "w@g.\"com", myrole, pwd, intVal},
			Result:        `create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass, int_val) values ("jinzhu?", 1, 999.99, true, "12345", "2020-02-23 11:10:10", "2020-02-23 11:10:10", NULL, "w@g.""com", "admin", "pass", 1)`,
		},
		{
			SQL:           "create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass, float_val) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			NumericRegexp: nil,
			Vars:          []interface{}{"jinzhu?", 1, 999.99, true, []byte("12345"), tt, &tt, nil, "w@g.\"com", myrole, pwd, floatVal},
			Result:        `create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass, float_val) values ("jinzhu?", 1, 999.99, true, "12345", "2020-02-23 11:10:10", "2020-02-23 11:10:10", NULL, "w@g.""com", "admin", "pass", 1.23)`,
		},
	}

	for idx, r := range results {
		if result := logger.ExplainSQL(r.SQL, r.NumericRegexp, `"`, r.Vars...); result != r.Result {
			t.Errorf("Explain SQL #%v expects %v, but got %v", idx, r.Result, result)
		}
	}
}
