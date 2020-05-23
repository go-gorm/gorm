package tests

import (
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

func Now() *time.Time {
	now := time.Now()
	return &now
}

func RunTestsSuit(t *testing.T, db *gorm.DB) {
	// TestCreate(t, db)
	TestFind(t, db)
	TestUpdate(t, db)
	TestDelete(t, db)

	TestGroupBy(t, db)
	TestJoins(t, db)
	TestAssociations(t, db)
}
