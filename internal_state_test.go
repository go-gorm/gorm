package gorm_test

import (
	"runtime"
	"testing"
)

type (
	Organization struct {
		Id   int64
		Name string `sql:"type:varchar(255);not null;"`
	}

	App struct {
		Id             int64
		Organization   Organization
		OrganizationId int64  `sql:"not null;"`
		Name           string `sql:"type:varchar(255);not null;"`
	}
)

// TestMultipleSingularTableInvocations runs `DB.SingularTable(true)' at the
// beginning and middle of operation.  The `SingularTable' call clears out all
// of the `gorm.modelStructs' internal package state, triggering the conditions
// where gorm must recover, or this test will not pass.
//
// Also see the `Scope.Fields' function in scope.go for additional information
// and context.
func TestMultipleSingularTableInvocations(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	DB.SingularTable(true)        // Invocation #1.
	defer DB.SingularTable(false) // Restore original plurality.
	//DB.LogMode(true)

	entities := []interface{}{
		&Organization{},
		&App{},
	}
	for i := len(entities) - 1; i >= 0; i-- {
		if err := DB.DropTableIfExists(entities[i]).Error; err != nil {
			t.Fatalf("Problem dropping table entity=%+v: %s", entities[i], err)
		}
	}
	if err := DB.AutoMigrate(entities...).Error; err != nil {
		t.Fatalf("Auto-migrate failed: %s", err)
	}

	createFixtures(t)
}

func createFixtures(t *testing.T) {
	DB.SingularTable(true) // Invocation #2.  Clobber/reset internal gorm state.

	org := &Organization{
		Name: "Some Organization for Testing",
	}
	if err := DB.Save(org).Error; err != nil {
		t.Fatal(err)
	}

	app := &App{
		OrganizationId: org.Id,
		Name:           "my-app-for-test-purposes",
	}
	if err := DB.Save(app).Error; err != nil {
		t.Fatal(err)
	}
}
