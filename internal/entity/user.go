package entity

type User struct {
	ID uint

	Name string `gorm:"column:name"`
	Email string `gorm:"column:email"`
}
