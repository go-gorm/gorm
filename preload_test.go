package gorm_test

import (
	"fmt"
	"testing"
)

func getPreloadUser(name string) User {
	return User{
		Name:            name,
		Role:            Role{"Preload"},
		BillingAddress:  Address{Address1: fmt.Sprintf("Billing Address %v", name)},
		ShippingAddress: Address{Address1: fmt.Sprintf("Shipping Address %v", name)},
		CreditCard:      CreditCard{Number: fmt.Sprintf("123456%v", name)},
		Emails: []Email{
			{Email: fmt.Sprintf("user_%v@example1.com", name)}, {Email: fmt.Sprintf("user_%v@example2.com", name)},
		},
		Languages: []Language{
			{Name: fmt.Sprintf("lang_1_%v", name)},
			{Name: fmt.Sprintf("lang_2_%v", name)},
		},
	}
}

func checkUserHasPreloadData(user User, t *testing.T) {
	u := getPreloadUser(user.Name)
	if user.BillingAddress.Address1 != u.BillingAddress.Address1 {
		t.Error("Failed to preload user's BillingAddress")
	}

	if user.ShippingAddress.Address1 != u.ShippingAddress.Address1 {
		t.Error("Failed to preload user's ShippingAddress")
	}

	if user.CreditCard.Number != u.CreditCard.Number {
		t.Error("Failed to preload user's CreditCard")
	}

	if len(user.Emails) != len(u.Emails) {
		t.Error("Failed to preload user's Emails")
	} else {
		var found int
		for _, e1 := range u.Emails {
			for _, e2 := range user.Emails {
				if e1.Email == e2.Email {
					found++
					break
				}
			}
		}
		if found != len(u.Emails) {
			t.Error("Failed to preload user's email details")
		}
	}
}

func TestPreload(t *testing.T) {
	user1 := getPreloadUser("user1")
	DB.Save(&user1)

	var user User
	DB.Where("role = ?", "Preload").Preload("BillingAddress").Preload("ShippingAddress").
		Preload("CreditCard").Preload("Emails").Find(&user)
	checkUserHasPreloadData(user, t)

	user2 := getPreloadUser("user2")
	DB.Save(&user2)

	user3 := getPreloadUser("user3")
	DB.Save(&user3)

	var users []User
	DB.Where("role = ?", "Preload").Preload("BillingAddress").Preload("ShippingAddress").
		Preload("CreditCard").Preload("Emails").Find(&users)

	for _, user := range users {
		checkUserHasPreloadData(user, t)
	}
}
