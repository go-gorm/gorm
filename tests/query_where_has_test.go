package tests_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

func TestQueryWhereHas(t *testing.T) {
	DB.Create(&Language{Code: "wh_ro", Name: "Romanian"})

	DB.Create(&[]User{
		{
			Name: "u1_has_pets_with_toy",
			Pets: []*Pet{
				{
					Name: "u1_pet_1_with_toy",
					Toy: Toy{
						Name: "u1_p1_toy_1",
					},
				},

				{
					Name: "u1_pet_2_with_toy",
					Toy: Toy{
						Name: "u1_p1_toy_2",
					},
				},
			},
			Toys: []Toy{
				{
					Name: "u1_toy_1",
				},
			},
			Languages: []Language{
				{
					Code: "wh_en",
					Name: "English",
				},
			},
		},

		{
			Name: "u2_has_pets_with_without_toy",
			Pets: []*Pet{
				{
					Name: "u2_pet_1_with_toy",
					Toy: Toy{
						Name: "u2_p1_toy_1",
					},
				},

				{
					Name: "u2_pet_2_without_toy",
				},
			},
			Toys: []Toy{
				{
					Name: "u2_toy_1",
				},
			},
			Languages: []Language{
				{
					Code: "wh_en",
					Name: "English",
				},
				{
					Code: "wh_it",
					Name: "Italian",
				},
			},
		},
		{
			Name: "u3_has_pets_without_toy",
			Pets: []*Pet{
				{
					Name: "u3_pet_1_without_toy",
				},

				{
					Name: "u3_pet_2_without_toy",
				},
			},
			Toys: []Toy{
				{
					Name: "u3_toy_1",
				},
			},
		},
	})

	t.Run("OneToOne", func(t *testing.T) {
		var err error

		var pet Pet
		petLookUpName := "u1_pet_1_with_toy"

		pet = Pet{}
		DB.Where("name = ?", petLookUpName).WhereHas("Toy").First(&pet)
		assert.Equal(t, petLookUpName, pet.Name)

		pet = Pet{}
		DB.Where("name = ?", petLookUpName).WhereHas("Toy", DB.Where("name = ?", "u1_p1_toy_1")).First(&pet)
		assert.Equal(t, petLookUpName, pet.Name)

		pet = Pet{}
		err = DB.Where("name = ?", petLookUpName).WhereDoesntHave("Toy").First(&pet).Error
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("HasMany", func(t *testing.T) {
		var err error
		var user User

		user = User{}
		DB.Where("name = ?", "u1_has_pets_with_toy").WhereHas("Pets").First(&user)
		assert.Equal(t, "u1_has_pets_with_toy", user.Name)

		user = User{}
		DB.Where("name = ?", "u1_has_pets_with_toy").WhereHas("Pets", DB.Where("name = ?", "u1_pet_1_with_toy")).First(&user)
		assert.Equal(t, "u1_has_pets_with_toy", user.Name)

		user = User{}
		err = DB.Where("name = ?", "u1_has_pets_with_toy").WhereDoesntHave("Pets").First(&user).Error
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("ManyToMany", func(t *testing.T) {
		var err error
		var user User

		user = User{}
		DB.Where("name = ?", "u1_has_pets_with_toy").WhereHas("Languages").First(&user)
		assert.Equal(t, "u1_has_pets_with_toy", user.Name)

		user = User{}
		err = DB.Where("name = ?", "u1_has_pets_with_toy").WhereDoesntHave("Languages").First(&user).Error
		assert.Equal(t, gorm.ErrRecordNotFound, err)

		user = User{}
		err = DB.Where("name = ?", "u3_has_pets_without_toy").WhereHas("Languages").First(&user).Error
		assert.Equal(t, gorm.ErrRecordNotFound, err)

		var users []User
		DB.WhereHas("Languages", DB.Where("code = ?", "wh_it")).Find(&users)
		assert.Equal(t, 1, len(users))
	})

	t.Run("Nested", func(t *testing.T) {
		var user User

		user = User{}
		DB.WhereHas("Pets", DB.WhereHas("Toy", DB.Where("name = ?", "u1_p1_toy_1"))).First(&user)
		assert.Equal(t, user.Name, "u1_has_pets_with_toy")
	})

	t.Run("Polymorphic", func(t *testing.T) {
		var user User

		user = User{}
		DB.Where("name = ?", "u1_has_pets_with_toy").WhereHas("Toys", DB.Where("name = ?", "u1_toy_1")).First(&user)
		assert.Equal(t, "u1_has_pets_with_toy", user.Name)

		var err error
		user = User{}
		err = DB.Where("name = ?", "u3_has_pets_without_toy").WhereDoesntHave("Toys").First(&user).Error
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("Chained", func(t *testing.T) {
		var users []User
		DB.Where("name LIKE ?", "u%_has_pets_%").WhereHas("Pets").WhereHas("Toys").Find(&users)
		for _, u := range users {
			assert.NotEmpty(t, u.Name)
		}
		assert.GreaterOrEqual(t, len(users), 1)
	})
}

func TestQueryWhereHasBelongsTo(t *testing.T) {
	company := Company{Name: "wh_belongs_company"}
	if err := DB.Create(&company).Error; err != nil {
		t.Fatalf("failed to create company: %v", err)
	}

	users := []*User{
		GetUser("wh_belongs_with_company", Config{}),
		GetUser("wh_belongs_no_company", Config{}),
	}
	users[0].CompanyID = &company.ID

	for _, user := range users {
		if err := DB.Create(user).Error; err != nil {
			t.Fatalf("failed to create user: %v", err)
		}
	}

	var user User
	DB.Where("name LIKE ?", "wh_belongs_%").WhereHas("Company").First(&user)
	assert.Equal(t, "wh_belongs_with_company", user.Name)

	var err error
	user = User{}
	err = DB.Where("name = ?", "wh_belongs_no_company").WhereHas("Company").First(&user).Error
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

func TestQueryWhereHasInvalidRelation(t *testing.T) {
	var user []User
	err := DB.WhereHas("NonExistentRelation").Find(&user).Error
	assert.NotNil(t, err)
}

func TestQueryWhereHasScopeFunc(t *testing.T) {
	petWithToy := Pet{Name: "wh_scope_pet_with_toy"}
	if err := DB.Create(&petWithToy).Error; err != nil {
		t.Fatalf("failed to create pet: %v", err)
	}
	toy := Toy{Name: "wh_scope_toy_1", OwnerID: fmt.Sprint(petWithToy.ID), OwnerType: "pets"}
	if err := DB.Create(&toy).Error; err != nil {
		t.Fatalf("failed to create toy: %v", err)
	}

	petWithoutToy := Pet{Name: "wh_scope_pet_without_toy"}
	if err := DB.Create(&petWithoutToy).Error; err != nil {
		t.Fatalf("failed to create pet: %v", err)
	}

	var pet Pet
	DB.Where("name LIKE ?", "wh_scope_pet_%").WhereHas("Toy", func(tx *gorm.DB) *gorm.DB {
		return tx.Where("name = ?", "wh_scope_toy_1")
	}).First(&pet)
	assert.Equal(t, "wh_scope_pet_with_toy", pet.Name)
}

func TestQueryWhereHasDryRun(t *testing.T) {
	sql := DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&User{}).WhereHas("Pets").Find(&[]User{})
	})

	assert.NotEmpty(t, sql)
	assert.True(t, strings.Contains(sql, "EXISTS"), "expected SQL to contain EXISTS, got: %s", sql)
}
