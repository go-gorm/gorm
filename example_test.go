package gorm_test

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type exampleUser struct {
	gorm.Model
	Name   string
	Active bool
}

// openExampleDB owns an isolated in-memory database for one example.
func openExampleDB() (*gorm.DB, func()) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		panic(err)
	}
	pool, err := db.DB()
	if err != nil {
		panic(err)
	}
	pool.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&exampleUser{}); err != nil {
		_ = pool.Close()
		panic(err)
	}
	return db, func() { _ = pool.Close() }
}

func ExampleDB_Session() {
	db, closeDB := openExampleDB()
	defer closeDB()
	users := []exampleUser{{Name: "alice", Active: true}, {Name: "bob", Active: true}}
	if err := db.Create(&users).Error; err != nil {
		panic(err)
	}

	// Session preserves the base predicate while isolating subsequent chains.
	active := db.Where("active = ?", true).Session(&gorm.Session{})
	var alice, bob exampleUser
	if err := active.Where("name = ?", "alice").First(&alice).Error; err != nil {
		panic(err)
	}
	if err := active.Where("name = ?", "bob").First(&bob).Error; err != nil {
		panic(err)
	}
	fmt.Println(alice.Name, bob.Name)
	// Output: alice bob
}

func ExampleDB_Updates() {
	db, closeDB := openExampleDB()
	defer closeDB()
	user := exampleUser{Name: "alice", Active: true}
	if err := db.Create(&user).Error; err != nil {
		panic(err)
	}

	// Map values include false. A struct's zero values are omitted by default.
	result := db.Model(&exampleUser{}).Where("id = ?", user.ID).
		Updates(map[string]interface{}{"active": false})
	if result.Error != nil {
		panic(result.Error)
	}
	var found exampleUser
	if err := db.First(&found, "id = ?", user.ID).Error; err != nil {
		panic(err)
	}
	fmt.Println(result.RowsAffected, found.Active)
	fmt.Println(errors.Is(db.Model(&exampleUser{}).Update("active", true).Error,
		gorm.ErrMissingWhereClause))
	// Output:
	// 1 false
	// true
}

func ExampleDB_Transaction() {
	db, closeDB := openExampleDB()
	defer closeDB()
	abort := errors.New("cancel this unit of work")
	err := db.Transaction(func(tx *gorm.DB) error {
		// Every operation in the unit of work uses tx.
		if err := tx.Create(&exampleUser{Name: "alice"}).Error; err != nil {
			return err
		}
		return abort
	})
	var count int64
	if err := db.Model(&exampleUser{}).Count(&count).Error; err != nil {
		panic(err)
	}
	fmt.Println(errors.Is(err, abort), count)
	// Output: true 0
}

func ExampleDB_Unscoped() {
	db, closeDB := openExampleDB()
	defer closeDB()
	user := exampleUser{Name: "alice"}
	if err := db.Create(&user).Error; err != nil {
		panic(err)
	}
	if err := db.Delete(&user).Error; err != nil {
		panic(err)
	}
	var count int64
	if err := db.Model(&exampleUser{}).Count(&count).Error; err != nil {
		panic(err)
	}
	fmt.Println("visible:", count)
	if err := db.Unscoped().Model(&exampleUser{}).Count(&count).Error; err != nil {
		panic(err)
	}
	fmt.Println("including deleted:", count)
	if err := db.Unscoped().Delete(&user).Error; err != nil {
		panic(err)
	}
	if err := db.Unscoped().Model(&exampleUser{}).Count(&count).Error; err != nil {
		panic(err)
	}
	fmt.Println("after physical delete:", count)
	// Output:
	// visible: 0
	// including deleted: 1
	// after physical delete: 0
}

func ExampleG() {
	db, closeDB := openExampleDB()
	defer closeDB()
	ctx := context.Background()
	user := exampleUser{Name: "alice", Active: true}
	if err := gorm.G[exampleUser](db).Create(ctx, &user); err != nil {
		panic(err)
	}
	active := gorm.G[exampleUser](db).Where("active = ?", true)
	found, err := active.Where("name = ?", "alice").First(ctx)
	if err != nil {
		panic(err)
	}
	_, err = active.Where("name = ?", "missing").First(ctx)
	fmt.Println(found.Name, errors.Is(err, gorm.ErrRecordNotFound))
	missing, err := active.Where("name = ?", "missing").Find(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println(len(missing))
	// Output:
	// alice true
	// 0
}

func ExampleDB_Rows() {
	db, closeDB := openExampleDB()
	defer closeDB()
	if err := db.Create(&exampleUser{Name: "alice"}).Error; err != nil {
		panic(err)
	}
	rows, err := db.Model(&exampleUser{}).Select("name").Order("id").Rows()
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			panic(err)
		}
	}()
	for rows.Next() {
		var user exampleUser
		if err := db.ScanRows(rows, &user); err != nil {
			panic(err)
		}
		fmt.Println(user.Name)
	}
	if err := rows.Err(); err != nil {
		panic(err)
	}
	// Output: alice
}
