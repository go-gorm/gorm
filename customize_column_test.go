package gorm_test

import (
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

type CustomizeColumn struct {
	ID   int64     `gorm:"column:mapped_id; primary_key:yes"`
	Name string    `gorm:"column:mapped_name"`
	Date time.Time `gorm:"column:mapped_time"`
}

// Make sure an ignored field does not interfere with another field's custom
// column name that matches the ignored field.
type CustomColumnAndIgnoredFieldClash struct {
	Body    string `sql:"-"`
	RawBody string `gorm:"column:body"`
}

func TestCustomizeColumn(t *testing.T) {
	col := "mapped_name"
	DB.DropTable(&CustomizeColumn{})
	DB.AutoMigrate(&CustomizeColumn{})

	scope := DB.NewScope(&CustomizeColumn{})
	if !scope.Dialect().HasColumn(scope.TableName(), col) {
		t.Errorf("CustomizeColumn should have column %s", col)
	}

	col = "mapped_id"
	if scope.PrimaryKey() != col {
		t.Errorf("CustomizeColumn should have primary key %s, but got %q", col, scope.PrimaryKey())
	}

	expected := "foo"
	cc := CustomizeColumn{ID: 666, Name: expected, Date: time.Now()}

	if count := DB.Create(&cc).RowsAffected; count != 1 {
		t.Error("There should be one record be affected when create record")
	}

	var cc1 CustomizeColumn
	DB.First(&cc1, 666)

	if cc1.Name != expected {
		t.Errorf("Failed to query CustomizeColumn")
	}

	cc.Name = "bar"
	DB.Save(&cc)

	var cc2 CustomizeColumn
	DB.First(&cc2, 666)
	if cc2.Name != "bar" {
		t.Errorf("Failed to query CustomizeColumn")
	}
}

func TestCustomColumnAndIgnoredFieldClash(t *testing.T) {
	DB.DropTable(&CustomColumnAndIgnoredFieldClash{})
	if err := DB.AutoMigrate(&CustomColumnAndIgnoredFieldClash{}).Error; err != nil {
		t.Errorf("Should not raise error: %s", err)
	}
}

type CustomizePerson struct {
	IdPerson string             `gorm:"column:idPerson;primary_key:true"`
	Accounts []CustomizeAccount `gorm:"many2many:PersonAccount;associationforeignkey:idAccount;foreignkey:idPerson"`
}

type CustomizeAccount struct {
	IdAccount string `gorm:"column:idAccount;primary_key:true"`
	Name      string
}

func TestManyToManyWithCustomizedColumn(t *testing.T) {
	DB.DropTable(&CustomizePerson{}, &CustomizeAccount{}, "PersonAccount")
	DB.AutoMigrate(&CustomizePerson{}, &CustomizeAccount{})

	account := CustomizeAccount{IdAccount: "account", Name: "id1"}
	person := CustomizePerson{
		IdPerson: "person",
		Accounts: []CustomizeAccount{account},
	}

	if err := DB.Create(&account).Error; err != nil {
		t.Errorf("no error should happen, but got %v", err)
	}

	if err := DB.Create(&person).Error; err != nil {
		t.Errorf("no error should happen, but got %v", err)
	}

	var person1 CustomizePerson
	scope := DB.NewScope(nil)
	if err := DB.Preload("Accounts").First(&person1, scope.Quote("idPerson")+" = ?", person.IdPerson).Error; err != nil {
		t.Errorf("no error should happen when preloading customized column many2many relations, but got %v", err)
	}

	if len(person1.Accounts) != 1 || person1.Accounts[0].IdAccount != "account" {
		t.Errorf("should preload correct accounts")
	}
}

type CustomizeUser struct {
	gorm.Model
	Email string `sql:"column:email_address"`
}

type CustomizeInvitation struct {
	gorm.Model
	Address string         `sql:"column:invitation"`
	Person  *CustomizeUser `gorm:"foreignkey:Email;associationforeignkey:invitation"`
}

func TestOneToOneWithCustomizedColumn(t *testing.T) {
	DB.DropTable(&CustomizeUser{}, &CustomizeInvitation{})
	DB.AutoMigrate(&CustomizeUser{}, &CustomizeInvitation{})

	user := CustomizeUser{
		Email: "hello@example.com",
	}
	invitation := CustomizeInvitation{
		Address: "hello@example.com",
	}

	DB.Create(&user)
	DB.Create(&invitation)

	var invitation2 CustomizeInvitation
	if err := DB.Preload("Person").Find(&invitation2, invitation.ID).Error; err != nil {
		t.Errorf("no error should happen, but got %v", err)
	}

	if invitation2.Person.Email != user.Email {
		t.Errorf("Should preload one to one relation with customize foreign keys")
	}
}

type PromotionDiscount struct {
	gorm.Model
	Name     string
	Coupons  []*PromotionCoupon `gorm:"ForeignKey:discount_id"`
	Rule     *PromotionRule     `gorm:"ForeignKey:discount_id"`
	Benefits []PromotionBenefit `gorm:"ForeignKey:promotion_id"`
}

type PromotionBenefit struct {
	gorm.Model
	Name        string
	PromotionID uint
	Discount    PromotionDiscount `gorm:"ForeignKey:promotion_id"`
}

type PromotionCoupon struct {
	gorm.Model
	Code       string
	DiscountID uint
	Discount   PromotionDiscount
}

type PromotionRule struct {
	gorm.Model
	Name       string
	Begin      *time.Time
	End        *time.Time
	DiscountID uint
	Discount   *PromotionDiscount
}

func TestOneToManyWithCustomizedColumn(t *testing.T) {
	DB.DropTable(&PromotionDiscount{}, &PromotionCoupon{})
	DB.AutoMigrate(&PromotionDiscount{}, &PromotionCoupon{})

	discount := PromotionDiscount{
		Name: "Happy New Year",
		Coupons: []*PromotionCoupon{
			{Code: "newyear1"},
			{Code: "newyear2"},
		},
	}

	if err := DB.Create(&discount).Error; err != nil {
		t.Errorf("no error should happen but got %v", err)
	}

	var discount1 PromotionDiscount
	if err := DB.Preload("Coupons").First(&discount1, "id = ?", discount.ID).Error; err != nil {
		t.Errorf("no error should happen but got %v", err)
	}

	if len(discount.Coupons) != 2 {
		t.Errorf("should find two coupons")
	}

	var coupon PromotionCoupon
	if err := DB.Preload("Discount").First(&coupon, "code = ?", "newyear1").Error; err != nil {
		t.Errorf("no error should happen but got %v", err)
	}

	if coupon.Discount.Name != "Happy New Year" {
		t.Errorf("should preload discount from coupon")
	}
}

func TestHasOneWithPartialCustomizedColumn(t *testing.T) {
	DB.DropTable(&PromotionDiscount{}, &PromotionRule{})
	DB.AutoMigrate(&PromotionDiscount{}, &PromotionRule{})

	var begin = time.Now()
	var end = time.Now().Add(24 * time.Hour)
	discount := PromotionDiscount{
		Name: "Happy New Year 2",
		Rule: &PromotionRule{
			Name:  "time_limited",
			Begin: &begin,
			End:   &end,
		},
	}

	if err := DB.Create(&discount).Error; err != nil {
		t.Errorf("no error should happen but got %v", err)
	}

	var discount1 PromotionDiscount
	if err := DB.Preload("Rule").First(&discount1, "id = ?", discount.ID).Error; err != nil {
		t.Errorf("no error should happen but got %v", err)
	}

	if discount.Rule.Begin.Format(time.RFC3339Nano) != begin.Format(time.RFC3339Nano) {
		t.Errorf("Should be able to preload Rule")
	}

	var rule PromotionRule
	if err := DB.Preload("Discount").First(&rule, "name = ?", "time_limited").Error; err != nil {
		t.Errorf("no error should happen but got %v", err)
	}

	if rule.Discount.Name != "Happy New Year 2" {
		t.Errorf("should preload discount from rule")
	}
}

func TestBelongsToWithPartialCustomizedColumn(t *testing.T) {
	DB.DropTable(&PromotionDiscount{}, &PromotionBenefit{})
	DB.AutoMigrate(&PromotionDiscount{}, &PromotionBenefit{})

	discount := PromotionDiscount{
		Name: "Happy New Year 3",
		Benefits: []PromotionBenefit{
			{Name: "free cod"},
			{Name: "free shipping"},
		},
	}

	if err := DB.Create(&discount).Error; err != nil {
		t.Errorf("no error should happen but got %v", err)
	}

	var discount1 PromotionDiscount
	if err := DB.Preload("Benefits").First(&discount1, "id = ?", discount.ID).Error; err != nil {
		t.Errorf("no error should happen but got %v", err)
	}

	if len(discount.Benefits) != 2 {
		t.Errorf("should find two benefits")
	}

	var benefit PromotionBenefit
	if err := DB.Preload("Discount").First(&benefit, "name = ?", "free cod").Error; err != nil {
		t.Errorf("no error should happen but got %v", err)
	}

	if benefit.Discount.Name != "Happy New Year 3" {
		t.Errorf("should preload discount from coupon")
	}
}
