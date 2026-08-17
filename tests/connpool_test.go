package tests_test

import (
	"context"
	"database/sql"
	"os"
	"reflect"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

type wrapperTx struct {
	*sql.Tx
	conn *wrapperConnPool
}

func (w *wrapperTx) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	w.conn.got = append(w.conn.got, query)
	return w.Tx.PrepareContext(ctx, query)
}

func (w *wrapperTx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	w.conn.got = append(w.conn.got, query)
	return w.Tx.ExecContext(ctx, query, args...)
}

func (w *wrapperTx) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	w.conn.got = append(w.conn.got, query)
	return w.Tx.QueryContext(ctx, query, args...)
}

func (w *wrapperTx) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	w.conn.got = append(w.conn.got, query)
	return w.Tx.QueryRowContext(ctx, query, args...)
}

type wrapperConnPool struct {
	db     *sql.DB
	got    []string
	expected []string
}

func (c *wrapperConnPool) Ping() error {
	return c.db.Ping()
}

// If you use BeginTx returned *sql.Tx as shown below then you can't record queries in a transaction.
//
//	func (c *wrapperConnPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
//		 return c.db.BeginTx(ctx, opts)
//	}
//
// You should use BeginTx returned gorm.Tx which could wrap *sql.Tx then you can record all queries.
func (w *wrapperConnPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	tx, err := w.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &wrapperTx{Tx: tx, conn: w}, nil
}

func (w *wrapperConnPool) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	w.got = append(w.got, query)
	return w.db.PrepareContext(ctx, query)
}

func (w *wrapperConnPool) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	w.got = append(w.got, query)
	return w.db.ExecContext(ctx, query, args...)
}

func (w *wrapperConnPool) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	w.got = append(w.got, query)
	return w.db.QueryContext(ctx, query, args...)
}

func (w *wrapperConnPool) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	w.got = append(w.got, query)
	return w.db.QueryRowContext(ctx, query, args...)
}

func TestConnPoolWrapper(t *testing.T) {
	dialect := os.Getenv("GORM_DIALECT")
	if dialect != "mysql" {
		t.SkipNow()
	}

	dbDSN := os.Getenv("GORM_DSN")
	if dbDSN == "" {
		dbDSN = "gorm:gorm@tcp(localhost:9910)/gorm?charset=utf8&parseTime=True&loc=Local"
	}
	nativeDB, err := sql.Open("mysql", dbDSN)
	if err != nil {
		t.Fatalf("Should open db success, but got %v", err)
	}

	conn := &wrapperConnPool{
		db: nativeDB,
		expected: []string{
			"SELECT VERSION()",
			"INSERT INTO `users` (`created_at`,`updated_at`,`deleted_at`,`name`,`age`,`birthday`,`company_id`,`manager_id`,`active`) VALUES (?,?,?,?,?,?,?,?,?)",
			"SELECT * FROM `users` WHERE name = ? AND `users`.`deleted_at` IS NULL ORDER BY `users`.`id` LIMIT ?",
			"INSERT INTO `users` (`created_at`,`updated_at`,`deleted_at`,`name`,`age`,`birthday`,`company_id`,`manager_id`,`active`) VALUES (?,?,?,?,?,?,?,?,?)",
			"SELECT * FROM `users` WHERE name = ? AND `users`.`deleted_at` IS NULL ORDER BY `users`.`id` LIMIT ?",
			"SELECT * FROM `users` WHERE name = ? AND `users`.`deleted_at` IS NULL ORDER BY `users`.`id` LIMIT ?",
			"INSERT INTO `users` (`created_at`,`updated_at`,`deleted_at`,`name`,`age`,`birthday`,`company_id`,`manager_id`,`active`) VALUES (?,?,?,?,?,?,?,?,?)",
			"SELECT * FROM `users` WHERE name = ? AND `users`.`deleted_at` IS NULL ORDER BY `users`.`id` LIMIT ?",
			"SELECT * FROM `users` WHERE name = ? AND `users`.`deleted_at` IS NULL ORDER BY `users`.`id` LIMIT ?",
		},
	}

	defer func() {
		if !reflect.DeepEqual(conn.got, conn.expected) {
			t.Errorf("expect %#v but got %#v", conn.expected, conn.got)
		}
	}()

	db, err := gorm.Open(mysql.New(mysql.Config{Conn: conn, DisableWithReturning: true}))
	db.Logger = DB.Logger
	if err != nil {
		t.Fatalf("Should open db success, but got %v", err)
	}

	tx := db.Begin()
	user := *GetUser("transaction", Config{})

	if err = tx.Save(&user).Error; err != nil {
		t.Fatalf("No error should raise, but got %v", err)
	}

	if err = tx.First(&User{}, "name = ?", "transaction").Error; err != nil {
		t.Fatalf("Should find saved record, but got %v", err)
	}

	user1 := *GetUser("transaction1-1", Config{})

	if err = tx.Save(&user1).Error; err != nil {
		t.Fatalf("No error should raise, but got %v", err)
	}

	if err = tx.First(&User{}, "name = ?", user1.Name).Error; err != nil {
		t.Fatalf("Should find saved record, but got %v", err)
	}

	if sqlTx, ok := tx.Statement.ConnPool.(gorm.TxCommitter); !ok || sqlTx == nil {
		t.Fatalf("Should return the underlying sql.Tx")
	}

	tx.Rollback()

	if err = db.First(&User{}, "name = ?", "transaction").Error; err == nil {
		t.Fatalf("Should not find record after rollback, but got %v", err)
	}

	txDB := db.Where("fake_name = ?", "fake_name")
	tx2 := txDB.Session(&gorm.Session{NewDB: true}).Begin()
	user2 := *GetUser("transaction-2", Config{})
	if err = tx2.Save(&user2).Error; err != nil {
		t.Fatalf("No error should raise, but got %v", err)
	}

	if err = tx2.First(&User{}, "name = ?", "transaction-2").Error; err != nil {
		t.Fatalf("Should find saved record, but got %v", err)
	}

	tx2.Commit()

	if err = db.First(&User{}, "name = ?", "transaction-2").Error; err != nil {
		t.Fatalf("Should be able to find committed record, but got %v", err)
	}
}
