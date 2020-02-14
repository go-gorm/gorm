package oci8

import (
	"os"
	"testing"

	"github.com/jinzhu/gorm"
)

func Test_Connection(t *testing.T) {
	dbDSN := os.Getenv("GORM_DSN")
	if dbDSN == "" {
		dbDSN = "gorm/gorm@localhost:1521/XEPDB1"
	}
	gDB, err := gorm.Open("oci8", dbDSN)
	if err != nil {
		t.Errorf("connection error: %s", err.Error())
	}
	db := gDB.DB()
	q := "select sysdate from dual"
	rows, err := db.Query(q)
	if err != nil {
		t.Errorf("query error: %s", err.Error())
	}
	defer rows.Close()
	var thedate string
	for rows.Next() {
		err := rows.Scan(&thedate)
		if err != nil {
			t.Errorf("scan error: %s", err.Error())
		}
	}
	t.Logf("The date is: %s\n", thedate)
}
