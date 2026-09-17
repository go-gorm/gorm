package schema

import (
	"sync"
	"testing"
)

type DialectTypeModel struct {
	ID           uint   `gorm:"primarykey"`
	Generic      string `gorm:"type:varchar(64)"`
	Override     string `gorm:"type:varchar(32);mysql:type:text;postgres:type:jsonb;sqlite:type:mediumtext"`
	PostgresOnly string `gorm:"postgresql:type:text[];mysql:type:json"`
}

type DialectTypeEmbeddedModel struct {
	ID uint `gorm:"primarykey"`
	Base
}

type Base struct {
	Content string `gorm:"type:varchar(16);mysql:type:longtext"`
}

// assertTagSettings checks that settings contains every key with the expected value.
func assertTagSettings(t *testing.T, settings map[string]string, want map[string]string) {
	t.Helper()

	for key, expected := range want {
		if v, ok := settings[key]; !ok || v != expected {
			t.Errorf("setting %s should be %q, got %#v", key, expected, v)
		}
	}
}

func TestParseTagSettingDialectType(t *testing.T) {
	settings := ParseTagSetting("type:varchar(32);mysql:type:json;postgres:type:text[];sqlite3:type:mediumtext;MSSQL:type:nvarchar(max)", ";")

	// the generic type tag is kept untouched, dialect-specific type tags are
	// stored as `<DIALECT>:TYPE`
	assertTagSettings(t, settings, map[string]string{
		"TYPE":          "varchar(32)",
		"MYSQL:TYPE":    "json",
		"POSTGRES:TYPE": "text[]",
		"SQLITE3:TYPE":  "mediumtext",
		"MSSQL:TYPE":    "nvarchar(max)",
	})

	// legacy plain settings are still stored for backward compatibility,
	// e.g. `index:type:fulltext` keeps working via settings["INDEX"]
	assertTagSettings(t, settings, map[string]string{
		"MYSQL":    "type:json",
		"POSTGRES": "type:text[]",
		"SQLITE3":  "type:mediumtext",
		"MSSQL":    "type:nvarchar(max)",
	})

	// types containing colons must not be treated as dialect tags
	settings = ParseTagSetting("type:enum('a:b')", ";")
	assertTagSettings(t, settings, map[string]string{"TYPE": "enum('a:b')"})
	if _, ok := settings["TYPE:TYPE"]; ok {
		t.Errorf("`type:enum('a:b')` should not produce a dialect-specific key")
	}
}

func TestDialectTypeSetting(t *testing.T) {
	settings := map[string]string{
		"TYPE":            "varchar(32)",
		"MYSQL:TYPE":      "text",
		"POSTGRESQL:TYPE": "jsonb",
		"SQLITE3:TYPE":    "mediumtext",
		"GAUSSDB:TYPE":    "text",
	}

	dialects := []struct {
		dialect  string
		expected string
		found    bool
	}{
		{dialect: "mysql", expected: "text", found: true},
		{dialect: "MySQL", expected: "text", found: true},             // case-insensitive
		{dialect: "mariadb", expected: "text", found: true},           // mariadb dialector accepts mysql tags
		{dialect: "tidb", expected: "text", found: true},              // tidb dialector accepts mysql tags
		{dialect: "postgres", expected: "jsonb", found: true},         // postgresql alias
		{dialect: "postgresql", expected: "jsonb", found: true},       // exact name match
		{dialect: "cockroachdb", expected: "jsonb", found: true},      // cockroachdb accepts postgres tags
		{dialect: "sqlite", expected: "mediumtext", found: true},      // sqlite3 alias
		{dialect: "sqlserver", expected: "varchar(32)", found: false}, // mssql tag not set here
		{dialect: "gaussdb", expected: "text", found: true},
		{dialect: "clickhouse", expected: "varchar(32)", found: false}, // unknown dialector falls back to generic
		{dialect: "", expected: "varchar(32)", found: false},           // no dialector, generic applies
	}

	for _, d := range dialects {
		if v, ok := dialectTypeSetting(d.dialect, settings); ok != d.found || (ok && v != d.expected) {
			t.Errorf("dialectTypeSetting(%q) = (%q, %v), want %q found=%v", d.dialect, v, ok, d.expected, d.found)
		}
	}

	// empty dialect-specific value is treated as absent
	if _, ok := dialectTypeSetting("mysql", map[string]string{"MYSQL:TYPE": ""}); ok {
		t.Errorf("empty dialect-specific type should be ignored")
	}
}

func TestFieldParseDialectType(t *testing.T) {
	// without a dialector, GORM's original rules apply
	schema, err := Parse(&DialectTypeModel{}, &sync.Map{}, NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	if dataType := schema.FieldsByDBName["generic"].DataType; dataType != "varchar(64)" {
		t.Errorf("generic field data type should be varchar(64), got %v", dataType)
	}
	if dataType := schema.FieldsByDBName["override"].DataType; dataType != "varchar(32)" {
		t.Errorf("override field data type should be varchar(32) without dialector, got %v", dataType)
	}
	if dataType := schema.FieldsByDBName["postgres_only"].DataType; dataType != String {
		t.Errorf("postgres_only field should fall back to the default string type, got %v", dataType)
	}

	// each dialector resolves its own type
	dialectDataTypes := map[string]map[string]DataType{
		"mysql":     {"generic": "varchar(64)", "override": "text", "postgres_only": "json"},
		"postgres":  {"generic": "varchar(64)", "override": "jsonb", "postgres_only": "text[]"},
		"sqlite":    {"generic": "varchar(64)", "override": "mediumtext", "postgres_only": String},
		"sqlserver": {"generic": "varchar(64)", "override": "varchar(32)", "postgres_only": String},
	}

	for dialect, expectations := range dialectDataTypes {
		schema, err := ParseWithSpecialTableNameAndDialect(&DialectTypeModel{}, &sync.Map{}, NamingStrategy{}, "", dialect)
		if err != nil {
			t.Fatal(err)
		}
		for column, expected := range expectations {
			if dataType := schema.FieldsByDBName[column].DataType; dataType != expected {
				t.Errorf("%s.%s data type should be %v, got %v", dialect, column, expected, dataType)
			}
		}
	}
}

// mustParseDialectSchema parses a model for the given dialector and fails the
// test on error.
func mustParseDialectSchema(t *testing.T, model interface{}, dialect string, cacheStore *sync.Map) *Schema {
	t.Helper()

	schema, err := ParseWithSpecialTableNameAndDialect(model, cacheStore, NamingStrategy{}, "", dialect)
	if err != nil {
		t.Fatal(err)
	}
	return schema
}

func TestFieldParseDialectTypeCacheIsolation(t *testing.T) {
	// one cache store shared by different dialects must not mix schemas
	cacheStore := &sync.Map{}

	mysqlSchema := mustParseDialectSchema(t, &DialectTypeModel{}, "mysql", cacheStore)
	postgresSchema := mustParseDialectSchema(t, &DialectTypeModel{}, "postgres", cacheStore)
	genericSchema, err := Parse(&DialectTypeModel{}, cacheStore, NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}

	expectations := []struct {
		schema *Schema
		column string
		want   DataType
	}{
		{schema: mysqlSchema, column: "override", want: "text"},
		{schema: postgresSchema, column: "override", want: "jsonb"},
		{schema: genericSchema, column: "override", want: "varchar(32)"},
	}
	for _, e := range expectations {
		if v := e.schema.FieldsByDBName[e.column].DataType; v != e.want {
			t.Errorf("override should be %v, got %v", e.want, v)
		}
	}

	if mysqlSchema == postgresSchema || mysqlSchema == genericSchema || postgresSchema == genericSchema {
		t.Errorf("schemas for different dialects should be cached separately")
	}

	// related parses through getOrParse hit the dialect-specific cache entry
	cached, err := getOrParse(&DialectTypeModel{}, cacheStore, NamingStrategy{}, "mysql")
	if err != nil {
		t.Fatal(err)
	}
	if cached != mysqlSchema {
		t.Errorf("getOrParse should reuse the mysql schema cache entry")
	}
}

func TestFieldParseDialectTypeEmbedded(t *testing.T) {
	schema, err := ParseWithSpecialTableNameAndDialect(&DialectTypeEmbeddedModel{}, &sync.Map{}, NamingStrategy{}, "", "mysql")
	if err != nil {
		t.Fatal(err)
	}
	if dataType := schema.FieldsByDBName["content"].DataType; dataType != "longtext" {
		t.Errorf("embedded field content should be longtext on mysql, got %v", dataType)
	}

	schema, err = ParseWithSpecialTableNameAndDialect(&DialectTypeEmbeddedModel{}, &sync.Map{}, NamingStrategy{}, "", "postgres")
	if err != nil {
		t.Fatal(err)
	}
	if dataType := schema.FieldsByDBName["content"].DataType; dataType != "varchar(16)" {
		t.Errorf("embedded field content should be varchar(16) on postgres, got %v", dataType)
	}
}
