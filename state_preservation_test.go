package gorm_test

import (
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

func TestMultipleSingularTableInvocations(t *testing.T) {
	DB.SingularTable(true) // Invocation #1.
	// DB.LogMode(true)

	entities := []interface{}{
		&Organization{},
		&App{},
	}
	for i := len(entities) - 1; i >= 0; i-- {
		if err := DB.DropTableIfExists(entities[i]).Error; err != nil {
			t.Fatalf("Drop table for entity=%+v: %s", entities[i], err)
		}
	}
	if err := DB.AutoMigrate(entities...).Error; err != nil {
		t.Fatalf("Auto-migrate failed for entity=%+v: %s", entity, err)
	}
	if err := DB.Model(&App{}).AddForeignKey("organization_id", "organization(id)", "RESTRICT", "RESTRICT"); err != nil {
		t.Fatalf("Problem adding OrganizationId foreign-key to App table: %s", err)
	}

	createFixtures(t)
}

func createFixtures(t *testing.T) {
	DB.SingularTable(true) // Invocation #2.  If this were to clobber internal gorm state it can break things.

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
