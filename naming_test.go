package gorm_test

import (
	"bytes"
	"testing"

	"github.com/jinzhu/gorm"
)

func TestTheNamingStrategy(t *testing.T) {

	cases := []struct {
		name     string
		namer    gorm.Namer
		expected string
	}{
		{name: "auth", expected: "auth", namer: gorm.TheNamingStrategy.DB},
		{name: "userRestrictions", expected: "user_restrictions", namer: gorm.TheNamingStrategy.Table},
		{name: "clientID", expected: "client_id", namer: gorm.TheNamingStrategy.Column},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := c.namer(c.name)
			if result != c.expected {
				t.Errorf("error in naming strategy. expected: %v got :%v\n", c.expected, result)
			}
		})
	}

}

func TestAddNamingStrategy(t *testing.T) {

	// users can set their custom namer
	custom := &gorm.NamingStrategy{
		Column: CustomNamer,
	}
	gorm.AddNamingStrategy(custom)

	// test
	cases := []struct {
		name     string
		namer    gorm.Namer
		expected string
	}{
		{name: "auth", expected: "auth", namer: gorm.TheNamingStrategy.DB},
		{name: "userRestrictions", expected: "user_restrictions", namer: gorm.TheNamingStrategy.Table},

		{name: "clientID", expected: "clientID", namer: gorm.TheNamingStrategy.Column},
		{name: "Client0ID", expected: "client0ID", namer: gorm.TheNamingStrategy.Column},
		{name: "_Client_ID_", expected: "_Client_ID_", namer: gorm.TheNamingStrategy.Column},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := c.namer(c.name)
			if result != c.expected {
				t.Errorf("error in naming strategy. expected: %v got :%v\n", c.expected, result)
			}
		})
	}

}

func CustomNamer(name string) string {
	// set `smap` public access and users can use it to cache
	if v := gorm.Smap.Get(name); v != "" {
		return v
	}

	const (
		lower = false
		upper = true
	)

	var (
		value    = name
		buf      = bytes.NewBufferString("")
		currCase bool
	)

	for i, v := range value {
		currCase = bool(value[i] >= 'A' && value[i] <= 'Z')
		if i == 0 && currCase == upper {
			buf.WriteRune(v + 32)
		} else {
			buf.WriteRune(v)
		}
	}

	s := buf.String()
	gorm.Smap.Set(name, s)

	return s
}

func TestNamingStrategy(t *testing.T) {

	dbNameNS := func(name string) string {
		return "db_" + name
	}
	tableNameNS := func(name string) string {
		return "tbl_" + name
	}
	columnNameNS := func(name string) string {
		return "col_" + name
	}

	ns := &gorm.NamingStrategy{
		DB:     dbNameNS,
		Table:  tableNameNS,
		Column: columnNameNS,
	}

	cases := []struct {
		name     string
		namer    gorm.Namer
		expected string
	}{
		{name: "auth", expected: "db_auth", namer: ns.DB},
		{name: "user", expected: "tbl_user", namer: ns.Table},
		{name: "password", expected: "col_password", namer: ns.Column},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := c.namer(c.name)
			if result != c.expected {
				t.Errorf("error in naming strategy. expected: %v got :%v\n", c.expected, result)
			}
		})
	}

}
