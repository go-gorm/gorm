package gorm_test

import "testing"

func TestHasOneAndHasManyAssociation(t *testing.T) {
	db.DropTable(Category{})
	db.DropTable(Post{})
	db.DropTable(Comment{})

	db.CreateTable(Category{})
	db.CreateTable(Post{})
	db.CreateTable(Comment{})

	post := Post{
		Title:        "post 1",
		Body:         "body 1",
		Comments:     []Comment{{Content: "Comment 1"}, {Content: "Comment 2"}},
		Category:     Category{Name: "Category 1"},
		MainCategory: Category{Name: "Main Category 1"},
	}

	if err := db.Save(&post).Error; err != nil {
		t.Errorf("Got errors when save post")
	}

	if db.First(&Category{}, "name = ?", "Category 1").Error != nil {
		t.Errorf("Category should be saved")
	}

	var p Post
	db.First(&p, post.Id)

	if post.CategoryId.Int64 == 0 || p.CategoryId.Int64 == 0 || post.MainCategoryId == 0 || p.MainCategoryId == 0 {
		t.Errorf("Category Id should exist")
	}

	if db.First(&Comment{}, "content = ?", "Comment 1").Error != nil {
		t.Errorf("Comment 1 should be saved")
	}
	if post.Comments[0].PostId == 0 {
		t.Errorf("Comment Should have post id")
	}

	var comment Comment
	if db.First(&comment, "content = ?", "Comment 2").Error != nil {
		t.Errorf("Comment 2 should be saved")
	}

	if comment.PostId == 0 {
		t.Errorf("Comment 2 Should have post id")
	}

	comment3 := Comment{Content: "Comment 3", Post: Post{Title: "Title 3", Body: "Body 3"}}
	db.Save(&comment3)
}

func TestRelated(t *testing.T) {
	user := User{
		Name:            "jinzhu",
		BillingAddress:  Address{Address1: "Billing Address - Address 1"},
		ShippingAddress: Address{Address1: "Shipping Address - Address 1"},
		Emails:          []Email{{Email: "jinzhu@example.com"}, {Email: "jinzhu-2@example@example.com"}},
		CreditCard:      CreditCard{Number: "1234567890"},
	}

	db.Save(&user)

	if user.CreditCard.Id == 0 {
		t.Errorf("After user save, credit card should have id")
	}

	if user.BillingAddress.Id == 0 {
		t.Errorf("After user save, billing address should have id")
	}

	if user.Emails[0].Id == 0 {
		t.Errorf("After user save, billing address should have id")
	}

	var emails []Email
	db.Model(&user).Related(&emails)
	if len(emails) != 2 {
		t.Errorf("Should have two emails")
	}

	var emails2 []Email
	db.Model(&user).Where("email = ?", "jinzhu@example.com").Related(&emails2)
	if len(emails2) != 1 {
		t.Errorf("Should have two emails")
	}

	var user1 User
	db.Model(&user).Related(&user1.Emails)
	if len(user1.Emails) != 2 {
		t.Errorf("Should have only one email match related condition")
	}

	var address1 Address
	db.Model(&user).Related(&address1, "BillingAddressId")
	if address1.Address1 != "Billing Address - Address 1" {
		t.Errorf("Should get billing address from user correctly")
	}

	user1 = User{}
	db.Model(&address1).Related(&user1, "BillingAddressId")
	if db.NewRecord(user1) {
		t.Errorf("Should get user from address correctly")
	}

	var user2 User
	db.Model(&emails[0]).Related(&user2)
	if user2.Id != user.Id || user2.Name != user.Name {
		t.Errorf("Should get user from email correctly")
	}

	var creditcard CreditCard
	var user3 User
	db.First(&creditcard, "number = ?", "1234567890")
	db.Model(&creditcard).Related(&user3)
	if user3.Id != user.Id || user3.Name != user.Name {
		t.Errorf("Should get user from credit card correctly")
	}

	if !db.Model(&CreditCard{}).Related(&User{}).RecordNotFound() {
		t.Errorf("RecordNotFound for Related")
	}
}

func TestManyToMany(t *testing.T) {
	db.Raw("delete from languages")
	var languages = []Language{{Name: "ZH"}, {Name: "EN"}}
	user := User{Name: "Many2Many", Languages: languages}
	db.Save(&user)

	// Query
	var newLanguages []Language
	db.Model(&user).Related(&newLanguages, "Languages")
	if len(newLanguages) != len([]string{"ZH", "EN"}) {
		t.Errorf("Query many to many relations")
	}

	newLanguages = []Language{}
	db.Model(&user).Association("Languages").Find(&newLanguages)
	if len(newLanguages) != len([]string{"ZH", "EN"}) {
		t.Errorf("Should be able to find many to many relations")
	}

	if db.Model(&user).Association("Languages").Count() != len([]string{"ZH", "EN"}) {
		t.Errorf("Count should return correct result")
	}

	// Append
	db.Model(&user).Association("Languages").Append(&Language{Name: "DE"})
	if db.Where("name = ?", "DE").First(&Language{}).RecordNotFound() {
		t.Errorf("New record should be saved when append")
	}

	languageA := Language{Name: "AA"}
	db.Save(&languageA)
	db.Model(&User{Id: user.Id}).Association("Languages").Append(languageA)
	languageC := Language{Name: "CC"}
	db.Save(&languageC)
	db.Model(&user).Association("Languages").Append(&[]Language{{Name: "BB"}, languageC})
	db.Model(&User{Id: user.Id}).Association("Languages").Append([]Language{{Name: "DD"}, {Name: "EE"}})

	totalLanguages := []string{"ZH", "EN", "DE", "AA", "BB", "CC", "DD", "EE"}

	if db.Model(&user).Association("Languages").Count() != len(totalLanguages) {
		t.Errorf("All appended languages should be saved")
	}

	// Delete
	var language Language
	db.Where("name = ?", "EE").First(&language)
	db.Model(&user).Association("Languages").Delete(language, &language)
	if db.Model(&user).Association("Languages").Count() != len(totalLanguages)-1 {
		t.Errorf("Relations should be deleted with Delete")
	}
	if db.Where("name = ?", "EE").First(&Language{}).RecordNotFound() {
		t.Errorf("Language EE should not be deleted")
	}

	languages = []Language{}
	db.Where("name IN (?)", []string{"CC", "DD"}).Find(&languages)
	db.Model(&user).Association("Languages").Delete(languages, &languages)
	if db.Model(&user).Association("Languages").Count() != len(totalLanguages)-3 {
		t.Errorf("Relations should be deleted with Delete")
	}

	// Replace
	var languageB Language
	db.Where("name = ?", "BB").First(&languageB)
	db.Model(&user).Association("Languages").Replace(languageB)
	if db.Model(&user).Association("Languages").Count() != 1 {
		t.Errorf("Relations should be replaced")
	}

	db.Model(&user).Association("Languages").Replace(&[]Language{{Name: "FF"}, {Name: "JJ"}})
	if db.Model(&user).Association("Languages").Count() != len([]string{"FF", "JJ"}) {
		t.Errorf("Relations should be replaced")
	}

	// Clear
	db.Model(&user).Association("Languages").Clear()
	if db.Model(&user).Association("Languages").Count() != 0 {
		t.Errorf("Relations should be cleared")
	}
}
