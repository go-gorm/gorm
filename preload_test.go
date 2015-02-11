package gorm_test

import "testing"

func TestPreload(t *testing.T) {
	user := User{Name: "PreloadUser", BillingAddress: Address{Address1: "Billing Address"}, ShippingAddress: Address{Address1: "Shipping Address"}, Languages: []Language{{Name: "Preload L1"}, {Name: "Preload L2"}}}
	DB.Save(&user)

	var users []User
	DB.Preload("BillingAddress").Preload("ShippingAddress").Preload("Languages").Find(&users)
}
