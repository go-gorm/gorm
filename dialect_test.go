package gorm_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/jinzhu/gorm"

	_ "github.com/jinzhu/gorm/dialects/mssql"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

func TestRegisterGetAndUnregisterDialect(t *testing.T) {
	commonDialect, ok := gorm.GetDialect("common")
	if !ok {
		t.Error("Expected to find dialect 'common' registered, but it is missing")
	}
	gorm.RegisterDialect("dialect_for_TestUnregisterDialect", commonDialect)

	// Check the test's dialect is there.
	testDialect, ok := gorm.GetDialect("dialect_for_TestUnregisterDialect")
	if !ok {
		t.Error("Expected to find the test dialect registered, but it is missing")
	}
	if testDialect != commonDialect {
		t.Error("Unexpected dialect returned by GetDialect")
	}

	// Remove the test dialect.
	gorm.UnregisterDialect("dialect_for_TestUnregisterDialect")

	// Check the test's dialect is now gone.
	testDialect, ok = gorm.GetDialect("dialect_for_TestUnregisterDialect")
	if ok {
		t.Errorf("Expected the test dialect to be removed, but it is still registered: %v", testDialect)
	}
}

func TestGetAllDialectsEmpty(t *testing.T) {
	// Clear the old dialects map, and reset it after the test ends.
	oldDialectsMap := gorm.GetAllDialects()
	for name := range oldDialectsMap {
		gorm.UnregisterDialect(name)
	}
	defer func() {
		for name, dialect := range oldDialectsMap {
			gorm.RegisterDialect(name, dialect)
		}
	}()

	// Empty case.
	dialects := gorm.GetAllDialects()
	if len(dialects) != 0 {
		t.Errorf("There should be no dialects registered when dialectsMap is empty, instead found: %v", dialects)
	}
}

func TestGetAllDialectNamesEmpty(t *testing.T) {
	// Clear the old dialects map, and reset it after the test ends.
	oldDialectsMap := gorm.GetAllDialects()
	for name := range oldDialectsMap {
		gorm.UnregisterDialect(name)
	}
	defer func() {
		for name, dialect := range oldDialectsMap {
			gorm.RegisterDialect(name, dialect)
		}
	}()

	// Empty case.
	allNames := gorm.GetAllDialectNames()
	if len(allNames) != 0 {
		t.Errorf("There should be no registered dialects when dialectsMap is empty, instead found: %v", allNames)
	}
}

func TestGetAllDialects(t *testing.T) {
	// Clear the old dialects map, and reset it after the test ends.
	oldDialectsMap := gorm.GetAllDialects()
	for name := range oldDialectsMap {
		gorm.UnregisterDialect(name)
	}
	defer func() {
		for name, dialect := range oldDialectsMap {
			gorm.RegisterDialect(name, dialect)
		}
	}()

	// Register some dialects.
	dialectNames := []string{
		"common",
		"mysql",
		"mssql",
		"postgres",
		"cloudsqlpostgres",
		"sqlite3",
	}
	for _, name := range dialectNames {
		oldDialect, ok := oldDialectsMap[name]
		if !ok {
			t.Errorf("Expected imports to register dialect '%s', but it is missing. Full map is: %v", name, oldDialectsMap)
		}
		gorm.RegisterDialect(name, oldDialect)
	}

	// Check the returned map
	dialects := gorm.GetAllDialects()
	if len(dialects) != 6 {
		t.Errorf("Expected to find 6 dialects registered, instead found %d. Full map is: %v", len(dialects), dialects)
	}
}

func TestGetAllDialectNames(t *testing.T) {
	// Clear the old dialects map, and reset it after the test ends.
	oldDialectsMap := gorm.GetAllDialects()
	for name := range oldDialectsMap {
		gorm.UnregisterDialect(name)
	}
	defer func() {
		for name, dialect := range oldDialectsMap {
			gorm.RegisterDialect(name, dialect)
		}
	}()

	// Register some dialects.
	dialectNames := []string{
		"cloudsqlpostgres",
		"common",
		"mssql",
		"mysql",
		"postgres",
		"sqlite3",
	}
	for _, name := range dialectNames {
		oldDialect, ok := oldDialectsMap[name]
		if !ok {
			t.Errorf("Expected imports to register dialect '%s', but it is missing. Full map is: %v", name, oldDialectsMap)
		}
		gorm.RegisterDialect(name, oldDialect)
	}

	// Check the returned map
	allNames := gorm.GetAllDialectNames()
	if len(allNames) != 6 {
		t.Errorf("Expected to find 6 dialect names, instead found %d. Full list is: %v", len(allNames), allNames)
	}

	// Sort both dialectNames and allNames
	sort.Strings(dialectNames)
	sort.Strings(allNames)
	if !reflect.DeepEqual(dialectNames, allNames) {
		t.Errorf("Unexpected list of dialects returned. Expected: %v, instead found: %v", dialectNames, allNames)
	}
}
