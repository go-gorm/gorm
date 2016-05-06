package main

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var db gorm.DB

// Profile ...
type Profile struct {
	gorm.Model
	Name string `sql:"type:varchar(40);not null"`
}

// User ...
type User struct {
	gorm.Model
	Username     string `sql:"type:varchar(100);not null;unique"`
	UserProfiles []*UserProfile
}

// UserProfile ...
type UserProfile struct {
	gorm.Model
	ProfileID sql.NullInt64 `sql:"index;not null"`
	UserID    sql.NullInt64 `sql:"index;not null"`
	Profile   *Profile
	User      *User
	State     string `sql:"index;not null"`
}

func init() {
	var err error
	db, err = gorm.Open("sqlite3", ":memory:")
	// db, err := gorm.Open("postgres", "user=username dbname=password sslmode=disable")
	// db, err := gorm.Open("mysql", "user:password@/dbname?charset=utf8&parseTime=True")
	if err != nil {
		panic(err)
	}
	db.LogMode(true)

	db.AutoMigrate(new(Profile), new(User), new(UserProfile))
}

func main() {
	buyerProfile := &Profile{Name: "buyer"}
	if err := db.Create(buyerProfile).Error; err != nil {
		panic(err)
	}
	sellerProfile := &Profile{Name: "seller"}
	if err := db.Create(sellerProfile).Error; err != nil {
		panic(err)
	}

	user := &User{
		Username: "username",
		UserProfiles: []*UserProfile{
			&UserProfile{
				Profile: buyerProfile,
				State:   "some_state",
			},
		},
	}
	if err := db.Create(user).Error; err != nil {
		panic(err)
	}

	// Now let's update the user
	tx := db.Begin()

	user.Username = "username_edited"

	user.UserProfiles = append(
		user.UserProfiles,
		&UserProfile{
			Profile: sellerProfile,
			State:   "some_state",
		},
	)

	if err := tx.Model(user).Association("UserProfiles").Append(&UserProfile{
		Profile: sellerProfile,
		State:   "some_state",
	}).Error; err != nil {
		tx.Rollback() // rollback the transaction
		panic(err)
	}

	// if err := tx.Save(user).Error; err != nil {
	// 	tx.Rollback() // rollback the transaction
	// 	panic(err)
	// }

	if err := tx.Commit().Error; err != nil {
		tx.Rollback() // rollback the transaction
		panic(err)
	}
}
