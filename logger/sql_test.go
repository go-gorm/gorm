package logger_test

import (
	"regexp"
	"testing"

	"github.com/jinzhu/gorm/logger"
	"github.com/jinzhu/now"
)

func TestExplainSQL(t *testing.T) {
	type role string
	type password []byte
	var (
		tt     = now.MustParse("2020-02-23 11:10:10")
		myrole = role("admin")
		pwd    = password([]byte("pass"))
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
			Vars:          []interface{}{"jinzhu", 1, 999.99, true, []byte("12345"), tt, &tt, nil, "w@g.com", myrole, pwd},
			Result:        `create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass) values ("jinzhu", 1, 999.990000, true, "12345", "2020-02-23 11:10:10", "2020-02-23 11:10:10", NULL, "w@g.com", "admin", "pass")`,
		},
		{
			SQL:           "create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass) values (@p0, @p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9, @p10)",
			NumericRegexp: regexp.MustCompile("@p(\\d+)"),
			Vars:          []interface{}{"jinzhu", 1, 999.99, true, []byte("12345"), tt, &tt, nil, "w@g.com", myrole, pwd},
			Result:        `create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass) values ("jinzhu", 1, 999.990000, true, "12345", "2020-02-23 11:10:10", "2020-02-23 11:10:10", NULL, "w@g.com", "admin", "pass")`,
		},
		{
			SQL:           "create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass) values ($2, $3, $0, $1, $6, $7, $4, $5, $8, $9, $10)",
			NumericRegexp: regexp.MustCompile("\\$(\\d+)"),
			Vars:          []interface{}{999.99, true, "jinzhu", 1, &tt, nil, []byte("12345"), tt, "w@g.com", myrole, pwd},
			Result:        `create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass) values ("jinzhu", 1, 999.990000, true, "12345", "2020-02-23 11:10:10", "2020-02-23 11:10:10", NULL, "w@g.com", "admin", "pass")`,
		},
		{
			SQL:           "create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass) values (@p0, @p10, @p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9)",
			NumericRegexp: regexp.MustCompile("@p(\\d+)"),
			Vars:          []interface{}{"jinzhu", 999.99, true, []byte("12345"), tt, &tt, nil, "w@g.com", myrole, pwd, 1},
			Result:        `create table users (name, age, height, actived, bytes, create_at, update_at, deleted_at, email, role, pass) values ("jinzhu", 1, 999.990000, true, "12345", "2020-02-23 11:10:10", "2020-02-23 11:10:10", NULL, "w@g.com", "admin", "pass")`,
		},
	}

	for idx, r := range results {
		if result := logger.ExplainSQL(r.SQL, r.NumericRegexp, `"`, r.Vars...); result != r.Result {
			t.Errorf("Explain SQL #%v expects %v, but got %v", idx, r.Result, result)
		}
	}
}
