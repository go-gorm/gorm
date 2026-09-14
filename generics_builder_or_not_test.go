package gorm_test

import (
	"context"
	"sort"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type orNotOwner struct {
	ID   uint `gorm:"primaryKey"`
	Name string
	Pets []orNotPet `gorm:"foreignKey:OwnerID"`
}

type orNotPet struct {
	ID      uint `gorm:"primaryKey"`
	OwnerID uint
	Name    string
}

type orNotUser struct {
	ID      uint `gorm:"primaryKey"`
	Name    string
	Company orNotCompany `gorm:"foreignKey:UserID"`
}

type orNotCompany struct {
	ID     uint `gorm:"primaryKey"`
	Name   string
	UserID uint
}

func petNames(pets []orNotPet) []string {
	names := make([]string, 0, len(pets))
	for _, p := range pets {
		names = append(names, p.Name)
	}
	sort.Strings(names)
	return names
}

func equalNames(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func openOrNotDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db failed: %v", err)
	}
	if err := db.AutoMigrate(&orNotOwner{}, &orNotPet{}, &orNotUser{}, &orNotCompany{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	if err := db.Where("1 = 1").Delete(&orNotPet{}).Error; err != nil {
		t.Fatalf("clean pets failed: %v", err)
	}
	if err := db.Where("1 = 1").Delete(&orNotOwner{}).Error; err != nil {
		t.Fatalf("clean owners failed: %v", err)
	}
	if err := db.Where("1 = 1").Delete(&orNotCompany{}).Error; err != nil {
		t.Fatalf("clean companies failed: %v", err)
	}
	if err := db.Where("1 = 1").Delete(&orNotUser{}).Error; err != nil {
		t.Fatalf("clean users failed: %v", err)
	}
	return db
}

func TestGenericsPreloadBuilderOrNot(t *testing.T) {
	ctx := context.Background()
	db := openOrNotDB(t)

	owner := orNotOwner{ID: 1, Name: "o1", Pets: []orNotPet{
		{Name: "cat"}, {Name: "dog"}, {Name: "bird"},
	}}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Or: preload pets named cat OR dog -> 2 pets
	owners, err := gorm.G[orNotOwner](db).Preload("Pets", func(pb gorm.PreloadBuilder) error {
		pb.Where("name = ?", "cat").Or("name = ?", "dog")
		return nil
	}).Where("id = ?", owner.ID).Find(ctx)
	if err != nil {
		t.Fatalf("preload Or failed: %v", err)
	}
	if len(owners) != 1 {
		t.Fatalf("expected 1 owner, got %d", len(owners))
	}
	if got := petNames(owners[0].Pets); !equalNames(got, []string{"cat", "dog"}) {
		t.Fatalf("preload Or expected pets [cat dog], got %v", got)
	}

	// Not alone: preload pets not named bird -> 2 pets (cat, dog)
	owners, err = gorm.G[orNotOwner](db).Preload("Pets", func(pb gorm.PreloadBuilder) error {
		pb.Not("name = ?", "bird")
		return nil
	}).Where("id = ?", owner.ID).Find(ctx)
	if err != nil {
		t.Fatalf("preload Not failed: %v", err)
	}
	if len(owners) != 1 {
		t.Fatalf("expected 1 owner, got %d", len(owners))
	}
	if got := petNames(owners[0].Pets); !equalNames(got, []string{"cat", "dog"}) {
		t.Fatalf("preload Not expected pets [cat dog], got %v", got)
	}

	// Where + Not: name = cat AND NOT name = dog -> 1 pet (cat)
	owners, err = gorm.G[orNotOwner](db).Preload("Pets", func(pb gorm.PreloadBuilder) error {
		pb.Where("name = ?", "cat").Not("name = ?", "dog")
		return nil
	}).Where("id = ?", owner.ID).Find(ctx)
	if err != nil {
		t.Fatalf("preload Where+Not failed: %v", err)
	}
	if len(owners) != 1 {
		t.Fatalf("expected 1 owner, got %d", len(owners))
	}
	if got := petNames(owners[0].Pets); !equalNames(got, []string{"cat"}) {
		t.Fatalf("preload Where+Not expected pets [cat], got %v", got)
	}
}

func TestGenericsJoinBuilderOrNot(t *testing.T) {
	ctx := context.Background()
	db := openOrNotDB(t)

	// Each company name is unique to its user, so results are independent
	// of SQL precedence between the FK condition and the OR branches.
	users := []orNotUser{
		{Name: "u1", Company: orNotCompany{Name: "alpha"}},
		{Name: "u2", Company: orNotCompany{Name: "beta"}},
		{Name: "u3", Company: orNotCompany{Name: "gamma"}},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("create failed: %v", err)
	}

	joinedNames := func(t *testing.T, build func(jb gorm.JoinBuilder, joinTable clause.Table, curTable clause.Table) error) []string {
		t.Helper()
		results, err := gorm.G[orNotUser](db).Joins(clause.Has("Company"), build).Find(ctx)
		if err != nil {
			t.Fatalf("joins failed: %v", err)
		}
		names := make([]string, 0, len(results))
		for _, u := range results {
			names = append(names, u.Name)
		}
		sort.Strings(names)
		return names
	}

	// Or: company name = alpha OR gamma -> users u1, u3
	got := joinedNames(t, func(jb gorm.JoinBuilder, joinTable clause.Table, curTable clause.Table) error {
		jb.Where("?.name = ?", joinTable, "alpha").Or("?.name = ?", joinTable, "gamma")
		return nil
	})
	if !equalNames(got, []string{"u1", "u3"}) {
		t.Fatalf("joins Or expected users [u1 u3], got %v", got)
	}

	// Not alone: company name NOT beta -> users u1, u3
	got = joinedNames(t, func(jb gorm.JoinBuilder, joinTable clause.Table, curTable clause.Table) error {
		jb.Not("?.name = ?", joinTable, "beta")
		return nil
	})
	if !equalNames(got, []string{"u1", "u3"}) {
		t.Fatalf("joins Not expected users [u1 u3], got %v", got)
	}

	// Where + Not: name = alpha AND NOT name = beta -> user u1
	got = joinedNames(t, func(jb gorm.JoinBuilder, joinTable clause.Table, curTable clause.Table) error {
		jb.Where("?.name = ?", joinTable, "alpha").Not("?.name = ?", joinTable, "beta")
		return nil
	})
	if !equalNames(got, []string{"u1"}) {
		t.Fatalf("joins Where+Not expected users [u1], got %v", got)
	}
}
