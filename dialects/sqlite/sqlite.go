package sqlite

import (
	"database/sql"
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/common/sqlbuilder"
)

// Dialect Sqlite3 Dialect for GORM
type Dialect struct {
	DB *sql.DB
}

// Quote quote for value
func (dialect Dialect) Quote(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}

// Insert insert
func (dialect *Dialect) Insert(tx *gorm.DB) (err error) {
	s, args, defaultFieldsSlice, err := sqlbuilder.BuildInsertSQL(tx)

	if err == nil {
		result, err := dialect.DB.Exec(s.String(), args...)

		if err == nil {
			var lastInsertID int64
			tx.RowsAffected, _ = result.RowsAffected()
			lastInsertID, err = result.LastInsertId()
			if len(defaultFieldsSlice) == int(tx.RowsAffected) {
				startID := lastInsertID - tx.RowsAffected + 1
				for i, defaultFields := range defaultFieldsSlice {
					if len(defaultFields) > 0 {
						tx.AddError(defaultFields[0].Set(startID + int64(i)))
					}
				}
			}
		}
	}

	return
}

// Query query
func (*Dialect) Query(tx *gorm.DB) error {
	return nil
}

// Update update
func (*Dialect) Update(tx *gorm.DB) error {
	return nil
}

// Delete delete
func (*Dialect) Delete(tx *gorm.DB) error {
	return nil
}
