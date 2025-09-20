package tests_test

import (
	"testing"

	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

func TestTransactionLeakWithScopesSession(t *testing.T) {
	var users []User

	// Get the underlying sql.DB to check connection stats
	sqldb, err := DB.DB()
	if err != nil {
		t.Error(err)
	}

	before := sqldb.Stats().InUse

	// This should not leak transactions - the issue case
	result := DB.Scopes(func(s *gorm.DB) *gorm.DB {
		return s.Session(&gorm.Session{
			DryRun: true,
		})
	}).Where("name = ?", "transaction-leak-test").Find(&users)

	if result.Error != nil {
		t.Errorf("Query failed: %v", result.Error)
	}

	after := sqldb.Stats().InUse

	if before != after {
		t.Errorf("Transaction leak detected: InUse connections before=%d, after=%d", before, after)
	}
}

func TestTransactionLeakWithScopesSessionDelete(t *testing.T) {
	// Get the underlying sql.DB to check connection stats
	sqldb, err := DB.DB()
	if err != nil {
		t.Error(err)
	}

	before := sqldb.Stats().InUse

	// This should not leak transactions - the more specific case from the playground
	result := DB.Scopes(func(s *gorm.DB) *gorm.DB {
		return s.Session(&gorm.Session{
			DryRun: true,
		})
	}).Where("name = ?", "transaction-leak-test").Delete(&User{})

	if result.Error != nil {
		t.Errorf("Delete failed: %v", result.Error)
	}

	after := sqldb.Stats().InUse

	if before != after {
		t.Errorf("Transaction leak detected: InUse connections before=%d, after=%d", before, after)
	}
}