package tests_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

func TestQueryWhereHas(t *testing.T) {
	DB.Create(&Language{Code: "ro", Name: "Romanian"})

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
					Code: "en",
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
					Code: "en",
					Name: "English",
				},
				{
					Code: "it",
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
		DB.WhereHas("Languages", DB.Where("code = ?", "it")).Find(&users)
		assert.Equal(t, 1, len(users))
	})

	t.Run("Nested", func(t *testing.T) {
		var user User

		user = User{}
		DB.WhereHas("Pets", DB.WhereHas("Toy", DB.Where("name = ?", "u1_p1_toy_1"))).First(&user)
		assert.Equal(t, user.Name, "u1_has_pets_with_toy")
	})
}
