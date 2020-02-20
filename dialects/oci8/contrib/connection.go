package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/mattn/go-oci8"
)

const maxAttemtps = 60
const wait = 10

func main() {
	var host string
	flag.StringVar(&host, "h", "oracle", "hostname for oracle connection")
	flag.Parse()
	dbDSN := os.Getenv("GORM_DSN")
	if dbDSN == "" {
		dbDSN = fmt.Sprintf("gorm/gorm@%s:1521/XEPDB1", host)
	}
	fmt.Println("connecting to: ", dbDSN)
	for i := 0; i < maxAttemtps; i++ {
		db, err := sql.Open("oci8", dbDSN)
		if err != nil {
			fmt.Println(err.Error())
			fmt.Println("sleeping: ", wait, " seconds...")
			time.Sleep(wait * time.Second)
			continue
		}
		q := "select sysdate from dual"
		rows, err := db.Query(q)
		if err != nil {
			fmt.Println(err.Error())
			fmt.Println("sleeping: ", wait, " seconds...")
			time.Sleep(wait * time.Second)
			continue
		}
		defer rows.Close()
		var thedate string
		for rows.Next() {
			err := rows.Scan(&thedate)
			if err != nil {
				fmt.Println(err.Error())
				fmt.Println("sleeping: ", wait, " seconds...")
				time.Sleep(wait * time.Second)
				continue
			}
		}
		fmt.Println("connected to oracle...")
		return
	}
	fmt.Println("unable to connect.")
	os.Exit(-1)
}
