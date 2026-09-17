package tests_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"gorm.io/gorm"
)

// covers the database-specific type tag rules:
//   - no type tag at all            -> GORM's default data type rules
//   - generic `type` only           -> applies to every dialect
//   - `<dialect>:type:` only        -> that dialect uses it, others fall back to GORM defaults
//   - generic `type` + `<dialect>:type:` -> that dialect uses the specific one, others use the generic one
type DialectTypeColumnStruct struct {
	gorm.Model
	Plain         string `gorm:""`
	Generic       string `gorm:"type:varchar(64)"`
	Override      string `gorm:"type:varchar(32);mysql:type:text;postgres:type:jsonb"`
	MysqlOnly     string `gorm:"mysql:type:varchar(10)"`
	PostgresAlias string `gorm:"postgresql:type:text"`
}

func TestDialectSpecificTypeTags(t *testing.T) {
	DB.Migrator().DropTable(&DialectTypeColumnStruct{})
	if err := DB.AutoMigrate(&DialectTypeColumnStruct{}); err != nil {
		t.Fatalf("failed to migrate, got error: %v", err)
	}

	columnTypes, err := DB.Migrator().ColumnTypes(&DialectTypeColumnStruct{})
	if err != nil {
		t.Fatalf("failed to get column types, got error: %v", err)
	}

	columnTypeMap := map[string]gorm.ColumnType{}
	for _, ct := range columnTypes {
		columnTypeMap[ct.Name()] = ct
	}

	// expected column data types per dialect, `length` is only checked when set
	type columnExpectation struct {
		dataType string
		length   int64
	}
	expectedTypes := map[string]map[string]columnExpectation{
		// mysql & tidb (tidb runs the mysql driver, so the dialector name is `mysql`)
		"mysql": {
			"plain":          {dataType: "longtext"},            // no type tag -> GORM default
			"generic":        {dataType: "varchar", length: 64}, // generic type tag
			"override":       {dataType: "text"},                // mysql:type wins over generic type
			"mysql_only":     {dataType: "varchar", length: 10}, // mysql-specific type tag
			"postgres_alias": {dataType: "longtext"},            // postgresql tag ignored on mysql -> GORM default
		},
		"postgres": {
			"plain":          {dataType: "text"},
			"generic":        {dataType: "varchar", length: 64},
			"override":       {dataType: "jsonb"}, // postgres:type wins over generic type
			"mysql_only":     {dataType: "text"},  // mysql tag ignored on postgres -> GORM default
			"postgres_alias": {dataType: "text"},  // `postgresql` alias resolves for the postgres dialector
		},
		// gaussdb is postgres-compatible, the postgres alias applies as well
		"gaussdb": {
			"plain":          {dataType: "text"},
			"generic":        {dataType: "varchar", length: 64},
			"override":       {dataType: "jsonb"},
			"mysql_only":     {dataType: "text"},
			"postgres_alias": {dataType: "text"},
		},
		"sqlserver": {
			"generic":  {dataType: "varchar", length: 64}, // generic type tag
			"override": {dataType: "varchar", length: 32}, // no mssql tag -> generic type applies
		},
	}

	dialect := DB.Dialector.Name()
	if expectations, ok := expectedTypes[dialect]; ok {
		for column, expected := range expectations {
			ct, ok := columnTypeMap[column]
			if !ok {
				t.Fatalf("column %s not found", column)
			}

			if dataType := strings.ToLower(ct.DatabaseTypeName()); dataType != expected.dataType {
				t.Errorf("%s: column %s data type should be %s, got %s", dialect, column, expected.dataType, dataType)
			}

			if expected.length > 0 {
				if length, ok := ct.Length(); !ok || length != expected.length {
					t.Errorf("%s: column %s length should be %d, got %d (%v)", dialect, column, expected.length, length, ok)
				}
			}
		}
	} else if dialect != "sqlite" {
		t.Logf("skip column type assertions for dialect %s", dialect)
	}

	// CRUD round-trip on the typed columns
	record := DialectTypeColumnStruct{
		Plain:         "plain value",
		Generic:       "generic value",
		Override:      `{"name":"gorm"}`, // valid JSON, required by the postgres jsonb column
		MysqlOnly:     "m",
		PostgresAlias: "alias value",
	}
	if err := DB.Create(&record).Error; err != nil {
		t.Fatalf("failed to create record, got error: %v", err)
	}

	var got DialectTypeColumnStruct
	if err := DB.First(&got, record.ID).Error; err != nil {
		t.Fatalf("failed to query record, got error: %v", err)
	}
	if got.Plain != "plain value" || got.Generic != "generic value" || !jsonEqual(t, got.Override, `{"name":"gorm"}`) ||
		got.MysqlOnly != "m" || got.PostgresAlias != "alias value" {
		t.Errorf("record round-trip mismatch, got %+v", got)
	}

	if err := DB.Model(&got).Update("override", `{"name":"jinzhu"}`).Error; err != nil {
		t.Fatalf("failed to update record, got error: %v", err)
	}
	var override string
	if err := DB.Model(&DialectTypeColumnStruct{}).Where("id = ?", got.ID).Pluck("override", &override).Error; err != nil {
		t.Fatalf("failed to pluck override, got error: %v", err)
	}
	if !jsonEqual(t, override, `{"name":"jinzhu"}`) {
		t.Errorf("override should be updated to jinzhu, got %s", override)
	}

	if err := DB.Migrator().DropTable(&DialectTypeColumnStruct{}); err != nil {
		t.Errorf("failed to drop table, got error: %v", err)
	}
}

// jsonEqual compares two JSON strings semantically, databases like postgres
// normalize jsonb values (e.g. `{"a":1}` becomes `{"a": 1}`) when storing them.
func jsonEqual(t *testing.T, expected, actual string) bool {
	t.Helper()

	var expectedValue, actualValue interface{}
	if err := json.Unmarshal([]byte(expected), &expectedValue); err != nil {
		t.Fatalf("failed to unmarshal expected json %q, got error: %v", expected, err)
	}
	if err := json.Unmarshal([]byte(actual), &actualValue); err != nil {
		t.Fatalf("failed to unmarshal actual json %q, got error: %v", actual, err)
	}

	if !reflect.DeepEqual(expectedValue, actualValue) {
		return false
	}
	return true
}
