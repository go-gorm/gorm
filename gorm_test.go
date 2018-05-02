package gorm

import (
	"fmt"
	"testing"

	"github.com/Jetereting/gorm"
	_ "github.com/go-sql-driver/mysql"
)

var (
	db  *gorm.DB
	err error
)

func init() {
	db, err = gorm.Open("mysql", "user:password@tcp(ip:port)/dbName?charset=utf8")
	if err != nil {
		fmt.Println(err)
		return
	}
}

// TestQuery 测试查询
func TestQuery(t *testing.T) {
	datas, e := db.RawMap("select * from users where user_id=?", 123)
	if e != nil {
		fmt.Println("err:", e)
	}
	fmt.Println("datas:", datas)
}

// TestIsCanInsert 测试插入
func TestIsCanInsert(t *testing.T) {
	_, e := db.RawMap("insert into users(user_id,user_name,user_tag) values (?,?,?)", 123, "testName", "testTag")
	if e != nil {
		fmt.Println("err:", e)
	}
	fmt.Println("It work!")
}

// TestIsCanUpdate 测试更新
func TestIsCanUpdate(t *testing.T) {
	_, e := db.RawMap("update users set user_name=? where user_id=?", "testName2", 123)
	if e != nil {
		fmt.Println("err:", e)
	}
	fmt.Println("It work!")
}

// TestTX 测试事物
func TestTX(t *testing.T) {
	tx := db.Begin()
	_, e := tx.RawMap("update users set user_name=? where user_id=?", "testName3", 123)
	if e != nil {
		fmt.Println("err:", e)
		tx.Rollback()
	}
	_, e = tx.RawMap("update users set user_name=? where user_id=?", "long text....long text....long text....long text....long text....long text....long text....", 123)
	if e != nil {
		fmt.Println("err:", e)
		tx.Rollback()
	}
	tx.Commit()
	fmt.Println("done!")
}
