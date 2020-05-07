package callbacks

import (
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/schema"
)

func preload(db *gorm.DB, preloadFields []string, rel *schema.Relationship) {
}
