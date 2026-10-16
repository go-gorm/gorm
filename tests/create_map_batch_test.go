package tests_test

import (
	"fmt"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestCreateInBatchesFromMap(t *testing.T) {
	type BatchMapUser struct {
		ID   uint
		Name string
		Age  int
	}
	if err := DB.Migrator().DropTable(&BatchMapUser{}); err != nil {
		t.Fatal(err)
	}
	if err := DB.AutoMigrate(&BatchMapUser{}); err != nil {
		t.Fatal(err)
	}

	for _, returning := range []string{"automatic", "all", "partial"} {
		if returning != "automatic" && DB.Dialector.Name() != "postgres" && DB.Dialector.Name() != "sqlite" {
			continue
		}
		for _, pointer := range []bool{false, true} {
			for _, configured := range []bool{false, true} {
				for _, size := range []struct{ count, batch int }{{1, 500}, {5, 2}} {
					t.Run(fmt.Sprintf("%s/pointer=%t/configured=%t/count=%d", returning, pointer, configured, size.count), func(t *testing.T) {
						// Keep spare capacity and a sentinel to detect writes beyond the input,
						// as well as corruption of subsequent batches sharing the same array.
						backing := make([]map[string]interface{}, size.count+1)
						backing[size.count] = map[string]interface{}{"sentinel": true}
						rows := backing[:size.count]
						for i := range rows {
							rows[i] = map[string]interface{}{"Name": fmt.Sprintf("%s_%d", t.Name(), i), "Age": 18 + i}
						}
						var value interface{} = rows
						if pointer {
							value = &rows
						}
						tx := DB.Model(&BatchMapUser{})
						if returning == "all" {
							tx = tx.Clauses(clause.Returning{})
						}
						if returning == "partial" {
							tx = tx.Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}, {Name: "age"}}})
						}
						var result *gorm.DB
						if configured {
							result = tx.Session(&gorm.Session{CreateBatchSize: size.batch}).Create(value)
						} else {
							result = tx.CreateInBatches(value, size.batch)
						}
						if result.Error != nil {
							t.Fatal(result.Error)
						}
						if result.RowsAffected != int64(size.count) || len(rows) != size.count {
							t.Fatalf("rows affected=%d, input length=%d, want %d", result.RowsAffected, len(rows), size.count)
						}
						if backing[size.count]["sentinel"] != true {
							t.Fatalf("overwrote spare capacity: %v", backing[size.count])
						}
						var stored []BatchMapUser
						if err := DB.Where("name LIKE ?", t.Name()+"_%").Order("id").Find(&stored).Error; err != nil {
							t.Fatal(err)
						}
						if len(stored) != size.count {
							t.Fatalf("stored %d rows, want %d: %+v", len(stored), size.count, stored)
						}
						for i, user := range stored {
							name := fmt.Sprintf("%s_%d", t.Name(), i)
							if user.Name != name || user.Age != 18+i || user.ID == 0 {
								t.Fatalf("row %d: %+v", i, user)
							}
							if fmt.Sprint(rows[i]["id"]) != fmt.Sprint(user.ID) {
								t.Fatalf("row %d: returned id=%v, stored id=%d", i, rows[i]["id"], user.ID)
							}
							nameKey, ageKey := "Name", "Age"
							if returning == "all" {
								nameKey, ageKey = "name", "age"
							}
							if rows[i][nameKey] != name || fmt.Sprint(rows[i][ageKey]) != fmt.Sprint(18+i) {
								t.Fatalf("row %d lost input fields: %v", i, rows[i])
							}
							if returning == "partial" && fmt.Sprint(rows[i]["age"]) != fmt.Sprint(user.Age) {
								t.Fatalf("row %d missing returned age: %v", i, rows[i])
							}
						}
					})
				}
			}
		}
	}
}

func TestCreateInBatchesFromMapRollback(t *testing.T) {
	type BatchMapUniqueUser struct {
		ID   uint
		Name string `gorm:"unique"`
	}
	for _, skip := range []bool{false, true} {
		for _, pointer := range []bool{false, true} {
			for _, configured := range []bool{false, true} {
				t.Run(fmt.Sprintf("skip=%t/pointer=%t/configured=%t", skip, pointer, configured), func(t *testing.T) {
					if err := DB.Migrator().DropTable(&BatchMapUniqueUser{}); err != nil {
						t.Fatal(err)
					}
					if err := DB.AutoMigrate(&BatchMapUniqueUser{}); err != nil {
						t.Fatal(err)
					}
					rows := []map[string]interface{}{{"Name": "alice"}, {"Name": "bob"}, {"Name": "alice"}}
					var value interface{} = rows
					if pointer {
						value = &rows
					}
					tx := DB.Session(&gorm.Session{SkipDefaultTransaction: skip}).Model(&BatchMapUniqueUser{})
					var result *gorm.DB
					if configured {
						result = tx.Session(&gorm.Session{CreateBatchSize: 2}).Create(value)
					} else {
						result = tx.CreateInBatches(value, 2)
					}
					if result.Error == nil {
						t.Fatal("expected unique constraint error in the second batch")
					}
					if result.RowsAffected != 2 {
						t.Fatalf("completed batch rows=%d, want 2", result.RowsAffected)
					}
					var count int64
					if err := DB.Model(&BatchMapUniqueUser{}).Count(&count).Error; err != nil {
						t.Fatal(err)
					}
					var want int64
					if skip {
						want = 2
					}
					if count != want {
						t.Fatalf("persisted %d rows, want %d", count, want)
					}
					if rows[2]["Name"] != "alice" {
						t.Fatalf("second batch corrupted: %v", rows[2])
					}
				})
			}
		}
	}
}

func TestScanMapSliceReplacesPreviousResults(t *testing.T) {
	type MapScanUser struct {
		ID   uint
		Name string
	}
	if err := DB.Migrator().DropTable(&MapScanUser{}); err != nil {
		t.Fatal(err)
	}
	if err := DB.AutoMigrate(&MapScanUser{}); err != nil {
		t.Fatal(err)
	}
	user := MapScanUser{Name: "alice"}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	rows := []map[string]interface{}{{"stale": true}}
	if err := DB.Model(&MapScanUser{}).Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["name"] != "alice" || rows[0]["stale"] != nil {
		t.Fatalf("unexpected results: %v", rows)
	}
	if err := DB.Model(&MapScanUser{}).Where("1 = 0").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected empty results, got %v", rows)
	}
}
