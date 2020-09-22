package tests_test

import (
	"testing"

	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

func TestCreateUniqueConstraint(t *testing.T) {
	user1 := GetUser("create-unique-constraint", Config{})
	if err := DB.Create(user1).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	user1Contact := &Contact{UserID: &user1.ID, Email: "create-unique-constraint@email"}
	if err := DB.Create(user1Contact).Error; err != nil {
		t.Fatalf("errors happened when create cotract: %v", err)
	}

	user2 := GetUser("create-unique-constraint2", Config{})
	if err := DB.Create(user2).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	user2Contact := &Contact{UserID: &user2.ID, Email: "create-unique-constraint@email"}
	err := DB.Create(user2Contact).Error
	if err == nil {
		t.Fatal("should return unique constraint error")
	}
	e, ok := err.(*gorm.ErrUniqueConstraint)
	if !ok {
		t.Fatalf("should return unique constraint error, got err %v", err)
	}

	if len(e.ConstraintName) > 0 {
		AssertEqual(t, e.ConstraintName, "idx_email")
	}

	if len(e.Columns) > 0 {
		AssertEqual(t, e.Columns[0], "contacts.email")
	}
}
