package tests

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// User has one `Account` (has one), many `Pets` (has many) and `Toys` (has many - polymorphic)
// He works in a Company (belongs to), he has a Manager (belongs to - single-table), and also managed a Team (has many - single-table)
// He speaks many languages (many to many) and has many friends (many to many - single-table)
// His pet also has one Toy (has one - polymorphic)
// NamedPet is a reference to a Named `Pets` (has many)
type User struct {
	gorm.Model
	Name      string
	Age       uint
	Birthday  *time.Time
	Account   Account
	Pets      []*Pet
	NamedPet  *Pet
	Toys      []Toy `gorm:"polymorphic:Owner"`
	CompanyID *int
	Company   Company
	ManagerID *uint
	Manager   *User
	Team      []User     `gorm:"foreignkey:ManagerID"`
	Languages []Language `gorm:"many2many:UserSpeak;"`
	Friends   []*User    `gorm:"many2many:user_friends;"`
	Active    bool
}

type Account struct {
	gorm.Model
	UserID sql.NullInt64
	Number string
}

type Pet struct {
	gorm.Model
	UserID *uint
	Name   string
	Toy    Toy `gorm:"polymorphic:Owner;"`
}

type Toy struct {
	gorm.Model
	Name      string
	OwnerID   string
	OwnerType string
}

type Company struct {
	ID   int
	Name string
}

type Language struct {
	Code string `gorm:"primarykey"`
	Name string
}

type Coupon struct {
	ID               int              `gorm:"primarykey; size:255"`
	AppliesToProduct []*CouponProduct `gorm:"foreignKey:CouponId;constraint:OnDelete:CASCADE"`
	AmountOff        uint32           `gorm:"column:amount_off"`
	PercentOff       float32          `gorm:"column:percent_off"`
}

type CouponProduct struct {
	CouponId  int    `gorm:"primarykey;size:255"`
	ProductId string `gorm:"primarykey;size:255"`
	Desc      string
}

type Order struct {
	gorm.Model
	Num      string
	Coupon   *Coupon
	CouponID string
}

type Parent struct {
	gorm.Model
	FavChildID uint
	FavChild   *Child
	Children   []*Child
}

type Child struct {
	gorm.Model
	Name     string
	ParentID *uint
	Parent   *Parent
}

type Meta interface {
	Scan(src interface{}) error
	Value() (driver.Value, error)
	GormDataType() string
}
type MotorMeta struct {
	Power string
}

func (meta *MotorMeta) Scan(src interface{}) error {
	bytes, ok := src.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSON value:", src))
	}
	result := MotorMeta{}
	err := json.Unmarshal(bytes, &result)
	*meta = result
	return err
}
func (meta *MotorMeta) Value() (driver.Value, error) {
	if meta == nil {
		return nil, nil
	}
	res, err := json.Marshal(meta)
	return string(res), err
}
func (MotorMeta) GormDataType() string {
	return "json"
}

type ManualMeta struct {
}

func (meta *ManualMeta) Scan(src interface{}) error {
	bytes, ok := src.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSON value:", src))
	}
	result := ManualMeta{}
	err := json.Unmarshal(bytes, &result)
	*meta = result
	return err
}
func (meta *ManualMeta) Value() (driver.Value, error) {
	if meta == nil {
		return nil, nil
	}
	res, err := json.Marshal(meta)
	return string(res), err
}
func (ManualMeta) GormDataType() string {
	return "json"
}

type Vehicle struct {
	gorm.Model
	Meta Meta
}
