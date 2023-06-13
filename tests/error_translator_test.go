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

	dialectors := map[string]bool{"sqlite": true, "postgres": true, "mysql": true, "sqlserver": true}
	if supported, found := dialectors[db.Dialector.Name()]; !(found && supported) {
		return
	}

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
	type Country struct {
		gorm.Model
		Name string `gorm:"unique"`
	}

	type City struct {
		gorm.Model
		Name      string `gorm:"unique"`
		CountryID uint
		Country   Country `gorm:"Constraint:OnUpdate:CASCADE,OnDelete:CASCADE;FOREIGNKEY:CountryID;References:ID"`
	}

	db, err := OpenTestConnection(&gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to connect database, got error %v", err)
	}

	dialectors := map[string]bool{"sqlite": true, "postgres": true, "mysql": false, "sqlserver": true}
	if supported, found := dialectors[db.Dialector.Name()]; !(found && supported) {
		return
	}

	if err = db.AutoMigrate(&Country{}, &City{}); err != nil {
		t.Fatalf("failed to migrate countries & cities tables, got error: %v", err)
	}

	country := &Country{Name: "Netherlands"}

	err = db.Create(country).Error
	if err != nil {
		t.Fatalf("failed to create country: %v", err)
	}

	err = db.Create(&City{Name: "Amsterdam", CountryID: country.ID}).Error
	if err != nil {
		t.Fatalf("failed to create city: %v", err)
	}

	err = db.Create(&City{Name: "Rotterdam", CountryID: 123}).Error
	if !errors.Is(err, gorm.ErrForeignKeyViolated) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrForeignKeyViolated, err)
	}
}
