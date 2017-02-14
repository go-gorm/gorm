package main

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var Db *gorm.DB

func init() {
	var err error
	Db, err = gorm.Open("sqlite3", "test.db")
	if err != nil {
		panic(err)
	}
	Db.LogMode(true)
}

func main() {
	facilityID := "LS342"
	clientID := "222017"

	type TableOptionList struct {
		ID   int `gorm:"primary_key"`
		Name string
	}

	type TableClient struct {
		gorm.Model
		FacilityID    string            `gorm:"primary_key"`
		ClientID      string            `gorm:"primary_key"`
		SpecificNeeds []TableOptionList `gorm:"many2many:options_specific_needs;ForeignKey:facility_id,id"`

		Client   TableOptionList
		Facility TableOptionList
	}
	Db.AutoMigrate(&TableClient{})
	Db.AutoMigrate(&TableOptionList{})

	var newClient TableClient
	Db.FirstOrCreate(&newClient, TableClient{FacilityID: facilityID, ClientID: clientID})

	Db.Model(&newClient).Association("SpecificNeeds").Append([]TableOptionList{TableOptionList{ID: 1, Name: "Lusaka"}})

	//Test with standard Preload - wrong SQL that won't work on Sqlite because
	//of the IN statement
	var DbClient TableClient
	Db.Where("facility_id = ? AND client_id = ?", facilityID, clientID).
		Preload("SpecificNeeds").
		First(&DbClient)

	//Try with custom SQL - instead it runs the custom SQL then appends the bad
	//SQL too with an AND statement.
	var DbClient2 TableClient
	Db.Where("facility_id = ? AND client_id = ?", facilityID, clientID).
		Preload("SpecificNeeds", func(s *gorm.DB) *gorm.DB {
		return s.Where(
			`SELECT *
			FROM "table_option_lists"
			INNER JOIN "options_specific_needs"
			ON "options_specific_needs"."table_option_list_id" = "table_option_lists"."id"
			WHERE ("options_specific_needs"."table_client_facility_id" = ? AND "options_specific_needs"."table_client_id" = ?)`, facilityID, clientID)
	}).
		First(&DbClient2)
}
