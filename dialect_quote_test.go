package gorm

import "testing"

func TestQuoteEscapesDelimiter(t *testing.T) {
	cases := []struct {
		dialect Dialect
		in      string
		want    string
	}{
		{&commonDialect{}, "name", `"name"`},
		{&commonDialect{}, `na"me`, `"na""me"`},
		{&commonDialect{}, `x" OR 1=1 --`, `"x"" OR 1=1 --"`},
		{&mysql{}, "name", "`name`"},
		{&mysql{}, "na`me", "`na``me`"},
		{&mysql{}, "x` OR 1=1 --", "`x`` OR 1=1 --`"},
		{&mssql{}, "name", "[name]"},
		{&mssql{}, "na]me", "[na]]me]"},
		{&mssql{}, "x] OR 1=1 --", "[x]] OR 1=1 --]"},
		{&postgres{}, `na"me`, `"na""me"`},
		{&sqlite3{}, `na"me`, `"na""me"`},
	}
	for _, c := range cases {
		if got := c.dialect.Quote(c.in); got != c.want {
			t.Errorf("%s.Quote(%q) = %q, want %q", c.dialect.GetName(), c.in, got, c.want)
		}
	}
}

func TestScopeQuoteEscapesEachSegment(t *testing.T) {
	db := &DB{parent: &DB{dialect: &commonDialect{}}}
	db.parent.parent = db.parent
	scope := db.NewScope(nil)
	if got, want := scope.Quote(`ta"ble.col"umn`), `"ta""ble"."col""umn"`; got != want {
		t.Errorf("scope.Quote = %q, want %q", got, want)
	}
}
