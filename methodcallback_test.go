package gorm_test

import "fmt"

func ExampleAfterScanMethodCallback() {
	fmt.Println(`package main

import (
	"fmt"
	"reflect"
	"github.com/jinzhu/gorm"
	"database/sql/driver"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Media struct {
	Name      string
	baseUrl   *string
	modelType reflect.Type
	model interface {
		GetID() int
	}
	fieldName *string
}

func (image *Media) Scan(value interface{}) error {
	image.Name = string(value.([]byte))
	return nil
}

func (image *Media) Value() (driver.Value, error) {
	return image.Name, nil
}

func (image *Media) AfterScan(scope *gorm.Scope, field *gorm.Field) {
	image.fieldName, image.model = &field.StructField.Name, scope.Value.(interface {
		GetID() int
	})
	baseUrl, _ := scope.DB().Get("base_url")
	image.baseUrl = baseUrl.(*string)
	image.modelType = reflect.TypeOf(scope.Value)
	for image.modelType.Kind() == reflect.Ptr {
		image.modelType = image.modelType.Elem()
	}
}

func (image *Media) URL() string {
	return fmt.Sprintf("%v/%v/%v/%v/%v", *image.baseUrl, image.modelType.Name(), image.model.GetID(), *image.fieldName, image.Name)
}

type User struct {
	ID        int
	MainImage Media
}

func (user *User) GetID() int {
	return user.ID
}

func main() {
	db, err := gorm.Open("sqlite3", "db.db")
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&User{})

	baseUrl := "http://example.com/media"
	db = db.Set("base_url", &baseUrl)

	var model User
	db_ := db.Where("id = ?", 1).First(&model)
	if db_.RecordNotFound() {
		db.Save(&User{MainImage: Media{Name: "picture.jpg"}})
		err = db.Where("id = ?", 1).First(&model).Error
		if err != nil {
			panic(err)
		}
	} else if db_.Error != nil {
		panic(db_.Error)
	}

	fmt.Println(model.MainImage.URL())
}`)
}
