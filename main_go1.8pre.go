// +build !go1.8

package gorm

// Begin starts a transaction
func (s *DB) Begin() *DB {
	c := s.clone()
	if db, ok := c.db.(sqlDb); ok && db != nil {
		tx, err := db.Begin()
		c.db = interface{}(tx).(SQLCommon)
		c.AddError(err)
	} else {
		c.AddError(ErrCantStartTransaction)
	}
	return c
}
