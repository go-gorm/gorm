package gorm

import (
	"database/sql"
	"fmt"
)

// TestHelper is the exported struct used for setting expectations
type TestHelper struct {
	gormDb   *DB
	mockDb   *sql.DB
	adapter  Adapter
	asserter Asserter
}

// Close closes the DB connection
func (h *TestHelper) Close() error {
	return h.adapter.Close()
}

func (h *TestHelper) ExpectFirst(model interface{}) Query {
	fmt.Printf("Expecting query: %s", "some query\n")
	return h.asserter.Query("some sql")
}

func (h *TestHelper) ExpectFind(model interface{}) {
	fmt.Println("Expecting query: %s", "some query involving Find")
}

// NewTestHelper returns a fresh TestHelper
func NewTestHelper(adapter Adapter) (error, *DB, *TestHelper) {
	err, mockDb, gormDb, asserter := adapter.Open()

	if err != nil {
		return err, nil, nil
	}

	return nil, gormDb, &TestHelper{gormDb: gormDb, mockDb: mockDb, adapter: adapter, asserter: asserter}
}
