package oci8

import (
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/matryer/is"
)

func Test_Connection(t *testing.T) {
	is := is.New(t)
	dbDSN := os.Getenv("GORM_DSN")
	if dbDSN == "" {
		dbDSN = "gorm/gorm@localhost:1521/XEPDB1"
	}
	gDB, err := gorm.Open("oci8", dbDSN)
	is.NoErr(err)
	db := gDB.DB()
	q := "select sysdate from dual"
	rows, err := db.Query(q)
	is.NoErr(err)
	defer rows.Close()
	var thedate string
	for rows.Next() {
		err := rows.Scan(&thedate)
		is.NoErr(err)
	}
	t.Logf("The date is: %s\n", thedate)
}
