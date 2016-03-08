package gorm_test

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/jinzhu/gorm"
)

func TestBelongsTo(t *testing.T) {
	post := Post{
		Title:        "post belongs to",
		Body:         "body belongs to",
		Category:     Category{Name: "Category 1"},
		MainCategory: Category{Name: "Main Category 1"},
	}

	if err := DB.Save(&post).Error; err != nil {
		t.Error("Got errors when save post", err)
	}

	if post.Category.ID == 0 || post.MainCategory.ID == 0 {
		t.Errorf("Category's primary key should be updated")
	}

	if post.CategoryId.Int64 == 0 || post.MainCategoryId == 0 {
		t.Errorf("post's foreign key should be updated")
	}

	// Query
	var category1 Category
	DB.Model(&post).Association("Category").Find(&category1)
	if category1.Name != "Category 1" {
		t.Errorf("Query belongs to relations with Association")
	}

	var mainCategory1 Category
	DB.Model(&post).Association("MainCategory").Find(&mainCategory1)
	if mainCategory1.Name != "Main Category 1" {
		t.Errorf("Query belongs to relations with Association")
	}

	var category11 Category
	DB.Model(&post).Related(&category11)
	if category11.Name != "Category 1" {
		t.Errorf("Query belongs to relations with Related")
	}

	if DB.Model(&post).Association("Category").Count() != 1 {
		t.Errorf("Post's category count should be 1")
	}

	if DB.Model(&post).Association("MainCategory").Count() != 1 {
		t.Errorf("Post's main category count should be 1")
	}

	// Append
	var category2 = Category{
		Name: "Category 2",
	}
	DB.Model(&post).Association("Category").Append(&category2)

	if category2.ID == 0 {
		t.Errorf("Category should has ID when created with Append")
	}

	var category21 Category
	DB.Model(&post).Related(&category21)

	if category21.Name != "Category 2" {
		t.Errorf("Category should be updated with Append")
	}

	if DB.Model(&post).Association("Category").Count() != 1 {
		t.Errorf("Post's category count should be 1")
	}

	// Replace
	var category3 = Category{
		Name: "Category 3",
	}
	DB.Model(&post).Association("Category").Replace(&category3)

	if category3.ID == 0 {
		t.Errorf("Category should has ID when created with Replace")
	}

	var category31 Category
	DB.Model(&post).Related(&category31)
	if category31.Name != "Category 3" {
		t.Errorf("Category should be updated with Replace")
	}

	if DB.Model(&post).Association("Category").Count() != 1 {
		t.Errorf("Post's category count should be 1")
	}

	// Delete
	DB.Model(&post).Association("Category").Delete(&category2)
	if DB.Model(&post).Related(&Category{}).RecordNotFound() {
		t.Errorf("Should not delete any category when Delete a unrelated Category")
	}

	if post.Category.Name == "" {
		t.Errorf("Post's category should not be reseted when Delete a unrelated Category")
	}

	DB.Model(&post).Association("Category").Delete(&category3)

	if post.Category.Name != "" {
		t.Errorf("Post's category should be reseted after Delete")
	}

	var category41 Category
	DB.Model(&post).Related(&category41)
	if category41.Name != "" {
		t.Errorf("Category should be deleted with Delete")
	}

	if count := DB.Model(&post).Association("Category").Count(); count != 0 {
		t.Errorf("Post's category count should be 0 after Delete, but got %v", count)
	}

	// Clear
	DB.Model(&post).Association("Category").Append(&Category{
		Name: "Category 2",
	})

	if DB.Model(&post).Related(&Category{}).RecordNotFound() {
		t.Errorf("Should find category after append")
	}

	if post.Category.Name == "" {
		t.Errorf("Post's category should has value after Append")
	}

	DB.Model(&post).Association("Category").Clear()

	if post.Category.Name != "" {
		t.Errorf("Post's category should be cleared after Clear")
	}

	if !DB.Model(&post).Related(&Category{}).RecordNotFound() {
		t.Errorf("Should not find any category after Clear")
	}

	if count := DB.Model(&post).Association("Category").Count(); count != 0 {
		t.Errorf("Post's category count should be 0 after Clear, but got %v", count)
	}

	// Check Association mode with soft delete
	category6 := Category{
		Name: "Category 6",
	}
	DB.Model(&post).Association("Category").Append(&category6)

	if count := DB.Model(&post).Association("Category").Count(); count != 1 {
		t.Errorf("Post's category count should be 1 after Append, but got %v", count)
	}

	DB.Delete(&category6)

	if count := DB.Model(&post).Association("Category").Count(); count != 0 {
		t.Errorf("Post's category count should be 0 after the category has been deleted, but got %v", count)
	}

	if err := DB.Model(&post).Association("Category").Find(&Category{}).Error; err == nil {
		t.Errorf("Post's category is not findable after Delete")
	}

	if count := DB.Unscoped().Model(&post).Association("Category").Count(); count != 1 {
		t.Errorf("Post's category count should be 1 when query with Unscoped, but got %v", count)
	}

	if err := DB.Unscoped().Model(&post).Association("Category").Find(&Category{}).Error; err != nil {
		t.Errorf("Post's category should be findable when query with Unscoped, got %v", err)
	}
}

func TestBelongsToOverrideForeignKey1(t *testing.T) {
	type Profile struct {
		gorm.Model
		Name string
	}

	type User struct {
		gorm.Model
		Profile      Profile `gorm:"ForeignKey:ProfileRefer"`
		ProfileRefer int
	}

	if relation, ok := DB.NewScope(&User{}).FieldByName("Profile"); ok {
		if relation.Relationship.Kind != "belongs_to" ||
			!reflect.DeepEqual(relation.Relationship.ForeignFieldNames, []string{"ProfileRefer"}) ||
			!reflect.DeepEqual(relation.Relationship.AssociationForeignFieldNames, []string{"ID"}) {
			t.Errorf("Override belongs to foreign key with tag")
		}
	}
}

func TestBelongsToOverrideForeignKey2(t *testing.T) {
	type Profile struct {
		gorm.Model
		Refer string
		Name  string
	}

	type User struct {
		gorm.Model
		Profile   Profile `gorm:"ForeignKey:ProfileID;AssociationForeignKey:Refer"`
		ProfileID int
	}

	if relation, ok := DB.NewScope(&User{}).FieldByName("Profile"); ok {
		if relation.Relationship.Kind != "belongs_to" ||
			!reflect.DeepEqual(relation.Relationship.ForeignFieldNames, []string{"ProfileID"}) ||
			!reflect.DeepEqual(relation.Relationship.AssociationForeignFieldNames, []string{"Refer"}) {
			t.Errorf("Override belongs to foreign key with tag")
		}
	}
}

func TestHasOne(t *testing.T) {
	user := User{
		Name:       "has one",
		CreditCard: CreditCard{Number: "411111111111"},
	}

	if err := DB.Save(&user).Error; err != nil {
		t.Error("Got errors when save user", err.Error())
	}

	if user.CreditCard.UserId.Int64 == 0 {
		t.Errorf("CreditCard's foreign key should be updated")
	}

	// Query
	var creditCard1 CreditCard
	DB.Model(&user).Related(&creditCard1)

	if creditCard1.Number != "411111111111" {
		t.Errorf("Query has one relations with Related")
	}

	var creditCard11 CreditCard
	DB.Model(&user).Association("CreditCard").Find(&creditCard11)

	if creditCard11.Number != "411111111111" {
		t.Errorf("Query has one relations with Related")
	}

	if DB.Model(&user).Association("CreditCard").Count() != 1 {
		t.Errorf("User's credit card count should be 1")
	}

	// Append
	var creditcard2 = CreditCard{
		Number: "411111111112",
	}
	DB.Model(&user).Association("CreditCard").Append(&creditcard2)

	if creditcard2.ID == 0 {
		t.Errorf("Creditcard should has ID when created with Append")
	}

	var creditcard21 CreditCard
	DB.Model(&user).Related(&creditcard21)
	if creditcard21.Number != "411111111112" {
		t.Errorf("CreditCard should be updated with Append")
	}

	if DB.Model(&user).Association("CreditCard").Count() != 1 {
		t.Errorf("User's credit card count should be 1")
	}

	// Replace
	var creditcard3 = CreditCard{
		Number: "411111111113",
	}
	DB.Model(&user).Association("CreditCard").Replace(&creditcard3)

	if creditcard3.ID == 0 {
		t.Errorf("Creditcard should has ID when created with Replace")
	}

	var creditcard31 CreditCard
	DB.Model(&user).Related(&creditcard31)
	if creditcard31.Number != "411111111113" {
		t.Errorf("CreditCard should be updated with Replace")
	}

	if DB.Model(&user).Association("CreditCard").Count() != 1 {
		t.Errorf("User's credit card count should be 1")
	}

	// Delete
	DB.Model(&user).Association("CreditCard").Delete(&creditcard2)
	var creditcard4 CreditCard
	DB.Model(&user).Related(&creditcard4)
	if creditcard4.Number != "411111111113" {
		t.Errorf("Should not delete credit card when Delete a unrelated CreditCard")
	}

	if DB.Model(&user).Association("CreditCard").Count() != 1 {
		t.Errorf("User's credit card count should be 1")
	}

	DB.Model(&user).Association("CreditCard").Delete(&creditcard3)
	if !DB.Model(&user).Related(&CreditCard{}).RecordNotFound() {
		t.Errorf("Should delete credit card with Delete")
	}

	if DB.Model(&user).Association("CreditCard").Count() != 0 {
		t.Errorf("User's credit card count should be 0 after Delete")
	}

	// Clear
	var creditcard5 = CreditCard{
		Number: "411111111115",
	}
	DB.Model(&user).Association("CreditCard").Append(&creditcard5)

	if DB.Model(&user).Related(&CreditCard{}).RecordNotFound() {
		t.Errorf("Should added credit card with Append")
	}

	if DB.Model(&user).Association("CreditCard").Count() != 1 {
		t.Errorf("User's credit card count should be 1")
	}

	DB.Model(&user).Association("CreditCard").Clear()
	if !DB.Model(&user).Related(&CreditCard{}).RecordNotFound() {
		t.Errorf("Credit card should be deleted with Clear")
	}

	if DB.Model(&user).Association("CreditCard").Count() != 0 {
		t.Errorf("User's credit card count should be 0 after Clear")
	}

	// Check Association mode with soft delete
	var creditcard6 = CreditCard{
		Number: "411111111116",
	}
	DB.Model(&user).Association("CreditCard").Append(&creditcard6)

	if count := DB.Model(&user).Association("CreditCard").Count(); count != 1 {
		t.Errorf("User's credit card count should be 1 after Append, but got %v", count)
	}

	DB.Delete(&creditcard6)

	if count := DB.Model(&user).Association("CreditCard").Count(); count != 0 {
		t.Errorf("User's credit card count should be 0 after credit card deleted, but got %v", count)
	}

	if err := DB.Model(&user).Association("CreditCard").Find(&CreditCard{}).Error; err == nil {
		t.Errorf("User's creditcard is not findable after Delete")
	}

	if count := DB.Unscoped().Model(&user).Association("CreditCard").Count(); count != 1 {
		t.Errorf("User's credit card count should be 1 when query with Unscoped, but got %v", count)
	}

	if err := DB.Unscoped().Model(&user).Association("CreditCard").Find(&CreditCard{}).Error; err != nil {
		t.Errorf("User's creditcard should be findable when query with Unscoped, got %v", err)
	}
}

func TestHasOneOverrideForeignKey1(t *testing.T) {
	type Profile struct {
		gorm.Model
		Name      string
		UserRefer uint
	}

	type User struct {
		gorm.Model
		Profile Profile `gorm:"ForeignKey:UserRefer"`
	}

	if relation, ok := DB.NewScope(&User{}).FieldByName("Profile"); ok {
		if relation.Relationship.Kind != "has_one" ||
			!reflect.DeepEqual(relation.Relationship.ForeignFieldNames, []string{"UserRefer"}) ||
			!reflect.DeepEqual(relation.Relationship.AssociationForeignFieldNames, []string{"ID"}) {
			t.Errorf("Override belongs to foreign key with tag")
		}
	}
}

func TestHasOneOverrideForeignKey2(t *testing.T) {
	type Profile struct {
		gorm.Model
		Name   string
		UserID uint
	}

	type User struct {
		gorm.Model
		Refer   string
		Profile Profile `gorm:"ForeignKey:UserID;AssociationForeignKey:Refer"`
	}

	if relation, ok := DB.NewScope(&User{}).FieldByName("Profile"); ok {
		if relation.Relationship.Kind != "has_one" ||
			!reflect.DeepEqual(relation.Relationship.ForeignFieldNames, []string{"UserID"}) ||
			!reflect.DeepEqual(relation.Relationship.AssociationForeignFieldNames, []string{"Refer"}) {
			t.Errorf("Override belongs to foreign key with tag")
		}
	}
}

func TestHasMany(t *testing.T) {
	post := Post{
		Title:    "post has many",
		Body:     "body has many",
		Comments: []*Comment{{Content: "Comment 1"}, {Content: "Comment 2"}},
	}

	if err := DB.Save(&post).Error; err != nil {
		t.Error("Got errors when save post", err)
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

	if DB.Model(&post).Association("Comments").Count() != 2 {
		t.Errorf("Post's comments count should be 2")
	}

	// Append
	DB.Model(&post).Association("Comments").Append(&Comment{Content: "Comment 3"})

	var comments2 []Comment
	DB.Model(&post).Related(&comments2)
	if !compareComments(comments2, []string{"Comment 1", "Comment 2", "Comment 3"}) {
		t.Errorf("Append new record to has many relations")
	}

	if DB.Model(&post).Association("Comments").Count() != 3 {
		t.Errorf("Post's comments count should be 3 after Append")
	}

	// Delete
	DB.Model(&post).Association("Comments").Delete(comments11)

	var comments3 []Comment
	DB.Model(&post).Related(&comments3)
	if !compareComments(comments3, []string{"Comment 3"}) {
		t.Errorf("Delete an existing resource for has many relations")
	}

	if DB.Model(&post).Association("Comments").Count() != 1 {
		t.Errorf("Post's comments count should be 1 after Delete 2")
	}

	// Replace
	DB.Model(&Post{Id: 999}).Association("Comments").Replace()

	var comments4 []Comment
	DB.Model(&post).Related(&comments4)
	if len(comments4) == 0 {
		t.Errorf("Replace for other resource should not clear all comments")
	}

	DB.Model(&post).Association("Comments").Replace(&Comment{Content: "Comment 4"}, &Comment{Content: "Comment 5"})

	var comments41 []Comment
	DB.Model(&post).Related(&comments41)
	if !compareComments(comments41, []string{"Comment 4", "Comment 5"}) {
		t.Errorf("Replace has many relations")
	}

	// Clear
	DB.Model(&Post{Id: 999}).Association("Comments").Clear()

	var comments5 []Comment
	DB.Model(&post).Related(&comments5)
	if len(comments5) == 0 {
		t.Errorf("Clear should not clear all comments")
	}

	DB.Model(&post).Association("Comments").Clear()

	var comments51 []Comment
	DB.Model(&post).Related(&comments51)
	if len(comments51) != 0 {
		t.Errorf("Clear has many relations")
	}

	// Check Association mode with soft delete
	var comment6 = Comment{
		Content: "comment 6",
	}
	DB.Model(&post).Association("Comments").Append(&comment6)

	if count := DB.Model(&post).Association("Comments").Count(); count != 1 {
		t.Errorf("post's comments count should be 1 after Append, but got %v", count)
	}

	DB.Delete(&comment6)

	if count := DB.Model(&post).Association("Comments").Count(); count != 0 {
		t.Errorf("post's comments count should be 0 after comment been deleted, but got %v", count)
	}

	var comments6 []Comment
	if DB.Model(&post).Association("Comments").Find(&comments6); len(comments6) != 0 {
		t.Errorf("post's comments count should be 0 when find with Find, but got %v", len(comments6))
	}

	if count := DB.Unscoped().Model(&post).Association("Comments").Count(); count != 1 {
		t.Errorf("post's comments count should be 1 when query with Unscoped, but got %v", count)
	}

	var comments61 []Comment
	if DB.Unscoped().Model(&post).Association("Comments").Find(&comments61); len(comments61) != 1 {
		t.Errorf("post's comments count should be 1 when query with Unscoped, but got %v", len(comments61))
	}
}

func TestHasManyOverrideForeignKey1(t *testing.T) {
	type Profile struct {
		gorm.Model
		Name      string
		UserRefer uint
	}

	type User struct {
		gorm.Model
		Profile []Profile `gorm:"ForeignKey:UserRefer"`
	}

	if relation, ok := DB.NewScope(&User{}).FieldByName("Profile"); ok {
		if relation.Relationship.Kind != "has_many" ||
			!reflect.DeepEqual(relation.Relationship.ForeignFieldNames, []string{"UserRefer"}) ||
			!reflect.DeepEqual(relation.Relationship.AssociationForeignFieldNames, []string{"ID"}) {
			t.Errorf("Override belongs to foreign key with tag")
		}
	}
}

func TestHasManyOverrideForeignKey2(t *testing.T) {
	type Profile struct {
		gorm.Model
		Name   string
		UserID uint
	}

	type User struct {
		gorm.Model
		Refer   string
		Profile []Profile `gorm:"ForeignKey:UserID;AssociationForeignKey:Refer"`
	}

	if relation, ok := DB.NewScope(&User{}).FieldByName("Profile"); ok {
		if relation.Relationship.Kind != "has_many" ||
			!reflect.DeepEqual(relation.Relationship.ForeignFieldNames, []string{"UserID"}) ||
			!reflect.DeepEqual(relation.Relationship.AssociationForeignFieldNames, []string{"Refer"}) {
			t.Errorf("Override belongs to foreign key with tag")
		}
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

	// Check Association mode with soft delete
	var language6 = Language{
		Name: "language 6",
	}
	DB.Model(&user).Association("Languages").Append(&language6)

	if count := DB.Model(&user).Association("Languages").Count(); count != 1 {
		t.Errorf("user's languages count should be 1 after Append, but got %v", count)
	}

	DB.Delete(&language6)

	if count := DB.Model(&user).Association("Languages").Count(); count != 0 {
		t.Errorf("user's languages count should be 0 after language been deleted, but got %v", count)
	}

	var languages6 []Language
	if DB.Model(&user).Association("Languages").Find(&languages6); len(languages6) != 0 {
		t.Errorf("user's languages count should be 0 when find with Find, but got %v", len(languages6))
	}

	if count := DB.Unscoped().Model(&user).Association("Languages").Count(); count != 1 {
		t.Errorf("user's languages count should be 1 when query with Unscoped, but got %v", count)
	}

	var languages61 []Language
	if DB.Unscoped().Model(&user).Association("Languages").Find(&languages61); len(languages61) != 1 {
		t.Errorf("user's languages count should be 1 when query with Unscoped, but got %v", len(languages61))
	}
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

	if err := DB.Save(&user).Error; err != nil {
		t.Errorf("No error should happen when saving user")
	}

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
