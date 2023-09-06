package tests_test

import (
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	. "gorm.io/gorm/utils/tests"
)

// only mysql support update join
func TestReasonUpdateJoinUpdatedAtIsAmbiguous(t *testing.T) {
	if DB.Dialector.Name() != "mysql" {
		return
	}

	if err := DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Model(&User{}).InnerJoins("Account", DB.Where("number = ?", 1)).Update("name", "jinzhu").Error; !strings.Contains(err.Error(), "Column 'updated_at' in field list is ambiguous") {
		t.Errorf(`Error should be column is ambiguous, but got: "%s"`, err)
	}
}

// only mysql support update join
func TestUpdateJoinWorksManuallySettingSetClauses(t *testing.T) {
	if DB.Dialector.Name() != "mysql" {
		return
	}

	var (
		users = []*User{
			GetUser("update-1", Config{Account: true}),
			GetUser("update-2", Config{Account: true}),
			GetUser("update-3", Config{}),
		}
		user = users[1]
	)

	if err := DB.Create(&users).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	} else if user.ID == 0 {
		t.Fatalf("user's primary value should not zero, %v", user.ID)
	} else if user.UpdatedAt.IsZero() {
		t.Fatalf("user's updated at should not zero, %v", user.UpdatedAt)
	}

	tx := DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Model(user).InnerJoins("Account", DB.Where("number = ?", user.Account.Number))
	tx.Statement.AddClause(clause.Set{
		{
			Column: clause.Column{
				Name:  "name",
				Table: "users",
			},
			Value: "franco",
		},
		{
			Column: clause.Column{
				Name:  "updated_at",
				Table: "users",
			},
			Value: time.Now(),
		},
	})

	if rowsAffected := tx.Updates(nil).RowsAffected; rowsAffected != 1 {
		t.Errorf("should only update one record, but got %v", rowsAffected)
	}

	var result User
	if err := DB.First(&result, "name = ?", "franco").Error; err != nil {
		t.Errorf("user's name should be updated")
	} else if result.UpdatedAt.UnixNano() == user.UpdatedAt.UnixNano() {
		t.Errorf("user's updated at should be changed, but got %v, was %v", result.UpdatedAt, user.UpdatedAt)
	}
}
