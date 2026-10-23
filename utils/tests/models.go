package tests

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

// User is the central association fixture. It embeds gorm.Model for a generated
// primary key, timestamps, and soft deletion. Associations cover nullable belongs-to
// keys, has-one/has-many records, polymorphic owners, and many-to-many join tables,
// including self-references. This graph is designed for ORM tests, not a production
// schema template.
type User struct {
	gorm.Model
	// Name is a scalar string used in filters, updates, and expected fixture values.
	Name string
	// Age is an unsigned scalar used to exercise numeric values and zero-value filtering.
	Age uint
	// Birthday is a nullable timestamp; nil and a pointer to the zero time are distinct
	// fixture values.
	Birthday *time.Time
	// Account is a has-one association through Account.UserID.
	Account Account
	// Pets is a has-many association through Pet.UserID, using pointer elements.
	Pets []*Pet
	// NamedPet is an optional has-one Pet using Pet.UserID; it shares the foreign-key
	// convention with Pets.
	NamedPet *Pet
	// Toys is a polymorphic has-many association using Toy.OwnerID and Toy.OwnerType.
	Toys []Toy `gorm:"polymorphic:Owner"`
	// Tools is a polymorphic has-many association with explicit CustomID and Type field
	// names.
	Tools []Tools `gorm:"polymorphicType:Type;polymorphicId:CustomID"`
	// CompanyID is the nullable belongs-to key referencing Company.ID.
	CompanyID *int
	// Company is the belongs-to association selected by CompanyID.
	Company Company
	// ManagerID is the nullable self-referential key identifying another User.
	ManagerID *uint
	// Manager is the belongs-to User selected by ManagerID.
	Manager *User
	// Team contains users whose ManagerID references this user; the foreign-key tag makes
	// the self-relation explicit.
	Team []User `gorm:"foreignkey:ManagerID"`
	// Languages is a many-to-many association through the UserSpeak join-table name,
	// subject to the configured naming strategy.
	Languages []Language `gorm:"many2many:UserSpeak;"`
	// Friends is a self-referential many-to-many association through user_friends.
	Friends []*User `gorm:"many2many:user_friends;"`
	// Active exercises boolean persistence and the difference between false and an omitted
	// struct update field.
	Active bool
}

// Account is a has-one User fixture with a nullable SQL foreign key and gorm.Model
// soft-delete metadata.
type Account struct {
	gorm.Model
	// UserID references User.ID when Valid is true; Valid=false represents SQL NULL.
	UserID sql.NullInt64
	// Number is the account fixture payload.
	Number string
}

// Pet belongs to a User and owns a polymorphic Toy. Its embedded gorm.Model supplies
// the key, timestamps, and soft-delete state.
type Pet struct {
	gorm.Model
	// UserID is a nullable reference to the owning User.
	UserID *uint
	// Name is the pet fixture payload.
	Name string
	// Toy is a polymorphic has-one association distinguished by OwnerID and OwnerType.
	Toy Toy `gorm:"polymorphic:Owner;"`
}

// Toy is a polymorphic association target shared by User and Pet fixtures. OwnerID is
// intentionally a string to exercise key conversion.
type Toy struct {
	gorm.Model
	// Name is the toy fixture payload.
	Name string
	// OwnerID holds the polymorphic owner key in string form.
	OwnerID string
	// OwnerType distinguishes the owning model/table according to the polymorphic relation
	// metadata.
	OwnerType string
}

// Tools is a polymorphic fixture with custom owner-key and discriminator names. The
// plural Go type name is intentional test data.
type Tools struct {
	gorm.Model
	// Name is the tool fixture payload.
	Name string
	// CustomID holds the owner key selected by the polymorphicId tag.
	CustomID string
	// Type is the discriminator selected by the polymorphicType tag.
	Type string
}

// Company is a minimal belongs-to target with an explicit integer key and no gorm.Model
// timestamps or soft deletion.
type Company struct {
	// ID is the primary key recognized by GORM naming conventions.
	ID int
	// Name is the company fixture payload.
	Name string
}

// Language is a many-to-many target with an explicitly tagged string primary key.
type Language struct {
	// Code is the non-integer primary key used by the User.Languages join table.
	Code string `gorm:"primarykey"`
	// Name is the language fixture payload.
	Name string
}

// Coupon exercises explicitly tagged keys, column names, and a has-many constraint with
// cascading deletion where the database enforces it.
type Coupon struct {
	// ID is the explicitly tagged primary key; the size tag is retained as schema-parser
	// fixture data.
	ID int `gorm:"primarykey; size:255"`
	// AppliesToProduct contains association rows linked by CouponProduct.CouponId, with an
	// OnDelete:CASCADE constraint tag.
	AppliesToProduct []*CouponProduct `gorm:"foreignKey:CouponId;constraint:OnDelete:CASCADE"`
	// AmountOff maps to the explicitly named amount_off column.
	AmountOff uint32 `gorm:"column:amount_off"`
	// PercentOff maps to the explicitly named percent_off column.
	PercentOff float32 `gorm:"column:percent_off"`
}

// CouponProduct uses a composite primary key to associate a coupon with a product
// identifier. It intentionally does not embed gorm.Model.
type CouponProduct struct {
	// CouponId is both the coupon foreign key and the integer component of the composite
	// primary key.
	CouponId int `gorm:"primarykey;size:255"`
	// ProductId is the string component of the composite primary key.
	ProductId string `gorm:"primarykey;size:255"`
	// Desc is an ordinary descriptive payload field.
	Desc string
}

// Order combines gorm.Model with a nullable Coupon association and an intentionally
// string-typed coupon key for schema and conversion tests.
type Order struct {
	gorm.Model
	// Num is the order fixture payload.
	Num string
	// Coupon is an optional belongs-to association selected by CouponID.
	Coupon *Coupon
	// CouponID is the string-typed belongs-to key, contrasting with Coupon.ID in the test
	// model.
	CouponID string
}

// Parent exercises two relations to Child: a selected favorite through FavChildID and a
// has-many collection through Child.ParentID.
type Parent struct {
	gorm.Model
	// FavChildID identifies the selected favorite Child; its zero value is a Go zero key,
	// not a nullable pointer.
	FavChildID uint
	// FavChild is the belongs-to association selected by FavChildID.
	FavChild *Child
	// Children is the has-many association through Child.ParentID.
	Children []*Child
}

// Child is the reciprocal side of the Parent fixture graph, with a nullable parent key
// and gorm.Model metadata.
type Child struct {
	gorm.Model
	// Name is the child fixture payload.
	Name string
	// ParentID is a nullable reference to Parent.ID.
	ParentID *uint
	// Parent is the optional belongs-to association selected by ParentID.
	Parent *Parent
}
