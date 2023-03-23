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
	if errors.Is(err, translatedErr) {
		t.Fatalf("expected err: %v got err: %v", translatedErr, err)
	}

	// it should translate error when the TranslateError flag is true
	db, _ = gorm.Open(tests.DummyDialector{TranslatedErr: translatedErr}, &gorm.Config{TranslateError: true})

	err = db.AddError(untranslatedErr)
	if !errors.Is(err, translatedErr) {
		t.Fatalf("expected err: %v got err: %v", translatedErr, err)
	}
}
