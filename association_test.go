package gorm_test

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

func TestBelongsTo(t *testing.T) {
	DB.DropTable(Category{}, Post{})
	DB.CreateTable(Category{}, Post{})

	post := Post{
		Title:        "post 1",
		Body:         "body 1",
		Category:     Category{Name: "Category 1"},
		MainCategory: Category{Name: "Main Category 1"},
	}

	if err := DB.Save(&post).Error; err != nil {
		t.Errorf("Got errors when save post", err.Error())
	}

	if post.Category.Id == 0 || post.MainCategory.Id == 0 {
		t.Errorf("Category's primary key should be updated")
	}

	if post.CategoryId.Int64 == 0 || post.MainCategoryId == 0 {
		t.Errorf("post's foreign key should be updated")
	}

	// Query
	var category1 Category
	DB.Model(&post).Association("Category").Find(&category1)
	if category1.Name != "Category 1" {
		t.Errorf("Query has one relations with Association")
	}

	var mainCategory1 Category
	DB.Model(&post).Association("MainCategory").Find(&mainCategory1)
	if mainCategory1.Name != "Main Category 1" {
		t.Errorf("Query has one relations with Association")
	}

	var category11 Category
	DB.Model(&post).Related(&category11)
	if category11.Name != "Category 1" {
		t.Errorf("Query has one relations with Related")
	}

	// Append
	var category2 = Category{
		Name: "Category 2",
	}
	DB.Model(&post).Association("Category").Append(&category2)

	if category2.Id == 0 {
		t.Errorf("Category should has ID when created with Append")
	}

	var category21 Category
	DB.Model(&post).Related(&category21)

	if category21.Name != "Category 2" {
		t.Errorf("Category should be updated with Append")
	}

	// Replace
	var category3 = Category{
		Name: "Category 3",
	}
	DB.Model(&post).Association("Category").Replace(&category3)

	if category3.Id == 0 {
		t.Errorf("Category should has ID when created with Replace")
	}

	var category31 Category
	DB.Model(&post).Related(&category31)
	if category31.Name != "Category 3" {
		t.Errorf("Category should be updated with Replace")
	}

	// Delete
	DB.Model(&post).Association("Category").Delete(&category2)
	DB.First(&post, post.Id)
	if DB.Model(&post).Related(&Category{}).RecordNotFound() {
		t.Errorf("Should not delete any category when Delete a unrelated Category")
	}

	DB.Model(&post).Association("Category").Delete(&category3)

	var category41 Category
	DB.Model(&post).Related(&category41)
	if category41.Name != "" {
		t.Errorf("Category should be deleted with Delete")
	}

	// Clear
	DB.Model(&post).Association("Category").Append(&Category{
		Name: "Category 2",
	})

	if DB.Model(&post).Related(&Category{}).RecordNotFound() {
		t.Errorf("Should find category after append")
	}

	DB.Model(&post).Association("Category").Clear()

	if !DB.Model(&post).Related(&Category{}).RecordNotFound() {
		t.Errorf("Should not find any category after Clear")
	}
}

func TestHasMany(t *testing.T) {
	DB.DropTable(Post{}, Comment{})
	DB.CreateTable(Post{}, Comment{})

	post := Post{
		Title:    "post 1",
		Body:     "body 1",
		Comments: []*Comment{{Content: "Comment 1"}, {Content: "Comment 2"}},
	}

	if err := DB.Save(&post).Error; err != nil {
		t.Errorf("Got errors when save post", err.Error())
	}

	for _, comment := range post.Comments {
		if comment.PostId == 0 {
			t.Errorf("comment's PostID should be updated")
		}
	}

	var compareComments = func(comments []Comment, contents []string) bool {
		var commentContents []string
		for _, comment := range comments {
			commentContents = append(commentContents, comment.Content)
		}
		sort.Strings(commentContents)
		sort.Strings(contents)
		return reflect.DeepEqual(commentContents, contents)
	}

	// Query
	if DB.First(&Comment{}, "content = ?", "Comment 1").Error != nil {
		t.Errorf("Comment 1 should be saved")
	}

	var comments1 []Comment
	DB.Model(&post).Association("Comments").Find(&comments1)
	if !compareComments(comments1, []string{"Comment 1", "Comment 2"}) {
		t.Errorf("Query has many relations with Association")
	}

	var comments11 []Comment
	DB.Model(&post).Related(&comments11)
	if !compareComments(comments11, []string{"Comment 1", "Comment 2"}) {
		t.Errorf("Query has many relations with Related")
	}

	// Append
	DB.Model(&post).Association("Comments").Append(&Comment{Content: "Comment 3"})

	var comments2 []Comment
	DB.Model(&post).Related(&comments2)
	if !compareComments(comments2, []string{"Comment 1", "Comment 2", "Comment 3"}) {
		t.Errorf("Append new record to has many relations")
	}

	// Delete
	DB.Model(&post).Association("Comments").Delete(comments11)

	var comments3 []Comment
	DB.Model(&post).Related(&comments3)
	if !compareComments(comments3, []string{"Comment 3"}) {
		t.Errorf("Delete an existing resource for has many relations")
	}

	// Replace
	DB.Model(&post).Association("Comments").Replace(&Comment{Content: "Comment 4"}, &Comment{Content: "Comment 5"})

	var comments4 []Comment
	DB.Model(&post).Related(&comments4)
	if !compareComments(comments4, []string{"Comment 4", "Comment 5"}) {
		t.Errorf("Replace has many relations")
	}

	// Clear
}

func TestHasOneAndHasManyAssociation(t *testing.T) {
	DB.DropTable(Category{}, Post{}, Comment{})
	DB.CreateTable(Category{}, Post{}, Comment{})

	post := Post{
		Title:        "post 1",
		Body:         "body 1",
		Comments:     []*Comment{{Content: "Comment 1"}, {Content: "Comment 2"}},
		Category:     Category{Name: "Category 1"},
		MainCategory: Category{Name: "Main Category 1"},
	}

	if err := DB.Save(&post).Error; err != nil {
		t.Errorf("Got errors when save post", err.Error())
	}

	if err := DB.First(&Category{}, "name = ?", "Category 1").Error; err != nil {
		t.Errorf("Category should be saved", err.Error())
	}

	var p Post
	DB.First(&p, post.Id)

	if post.CategoryId.Int64 == 0 || p.CategoryId.Int64 == 0 || post.MainCategoryId == 0 || p.MainCategoryId == 0 {
		t.Errorf("Category Id should exist")
	}

	if DB.First(&Comment{}, "content = ?", "Comment 1").Error != nil {
		t.Errorf("Comment 1 should be saved")
	}
	if post.Comments[0].PostId == 0 {
		t.Errorf("Comment Should have post id")
	}

	var comment Comment
	if DB.First(&comment, "content = ?", "Comment 2").Error != nil {
		t.Errorf("Comment 2 should be saved")
	}

	if comment.PostId == 0 {
		t.Errorf("Comment 2 Should have post id")
	}

	comment3 := Comment{Content: "Comment 3", Post: Post{Title: "Title 3", Body: "Body 3"}}
	DB.Save(&comment3)
}

func TestRelated(t *testing.T) {
	user := User{
		Name:            "jinzhu",
		BillingAddress:  Address{Address1: "Billing Address - Address 1"},
		ShippingAddress: Address{Address1: "Shipping Address - Address 1"},
		Emails:          []Email{{Email: "jinzhu@example.com"}, {Email: "jinzhu-2@example@example.com"}},
		CreditCard:      CreditCard{Number: "1234567890"},
		Company:         Company{Name: "company1"},
	}

	DB.Save(&user)

	if user.CreditCard.ID == 0 {
		t.Errorf("After user save, credit card should have id")
	}

	if user.BillingAddress.ID == 0 {
		t.Errorf("After user save, billing address should have id")
	}

	if user.Emails[0].Id == 0 {
		t.Errorf("After user save, billing address should have id")
	}

	var emails []Email
	DB.Model(&user).Related(&emails)
	if len(emails) != 2 {
		t.Errorf("Should have two emails")
	}

	var emails2 []Email
	DB.Model(&user).Where("email = ?", "jinzhu@example.com").Related(&emails2)
	if len(emails2) != 1 {
		t.Errorf("Should have two emails")
	}

	var emails3 []*Email
	DB.Model(&user).Related(&emails3)
	if len(emails3) != 2 {
		t.Errorf("Should have two emails")
	}

	var user1 User
	DB.Model(&user).Related(&user1.Emails)
	if len(user1.Emails) != 2 {
		t.Errorf("Should have only one email match related condition")
	}

	var address1 Address
	DB.Model(&user).Related(&address1, "BillingAddressId")
	if address1.Address1 != "Billing Address - Address 1" {
		t.Errorf("Should get billing address from user correctly")
	}

	user1 = User{}
	DB.Model(&address1).Related(&user1, "BillingAddressId")
	if DB.NewRecord(user1) {
		t.Errorf("Should get user from address correctly")
	}

	var user2 User
	DB.Model(&emails[0]).Related(&user2)
	if user2.Id != user.Id || user2.Name != user.Name {
		t.Errorf("Should get user from email correctly")
	}

	var creditcard CreditCard
	var user3 User
	DB.First(&creditcard, "number = ?", "1234567890")
	DB.Model(&creditcard).Related(&user3)
	if user3.Id != user.Id || user3.Name != user.Name {
		t.Errorf("Should get user from credit card correctly")
	}

	if !DB.Model(&CreditCard{}).Related(&User{}).RecordNotFound() {
		t.Errorf("RecordNotFound for Related")
	}

	var company Company
	if DB.Model(&user).Related(&company, "Company").RecordNotFound() || company.Name != "company1" {
		t.Errorf("RecordNotFound for Related")
	}
}

func TestManyToMany(t *testing.T) {
	DB.Raw("delete from languages")
	var languages = []Language{{Name: "ZH"}, {Name: "EN"}}
	user := User{Name: "Many2Many", Languages: languages}
	DB.Save(&user)

	// Query
	var newLanguages []Language
	DB.Model(&user).Related(&newLanguages, "Languages")
	if len(newLanguages) != len([]string{"ZH", "EN"}) {
		t.Errorf("Query many to many relations")
	}

	DB.Model(&user).Association("Languages").Find(&newLanguages)
	if len(newLanguages) != len([]string{"ZH", "EN"}) {
		t.Errorf("Should be able to find many to many relations")
	}

	if DB.Model(&user).Association("Languages").Count() != len([]string{"ZH", "EN"}) {
		t.Errorf("Count should return correct result")
	}

	// Append
	DB.Model(&user).Association("Languages").Append(&Language{Name: "DE"})
	if DB.Where("name = ?", "DE").First(&Language{}).RecordNotFound() {
		t.Errorf("New record should be saved when append")
	}

	languageA := Language{Name: "AA"}
	DB.Save(&languageA)
	DB.Model(&User{Id: user.Id}).Association("Languages").Append(&languageA)

	languageC := Language{Name: "CC"}
	DB.Save(&languageC)
	DB.Model(&user).Association("Languages").Append(&[]Language{{Name: "BB"}, languageC})

	DB.Model(&User{Id: user.Id}).Association("Languages").Append(&[]Language{{Name: "DD"}, {Name: "EE"}})

	totalLanguages := []string{"ZH", "EN", "DE", "AA", "BB", "CC", "DD", "EE"}

	if DB.Model(&user).Association("Languages").Count() != len(totalLanguages) {
		t.Errorf("All appended languages should be saved")
	}

	// Delete
	user.Languages = []Language{}
	DB.Model(&user).Association("Languages").Find(&user.Languages)

	var language Language
	DB.Where("name = ?", "EE").First(&language)
	DB.Model(&user).Association("Languages").Delete(language, &language)

	if DB.Model(&user).Association("Languages").Count() != len(totalLanguages)-1 || len(user.Languages) != len(totalLanguages)-1 {
		t.Errorf("Relations should be deleted with Delete")
	}
	if DB.Where("name = ?", "EE").First(&Language{}).RecordNotFound() {
		t.Errorf("Language EE should not be deleted")
	}

	DB.Where("name IN (?)", []string{"CC", "DD"}).Find(&languages)

	user2 := User{Name: "Many2Many_User2", Languages: languages}
	DB.Save(&user2)

	DB.Model(&user).Association("Languages").Delete(languages, &languages)
	if DB.Model(&user).Association("Languages").Count() != len(totalLanguages)-3 || len(user.Languages) != len(totalLanguages)-3 {
		t.Errorf("Relations should be deleted with Delete")
	}

	if DB.Model(&user2).Association("Languages").Count() == 0 {
		t.Errorf("Other user's relations should not be deleted")
	}

	// Replace
	var languageB Language
	DB.Where("name = ?", "BB").First(&languageB)
	DB.Model(&user).Association("Languages").Replace(languageB)
	if len(user.Languages) != 1 || DB.Model(&user).Association("Languages").Count() != 1 {
		t.Errorf("Relations should be replaced")
	}

	DB.Model(&user).Association("Languages").Replace()
	if len(user.Languages) != 0 || DB.Model(&user).Association("Languages").Count() != 0 {
		t.Errorf("Relations should be replaced with empty")
	}

	DB.Model(&user).Association("Languages").Replace(&[]Language{{Name: "FF"}, {Name: "JJ"}})
	if len(user.Languages) != 2 || DB.Model(&user).Association("Languages").Count() != len([]string{"FF", "JJ"}) {
		t.Errorf("Relations should be replaced")
	}

	// Clear
	DB.Model(&user).Association("Languages").Clear()
	if len(user.Languages) != 0 || DB.Model(&user).Association("Languages").Count() != 0 {
		t.Errorf("Relations should be cleared")
	}
}

func TestForeignKey(t *testing.T) {
	for _, structField := range DB.NewScope(&User{}).GetStructFields() {
		for _, foreignKey := range []string{"BillingAddressID", "ShippingAddressId", "CompanyID"} {
			if structField.Name == foreignKey && !structField.IsForeignKey {
				t.Errorf(fmt.Sprintf("%v should be foreign key", foreignKey))
			}
		}
	}

	for _, structField := range DB.NewScope(&Email{}).GetStructFields() {
		for _, foreignKey := range []string{"UserId"} {
			if structField.Name == foreignKey && !structField.IsForeignKey {
				t.Errorf(fmt.Sprintf("%v should be foreign key", foreignKey))
			}
		}
	}

	for _, structField := range DB.NewScope(&Post{}).GetStructFields() {
		for _, foreignKey := range []string{"CategoryId", "MainCategoryId"} {
			if structField.Name == foreignKey && !structField.IsForeignKey {
				t.Errorf(fmt.Sprintf("%v should be foreign key", foreignKey))
			}
		}
	}

	for _, structField := range DB.NewScope(&Comment{}).GetStructFields() {
		for _, foreignKey := range []string{"PostId"} {
			if structField.Name == foreignKey && !structField.IsForeignKey {
				t.Errorf(fmt.Sprintf("%v should be foreign key", foreignKey))
			}
		}
	}
}
