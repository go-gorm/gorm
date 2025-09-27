package tests_test

import (
	"errors"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	. "gorm.io/gorm/utils/tests"
)

func TestDelete(t *testing.T) {
	users := []User{*GetUser("delete", Config{}), *GetUser("delete", Config{}), *GetUser("delete", Config{})}

	if err := DB.Create(&users).Error; err != nil {
		t.Errorf("errors happened when create: %v", err)
	}

	for _, user := range users {
		if user.ID == 0 {
			t.Fatalf("user's primary key should has value after create, got : %v", user.ID)
		}
	}

	if res := DB.Delete(&users[1]); res.Error != nil || res.RowsAffected != 1 {
		t.Errorf("errors happened when delete: %v, affected: %v", res.Error, res.RowsAffected)
	}

	var result User
	if err := DB.Where("id = ?", users[1].ID).First(&result).Error; err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("should returns record not found error, but got %v", err)
	}

	for _, user := range []User{users[0], users[2]} {
		result = User{}
		if err := DB.Where("id = ?", user.ID).First(&result).Error; err != nil {
			t.Errorf("no error should returns when query %v, but got %v", user.ID, err)
		}
	}

	for _, user := range []User{users[0], users[2]} {
		result = User{}
		if err := DB.Where("id = ?", user.ID).First(&result).Error; err != nil {
			t.Errorf("no error should returns when query %v, but got %v", user.ID, err)
		}
	}

	if err := DB.Delete(&users[0]).Error; err != nil {
		t.Errorf("errors happened when delete: %v", err)
	}

	if err := DB.Delete(&User{}).Error; err != gorm.ErrMissingWhereClause {
		t.Errorf("errors happened when delete: %v", err)
	}

	if err := DB.Where("id = ?", users[0].ID).First(&result).Error; err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("should returns record not found error, but got %v", err)
	}
}

func TestDeleteWithTable(t *testing.T) {
	type UserWithDelete struct {
		gorm.Model
		Name string
	}

	DB.Table("deleted_users").Migrator().DropTable(UserWithDelete{})
	DB.Table("deleted_users").AutoMigrate(UserWithDelete{})

	user := UserWithDelete{Name: "delete1"}
	DB.Table("deleted_users").Create(&user)

	var result UserWithDelete
	if err := DB.Table("deleted_users").First(&result).Error; err != nil {
		t.Errorf("failed to find deleted user, got error %v", err)
	}

	AssertEqual(t, result, user)

	if err := DB.Table("deleted_users").Delete(&result).Error; err != nil {
		t.Errorf("failed to delete user, got error %v", err)
	}

	var result2 UserWithDelete
	if err := DB.Table("deleted_users").First(&result2, user.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("should raise record not found error, but got error %v", err)
	}

	var result3 UserWithDelete
	if err := DB.Table("deleted_users").Unscoped().First(&result3, user.ID).Error; err != nil {
		t.Fatalf("failed to find record, got error %v", err)
	}

	if err := DB.Table("deleted_users").Unscoped().Delete(&result).Error; err != nil {
		t.Errorf("failed to delete user with unscoped, got error %v", err)
	}

	var result4 UserWithDelete
	if err := DB.Table("deleted_users").Unscoped().First(&result4, user.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("should raise record not found error, but got error %v", err)
	}
}

func TestInlineCondDelete(t *testing.T) {
	user1 := *GetUser("inline_delete_1", Config{})
	user2 := *GetUser("inline_delete_2", Config{})
	DB.Save(&user1).Save(&user2)

	if DB.Delete(&User{}, user1.ID).Error != nil {
		t.Errorf("No error should happen when delete a record")
	} else if err := DB.Where("name = ?", user1.Name).First(&User{}).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("User can't be found after delete")
	}

	if err := DB.Delete(&User{}, "name = ?", user2.Name).Error; err != nil {
		t.Errorf("No error should happen when delete a record, err=%s", err)
	} else if err := DB.Where("name = ?", user2.Name).First(&User{}).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("User can't be found after delete")
	}
}

func TestBlockGlobalDelete(t *testing.T) {
	if err := DB.Delete(&User{}).Error; err == nil || !errors.Is(err, gorm.ErrMissingWhereClause) {
		t.Errorf("should returns missing WHERE clause while deleting error")
	}

	if err := DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&User{}).Error; err != nil {
		t.Errorf("should returns no error while enable global update, but got err %v", err)
	}
}

func TestDeleteWithAssociations(t *testing.T) {
	user := GetUser("delete_with_associations", Config{Account: true, Pets: 2, Toys: 4, Company: true, Manager: true, Team: 1, Languages: 1, Friends: 1})

	if err := DB.Create(user).Error; err != nil {
		t.Fatalf("failed to create user, got error %v", err)
	}

	if err := DB.Select(clause.Associations, "Pets.Toy").Delete(&user).Error; err != nil {
		t.Fatalf("failed to delete user, got error %v", err)
	}

	for key, value := range map[string]int64{"Account": 1, "Pets": 2, "Toys": 4, "Company": 1, "Manager": 1, "Team": 1, "Languages": 0, "Friends": 0} {
		if count := DB.Unscoped().Model(&user).Association(key).Count(); count != value {
			t.Errorf("user's %v expects: %v, got %v", key, value, count)
		}
	}

	for key, value := range map[string]int64{"Account": 0, "Pets": 0, "Toys": 0, "Company": 1, "Manager": 1, "Team": 0, "Languages": 0, "Friends": 0} {
		if count := DB.Model(&user).Association(key).Count(); count != value {
			t.Errorf("user's %v expects: %v, got %v", key, value, count)
		}
	}
}

func TestDeleteAssociationsWithUnscoped(t *testing.T) {
	user := GetUser("unscoped_delete_with_associations", Config{Account: true, Pets: 2, Toys: 4, Company: true, Manager: true, Team: 1, Languages: 1, Friends: 1})

	if err := DB.Create(user).Error; err != nil {
		t.Fatalf("failed to create user, got error %v", err)
	}

	if err := DB.Unscoped().Select(clause.Associations, "Pets.Toy").Delete(&user).Error; err != nil {
		t.Fatalf("failed to delete user, got error %v", err)
	}

	for key, value := range map[string]int64{"Account": 0, "Pets": 0, "Toys": 0, "Company": 1, "Manager": 1, "Team": 0, "Languages": 0, "Friends": 0} {
		if count := DB.Unscoped().Model(&user).Association(key).Count(); count != value {
			t.Errorf("user's %v expects: %v, got %v", key, value, count)
		}
	}

	for key, value := range map[string]int64{"Account": 0, "Pets": 0, "Toys": 0, "Company": 1, "Manager": 1, "Team": 0, "Languages": 0, "Friends": 0} {
		if count := DB.Model(&user).Association(key).Count(); count != value {
			t.Errorf("user's %v expects: %v, got %v", key, value, count)
		}
	}
}

func TestDeleteSliceWithAssociations(t *testing.T) {
	users := []User{
		*GetUser("delete_slice_with_associations1", Config{Account: true, Pets: 4, Toys: 1, Company: true, Manager: true, Team: 1, Languages: 1, Friends: 4}),
		*GetUser("delete_slice_with_associations2", Config{Account: true, Pets: 3, Toys: 2, Company: true, Manager: true, Team: 2, Languages: 2, Friends: 3}),
		*GetUser("delete_slice_with_associations3", Config{Account: true, Pets: 2, Toys: 3, Company: true, Manager: true, Team: 3, Languages: 3, Friends: 2}),
		*GetUser("delete_slice_with_associations4", Config{Account: true, Pets: 1, Toys: 4, Company: true, Manager: true, Team: 4, Languages: 4, Friends: 1}),
	}

	if err := DB.Create(users).Error; err != nil {
		t.Fatalf("failed to create user, got error %v", err)
	}

	if err := DB.Select(clause.Associations).Delete(&users).Error; err != nil {
		t.Fatalf("failed to delete user, got error %v", err)
	}

	for key, value := range map[string]int64{"Account": 4, "Pets": 10, "Toys": 10, "Company": 4, "Manager": 4, "Team": 10, "Languages": 0, "Friends": 0} {
		if count := DB.Unscoped().Model(&users).Association(key).Count(); count != value {
			t.Errorf("user's %v expects: %v, got %v", key, value, count)
		}
	}

	for key, value := range map[string]int64{"Account": 0, "Pets": 0, "Toys": 0, "Company": 4, "Manager": 4, "Team": 0, "Languages": 0, "Friends": 0} {
		if count := DB.Model(&users).Association(key).Count(); count != value {
			t.Errorf("user's %v expects: %v, got %v", key, value, count)
		}
	}
}

// only sqlite, postgres, gaussdb, sqlserver support returning
func TestSoftDeleteReturning(t *testing.T) {
	if DB.Dialector.Name() != "sqlite" && DB.Dialector.Name() != "postgres" && DB.Dialector.Name() != "gaussdb" && DB.Dialector.Name() != "sqlserver" {
		return
	}

	users := []*User{
		GetUser("delete-returning-1", Config{}),
		GetUser("delete-returning-2", Config{}),
		GetUser("delete-returning-3", Config{}),
	}
	DB.Create(&users)

	var results []User
	DB.Where("name IN ?", []string{users[0].Name, users[1].Name}).Clauses(clause.Returning{}).Delete(&results)
	if len(results) != 2 {
		t.Errorf("failed to return delete data, got %v", results)
	}

	var count int64
	DB.Model(&User{}).Where("name IN ?", []string{users[0].Name, users[1].Name, users[2].Name}).Count(&count)
	if count != 1 {
		t.Errorf("failed to delete data, current count %v", count)
	}
}

func TestDeleteReturning(t *testing.T) {
	if DB.Dialector.Name() != "sqlite" && DB.Dialector.Name() != "postgres" && DB.Dialector.Name() != "gaussdb" && DB.Dialector.Name() != "sqlserver" {
		return
	}

	companies := []Company{
		{Name: "delete-returning-1"},
		{Name: "delete-returning-2"},
		{Name: "delete-returning-3"},
	}
	DB.Create(&companies)

	var results []Company
	DB.Where("name IN ?", []string{companies[0].Name, companies[1].Name}).Clauses(clause.Returning{}).Delete(&results)
	if len(results) != 2 {
		t.Errorf("failed to return delete data, got %v", results)
	}

	var count int64
	DB.Model(&Company{}).Where("name IN ?", []string{companies[0].Name, companies[1].Name, companies[2].Name}).Count(&count)
	if count != 1 {
		t.Errorf("failed to delete data, current count %v", count)
	}
}

func TestNestedDelete(t *testing.T) {
	type NestedProfile struct {
		gorm.Model
		Name           string
		NestedUserID   uint
	}

	type NestedUser struct {
		gorm.Model
		Name     string
		Profiles []NestedProfile `gorm:"foreignKey:NestedUserID"`
	}

	DB.Migrator().DropTable(&NestedProfile{}, &NestedUser{})
	if err := DB.AutoMigrate(&NestedUser{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}
	if err := DB.AutoMigrate(&NestedProfile{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	user := NestedUser{Name: "nested_delete_test", Profiles: []NestedProfile{
		{Name: "Profile1"},
		{Name: "Profile2"},
	}}

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user, got error %v", err)
	}
	t.Logf("Created user with ID: %d", user.ID)

	var deletedUser NestedUser
	result := DB.Select("Profiles").Delete(&deletedUser, user.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete user with nested select, got error %v", result.Error)
	}

	var count int64
	DB.Model(&NestedUser{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 users after nested delete, got %d", count)
	}

	DB.Model(&NestedProfile{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 profiles after nested delete, got %d", count)
	}
}


func TestNestedDeleteWithBelongsTo(t *testing.T) {
	type Author struct {
		gorm.Model
		Name string
	}

	type Book struct {
		gorm.Model
		Title    string
		AuthorID uint
		Author   Author
	}

	DB.Migrator().DropTable(&Author{}, &Book{})
	if err := DB.AutoMigrate(&Author{}, &Book{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	author := Author{Name: "Test Author"}
	DB.Create(&author)

	book := Book{Title: "Test Book", AuthorID: author.ID}
	DB.Create(&book)

	var deletedBook Book
	result := DB.Select("Author").Delete(&deletedBook, book.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete book with nested BelongsTo, got error %v", result.Error)
	}

	var count int64
	DB.Model(&Book{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 books after nested delete with BelongsTo, got %d", count)
	}

	DB.Model(&Author{}).Count(&count)
	if count != 1 {
		t.Fatalf("Expected 1 author after nested delete with BelongsTo, got %d", count)
	}
}

func TestNestedDeleteWithManyToMany(t *testing.T) {
	type Tag struct {
		gorm.Model
		Name string
	}

	type Post struct {
		gorm.Model
		Title string
		Tags  []Tag `gorm:"many2many:post_tags;"`
	}

	DB.Migrator().DropTable(&Tag{}, &Post{}, "post_tags")
	if err := DB.AutoMigrate(&Tag{}, &Post{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	tag1 := Tag{Name: "Tag1"}
	tag2 := Tag{Name: "Tag2"}
	DB.Create(&tag1)
	DB.Create(&tag2)

	post := Post{Title: "Test Post", Tags: []Tag{tag1, tag2}}
	DB.Create(&post)

	var deletedPost Post
	result := DB.Select("Tags").Delete(&deletedPost, post.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete post with nested ManyToMany, got error %v", result.Error)
	}

	var count int64
	DB.Model(&Post{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 posts after nested delete with ManyToMany, got %d", count)
	}

	DB.Model(&Tag{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 tags after nested delete with ManyToMany, got %d", count)
	}

	DB.Table("post_tags").Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 join table records after nested delete with ManyToMany, got %d", count)
	}
}

func TestNestedDeleteWithEmbeddedStruct(t *testing.T) {
	type Address struct {
		Street string
		City   string
	}

	type User struct {
		gorm.Model
		Name    string
		Address Address `gorm:"embedded"`
	}

	DB.Migrator().DropTable(&User{})
	if err := DB.AutoMigrate(&User{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	user := User{
		Name: "embedded_delete_test",
		Address: Address{
			Street: "123 Main St",
			City:   "Test City",
		},
	}

	DB.Create(&user)

	var deletedUser User
	result := DB.Delete(&deletedUser, user.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete user with embedded struct, got error %v", result.Error)
	}

	var count int64
	DB.Model(&User{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 users after delete with embedded struct, got %d", count)
	}
}

func TestNestedDeleteDeepNesting(t *testing.T) {
	type Comment struct {
		gorm.Model
		Content string
		PostID  uint
	}

	type Post struct {
		gorm.Model
		Title    string
		UserID   uint
		Comments []Comment
	}

	type User struct {
		gorm.Model
		Name  string
		Posts []Post
	}

	DB.Migrator().DropTable(&Comment{}, &Post{}, &User{})
	if err := DB.AutoMigrate(&User{}, &Post{}, &Comment{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	user := User{Name: "deep_nesting_test", Posts: []Post{
		{Title: "Post1", Comments: []Comment{
			{Content: "Comment1"},
			{Content: "Comment2"},
		}},
		{Title: "Post2", Comments: []Comment{
			{Content: "Comment3"},
		}},
	}}
	DB.Create(&user)

	var deletedUser User
	result := DB.Select("Posts.Comments").Delete(&deletedUser, user.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete user with deep nesting, got error %v", result.Error)
	}

	var count int64
	DB.Model(&User{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 users after deep nested delete, got %d", count)
	}
	DB.Model(&Post{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 posts after deep nested delete, got %d", count)
	}
	DB.Model(&Comment{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 comments after deep nested delete, got %d", count)
	}
}

func TestNestedDeleteMultipleRelations(t *testing.T) {
	type MultiProfile struct {
		gorm.Model
		Name        string
		MultiUserID uint
	}

	type MultiPost struct {
		gorm.Model
		Title       string
		MultiUserID uint
	}

	type MultiUser struct {
		gorm.Model
		Name     string
		Profiles []MultiProfile `gorm:"foreignKey:MultiUserID"`
		Posts    []MultiPost    `gorm:"foreignKey:MultiUserID"`
	}

	DB.Migrator().DropTable(&MultiProfile{}, &MultiPost{}, &MultiUser{})
	if err := DB.AutoMigrate(&MultiUser{}, &MultiPost{}, &MultiProfile{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	user1 := MultiUser{Name: "multi_relation_test1", Profiles: []MultiProfile{{Name: "Profile1"}}}
	DB.Create(&user1)

	var deletedUser1 MultiUser
	result := DB.Select("Profiles").Delete(&deletedUser1, user1.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete user with Profiles relation, got error %v", result.Error)
	}
	
	user2 := MultiUser{Name: "multi_relation_test2", Posts: []MultiPost{{Title: "Post1"}}}
	DB.Create(&user2)
	
	var deletedUser2 MultiUser
	result = DB.Select("Posts").Delete(&deletedUser2, user2.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete user with Posts relation, got error %v", result.Error)
	}

	var count int64
	DB.Model(&MultiUser{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 users after multi-relation delete, got %d", count)
	}
	DB.Model(&MultiPost{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 posts after multi-relation delete, got %d", count)
	}
	DB.Model(&MultiProfile{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 profiles after multi-relation delete, got %d", count)
	}
}


func TestNestedDeleteWithPolymorphic(t *testing.T) {
	type Toy struct {
		gorm.Model
		Name      string
		OwnerID   uint
		OwnerType string
	}

	type Cat struct {
		gorm.Model
		Name string
		Toys []Toy `gorm:"polymorphic:Owner;"`
	}

	DB.Migrator().DropTable(&Toy{}, &Cat{})
	if err := DB.AutoMigrate(&Cat{}, &Toy{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	cat := Cat{Name: "Fluffy", Toys: []Toy{{Name: "Ball"}, {Name: "Mouse"}}}
	DB.Create(&cat)

	var deletedCat Cat
	result := DB.Select("Toys").Delete(&deletedCat, cat.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete cat with polymorphic toys, got error %v", result.Error)
	}

	var count int64
	DB.Model(&Cat{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 cats after polymorphic nested delete, got %d", count)
	}
	DB.Model(&Toy{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 toys after polymorphic nested delete, got %d", count)
	}
}

func TestNestedDeleteErrorHandling(t *testing.T) {
	type User struct {
		gorm.Model
		Name string
	}

	DB.Migrator().DropTable(&User{})
	if err := DB.AutoMigrate(&User{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	var user User
	result := DB.Select("NonExistentRelation").Delete(&user, 999)
	if result.Error == nil {
		t.Fatalf("Expected error for non-existent relationship, but got none")
	}

	result = DB.Select("Name").Delete(&user, 999)
	if result.Error == nil {
		t.Fatalf("Expected error for non-existent record, but got none")
	}
}

func TestNestedDeleteWithSelfReferential(t *testing.T) {
	type Category struct {
		gorm.Model
		Name       string
		ParentID   *uint
		Parent     *Category
		Children   []Category `gorm:"foreignKey:ParentID"`
	}

	DB.Migrator().DropTable(&Category{})
	if err := DB.AutoMigrate(&Category{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	parent := Category{Name: "Parent"}
	DB.Create(&parent)
	
	child1 := Category{Name: "Child1", ParentID: &parent.ID}
	child2 := Category{Name: "Child2", ParentID: &parent.ID}
	DB.Create(&child1)
	DB.Create(&child2)

	var deletedParent Category
	result := DB.Select("Children").Delete(&deletedParent, parent.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete parent with children, got error %v", result.Error)
	}

	var count int64
	DB.Model(&Category{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 categories after self-referential nested delete, got %d", count)
	}
}
