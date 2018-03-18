package sqlite

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/tests"
)

var DB *gorm.DB

func init() {
	var err error
	DB, err = Open(filepath.Join(os.TempDir(), "gorm.db"), Config{})
	if err != nil {
		panic(fmt.Sprintf("No error should happen when connecting to test database, but got err=%+v", err))
	}
}

func TestAll(t *testing.T) {
	tests.RunCreateSuit(DB, t)
}
