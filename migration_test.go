package gorm_test

import (
	"fmt"
	"testing"
	"time"
)

func runMigration() {
	if err := db.DropTable(&User{}).Error; err != nil {
		fmt.Printf("Got error when try to delete table users, %+v\n", err)
	}

	db.Exec("drop table products;")
	db.Exec("drop table emails;")
	db.Exec("drop table addresses")
	db.Exec("drop table credit_cards")
	db.Exec("drop table roles")
	db.Exec("drop table companies")
	db.Exec("drop table animals")
	db.Exec("drop table user_languages")
	db.Exec("drop table languages")

	if err := db.CreateTable(&Animal{}).Error; err != nil {
		panic(fmt.Sprintf("No error should happen when create table, but got %+v", err))
	}

	if err := db.CreateTable(User{}).Error; err != nil {
		panic(fmt.Sprintf("No error should happen when create table, but got %+v", err))
	}

	if err := db.AutoMigrate(&Product{}, Email{}, Address{}, CreditCard{}, Company{}, Role{}, Language{}).Error; err != nil {
		panic(fmt.Sprintf("No error should happen when create table, but got %+v", err))
	}
}

func TestIndexes(t *testing.T) {
	if err := db.Model(&Email{}).AddIndex("idx_email_email", "email").Error; err != nil {
		t.Errorf("Got error when tried to create index: %+v", err)
	}

	if err := db.Model(&Email{}).RemoveIndex("idx_email_email").Error; err != nil {
		t.Errorf("Got error when tried to remove index: %+v", err)
	}

	if err := db.Model(&Email{}).AddIndex("idx_email_email_and_user_id", "user_id", "email").Error; err != nil {
		t.Errorf("Got error when tried to create index: %+v", err)
	}

	if err := db.Model(&Email{}).RemoveIndex("idx_email_email_and_user_id").Error; err != nil {
		t.Errorf("Got error when tried to remove index: %+v", err)
	}

	if err := db.Model(&Email{}).AddUniqueIndex("idx_email_email_and_user_id", "user_id", "email").Error; err != nil {
		t.Errorf("Got error when tried to create index: %+v", err)
	}

	if db.Save(&User{Name: "unique_indexes", Emails: []Email{{Email: "user1@example.comiii"}, {Email: "user1@example.com"}, {Email: "user1@example.com"}}}).Error == nil {
		t.Errorf("Should get to create duplicate record when having unique index")
	}

	if err := db.Model(&Email{}).RemoveIndex("idx_email_email_and_user_id").Error; err != nil {
		t.Errorf("Got error when tried to remove index: %+v", err)
	}

	if db.Save(&User{Name: "unique_indexes", Emails: []Email{{Email: "user1@example.com"}, {Email: "user1@example.com"}}}).Error != nil {
		t.Errorf("Should be able to create duplicated emails after remove unique index")
	}
}

type BigEmail struct {
	Id           int64
	UserId       int64
	Email        string
	UserAgent    string
	RegisteredAt time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (b BigEmail) TableName() string {
	return "emails"
}

func TestAutoMigration(t *testing.T) {
	db.AutoMigrate(Address{})
	if err := db.Table("emails").AutoMigrate(BigEmail{}).Error; err != nil {
		t.Errorf("Auto Migrate should not raise any error")
	}

	db.Save(&BigEmail{Email: "jinzhu@example.org", UserAgent: "pc", RegisteredAt: time.Now()})

	var bigemail BigEmail
	db.First(&bigemail, "user_agent = ?", "pc")
	if bigemail.Email != "jinzhu@example.org" || bigemail.UserAgent != "pc" || bigemail.RegisteredAt.IsZero() {
		t.Error("Big Emails should be saved and fetched correctly")
	}
}
