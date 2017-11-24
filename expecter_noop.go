package gorm

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"sync"
)

var pool *NoopDriver

func init() {
	pool = &NoopDriver{
		conns: make(map[string]*NoopConnection),
	}

	sql.Register("noop", pool)
}

// NoopDriver implements sql/driver.Driver
type NoopDriver struct {
	sync.Mutex
	counter int
	conns   map[string]*NoopConnection
}

// Open implements sql/driver.Driver
func (d *NoopDriver) Open(dsn string) (driver.Conn, error) {
	d.Lock()
	defer d.Unlock()

	c, ok := d.conns[dsn]

	if !ok {
		return c, fmt.Errorf("No connection available")
	}

	c.opened++
	return c, nil
}

// NoopResult is a noop struct that satisfies sql.Result
type NoopResult struct{}

// LastInsertId is a noop method for satisfying drive.Result
func (r NoopResult) LastInsertId() (int64, error) {
	return 0, nil
}

// RowsAffected is a noop method for satisfying drive.Result
func (r NoopResult) RowsAffected() (int64, error) {
	return 0, nil
}

// NoopRows implements driver.Rows
type NoopRows struct {
	pos int
}

// Columns implements driver.Rows
func (r *NoopRows) Columns() []string {
	return []string{"foo", "bar", "baz", "lol", "kek", "zzz"}
}

// Close implements driver.Rows
func (r *NoopRows) Close() error {
	return nil
}

// Next implements driver.Rows and alwys returns only one row
func (r *NoopRows) Next(dest []driver.Value) error {
	if r.pos == 1 {
		return io.EOF
	}
	cols := []string{"foo", "bar", "baz", "lol", "kek", "zzz"}

	for i, col := range cols {
		dest[i] = col
	}

	r.pos++

	return nil
}

// NoopStmt implements driver.Stmt
type NoopStmt struct{}

// Close implements driver.Stmt
func (s *NoopStmt) Close() error {
	return nil
}

// NumInput implements driver.Stmt
func (s *NoopStmt) NumInput() int {
	return 1
}

// Exec implements driver.Stmt
func (s *NoopStmt) Exec(args []driver.Value) (driver.Result, error) {
	return &NoopResult{}, nil
}

// Query implements driver.Stmt
func (s *NoopStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &NoopRows{}, nil
}

// NewNoopDB initialises a new DefaultNoopDB
func NewNoopDB() (SQLCommon, error) {
	pool.Lock()
	dsn := fmt.Sprintf("noop_db_%d", pool.counter)
	pool.counter++

	noop := &NoopConnection{dsn: dsn, drv: pool}
	pool.conns[dsn] = noop
	pool.Unlock()

	db, err := noop.open()

	return db, err
}

// NoopConnection implements sql/driver.Conn
// for our purposes, the noop connection never returns an error, as we only
// require it for generating queries. It is necessary because eagerloading
// will fail if any operation returns an error
type NoopConnection struct {
	dsn    string
	drv    *NoopDriver
	opened int
}

func (c *NoopConnection) open() (*sql.DB, error) {
	db, err := sql.Open("noop", c.dsn)

	if err != nil {
		return db, err
	}

	return db, db.Ping()
}

// Close implements sql/driver.Conn
func (c *NoopConnection) Close() error {
	c.drv.Lock()
	defer c.drv.Unlock()

	c.opened--
	if c.opened == 0 {
		delete(c.drv.conns, c.dsn)
	}

	return nil
}

// Begin implements sql/driver.Conn
func (c *NoopConnection) Begin() (driver.Tx, error) {
	return c, nil
}

// Exec implements sql/driver.Conn
func (c *NoopConnection) Exec(query string, args []driver.Value) (driver.Result, error) {
	return NoopResult{}, nil
}

// Prepare implements sql/driver.Conn
func (c *NoopConnection) Prepare(query string) (driver.Stmt, error) {
	return &NoopStmt{}, nil
}

// Query implements sql/driver.Conn
func (c *NoopConnection) Query(query string, args []driver.Value) (driver.Rows, error) {
	return &NoopRows{}, nil
}

// Commit implements sql/driver.Conn
func (c *NoopConnection) Commit() error {
	return nil
}

// Rollback implements sql/driver.Conn
func (c *NoopConnection) Rollback() error {
	return nil
}
