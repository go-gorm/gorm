package tests_test

import (
	"errors"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

func TestDialectorWithErrorTranslatorSupport(t *testing.T) {
	// it shouldn't translate error when the TranslateError flag is false
	translatedErr := errors.New("translated error")
	untranslatedErr := errors.New("some random error")
	db, _ := gorm.Open(tests.DummyDialector{TranslatedErr: translatedErr})

	err := db.AddError(untranslatedErr)
	if !errors.Is(err, untranslatedErr) {
		t.Fatalf("expected err: %v got err: %v", untranslatedErr, err)
	}

	// it should translate error when the TranslateError flag is true
	db, _ = gorm.Open(tests.DummyDialector{TranslatedErr: translatedErr}, &gorm.Config{TranslateError: true})

	err = db.AddError(untranslatedErr)
	if !errors.Is(err, translatedErr) {
		t.Fatalf("expected err: %v got err: %v", translatedErr, err)
	}
}

func TestSupportedDialectorWithErrDuplicatedKey(t *testing.T) {
	type City struct {
		gorm.Model
		Name string `gorm:"unique"`
	}

	db, err := OpenTestConnection(&gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to connect database, got error %v", err)
	}

	dialectors := map[string]bool{"sqlite": true, "postgres": true, "gaussdb": true, "mysql": true, "sqlserver": true}
	if supported, found := dialectors[db.Dialector.Name()]; !(found && supported) {
		return
	}

	DB.Migrator().DropTable(&City{})

	if err = db.AutoMigrate(&City{}); err != nil {
		t.Fatalf("failed to migrate cities table, got error: %v", err)
	}

	err = db.Create(&City{Name: "Kabul"}).Error
	if err != nil {
		t.Fatalf("failed to create record: %v", err)
	}

	err = db.Create(&City{Name: "Kabul"}).Error
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrDuplicatedKey, err)
	}
}

func TestSupportedDialectorWithErrForeignKeyViolated(t *testing.T) {
	tidbSkip(t, "not support the foreign key feature")

	type City struct {
		gorm.Model
		Name string `gorm:"unique"`
	}

	type Museum struct {
		gorm.Model
		Name   string `gorm:"unique"`
		CityID uint
		City   City `gorm:"Constraint:OnUpdate:CASCADE,OnDelete:CASCADE;FOREIGNKEY:CityID;References:ID"`
	}

	db, err := OpenTestConnection(&gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to connect database, got error %v", err)
	}

	dialectors := map[string]bool{"sqlite": true, "postgres": true, "gaussdb": true, "mysql": true, "sqlserver": true}
	if supported, found := dialectors[db.Dialector.Name()]; !(found && supported) {
		return
	}

	DB.Migrator().DropTable(&City{}, &Museum{})

	if err = db.AutoMigrate(&City{}, &Museum{}); err != nil {
		t.Fatalf("failed to migrate countries & cities tables, got error: %v", err)
	}

	city := City{Name: "Amsterdam"}

	err = db.Create(&city).Error
	if err != nil {
		t.Fatalf("failed to create city: %v", err)
	}

	err = db.Create(&Museum{Name: "Eye Filmmuseum", CityID: city.ID}).Error
	if err != nil {
		t.Fatalf("failed to create museum: %v", err)
	}

	err = db.Create(&Museum{Name: "Dungeon", CityID: 123}).Error
	if !errors.Is(err, gorm.ErrForeignKeyViolated) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrForeignKeyViolated, err)
	}
}
