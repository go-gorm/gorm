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
	if !scope.Dialect().HasColumn(scope, scope.TableName(), col) {
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
