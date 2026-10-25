package gorm_test

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type assocReplaceCompany struct {
	ID        uint `gorm:"primarykey"`
	Name      string
	DeletedAt gorm.DeletedAt
}

type assocReplaceEmpPtr struct {
	ID        uint `gorm:"primarykey"`
	Name      string
	CompanyID *uint                // pointer FK: the identity map captures the pointer itself
	Company   *assocReplaceCompany `gorm:"foreignKey:CompanyID"`
}

type assocReplaceEmpVal struct {
	ID        uint `gorm:"primarykey"`
	Name      string
	CompanyID uint                 // value FK: control group
	Company   *assocReplaceCompany `gorm:"foreignKey:CompanyID"`
}

func assocReplaceSetup(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db failed: %v", err)
	}
	if err := db.Migrator().DropTable(&assocReplaceEmpPtr{}, &assocReplaceEmpVal{}, &assocReplaceCompany{}); err != nil {
		t.Fatalf("drop tables failed: %v", err)
	}
	if err := db.AutoMigrate(&assocReplaceCompany{}, &assocReplaceEmpPtr{}, &assocReplaceEmpVal{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func assertAssocReplace(t *testing.T, db *gorm.DB, table string) {
	t.Helper()

	// the NEW company must survive the replace (bug: the delete condition
	// aliased the FK pointer, which had been written through to the new value,
	// so the delete removed the new row instead of the old one)
	var gotNew assocReplaceCompany
	if err := db.First(&gotNew, 2).Error; err != nil {
		t.Fatalf("[%s] new company should remain after replace: %v", table, err)
	}
	if gotNew.DeletedAt.Valid {
		t.Errorf("[%s] new company should not be soft-deleted, got %+v", table, gotNew)
	}

	// the OLD company must be gone from the live rows
	var oldLive int64
	if err := db.Model(&assocReplaceCompany{}).Where("id = ?", 1).Count(&oldLive).Error; err != nil {
		t.Fatalf("[%s] count old company failed: %v", table, err)
	}
	if oldLive != 0 {
		t.Errorf("[%s] old company should have been deleted, still visible", table)
	}
}

// Unscoped Replace on a BelongsTo association captured pointer-typed foreign
// keys by reference. saveAssociation then updated the FK by writing through
// that same pointer, so the deferred delete condition pointed at the NEW
// associated row: the new row was deleted and the old one survived.
func TestAssociationUnscopedReplacePointerFK(t *testing.T) {
	db := assocReplaceSetup(t)

	oldC := assocReplaceCompany{ID: 1, Name: "old"}
	newC := assocReplaceCompany{ID: 2, Name: "new"}
	if err := db.Create(&oldC).Error; err != nil {
		t.Fatalf("create old failed: %v", err)
	}
	if err := db.Create(&newC).Error; err != nil {
		t.Fatalf("create new failed: %v", err)
	}

	emp := assocReplaceEmpPtr{ID: 10, Name: "e", CompanyID: &oldC.ID}
	if err := db.Create(&emp).Error; err != nil {
		t.Fatalf("create employee failed: %v", err)
	}

	if err := db.Model(&emp).Association("Company").Unscoped().Replace(&newC); err != nil {
		t.Fatalf("replace failed: %v", err)
	}
	assertAssocReplace(t, db, "emp_pointer_fk")

	// the FK must point at the (still existing) new company, not at a deleted row
	var e2 assocReplaceEmpPtr
	if err := db.First(&e2, 10).Error; err != nil {
		t.Fatalf("reload employee failed: %v", err)
	}
	if e2.CompanyID == nil || *e2.CompanyID != 2 {
		t.Fatalf("employee FK should point to company 2, got %v", e2.CompanyID)
	}
}

// value-typed foreign key: the same flow must keep working (control group for
// the pointer-aliasing fix)
func TestAssociationUnscopedReplaceValueFK(t *testing.T) {
	db := assocReplaceSetup(t)

	oldC := assocReplaceCompany{ID: 1, Name: "old"}
	newC := assocReplaceCompany{ID: 2, Name: "new"}
	if err := db.Create(&oldC).Error; err != nil {
		t.Fatalf("create old failed: %v", err)
	}
	if err := db.Create(&newC).Error; err != nil {
		t.Fatalf("create new failed: %v", err)
	}

	emp := assocReplaceEmpVal{ID: 20, Name: "e", CompanyID: 1}
	if err := db.Create(&emp).Error; err != nil {
		t.Fatalf("create employee failed: %v", err)
	}

	if err := db.Model(&emp).Association("Company").Unscoped().Replace(&newC); err != nil {
		t.Fatalf("replace failed: %v", err)
	}
	assertAssocReplace(t, db, "emp_value_fk")

	var e2 assocReplaceEmpVal
	if err := db.First(&e2, 20).Error; err != nil {
		t.Fatalf("reload employee failed: %v", err)
	}
	if e2.CompanyID != 2 {
		t.Fatalf("employee FK should point to company 2, got %d", e2.CompanyID)
	}
}
