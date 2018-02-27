package sqlite

import (
	"fmt"
	"os"
	"path/filepath"
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

func TestInsert(t *testing.T) {
	type User struct {
		gorm.Model
		Name string
		Age  int
	}

	DB.Create([]*User{{Name: "name1", Age: 10}, {Name: "name2", Age: 20}})
}
