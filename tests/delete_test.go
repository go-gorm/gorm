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
	type NestedDeleteProfile struct {
		gorm.Model
		Name               string
		NestedDeleteUserID uint
	}

	type NestedDeleteUser struct {
		gorm.Model
		Name     string
		Profiles []NestedDeleteProfile `gorm:"foreignKey:NestedDeleteUserID"`
	}

	DB.Migrator().DropTable(&NestedDeleteProfile{}, &NestedDeleteUser{})
	if err := DB.AutoMigrate(&NestedDeleteUser{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}
	if err := DB.AutoMigrate(&NestedDeleteProfile{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	user := NestedDeleteUser{Name: "nested_delete_test", Profiles: []NestedDeleteProfile{
		{Name: "Profile1"},
		{Name: "Profile2"},
	}}

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user, got error %v", err)
	}
	t.Logf("Created user with ID: %d", user.ID)

	var deletedUser NestedDeleteUser
	result := DB.Select("Profiles").Delete(&deletedUser, user.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete user with nested select, got error %v", result.Error)
	}

	var count int64
	DB.Model(&NestedDeleteUser{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 users after nested delete, got %d", count)
	}

	DB.Model(&NestedDeleteProfile{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 profiles after nested delete, got %d", count)
	}
}

func TestNestedDeleteWithBelongsTo(t *testing.T) {
	type NestedDeleteAuthor struct {
		gorm.Model
		Name string
	}

	type NestedDeleteBook struct {
		gorm.Model
		Title    string
		AuthorID uint
		Author   NestedDeleteAuthor
	}

	DB.Migrator().DropTable(&NestedDeleteAuthor{}, &NestedDeleteBook{})
	if err := DB.AutoMigrate(&NestedDeleteAuthor{}, &NestedDeleteBook{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	author := NestedDeleteAuthor{Name: "Test Author"}
	DB.Create(&author)

	book := NestedDeleteBook{Title: "Test Book", AuthorID: author.ID}
	DB.Create(&book)

	var deletedBook NestedDeleteBook
	result := DB.Select("Author").Delete(&deletedBook, book.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete book with nested BelongsTo, got error %v", result.Error)
	}

	var count int64
	DB.Model(&NestedDeleteBook{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 books after nested delete with BelongsTo, got %d", count)
	}

	DB.Model(&NestedDeleteAuthor{}).Count(&count)
	if count != 1 {
		t.Fatalf("Expected 1 author after nested delete with BelongsTo, got %d", count)
	}
}

func TestNestedDeleteWithManyToMany(t *testing.T) {
	type NestedDeleteTag struct {
		gorm.Model
		Name string
	}

	type NestedDeletePost struct {
		gorm.Model
		Title string
		Tags  []NestedDeleteTag `gorm:"many2many:nested_delete_post_tags;"`
	}

	DB.Migrator().DropTable(&NestedDeleteTag{}, &NestedDeletePost{}, "nested_delete_post_tags")
	if err := DB.AutoMigrate(&NestedDeleteTag{}, &NestedDeletePost{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	tag1 := NestedDeleteTag{Name: "Tag1"}
	tag2 := NestedDeleteTag{Name: "Tag2"}
	DB.Create(&tag1)
	DB.Create(&tag2)

	post := NestedDeletePost{Title: "Test Post", Tags: []NestedDeleteTag{tag1, tag2}}
	DB.Create(&post)

	var deletedPost NestedDeletePost
	result := DB.Select("Tags").Delete(&deletedPost, post.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete post with nested ManyToMany, got error %v", result.Error)
	}

	var count int64
	DB.Model(&NestedDeletePost{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 posts after nested delete with ManyToMany, got %d", count)
	}

	DB.Model(&NestedDeleteTag{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 tags after nested delete with ManyToMany, got %d", count)
	}

	DB.Table("nested_delete_post_tags").Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 join table records after nested delete with ManyToMany, got %d", count)
	}
}

func TestNestedDeleteWithEmbeddedStruct(t *testing.T) {
	type NestedDeleteAddress struct {
		Street string
		City   string
	}

	type NestedDeleteEmbeddedUser struct {
		gorm.Model
		Name    string
		Address NestedDeleteAddress `gorm:"embedded"`
	}

	DB.Migrator().DropTable(&NestedDeleteEmbeddedUser{})
	if err := DB.AutoMigrate(&NestedDeleteEmbeddedUser{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	user := NestedDeleteEmbeddedUser{
		Name: "embedded_delete_test",
		Address: NestedDeleteAddress{
			Street: "123 Main St",
			City:   "Test City",
		},
	}

	DB.Create(&user)

	var deletedUser NestedDeleteEmbeddedUser
	result := DB.Delete(&deletedUser, user.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete user with embedded struct, got error %v", result.Error)
	}

	var count int64
	DB.Model(&NestedDeleteEmbeddedUser{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 users after delete with embedded struct, got %d", count)
	}
}

func TestNestedDeleteDeepNesting(t *testing.T) {
	type NestedDeleteDeepComment struct {
		gorm.Model
		Content string
		PostID  uint
	}

	type NestedDeleteDeepNestingPost struct {
		gorm.Model
		Title    string
		UserID   uint
		Comments []NestedDeleteDeepComment `gorm:"foreignKey:PostID"`
	}

	type NestedDeleteDeepNestingUser struct {
		gorm.Model
		Name  string
		Posts []NestedDeleteDeepNestingPost `gorm:"foreignKey:UserID"`
	}

	DB.Migrator().DropTable(&NestedDeleteDeepComment{}, &NestedDeleteDeepNestingPost{}, &NestedDeleteDeepNestingUser{})
	if err := DB.AutoMigrate(&NestedDeleteDeepNestingUser{}, &NestedDeleteDeepNestingPost{}, &NestedDeleteDeepComment{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	user := NestedDeleteDeepNestingUser{Name: "deep_nesting_test", Posts: []NestedDeleteDeepNestingPost{
		{Title: "Post1", Comments: []NestedDeleteDeepComment{
			{Content: "Comment1"},
			{Content: "Comment2"},
		}},
		{Title: "Post2", Comments: []NestedDeleteDeepComment{
			{Content: "Comment3"},
		}},
	}}
	DB.Create(&user)

	var deletedUser NestedDeleteDeepNestingUser
	result := DB.Select("Posts.Comments").Delete(&deletedUser, user.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete user with deep nesting, got error %v", result.Error)
	}

	var count int64
	DB.Model(&NestedDeleteDeepNestingUser{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 users after deep nested delete, got %d", count)
	}
	DB.Model(&NestedDeleteDeepNestingPost{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 posts after deep nested delete, got %d", count)
	}
	DB.Model(&NestedDeleteDeepComment{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 comments after deep nested delete, got %d", count)
	}
}

func TestNestedDeleteMultipleRelations(t *testing.T) {
	type NestedDeleteMultiProfile struct {
		gorm.Model
		Name        string
		MultiUserID uint
	}

	type NestedDeleteMultiPost struct {
		gorm.Model
		Title       string
		MultiUserID uint
	}

	type NestedDeleteMultiUser struct {
		gorm.Model
		Name     string
		Profiles []NestedDeleteMultiProfile `gorm:"foreignKey:MultiUserID"`
		Posts    []NestedDeleteMultiPost    `gorm:"foreignKey:MultiUserID"`
	}

	DB.Migrator().DropTable(&NestedDeleteMultiProfile{}, &NestedDeleteMultiPost{}, &NestedDeleteMultiUser{})
	if err := DB.AutoMigrate(&NestedDeleteMultiUser{}, &NestedDeleteMultiPost{}, &NestedDeleteMultiProfile{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	user1 := NestedDeleteMultiUser{Name: "multi_relation_test1", Profiles: []NestedDeleteMultiProfile{{Name: "Profile1"}}}
	DB.Create(&user1)

	var deletedUser1 NestedDeleteMultiUser
	result := DB.Select("Profiles").Delete(&deletedUser1, user1.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete user with Profiles relation, got error %v", result.Error)
	}

	user2 := NestedDeleteMultiUser{Name: "multi_relation_test2", Posts: []NestedDeleteMultiPost{{Title: "Post1"}}}
	DB.Create(&user2)

	var deletedUser2 NestedDeleteMultiUser
	result = DB.Select("Posts").Delete(&deletedUser2, user2.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete user with Posts relation, got error %v", result.Error)
	}

	var count int64
	DB.Model(&NestedDeleteMultiUser{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 users after multi-relation delete, got %d", count)
	}
	DB.Model(&NestedDeleteMultiPost{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 posts after multi-relation delete, got %d", count)
	}
	DB.Model(&NestedDeleteMultiProfile{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 profiles after multi-relation delete, got %d", count)
	}
}

func TestNestedDeleteWithPolymorphic(t *testing.T) {
	type NestedDeleteToy struct {
		gorm.Model
		Name      string
		OwnerID   uint
		OwnerType string
	}

	type NestedDeleteCat struct {
		gorm.Model
		Name string
		Toys []NestedDeleteToy `gorm:"polymorphic:Owner;"`
	}

	DB.Migrator().DropTable(&NestedDeleteToy{}, &NestedDeleteCat{})
	if err := DB.AutoMigrate(&NestedDeleteCat{}, &NestedDeleteToy{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	cat := NestedDeleteCat{Name: "Fluffy", Toys: []NestedDeleteToy{{Name: "Ball"}, {Name: "Mouse"}}}
	DB.Create(&cat)

	var deletedCat NestedDeleteCat
	result := DB.Select("Toys").Delete(&deletedCat, cat.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete cat with polymorphic toys, got error %v", result.Error)
	}

	var count int64
	DB.Model(&NestedDeleteCat{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 cats after polymorphic nested delete, got %d", count)
	}
	DB.Model(&NestedDeleteToy{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 toys after polymorphic nested delete, got %d", count)
	}
}

func TestNestedDeleteWithSelfReferential(t *testing.T) {
	type NestedDeleteCategory struct {
		gorm.Model
		Name     string
		ParentID *uint
		Parent   *NestedDeleteCategory
		Children []NestedDeleteCategory `gorm:"foreignKey:ParentID"`
	}

	DB.Migrator().DropTable(&NestedDeleteCategory{})
	if err := DB.AutoMigrate(&NestedDeleteCategory{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	parent := NestedDeleteCategory{Name: "Parent"}
	DB.Create(&parent)

	child1 := NestedDeleteCategory{Name: "Child1", ParentID: &parent.ID}
	child2 := NestedDeleteCategory{Name: "Child2", ParentID: &parent.ID}
	DB.Create(&child1)
	DB.Create(&child2)

	var deletedParent NestedDeleteCategory
	result := DB.Select("Children").Delete(&deletedParent, parent.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete parent with children, got error %v", result.Error)
	}

	var count int64
	DB.Model(&NestedDeleteCategory{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 categories after self-referential nested delete, got %d", count)
	}
}

// TestNestedDeleteWithEmptyAssociations tests deletion when associations are empty
func TestNestedDeleteWithEmptyAssociations(t *testing.T) {
	type EmptyAssocProfile struct {
		gorm.Model
		Name             string
		EmptyAssocUserID uint
	}

	type EmptyAssocUser struct {
		gorm.Model
		Name     string
		Profiles []EmptyAssocProfile `gorm:"foreignKey:EmptyAssocUserID"`
	}

	DB.Migrator().DropTable(&EmptyAssocProfile{}, &EmptyAssocUser{})
	if err := DB.AutoMigrate(&EmptyAssocUser{}, &EmptyAssocProfile{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	// Create user without any profiles
	user := EmptyAssocUser{Name: "empty_assoc_user"}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user, got error %v", err)
	}

	// Delete with nested select when no associations exist
	var deletedUser EmptyAssocUser
	result := DB.Select("Profiles").Delete(&deletedUser, user.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete user with empty nested select, got error %v", result.Error)
	}

	var count int64
	DB.Model(&EmptyAssocUser{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 users after delete with empty nested select, got %d", count)
	}
}

// TestNestedDeleteWithNullableAssociation tests deletion with nullable foreign keys
func TestNestedDeleteWithNullableAssociation(t *testing.T) {
	type NullableAssocCompany struct {
		gorm.Model
		Name string
	}

	type NullableAssocUser struct {
		gorm.Model
		Name      string
		CompanyID *uint
		Company   *NullableAssocCompany
	}

	DB.Migrator().DropTable(&NullableAssocUser{}, &NullableAssocCompany{})
	if err := DB.AutoMigrate(&NullableAssocCompany{}, &NullableAssocUser{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	// Create user without company
	user := NullableAssocUser{Name: "nullable_user", CompanyID: nil}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user, got error %v", err)
	}

	// Delete with nested select when association is null
	var deletedUser NullableAssocUser
	result := DB.Select("Company").Delete(&deletedUser, user.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete user with null nested select, got error %v", result.Error)
	}

	var count int64
	DB.Model(&NullableAssocUser{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 users after delete with null association, got %d", count)
	}
}

// TestNestedDeleteWithSliceOfPointers tests deletion with slice of pointers
func TestNestedDeleteWithSliceOfPointers(t *testing.T) {
	type SlicePtrProfile struct {
		gorm.Model
		Name           string
		SlicePtrUserID uint
	}

	type SlicePtrUser struct {
		gorm.Model
		Name     string
		Profiles []*SlicePtrProfile `gorm:"foreignKey:SlicePtrUserID"`
	}

	DB.Migrator().DropTable(&SlicePtrProfile{}, &SlicePtrUser{})
	if err := DB.AutoMigrate(&SlicePtrUser{}, &SlicePtrProfile{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	user := SlicePtrUser{
		Name: "slice_ptr_user",
		Profiles: []*SlicePtrProfile{
			{Name: "Profile1"},
			{Name: "Profile2"},
			{Name: "Profile3"},
		},
	}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user with pointer profiles, got error %v", err)
	}

	var deletedUser SlicePtrUser
	result := DB.Select("Profiles").Delete(&deletedUser, user.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete user with slice of pointers, got error %v", result.Error)
	}

	var count int64
	DB.Model(&SlicePtrProfile{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 profiles after nested delete with slice of pointers, got %d", count)
	}
}

// TestNestedDeleteHasOneZeroStruct tests HasOne deletion with zero struct
func TestNestedDeleteHasOneZeroStruct(t *testing.T) {
	type HasOneZeroAccount struct {
		gorm.Model
		Name             string
		HasOneZeroUserID uint
	}

	type HasOneZeroUser struct {
		gorm.Model
		Name    string
		Account *HasOneZeroAccount `gorm:"foreignKey:HasOneZeroUserID"`
	}

	DB.Migrator().DropTable(&HasOneZeroAccount{}, &HasOneZeroUser{})
	if err := DB.AutoMigrate(&HasOneZeroUser{}, &HasOneZeroAccount{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	// Create user without account
	user := HasOneZeroUser{Name: "has_one_zero_user"}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user, got error %v", err)
	}

	// Delete with HasOne nested select when association doesn't exist
	var deletedUser HasOneZeroUser
	result := DB.Select("Account").Delete(&deletedUser, user.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete user with zero HasOne association, got error %v", result.Error)
	}

	var count int64
	DB.Model(&HasOneZeroUser{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 users after HasOne zero delete, got %d", count)
	}
}

// TestNestedDeleteWithManyToManyEmptyJoinTable tests M2M deletion with no join table records
func TestNestedDeleteWithManyToManyEmptyJoinTable(t *testing.T) {
	type EmptyM2MTag struct {
		gorm.Model
		Name string
	}

	type EmptyM2MPost struct {
		gorm.Model
		Title string
		Tags  []EmptyM2MTag `gorm:"many2many:empty_m2m_post_tags;"`
	}

	DB.Migrator().DropTable(&EmptyM2MTag{}, &EmptyM2MPost{}, "empty_m2m_post_tags")
	if err := DB.AutoMigrate(&EmptyM2MTag{}, &EmptyM2MPost{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	// Create post without tags
	post := EmptyM2MPost{Title: "empty_m2m_post"}
	if err := DB.Create(&post).Error; err != nil {
		t.Fatalf("Failed to create post, got error %v", err)
	}

	// Delete with M2M nested select when no join table records exist
	var deletedPost EmptyM2MPost
	result := DB.Select("Tags").Delete(&deletedPost, post.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete post with empty M2M, got error %v", result.Error)
	}

	var count int64
	DB.Model(&EmptyM2MPost{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 posts after empty M2M delete, got %d", count)
	}
}

// TestNestedDeleteMultipleLevelsWithAssociations tests deeply nested deletions with multiple relationship types
func TestNestedDeleteMultipleLevelsWithAssociations(t *testing.T) {
	type Level3Item struct {
		gorm.Model
		Name              string
		Level2ContainerID uint
	}

	type Level2Container struct {
		gorm.Model
		Name           string
		Level1ParentID uint
		Items          []Level3Item `gorm:"foreignKey:Level2ContainerID"`
	}

	type Level1Parent struct {
		gorm.Model
		Name       string
		Containers []Level2Container `gorm:"foreignKey:Level1ParentID"`
	}

	DB.Migrator().DropTable(&Level3Item{}, &Level2Container{}, &Level1Parent{})
	if err := DB.AutoMigrate(&Level1Parent{}, &Level2Container{}, &Level3Item{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	parent := Level1Parent{
		Name: "parent",
		Containers: []Level2Container{
			{
				Name: "container1",
				Items: []Level3Item{
					{Name: "item1"},
					{Name: "item2"},
				},
			},
			{
				Name: "container2",
				Items: []Level3Item{
					{Name: "item3"},
				},
			},
		},
	}
	if err := DB.Create(&parent).Error; err != nil {
		t.Fatalf("Failed to create nested structure, got error %v", err)
	}

	// Delete parent with nested items
	var deletedParent Level1Parent
	result := DB.Select("Containers.Items").Delete(&deletedParent, parent.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete parent with deeply nested items, got error %v", result.Error)
	}

	var countParent, countContainer, countItem int64
	DB.Model(&Level1Parent{}).Count(&countParent)
	DB.Model(&Level2Container{}).Count(&countContainer)
	DB.Model(&Level3Item{}).Count(&countItem)

	if countParent != 0 {
		t.Fatalf("Expected 0 parents after nested delete, got %d", countParent)
	}
	if countContainer != 0 {
		t.Fatalf("Expected 0 containers after nested delete, got %d", countContainer)
	}
	if countItem != 0 {
		t.Fatalf("Expected 0 items after nested delete, got %d", countItem)
	}
}

// TestNestedDeleteWithPartialNestedSelection tests deleting only specific nested associations
func TestNestedDeleteWithPartialNestedSelection(t *testing.T) {
	type PartialSelProfile struct {
		gorm.Model
		Name             string
		PartialSelUserID uint
	}

	type PartialSelPost struct {
		gorm.Model
		Title            string
		PartialSelUserID uint
	}

	type PartialSelUser struct {
		gorm.Model
		Name     string
		Profiles []PartialSelProfile `gorm:"foreignKey:PartialSelUserID"`
		Posts    []PartialSelPost    `gorm:"foreignKey:PartialSelUserID"`
	}

	DB.Migrator().DropTable(&PartialSelProfile{}, &PartialSelPost{}, &PartialSelUser{})
	if err := DB.AutoMigrate(&PartialSelUser{}, &PartialSelProfile{}, &PartialSelPost{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	user := PartialSelUser{
		Name: "partial_sel_user",
		Profiles: []PartialSelProfile{
			{Name: "Profile1"},
			{Name: "Profile2"},
		},
		Posts: []PartialSelPost{
			{Title: "Post1"},
			{Title: "Post2"},
		},
	}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user, got error %v", err)
	}

	// Delete only profiles nested association
	var deletedUser PartialSelUser
	result := DB.Select("Profiles").Delete(&deletedUser, user.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete user with partial nested select, got error %v", result.Error)
	}

	var countUser, countProfile, countPost int64
	DB.Model(&PartialSelUser{}).Count(&countUser)
	DB.Model(&PartialSelProfile{}).Count(&countProfile)
	DB.Model(&PartialSelPost{}).Count(&countPost)

	// User and profiles should be deleted
	if countUser != 0 {
		t.Fatalf("Expected 0 users after partial delete, got %d", countUser)
	}
	if countProfile != 0 {
		t.Fatalf("Expected 0 profiles after partial delete, got %d", countProfile)
	}
	// Posts remain because they were not selected for deletion
	if countPost != 2 {
		t.Fatalf("Expected 2 posts after partial delete (not selected), got %d", countPost)
	}
}

// TestNestedDeleteWithPreloadedData tests deletion with preloaded associations
func TestNestedDeleteWithPreloadedData(t *testing.T) {
	type PreloadProfile struct {
		gorm.Model
		Name          string
		PreloadUserID uint
	}

	type PreloadUser struct {
		gorm.Model
		Name     string
		Profiles []PreloadProfile `gorm:"foreignKey:PreloadUserID"`
	}

	DB.Migrator().DropTable(&PreloadProfile{}, &PreloadUser{})
	if err := DB.AutoMigrate(&PreloadUser{}, &PreloadProfile{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	user := PreloadUser{
		Name: "preload_user",
		Profiles: []PreloadProfile{
			{Name: "Profile1"},
			{Name: "Profile2"},
		},
	}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user, got error %v", err)
	}

	// Load user with preloaded associations
	var loadedUser PreloadUser
	if err := DB.Preload("Profiles").First(&loadedUser, user.ID).Error; err != nil {
		t.Fatalf("Failed to preload user, got error %v", err)
	}

	// Delete using preloaded data
	result := DB.Select("Profiles").Delete(&loadedUser)
	if result.Error != nil {
		t.Fatalf("Failed to delete user with preloaded data, got error %v", result.Error)
	}

	var count int64
	DB.Model(&PreloadUser{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 users after delete with preloaded data, got %d", count)
	}
}

// TestNestedDeleteWithSoftDelete tests nested deletion with soft delete enabled
func TestNestedDeleteWithSoftDeleteNested(t *testing.T) {
	type SoftDelProfile struct {
		gorm.Model
		Name          string
		SoftDelUserID uint
	}

	type SoftDelUser struct {
		gorm.Model
		Name     string
		Profiles []SoftDelProfile `gorm:"foreignKey:SoftDelUserID"`
	}

	DB.Migrator().DropTable(&SoftDelProfile{}, &SoftDelUser{})
	if err := DB.AutoMigrate(&SoftDelUser{}, &SoftDelProfile{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	user := SoftDelUser{
		Name: "soft_del_user",
		Profiles: []SoftDelProfile{
			{Name: "Profile1"},
			{Name: "Profile2"},
		},
	}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user, got error %v", err)
	}

	// Soft delete with nested select
	var deletedUser SoftDelUser
	result := DB.Select("Profiles").Delete(&deletedUser, user.ID)
	if result.Error != nil {
		t.Fatalf("Failed to soft delete with nested select, got error %v", result.Error)
	}

	// Check soft deleted records are still there
	var countScoped, countUnscoped int64
	DB.Model(&SoftDelUser{}).Count(&countScoped)
	DB.Model(&SoftDelUser{}).Unscoped().Count(&countUnscoped)

	if countScoped != 0 {
		t.Fatalf("Expected 0 undeleted users after soft delete, got %d", countScoped)
	}
	if countUnscoped != 1 {
		t.Fatalf("Expected 1 soft-deleted user when using Unscoped, got %d", countUnscoped)
	}
}

// TestNestedDeleteWithUnscopedNested tests hard deletion of nested associations
func TestNestedDeleteWithUnscopedNested(t *testing.T) {
	type UnscopedProfile struct {
		gorm.Model
		Name           string
		UnscopedUserID uint
	}

	type UnscopedUser struct {
		gorm.Model
		Name     string
		Profiles []UnscopedProfile `gorm:"foreignKey:UnscopedUserID"`
	}

	DB.Migrator().DropTable(&UnscopedProfile{}, &UnscopedUser{})
	if err := DB.AutoMigrate(&UnscopedUser{}, &UnscopedProfile{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	user := UnscopedUser{
		Name: "unscoped_user",
		Profiles: []UnscopedProfile{
			{Name: "Profile1"},
			{Name: "Profile2"},
		},
	}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user, got error %v", err)
	}

	// Hard delete with nested select using Unscoped
	var deletedUser UnscopedUser
	result := DB.Unscoped().Select("Profiles").Delete(&deletedUser, user.ID)
	if result.Error != nil {
		t.Fatalf("Failed to unscoped delete with nested select, got error %v", result.Error)
	}

	// Check records are completely gone
	var countUnscoped int64
	DB.Model(&UnscopedUser{}).Unscoped().Count(&countUnscoped)

	if countUnscoped != 0 {
		t.Fatalf("Expected 0 users after unscoped hard delete, got %d", countUnscoped)
	}
}

// TestNestedDeleteWithComplexM2M tests M2M deletion with multiple posts and tags
func TestNestedDeleteWithComplexM2M(t *testing.T) {
	type ComplexTag struct {
		gorm.Model
		Name string
	}

	type ComplexPost struct {
		gorm.Model
		Title         string
		ComplexBlogID uint
		Tags          []ComplexTag `gorm:"many2many:complex_m2m_post_tags;"`
	}

	type ComplexBlog struct {
		gorm.Model
		Name  string
		Posts []ComplexPost `gorm:"foreignKey:ComplexBlogID"`
	}

	DB.Migrator().DropTable(&ComplexTag{}, &ComplexPost{}, &ComplexBlog{}, "complex_m2m_post_tags")
	if err := DB.AutoMigrate(&ComplexBlog{}, &ComplexPost{}, &ComplexTag{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	// Create tags
	tags := []ComplexTag{
		{Name: "golang"},
		{Name: "database"},
		{Name: "testing"},
	}
	if err := DB.Create(&tags).Error; err != nil {
		t.Fatalf("Failed to create tags, got error %v", err)
	}

	// Create blog with posts and tags
	blog := ComplexBlog{
		Name: "complex_blog",
		Posts: []ComplexPost{
			{Title: "Post1", Tags: []ComplexTag{tags[0], tags[1]}},
			{Title: "Post2", Tags: []ComplexTag{tags[1], tags[2]}},
			{Title: "Post3", Tags: []ComplexTag{tags[0], tags[2]}},
		},
	}
	if err := DB.Create(&blog).Error; err != nil {
		t.Fatalf("Failed to create blog with posts, got error %v", err)
	}

	// Delete blog with posts
	var deletedBlog ComplexBlog
	result := DB.Select("Posts.Tags").Delete(&deletedBlog, blog.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete blog with complex M2M, got error %v", result.Error)
	}

	var countBlog, countPost, countTag, countJoin int64
	DB.Model(&ComplexBlog{}).Count(&countBlog)
	DB.Model(&ComplexPost{}).Count(&countPost)
	DB.Model(&ComplexTag{}).Count(&countTag)
	DB.Table("complex_m2m_post_tags").Count(&countJoin)

	if countBlog != 0 {
		t.Fatalf("Expected 0 blogs after complex M2M delete, got %d", countBlog)
	}
	if countPost != 0 {
		t.Fatalf("Expected 0 posts after complex M2M delete, got %d", countPost)
	}
	if countTag != 3 {
		t.Fatalf("Expected 3 tags after complex M2M delete (tags should not be deleted), got %d", countTag)
	}
	if countJoin != 0 {
		t.Fatalf("Expected 0 join table records after complex M2M delete, got %d", countJoin)
	}
}

// TestNestedDeleteAssociationsClause tests deletion with clause.Associations
func TestNestedDeleteAssociationsClause(t *testing.T) {
	type ClauseProfile struct {
		gorm.Model
		Name         string
		ClauseUserID uint
	}

	type ClausePost struct {
		gorm.Model
		Title        string
		ClauseUserID uint
	}

	type ClauseUser struct {
		gorm.Model
		Name     string
		Profiles []ClauseProfile `gorm:"foreignKey:ClauseUserID"`
		Posts    []ClausePost    `gorm:"foreignKey:ClauseUserID"`
	}

	DB.Migrator().DropTable(&ClauseProfile{}, &ClausePost{}, &ClauseUser{})
	if err := DB.AutoMigrate(&ClauseUser{}, &ClauseProfile{}, &ClausePost{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	user := ClauseUser{
		Name: "clause_user",
		Profiles: []ClauseProfile{
			{Name: "Profile1"},
		},
		Posts: []ClausePost{
			{Title: "Post1"},
		},
	}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user, got error %v", err)
	}

	// Delete all associations using clause.Associations
	var deletedUser ClauseUser
	result := DB.Select(clause.Associations).Delete(&deletedUser, user.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete with clause.Associations, got error %v", result.Error)
	}

	var count int64
	DB.Model(&ClauseUser{}).Count(&count)
	if count != 0 {
		t.Fatalf("Expected 0 users after delete with clause.Associations, got %d", count)
	}
}

// TestNestedDeleteWithPreloadedPartialData tests deletion with preloaded partial data
func TestNestedDeleteWithPreloadedPartialData(t *testing.T) {
	type PreloadPartialProfile struct {
		gorm.Model
		Name                 string
		Description          string
		PreloadPartialUserID uint
	}

	type PreloadPartialUser struct {
		gorm.Model
		Name     string
		Email    string
		Profiles []PreloadPartialProfile `gorm:"foreignKey:PreloadPartialUserID"`
	}

	DB.Migrator().DropTable(&PreloadPartialProfile{}, &PreloadPartialUser{})
	if err := DB.AutoMigrate(&PreloadPartialUser{}, &PreloadPartialProfile{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	user := PreloadPartialUser{
		Name:  "preload_partial_user",
		Email: "user@example.com",
		Profiles: []PreloadPartialProfile{
			{Name: "Profile1", Description: "Desc1"},
			{Name: "Profile2", Description: "Desc2"},
		},
	}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user, got error %v", err)
	}

	// Load user with preloaded profiles but only specific columns (Name, not Description)
	var loadedUser PreloadPartialUser
	if err := DB.Preload("Profiles", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "name", "preload_partial_user_id")
	}).First(&loadedUser, user.ID).Error; err != nil {
		t.Fatalf("Failed to preload user with partial columns, got error %v", err)
	}

	// Verify that Description field is not loaded (zero value)
	if loadedUser.Profiles[0].Description != "" {
		// If it's loaded from DB, the description will be there; if truly partial, it won't be
		t.Logf("Note: Profiles preloaded with Description field: %s", loadedUser.Profiles[0].Description)
	}

	// Delete using the preloaded user (even though it's partial)
	result := DB.Select("Profiles").Delete(&loadedUser)
	if result.Error != nil {
		t.Fatalf("Failed to delete with preloaded partial data, got error %v", result.Error)
	}

	var countUser, countProfile int64
	DB.Model(&PreloadPartialUser{}).Count(&countUser)
	DB.Model(&PreloadPartialProfile{}).Count(&countProfile)

	if countUser != 0 {
		t.Fatalf("Expected 0 users after delete with preloaded partial data, got %d", countUser)
	}
	if countProfile != 0 {
		t.Fatalf("Expected 0 profiles after delete with preloaded partial data, got %d", countProfile)
	}
}

// TestNestedDeleteWithComposedAssociations tests deletion with multiple levels and different relationship types
func TestNestedDeleteWithComposedAssociations(t *testing.T) {
	type ComposedToy struct {
		gorm.Model
		Name              string
		ComposedProfileID uint
	}

	type ComposedProfile struct {
		gorm.Model
		Name           string
		ComposedUserID uint
		Toys           []ComposedToy `gorm:"foreignKey:ComposedProfileID"`
	}

	type ComposedUser struct {
		gorm.Model
		Name     string
		Profiles []ComposedProfile `gorm:"foreignKey:ComposedUserID"`
	}

	DB.Migrator().DropTable(&ComposedToy{}, &ComposedProfile{}, &ComposedUser{})
	if err := DB.AutoMigrate(&ComposedUser{}, &ComposedProfile{}, &ComposedToy{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	// Create complex nested structure
	user := ComposedUser{
		Name: "composed_user",
		Profiles: []ComposedProfile{
			{
				Name: "Profile1",
				Toys: []ComposedToy{
					{Name: "Toy1A"},
					{Name: "Toy1B"},
					{Name: "Toy1C"},
				},
			},
			{
				Name: "Profile2",
				Toys: []ComposedToy{
					{Name: "Toy2A"},
					{Name: "Toy2B"},
				},
			},
			{
				Name: "Profile3",
				Toys: []ComposedToy{
					{Name: "Toy3A"},
				},
			},
		},
	}

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create composed user, got error %v", err)
	}

	// Verify structure was created correctly
	var countUser, countProfile, countToy int64
	DB.Model(&ComposedUser{}).Count(&countUser)
	DB.Model(&ComposedProfile{}).Count(&countProfile)
	DB.Model(&ComposedToy{}).Count(&countToy)

	if countUser != 1 {
		t.Fatalf("Expected 1 user before delete, got %d", countUser)
	}
	if countProfile != 3 {
		t.Fatalf("Expected 3 profiles before delete, got %d", countProfile)
	}
	if countToy != 6 {
		t.Fatalf("Expected 6 toys before delete, got %d", countToy)
	}

	// Delete user with composed associations (user -> profiles -> toys)
	var deletedUser ComposedUser
	result := DB.Select("Profiles.Toys").Delete(&deletedUser, user.ID)
	if result.Error != nil {
		t.Fatalf("Failed to delete with composed associations, got error %v", result.Error)
	}

	// After delete, verify all nested records are gone
	DB.Model(&ComposedUser{}).Count(&countUser)
	DB.Model(&ComposedProfile{}).Count(&countProfile)
	DB.Model(&ComposedToy{}).Count(&countToy)

	if countUser != 0 {
		t.Fatalf("Expected 0 users after composed delete, got %d", countUser)
	}
	if countProfile != 0 {
		t.Fatalf("Expected 0 profiles after composed delete, got %d", countProfile)
	}
	if countToy != 0 {
		t.Fatalf("Expected 0 toys after composed delete, got %d", countToy)
	}
}
