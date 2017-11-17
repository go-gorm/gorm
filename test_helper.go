package gorm

import (
	"fmt"
)

// TestHelper is the exported struct used for setting expectations
type TestHelper struct {
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

// NewTestHelper returns a fresh TestHelper with an arbitary Adapter
func NewTestHelper(adapter Adapter) (error, *DB, *TestHelper) {
	err, asserter := adapter.Open()

	if err != nil {
		return err, nil, nil
	}

	gormDb, err := Open("sqlmock", "mock_gorm_dsn")

	if err != nil {
		return err, nil, nil
	}

	return nil, gormDb, &TestHelper{adapter: adapter, asserter: asserter}
}

// NewDefaultTestHelper returns a TestHelper powered by go-sqlmock
func NewDefaultTestHelper() (error, *DB, *TestHelper) {
	adapter := &SqlmockAdapter{}
	err, asserter := adapter.Open()

	if err != nil {
		return err, nil, nil
	}

	gormDb, err := Open("sqlmock", "mock_gorm_dsn")

	if err != nil {
		return err, nil, nil
	}

	return nil, gormDb, &TestHelper{adapter: adapter, asserter: asserter}
}
