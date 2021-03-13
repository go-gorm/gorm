package gorm

import (
	"context"
	"reflect"
	"testing"
)

func TestContext_success(t *testing.T) {
	db := &DB{}

	ctx := WithContext(context.Background(), db)
	extractedDB, err := FromContext(ctx)

	if err != nil {
		t.Errorf("expected err to be nil. Got: %v", err)
	}

	if !reflect.DeepEqual(db, extractedDB) {
		t.Errorf("db and extractedDB are not the same")
	}
}

func TestContext_failure(t *testing.T) {
	extractedDB, err := FromContext(context.Background())

	if extractedDB != nil {
		t.Errorf("expected extractedDB to nil. Got: %v", extractedDB)
	}

	if err != ErrDBNotFoundInContext {
		t.Errorf("expected err to be ErrDBNotFoundInContext. Got: %v", err)
	}
}
