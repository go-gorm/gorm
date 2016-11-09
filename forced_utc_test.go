package gorm_test

import (
	"os"
	"testing"
	"time"
)

// table that is not created automatically
type ExternalData struct {
	Id   int
	Time time.Time
}

func TestForcedUTC(t *testing.T) {
	if dialect := os.Getenv("GORM_DIALECT"); dialect != "postgres" {
		t.Skip("Skipping this because this is to test postgres issues with timestamps without timezones")
	}
	db := DB.New()
	db.DropTableIfExists(&ExternalData{})
	db.Exec(`
		CREATE TABLE IF NOT EXISTS external_data(
		  id serial PRIMARY KEY,
		  time timestamp without time zone NOT NULL
		)`)

	tm := time.Date(2000, 1, 1, 1, 0, 0, 0, time.FixedZone("test location", +7200))

	//Test without forcing utc
	elem0 := ExternalData{Time: tm}
	db.Save(&elem0)

	elem := ExternalData{}
	db.Find(&elem, elem0.Id)
	if elem.Time.Equal(tm) {
		t.Errorf("Times should not be equal (timezones)")
	}

	db.Model(&elem).Update("time", tm)
	elem = ExternalData{}
	db.Find(&elem, elem0.Id)
	if elem.Time.Equal(tm) {
		t.Errorf("Times should not be equal (timezones)")
	}

	cnt := 0

	db.Model(&ExternalData{}).Where("time = ?", tm).Count(&cnt)
	if cnt == 0 {
		t.Errorf("Timezone is cut off, data still should be found (timezones)")
	}

	db.Model(&ExternalData{}).Where("time = ?", tm.UTC()).Count(&cnt)
	if cnt != 0 {
		t.Errorf("UTC normalized time should not be found (timezones)")
	}

	//Test with forcing utc
	db.ForceUTC(true)

	elem0 = ExternalData{Time: tm}
	db.Save(&elem0)

	elem = ExternalData{}
	db.Find(&elem, elem0.Id)
	if !elem.Time.Equal(tm) {
		t.Errorf("Times should be equal (forced UTC)")
	}

	db.Model(&elem).Update("time", tm)
	elem = ExternalData{}
	db.Find(&elem, elem0.Id)
	if !elem.Time.Equal(tm) {
		t.Errorf("Times should be equal (forced UTC)")
	}

	db.Model(&ExternalData{}).Where("time = ?", tm).Count(&cnt)
	if cnt != 1 {
		t.Errorf("Record should be found (forced UTC)")
	}

	db.Model(&ExternalData{}).Where("time = ?", tm.UTC()).Count(&cnt)
	if cnt != 1 {
		t.Errorf("UTC normalized time should be found (forcedUTC)")
	}
}
