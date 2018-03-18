package sqlite

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jinzhu/gorm"
)

var DB *gorm.DB

func init() {
	var err error
	DB, err = Open(filepath.Join(os.TempDir(), "gorm.db"), Config{})
	if err != nil {
		panic(fmt.Sprintf("No error should happen when connecting to test database, but got err=%+v", err))
	}
}

func TestBatchInsert(t *testing.T) {
	type User struct {
		gorm.Model
		Name string
		Age  int
	}

	users := []*User{{Name: "name1", Age: 10}, {Name: "name2", Age: 20}, {Name: "name3", Age: 30}}

	DB.Create(users)

	for _, user := range users {
		if user.ID == 0 {
			t.Errorf("User should have primary key")
		}

		var newUser User
		if err := DB.Find(&newUser, "id = ?", user.ID).Error; err != nil {
			t.Error(err)
		}

		if !reflect.DeepEqual(&newUser, user) {
			t.Errorf("User should be equal, but got %#v, should be %#v", newUser, user)
		}
	}
}
