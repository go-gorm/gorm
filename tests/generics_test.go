package tests_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	. "gorm.io/gorm/utils/tests"
)

func TestGenericsCreate(t *testing.T) {
	ctx := context.Background()

	user := User{Name: "TestGenericsCreate", Age: 18}
	err := gorm.G[User](DB).Create(ctx, &user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if user.ID == 0 {
		t.Fatalf("no primary key found for %v", user)
	}

	if u, err := gorm.G[User](DB).Where("name = ?", user.Name).First(ctx); err != nil {
		t.Fatalf("failed to find user, got error: %v", err)
	} else if u.Name != user.Name || u.ID != user.ID {
		t.Errorf("found invalid user, got %v, expect %v", u, user)
	}

	if u, err := gorm.G[User](DB).Where("name = ?", user.Name).Take(ctx); err != nil {
		t.Fatalf("failed to find user, got error: %v", err)
	} else if u.Name != user.Name || u.ID != user.ID {
		t.Errorf("found invalid user, got %v, expect %v", u, user)
	}

	if u, err := gorm.G[User](DB).Select("name").Where("name = ?", user.Name).First(ctx); err != nil {
		t.Fatalf("failed to find user, got error: %v", err)
	} else if u.Name != user.Name || u.Age != 0 {
		t.Errorf("found invalid user, got %v, expect %v", u, user)
	}

	if u, err := gorm.G[User](DB).Omit("name").Where("name = ?", user.Name).First(ctx); err != nil {
		t.Fatalf("failed to find user, got error: %v", err)
	} else if u.Name != "" || u.Age != user.Age {
		t.Errorf("found invalid user, got %v, expect %v", u, user)
	}

	result := struct {
		ID   int
		Name string
	}{}
	if err := gorm.G[User](DB).Where("name = ?", user.Name).Scan(ctx, &result); err != nil {
		t.Fatalf("failed to scan user, got error: %v", err)
	} else if result.Name != user.Name || uint(result.ID) != user.ID {
		t.Errorf("found invalid user, got %v, expect %v", result, user)
	}

	mapResult, err := gorm.G[map[string]interface{}](DB).Table("users").Where("name = ?", user.Name).MapColumns(map[string]string{"name": "user_name"}).Take(ctx)
	if v := mapResult["user_name"]; fmt.Sprint(v) != user.Name {
		t.Errorf("failed to find map results, got %v, err %v", mapResult, err)
	}

	selectOnly := User{Name: "GenericsCreateSelectOnly", Age: 99}
	if err := gorm.G[User](DB).Select("name").Create(ctx, &selectOnly); err != nil {
		t.Fatalf("failed to create with Select, got error: %v", err)
	}

	if selectOnly.ID == 0 {
		t.Fatalf("no primary key found for select-only user: %v", selectOnly)
	}

	if stored, err := gorm.G[User](DB).Where("id = ?", selectOnly.ID).First(ctx); err != nil {
		t.Fatalf("failed to reload select-only user, got error: %v", err)
	} else if stored.Name != selectOnly.Name || stored.Age != 0 {
		t.Errorf("unexpected select-only user state, got %#v", stored)
	}

	omitAge := User{Name: "GenericsCreateOmitAge", Age: 88}
	if err := gorm.G[User](DB).Omit("age").Create(ctx, &omitAge); err != nil {
		t.Fatalf("failed to create with Omit, got error: %v", err)
	}

	if omitAge.ID == 0 {
		t.Fatalf("no primary key found for omit-age user: %v", omitAge)
	}

	if stored, err := gorm.G[User](DB).Where("id = ?", omitAge.ID).First(ctx); err != nil {
		t.Fatalf("failed to reload omit-age user, got error: %v", err)
	} else if stored.Name != omitAge.Name || stored.Age != 0 {
		t.Errorf("unexpected omit-age user state, got %#v", stored)
	}
}

func TestGenericsCreateInBatches(t *testing.T) {
	batch := []User{
		{Name: "GenericsCreateInBatches1"},
		{Name: "GenericsCreateInBatches2"},
		{Name: "GenericsCreateInBatches3"},
	}
	ctx := context.Background()

	if err := gorm.G[User](DB).CreateInBatches(ctx, &batch, 2); err != nil {
		t.Fatalf("CreateInBatches failed: %v", err)
	}

	for _, u := range batch {
		if u.ID == 0 {
			t.Fatalf("no primary key found for %v", u)
		}
	}

	count, err := gorm.G[User](DB).Where("name like ?", "GenericsCreateInBatches%").Count(ctx, "*")
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 records, got %d", count)
	}

	found, err := gorm.G[User](DB).Raw("SELECT * FROM users WHERE name LIKE ?", "GenericsCreateInBatches%").Find(ctx)
	if len(found) != len(batch) {
		t.Errorf("expected %d from Raw Find, got %d", len(batch), len(found))
	}

	found, err = gorm.G[User](DB).Where("name like ?", "GenericsCreateInBatches%").Limit(2).Find(ctx)
	if len(found) != 2 {
		t.Errorf("expected %d from Raw Find, got %d", 2, len(found))
	}

	found, err = gorm.G[User](DB).Where("name like ?", "GenericsCreateInBatches%").Offset(2).Limit(2).Find(ctx)
	if len(found) != 1 {
		t.Errorf("expected %d from Raw Find, got %d", 1, len(found))
	}
}

func TestGenericsExecAndUpdate(t *testing.T) {
	ctx := context.Background()

	name := "GenericsExec"
	if err := gorm.G[User](DB).Exec(ctx, "INSERT INTO users(name) VALUES(?)", name); err != nil {
		t.Fatalf("Exec insert failed: %v", err)
	}

	name2 := "GenericsExec2"
	if err := gorm.G[User](DB).Exec(ctx, "INSERT INTO ?(name) VALUES(?)", clause.Table{Name: clause.CurrentTable}, name2); err != nil {
		t.Fatalf("Exec insert failed: %v", err)
	}

	u, err := gorm.G[User](DB).Table("users as u").Where("u.name = ?", name).First(ctx)
	if err != nil {
		t.Fatalf("failed to find user, got error: %v", err)
	} else if u.Name != name || u.ID == 0 {
		t.Errorf("found invalid user, got %v", u)
	}

	name += "Update"
	rows, err := gorm.G[User](DB).Where("id = ?", u.ID).Update(ctx, "name", name)
	if rows != 1 {
		t.Fatalf("failed to get affected rows, got %d, should be %d", rows, 1)
	}

	nu, err := gorm.G[User](DB).Where("name = ?", name).First(ctx)
	if err != nil {
		t.Fatalf("failed to find user, got error: %v", err)
	} else if nu.Name != name || u.ID != nu.ID {
		t.Fatalf("found invalid user, got %v, expect %v", nu.ID, u.ID)
	}

	rows, err = gorm.G[User](DB).Where("id = ?", u.ID).Updates(ctx, User{Name: "GenericsExecUpdates", Age: 18})
	if rows != 1 {
		t.Fatalf("failed to get affected rows, got %d, should be %d", rows, 1)
	}

	nu, err = gorm.G[User](DB).Where("id = ?", u.ID).Last(ctx)
	if err != nil {
		t.Fatalf("failed to find user, got error: %v", err)
	} else if nu.Name != "GenericsExecUpdates" || nu.Age != 18 || u.ID != nu.ID {
		t.Fatalf("found invalid user, got %v, expect %v", nu.ID, u.ID)
	}
}

func TestGenericsRow(t *testing.T) {
	ctx := context.Background()

	user := User{Name: "GenericsRow"}
	if err := gorm.G[User](DB).Create(ctx, &user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	rawSQLUserRow := gorm.G[User](DB).Raw("SELECT name FROM ? WHERE id = ?", clause.Table{Name: clause.CurrentTable}, user.ID).Row(ctx)
	var name string
	if err := rawSQLUserRow.Scan(&name); err != nil {
		t.Fatalf("rawSQLUserRow scan failed: %v", err)
	}
	if name != user.Name {
		t.Errorf("expected %s, got %s", user.Name, name)
	}

	var scannedUserName string
	selectUserRow := gorm.G[User](DB).Select("name").Where("name = ?", user.Name).Row(ctx)
	if err := selectUserRow.Scan(&scannedUserName); err != nil {
		t.Fatalf("selectUserRow scan failed: %v", err)
	}
	if name != user.Name {
		t.Errorf("expected %s, got %s", user.Name, scannedUserName)
	}

	user2 := User{Name: "GenericsRow2"}
	if err := gorm.G[User](DB).Create(ctx, &user2); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	rawSQLUserRows, err := gorm.G[User](DB).Raw("SELECT name FROM users WHERE id IN ?", []uint{user.ID, user2.ID}).Rows(ctx)
	if err != nil {
		t.Fatalf("rawSQLUserRows failed: %v", err)
	}

	count := 0
	for rawSQLUserRows.Next() {
		var name string
		if err := rawSQLUserRows.Scan(&name); err != nil {
			t.Fatalf("rawSQLUserRows.Scan failed: %v", err)
		}
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 rows, got %d", count)
	}

	selectNameUserRows, err := gorm.G[User](DB).Select("name").Where("id IN ?", []uint{user.ID, user2.ID}).Rows(ctx)
	if err != nil {
		t.Fatalf("selectNameUserRows failed: %v", err)
	}
	count = 0
	for selectNameUserRows.Next() {
		var name string
		if err := selectNameUserRows.Scan(&name); err != nil {
			t.Fatalf("selectNameUserRows.Scan failed: %v", err)
		}
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 rows, got %d", count)
	}

	fullUserRows, err := gorm.G[User](DB).Where("id IN ?", []uint{user.ID, user2.ID}).Rows(ctx)
	if err != nil {
		t.Fatalf("Rows failed: %v", err)
	}
	count = 0
	for fullUserRows.Next() {
		var scannedUser User
		if err := DB.ScanRows(fullUserRows, &scannedUser); err != nil {
			t.Fatalf("DB.ScanRows failed: %v", err)
		}
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 rows, got %d", count)
	}
}

func TestGenericsDelete(t *testing.T) {
	ctx := context.Background()

	u := User{Name: "GenericsDelete"}
	if err := gorm.G[User](DB).Create(ctx, &u); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	rows, err := gorm.G[User](DB).Where("id = ?", u.ID).Delete(ctx)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if rows != 1 {
		t.Errorf("expected 1 row deleted, got %d", rows)
	}

	_, err = gorm.G[User](DB).Where("id = ?", u.ID).First(ctx)
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("User after delete failed: %v", err)
	}
}

func TestGenericsFindInBatches(t *testing.T) {
	ctx := context.Background()

	users := []User{
		{Name: "GenericsFindBatchA"},
		{Name: "GenericsFindBatchB"},
		{Name: "GenericsFindBatchC"},
		{Name: "GenericsFindBatchD"},
		{Name: "GenericsFindBatchE"},
	}
	if err := gorm.G[User](DB).CreateInBatches(ctx, &users, len(users)); err != nil {
		t.Fatalf("CreateInBatches failed: %v", err)
	}

	total := 0
	err := gorm.G[User](DB).Where("name like ?", "GenericsFindBatch%").FindInBatches(ctx, 2, func(chunk []User, batch int) error {
		if len(chunk) > 2 {
			t.Errorf("batch size exceed 2: got %d", len(chunk))
		}

		total += len(chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("FindInBatches failed: %v", err)
	}

	if total != len(users) {
		t.Errorf("expected total %d, got %d", len(users), total)
	}
}

func TestGenericsScopes(t *testing.T) {
	ctx := context.Background()

	users := []User{{Name: "GenericsScopes1"}, {Name: "GenericsScopes2"}, {Name: "GenericsScopes3"}}
	err := gorm.G[User](DB).CreateInBatches(ctx, &users, len(users))
	if err != nil {
		t.Fatalf("CreateInBatches failed: %v", err)
	}

	filterName1 := func(stmt *gorm.Statement) {
		stmt.Where("name = ?", "GenericsScopes1")
	}

	results, err := gorm.G[User](DB).Scopes(filterName1).Find(ctx)
	if err != nil {
		t.Fatalf("Scopes failed: %v", err)
	}
	if len(results) != 1 || results[0].Name != "GenericsScopes1" {
		t.Fatalf("Scopes expected 1, got %d", len(results))
	}

	notResult, err := gorm.G[User](DB).Where("name like ?", "GenericsScopes%").Not("name = ?", "GenericsScopes1").Order("name").Find(ctx)
	if len(notResult) != 2 {
		t.Fatalf("expected 2 results, got %d", len(notResult))
	} else if notResult[0].Name != "GenericsScopes2" || notResult[1].Name != "GenericsScopes3" {
		t.Fatalf("expected names 'GenericsScopes2' and 'GenericsScopes3', got %s and %s", notResult[0].Name, notResult[1].Name)
	}

	orResult, err := gorm.G[User](DB).Or("name = ?", "GenericsScopes1").Or("name = ?", "GenericsScopes2").Order("name").Find(ctx)
	if len(orResult) != 2 {
		t.Fatalf("expected 2 results, got %d", len(notResult))
	} else if orResult[0].Name != "GenericsScopes1" || orResult[1].Name != "GenericsScopes2" {
		t.Fatalf("expected names 'GenericsScopes2' and 'GenericsScopes3', got %s and %s", orResult[0].Name, orResult[1].Name)
	}
}

func TestGenericsJoins(t *testing.T) {
	ctx := context.Background()
	db := gorm.G[User](DB)

	u := User{Name: "GenericsJoins", Company: Company{Name: "GenericsCompany"}}
	u2 := User{Name: "GenericsJoins_2", Company: Company{Name: "GenericsCompany_2"}}
	u3 := User{Name: "GenericsJoins_3", Company: Company{Name: "GenericsCompany_3"}}
	db.CreateInBatches(ctx, &[]User{u3, u, u2}, 10)

	// Inner JOIN + WHERE
	result, err := db.Joins(clause.Has("Company"), func(db gorm.JoinBuilder, joinTable clause.Table, curTable clause.Table) error {
		db.Where("?.name = ?", joinTable, u.Company.Name)
		return nil
	}).First(ctx)
	if err != nil {
		t.Fatalf("Joins failed: %v", err)
	}
	if result.Name != u.Name || result.Company.Name != u.Company.Name {
		t.Fatalf("Joins expected %s, got %+v", u.Name, result)
	}

	// Inner JOIN + WHERE with map
	result, err = db.Joins(clause.Has("Company"), func(db gorm.JoinBuilder, joinTable clause.Table, curTable clause.Table) error {
		db.Where(map[string]any{"name": u.Company.Name})
		return nil
	}).First(ctx)
	if err != nil {
		t.Fatalf("Joins failed: %v", err)
	}
	if result.Name != u.Name || result.Company.Name != u.Company.Name {
		t.Fatalf("Joins expected %s, got %+v", u.Name, result)
	}

	// Left JOIN w/o WHERE
	result, err = db.Joins(clause.LeftJoin.Association("Company"), nil).Where(map[string]any{"name": u.Name}).First(ctx)
	if err != nil {
		t.Fatalf("Joins failed: %v", err)
	}
	if result.Name != u.Name || result.Company.Name != u.Company.Name {
		t.Fatalf("Joins expected %s, got %+v", u.Name, result)
	}

	// Left JOIN + Alias WHERE
	result, err = db.Joins(clause.LeftJoin.Association("Company").As("t"), func(db gorm.JoinBuilder, joinTable clause.Table, curTable clause.Table) error {
		if joinTable.Name != "t" {
			t.Fatalf("Join table should be t, but got %v", joinTable.Name)
		}
		db.Where("?.name = ?", joinTable, u.Company.Name)
		return nil
	}).Where(map[string]any{"name": u.Name}).First(ctx)
	if err != nil {
		t.Fatalf("Joins failed: %v", err)
	}
	if result.Name != u.Name || result.Company.Name != u.Company.Name {
		t.Fatalf("Joins expected %s, got %+v", u.Name, result)
	}

	// Raw Subquery JOIN + WHERE
	result, err = db.Joins(clause.LeftJoin.AssociationFrom("Company", gorm.G[Company](DB)).As("t"),
		func(db gorm.JoinBuilder, joinTable clause.Table, curTable clause.Table) error {
			if joinTable.Name != "t" {
				t.Fatalf("Join table should be t, but got %v", joinTable.Name)
			}
			db.Where("?.name = ?", joinTable, u.Company.Name)
			return nil
		},
	).Where(map[string]any{"name": u2.Name}).First(ctx)
	if err != nil {
		t.Fatalf("Raw subquery join failed: %v", err)
	}
	if result.Name != u2.Name || result.Company.Name != u.Company.Name || result.Company.ID == 0 {
		t.Fatalf("Joins expected %s, got %+v", u.Name, result)
	}

	// Raw Subquery JOIN + WHERE + Select
	result, err = db.Joins(clause.LeftJoin.AssociationFrom("Company", gorm.G[Company](DB).Select("Name")).As("t"),
		func(db gorm.JoinBuilder, joinTable clause.Table, curTable clause.Table) error {
			if joinTable.Name != "t" {
				t.Fatalf("Join table should be t, but got %v", joinTable.Name)
			}
			db.Where("?.name = ?", joinTable, u.Company.Name)
			return nil
		},
	).Where(map[string]any{"name": u2.Name}).First(ctx)
	if err != nil {
		t.Fatalf("Raw subquery join failed: %v", err)
	}
	if result.Name != u2.Name || result.Company.Name != u.Company.Name || result.Company.ID != 0 {
		t.Fatalf("Joins expected %s, got %+v", u.Name, result)
	}

	_, err = db.Joins(clause.Has("Company"), func(db gorm.JoinBuilder, joinTable clause.Table, curTable clause.Table) error {
		return errors.New("join error")
	}).First(ctx)
	if err == nil {
		t.Fatalf("Joins should got error, but got nil")
	}
}

func TestGenericsNestedJoins(t *testing.T) {
	users := []User{
		{
			Name: "generics-nested-joins-1",
			Manager: &User{
				Name: "generics-nested-joins-manager-1",
				Company: Company{
					Name: "generics-nested-joins-manager-company-1",
				},
				NamedPet: &Pet{
					Name: "generics-nested-joins-manager-namepet-1",
					Toy: Toy{
						Name: "generics-nested-joins-manager-namepet-toy-1",
					},
				},
			},
			NamedPet: &Pet{Name: "generics-nested-joins-namepet-1", Toy: Toy{Name: "generics-nested-joins-namepet-toy-1"}},
		},
		{
			Name:     "generics-nested-joins-2",
			Manager:  GetUser("generics-nested-joins-manager-2", Config{Company: true, NamedPet: true}),
			NamedPet: &Pet{Name: "generics-nested-joins-namepet-2", Toy: Toy{Name: "generics-nested-joins-namepet-toy-2"}},
		},
	}

	ctx := context.Background()
	db := gorm.G[User](DB)
	db.CreateInBatches(ctx, &users, 100)

	var userIDs []uint
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}

	users2, err := db.Joins(clause.LeftJoin.Association("Manager"), nil).
		Joins(clause.LeftJoin.Association("Manager.Company"), nil).
		Joins(clause.LeftJoin.Association("Manager.NamedPet.Toy"), nil).
		Joins(clause.LeftJoin.Association("NamedPet.Toy"), nil).
		Joins(clause.LeftJoin.Association("NamedPet").As("t"), nil).
		Where(map[string]any{"id": userIDs}).Find(ctx)

	if err != nil {
		t.Fatalf("Failed to load with joins, got error: %v", err)
	} else if len(users2) != len(users) {
		t.Fatalf("Failed to load join users, got: %v, expect: %v", len(users2), len(users))
	}

	sort.Slice(users2, func(i, j int) bool {
		return users2[i].ID > users2[j].ID
	})

	sort.Slice(users, func(i, j int) bool {
		return users[i].ID > users[j].ID
	})

	for idx, user := range users {
		// user
		CheckUser(t, user, users2[idx])
		if users2[idx].Manager == nil {
			t.Fatalf("Failed to load Manager")
		}
		// manager
		CheckUser(t, *user.Manager, *users2[idx].Manager)
		// user pet
		if users2[idx].NamedPet == nil {
			t.Fatalf("Failed to load NamedPet")
		}
		CheckPet(t, *user.NamedPet, *users2[idx].NamedPet)
		// manager pet
		if users2[idx].Manager.NamedPet == nil {
			t.Fatalf("Failed to load NamedPet")
		}
		CheckPet(t, *user.Manager.NamedPet, *users2[idx].Manager.NamedPet)
	}
}

func TestGenericsPreloads(t *testing.T) {
	ctx := context.Background()
	db := gorm.G[User](DB)

	u := *GetUser("GenericsPreloads_1", Config{Company: true, Pets: 3, Friends: 7})
	u2 := *GetUser("GenericsPreloads_2", Config{Company: true, Pets: 5, Friends: 5})
	u3 := *GetUser("GenericsPreloads_3", Config{Company: true, Pets: 7, Friends: 3})
	names := []string{u.Name, u2.Name, u3.Name}

	db.CreateInBatches(ctx, &[]User{u3, u, u2}, 10)

	result, err := db.Preload("Company", nil).Preload("Pets", nil).Where("name = ?", u.Name).First(ctx)
	if err != nil {
		t.Fatalf("Preload failed: %v", err)
	}

	if result.Name != u.Name || result.Company.Name != u.Company.Name || len(result.Pets) != len(u.Pets) {
		t.Fatalf("Preload expected %s, got %+v", u.Name, result)
	}

	results, err := db.Preload("Company", func(db gorm.PreloadBuilder) error {
		db.Where("name = ?", u.Company.Name)
		return nil
	}).Where("name in ?", names).Find(ctx)
	if err != nil {
		t.Fatalf("Preload failed: %v", err)
	}
	for _, result := range results {
		if result.Name == u.Name {
			if result.Company.Name != u.Company.Name {
				t.Fatalf("Preload user %v company should be %v, but got %+v", u.Name, u.Company.Name, result.Company.Name)
			}
		} else if result.Company.Name != "" {
			t.Fatalf("Preload other company should not loaded, user %v company expect %v but got %+v", u.Name, u.Company.Name, result.Company.Name)
		}
	}

	_, err = db.Preload("Company", func(db gorm.PreloadBuilder) error {
		return errors.New("preload error")
	}).Where("name in ?", names).Find(ctx)
	if err == nil {
		t.Fatalf("Preload should failed, but got nil")
	}

	if DB.Dialector.Name() == "mysql" {
		// mysql 5.7 doesn't support row_number()
		if strings.HasPrefix(DB.Dialector.(*mysql.Dialector).ServerVersion, "5.7") {
			return
		}
	}
	results, err = db.Preload("Pets", func(db gorm.PreloadBuilder) error {
		db.LimitPerRecord(5)
		return nil
	}).Where("name in ?", names).Find(ctx)

	for _, result := range results {
		if result.Name == u.Name {
			if len(result.Pets) != len(u.Pets) {
				t.Fatalf("Preload user %v pets should be %v, but got %+v", u.Name, u.Pets, result.Pets)
			}
		} else if len(result.Pets) != 5 {
			t.Fatalf("Preload user %v pets should be 5, but got %+v", result.Name, result.Pets)
		}
	}

	if DB.Dialector.Name() == "sqlserver" {
		// sqlserver doesn't support order by in subquery
		return
	}
	results, err = db.Preload("Pets", func(db gorm.PreloadBuilder) error {
		db.Order("name desc").LimitPerRecord(5)
		return nil
	}).Where("name in ?", names).Find(ctx)

	for _, result := range results {
		if result.Name == u.Name {
			if len(result.Pets) != len(u.Pets) {
				t.Fatalf("Preload user %v pets should be %v, but got %+v", u.Name, u.Pets, result.Pets)
			}
		} else if len(result.Pets) != 5 {
			t.Fatalf("Preload user %v pets should be 5, but got %+v", result.Name, result.Pets)
		}
		for i := 1; i < len(result.Pets); i++ {
			if result.Pets[i-1].Name < result.Pets[i].Name {
				t.Fatalf("Preload user %v pets not ordered correctly, last %v, cur %v", result.Name, result.Pets[i-1], result.Pets[i])
			}
		}
	}

	results, err = db.Preload("Pets", func(db gorm.PreloadBuilder) error {
		db.Order("name").LimitPerRecord(5)
		return nil
	}).Preload("Friends", func(db gorm.PreloadBuilder) error {
		db.Order("name")
		return nil
	}).Where("name in ?", names).Find(ctx)

	for _, result := range results {
		if result.Name == u.Name {
			if len(result.Pets) != len(u.Pets) {
				t.Fatalf("Preload user %v pets should be %v, but got %+v", u.Name, u.Pets, result.Pets)
			}
			if len(result.Friends) != len(u.Friends) {
				t.Fatalf("Preload user %v pets should be %v, but got %+v", u.Name, u.Pets, result.Pets)
			}
		} else if len(result.Pets) != 5 || len(result.Friends) == 0 {
			t.Fatalf("Preload user %v pets should be 5, but got %+v", result.Name, result.Pets)
		}
		for i := 1; i < len(result.Pets); i++ {
			if result.Pets[i-1].Name > result.Pets[i].Name {
				t.Fatalf("Preload user %v pets not ordered correctly, last %v, cur %v", result.Name, result.Pets[i-1], result.Pets[i])
			}
		}
		for i := 1; i < len(result.Pets); i++ {
			if result.Pets[i-1].Name > result.Pets[i].Name {
				t.Fatalf("Preload user %v friends not ordered correctly, last %v, cur %v", result.Name, result.Pets[i-1], result.Pets[i])
			}
		}
	}
}

func TestGenericsNestedPreloads(t *testing.T) {
	user := *GetUser("generics_nested_preload", Config{Pets: 2})
	user.Friends = []*User{GetUser("generics_nested_preload", Config{Pets: 5})}

	ctx := context.Background()
	db := gorm.G[User](DB)

	for idx, pet := range user.Pets {
		pet.Toy = Toy{Name: "toy_nested_preload_" + strconv.Itoa(idx+1)}
	}

	if err := db.Create(ctx, &user); err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	user2, err := db.Preload("Pets.Toy", nil).Preload("Friends.Pets", func(db gorm.PreloadBuilder) error {
		return nil
	}).Where(user.ID).Take(ctx)
	if err != nil {
		t.Errorf("failed to nested preload user")
	}
	CheckUser(t, user2, user)
	if len(user.Pets) == 0 || len(user.Friends) == 0 || len(user.Friends[0].Pets) == 0 {
		t.Fatalf("failed to nested preload")
	}

	if DB.Dialector.Name() == "mysql" {
		// mysql 5.7 doesn't support row_number()
		if strings.HasPrefix(DB.Dialector.(*mysql.Dialector).ServerVersion, "5.7") {
			return
		}
	}
	if DB.Dialector.Name() == "sqlserver" {
		// sqlserver doesn't support order by in subquery
		return
	}

	user3, err := db.Preload("Pets.Toy", nil).Preload("Friends.Pets", func(db gorm.PreloadBuilder) error {
		db.LimitPerRecord(3)
		return nil
	}).Where(user.ID).Take(ctx)
	if err != nil {
		t.Errorf("failed to nested preload user")
	}
	CheckUser(t, user3, user)

	if len(user3.Friends) != 1 || len(user3.Friends[0].Pets) != 3 {
		t.Errorf("failed to nested preload with limit per record")
	}
}

func TestGenericsDistinct(t *testing.T) {
	ctx := context.Background()

	batch := []User{
		{Name: "GenericsDistinctDup"},
		{Name: "GenericsDistinctDup"},
		{Name: "GenericsDistinctUnique"},
	}
	if err := gorm.G[User](DB).CreateInBatches(ctx, &batch, len(batch)); err != nil {
		t.Fatalf("CreateInBatches failed: %v", err)
	}

	results, err := gorm.G[User](DB).Where("name like ?", "GenericsDistinct%").Distinct("name").Find(ctx)
	if err != nil {
		t.Fatalf("Distinct Find failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 distinct names, got %d", len(results))
	}

	var names []string
	for _, u := range results {
		names = append(names, u.Name)
	}
	sort.Strings(names)
	expected := []string{"GenericsDistinctDup", "GenericsDistinctUnique"}
	if !reflect.DeepEqual(names, expected) {
		t.Errorf("expected names %v, got %v", expected, names)
	}
}

func TestGenericsSetCreate(t *testing.T) {
	ctx := context.Background()

	name := "GenericsSetCreate"
	age := uint(21)

	err := gorm.G[User](DB).Set(
		clause.Assignment{Column: clause.Column{Name: "name"}, Value: name},
		clause.Assignment{Column: clause.Column{Name: "age"}, Value: age},
	).Create(ctx)
	if err != nil {
		t.Fatalf("Set Create failed: %v", err)
	}

	u, err := gorm.G[User](DB).Where("name = ?", name).First(ctx)
	if err != nil {
		t.Fatalf("failed to find created user: %v", err)
	}
	if u.ID == 0 || u.Name != name || u.Age != age {
		t.Fatalf("created user mismatch, got %+v", u)
	}
}

func TestGenericsSetUpdate(t *testing.T) {
	ctx := context.Background()

	// prepare
	u := User{Name: "GenericsSetUpdate_Before", Age: 30}
	if err := gorm.G[User](DB).Create(ctx, &u); err != nil {
		t.Fatalf("prepare user failed: %v", err)
	}

	// update with Set after chain
	newName := "GenericsSetUpdate_After"
	newAge := uint(31)
	rows, err := gorm.G[User](DB).
		Where("id = ?", u.ID).
		Set(
			clause.Assignment{Column: clause.Column{Name: "name"}, Value: newName},
			clause.Assignment{Column: clause.Column{Name: "age"}, Value: newAge},
		).
		Update(ctx)
	if err != nil {
		t.Fatalf("Set Update failed: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 row affected, got %d", rows)
	}

	nu, err := gorm.G[User](DB).Where("id = ?", u.ID).First(ctx)
	if err != nil {
		t.Fatalf("failed to query updated user: %v", err)
	}
	if nu.Name != newName || nu.Age != newAge {
		t.Fatalf("updated user mismatch, got %+v", nu)
	}
}

func TestGenericsGroupHaving(t *testing.T) {
	ctx := context.Background()

	batch := []User{
		{Name: "GenericsGroupHavingMulti"},
		{Name: "GenericsGroupHavingMulti"},
		{Name: "GenericsGroupHavingSingle"},
	}
	if err := gorm.G[User](DB).CreateInBatches(ctx, &batch, len(batch)); err != nil {
		t.Fatalf("CreateInBatches failed: %v", err)
	}

	grouped, err := gorm.G[User](DB).Select("name").Where("name like ?", "GenericsGroupHaving%").Group("name").Having("COUNT(id) > ?", 1).Find(ctx)
	if err != nil {
		t.Fatalf("Group+Having Find failed: %v", err)
	}

	if len(grouped) != 1 {
		t.Errorf("expected 1 group with count>1, got %d", len(grouped))
	} else if grouped[0].Name != "GenericsGroupHavingMulti" {
		t.Errorf("expected group name 'GenericsGroupHavingMulti', got '%s'", grouped[0].Name)
	}
}

func TestGenericsSubQuery(t *testing.T) {
	ctx := context.Background()
	users := []User{
		{Name: "GenericsSubquery_1", Age: 10},
		{Name: "GenericsSubquery_2", Age: 20},
		{Name: "GenericsSubquery_3", Age: 30},
		{Name: "GenericsSubquery_4", Age: 40},
	}

	if err := gorm.G[User](DB).CreateInBatches(ctx, &users, len(users)); err != nil {
		t.Fatalf("CreateInBatches failed: %v", err)
	}

	results, err := gorm.G[User](DB).Where("name IN (?)", gorm.G[User](DB).Select("name").Where("name LIKE ?", "GenericsSubquery%")).Find(ctx)
	if err != nil {
		t.Fatalf("got error: %v", err)
	}

	if len(results) != 4 {
		t.Errorf("Four users should be found, instead found %d", len(results))
	}

	results, err = gorm.G[User](DB).Where("name IN (?)", gorm.G[User](DB).Select("name").Where("name IN ?", []string{"GenericsSubquery_1", "GenericsSubquery_2"}).Or("name = ?", "GenericsSubquery_3")).Find(ctx)
	if err != nil {
		t.Fatalf("got error: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Three users should be found, instead found %d", len(results))
	}
}

func TestGenericsUpsert(t *testing.T) {
	ctx := context.Background()
	lang := Language{Code: "upsert", Name: "Upsert"}

	if err := gorm.G[Language](DB, clause.OnConflict{DoNothing: true}).Create(ctx, &lang); err != nil {
		t.Fatalf("failed to upsert, got %v", err)
	}

	lang2 := Language{Code: "upsert", Name: "Upsert"}
	if err := gorm.G[Language](DB, clause.OnConflict{DoNothing: true}).Create(ctx, &lang2); err != nil {
		t.Fatalf("failed to upsert, got %v", err)
	}

	langs, err := gorm.G[Language](DB).Where("code = ?", lang.Code).Find(ctx)
	if err != nil {
		t.Errorf("no error should happen when find languages with code, but got %v", err)
	} else if len(langs) != 1 {
		t.Errorf("should only find only 1 languages, but got %+v", langs)
	}

	lang3 := Language{Code: "upsert", Name: "Upsert"}
	if err := gorm.G[Language](DB, clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"name": "upsert-new"}),
	}).Create(ctx, &lang3); err != nil {
		t.Fatalf("failed to upsert, got %v", err)
	}

	if langs, err := gorm.G[Language](DB).Where("code = ?", lang.Code).Find(ctx); err != nil {
		t.Errorf("no error should happen when find languages with code, but got %v", err)
	} else if len(langs) != 1 {
		t.Errorf("should only find only 1 languages, but got %+v", langs)
	} else if langs[0].Name != "upsert-new" {
		t.Errorf("should update name on conflict, but got name %+v", langs[0].Name)
	}
}

func TestGenericsWithResult(t *testing.T) {
	ctx := context.Background()
	users := []User{{Name: "TestGenericsWithResult", Age: 18}, {Name: "TestGenericsWithResult2", Age: 18}}

	result := gorm.WithResult()
	err := gorm.G[User](DB, result).CreateInBatches(ctx, &users, 2)
	if err != nil {
		t.Errorf("failed to create users WithResult")
	}

	if result.RowsAffected != 2 {
		t.Errorf("failed to get affected rows, got %d, should be %d", result.RowsAffected, 2)
	}
}

func TestGenericsReuse(t *testing.T) {
	ctx := context.Background()
	users := []User{{Name: "TestGenericsReuse1", Age: 18}, {Name: "TestGenericsReuse2", Age: 18}}

	err := gorm.G[User](DB).CreateInBatches(ctx, &users, 2)
	if err != nil {
		t.Errorf("failed to create users")
	}

	reusedb := gorm.G[User](DB).Where("name like ?", "TestGenericsReuse%")

	sg := sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		sg.Add(1)

		go func() {
			if u1, err := reusedb.Where("id = ?", users[0].ID).First(ctx); err != nil {
				t.Errorf("failed to find user, got error: %v", err)
			} else if u1.Name != users[0].Name || u1.ID != users[0].ID {
				t.Errorf("found invalid user, got %v, expect %v", u1, users[0])
			}

			if u2, err := reusedb.Where("id = ?", users[1].ID).First(ctx); err != nil {
				t.Errorf("failed to find user, got error: %v", err)
			} else if u2.Name != users[1].Name || u2.ID != users[1].ID {
				t.Errorf("found invalid user, got %v, expect %v", u2, users[1])
			}

			if users, err := reusedb.Where("id IN ?", []uint{users[0].ID, users[1].ID}).Find(ctx); err != nil {
				t.Errorf("failed to find user, got error: %v", err)
			} else if len(users) != 2 {
				t.Errorf("should find 2 users, but got %d", len(users))
			}
			sg.Done()
		}()
	}
	sg.Wait()
}

func TestGenericsWithTransaction(t *testing.T) {
	ctx := context.Background()
	tx := DB.Begin()
	if tx.Error != nil {
		t.Fatalf("failed to begin transaction: %v", tx.Error)
	}

	users := []User{{Name: "TestGenericsTransaction", Age: 18}, {Name: "TestGenericsTransaction2", Age: 18}}
	err := gorm.G[User](tx).CreateInBatches(ctx, &users, 2)

	count, err := gorm.G[User](tx).Where("name like ?", "TestGenericsTransaction%").Count(ctx, "*")
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 records, got %d", count)
	}

	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("failed to rollback transaction: %v", err)
	}

	count2, err := gorm.G[User](DB).Where("name like ?", "TestGenericsTransaction%").Count(ctx, "*")
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count2 != 0 {
		t.Errorf("expected 0 records after rollback, got %d", count2)
	}
}

func TestGenericsToSQL(t *testing.T) {
	ctx := context.Background()
	sql := DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		gorm.G[User](tx).Limit(10).Find(ctx)
		return tx
	})

	if !regexp.MustCompile("SELECT \\* FROM .users..* 10").MatchString(sql) {
		t.Errorf("ToSQL: got wrong sql with Generics API %v", sql)
	}
}

func TestGenericsScanUUID(t *testing.T) {
	ctx := context.Background()
	users := []User{
		{Name: uuid.NewString(), Age: 21},
		{Name: uuid.NewString(), Age: 22},
		{Name: uuid.NewString(), Age: 23},
	}

	if err := gorm.G[User](DB).CreateInBatches(ctx, &users, 2); err != nil {
		t.Fatalf("CreateInBatches failed: %v", err)
	}

	userIds := []uuid.UUID{}
	if err := gorm.G[User](DB).Select("name").Where("id in ?", []uint{users[0].ID, users[1].ID, users[2].ID}).Order("age").Scan(ctx, &userIds); err != nil || len(users) != 3 {
		t.Fatalf("Scan failed: %v, userids %v", err, userIds)
	}

	if userIds[0].String() != users[0].Name || userIds[1].String() != users[1].Name || userIds[2].String() != users[2].Name {
		t.Fatalf("wrong uuid scanned")
	}
}

func TestGenericsCount(t *testing.T) {
	ctx := context.Background()

	// Just test that the API can be called
	_, err := gorm.G[User](DB).Count(ctx, "*")
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
}

func TestGenericsUpdate(t *testing.T) {
	ctx := context.Background()

	// Just test that the API can be called
	_, err := gorm.G[User](DB).Where("id = ?", 1).Update(ctx, "name", "test")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
}

func TestGenericsUpdates(t *testing.T) {
	ctx := context.Background()

	// Just test that the API can be called
	_, err := gorm.G[User](DB).Where("id = ?", 1).Updates(ctx, User{Name: "test"})
	if err != nil {
		t.Fatalf("Updates failed: %v", err)
	}
}

func TestGenericsDeleteAPI(t *testing.T) {
	ctx := context.Background()

	// Just test that the API can be called
	_, err := gorm.G[User](DB).Where("id = ?", 1).Delete(ctx)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestGenericsAssociation(t *testing.T) {
	// Test basic Association creation
	assoc := clause.Association{
		Association: "Orders",
		Type:        clause.OpCreate,
		Set: []clause.Assignment{
			{Column: clause.Column{Name: "amount"}, Value: 100},
			{Column: clause.Column{Name: "state"}, Value: "new"},
		},
	}

	// Verify it implements Assigner interface
	assignments := assoc.Assignments()
	if len(assignments) != 0 {
		t.Errorf("Association.Assignments() should return empty slice, got %v", assignments)
	}

	// Verify it implements AssociationAssigner interface
	assocAssignments := assoc.AssociationAssignments()
	if len(assocAssignments) != 1 {
		t.Errorf("Association.AssociationAssignments() should return slice with one element, got %v", assocAssignments)
	}

	if assocAssignments[0].Association != "Orders" {
		t.Errorf("Association.AssociationAssignments()[0].Association should be 'Orders', got %v", assocAssignments[0].Association)
	}

	// Test different association operation types
	operations := []struct {
		Type     clause.AssociationOpType
		TypeName string
	}{
		{clause.OpUnlink, "OpUnlink"},
		{clause.OpDelete, "OpDelete"},
		{clause.OpUpdate, "OpUpdate"},
		{clause.OpCreate, "OpCreate"},
	}

	for _, op := range operations {
		assoc := clause.Association{
			Association: "Orders",
			Type:        op.Type,
		}

		if assoc.Type != op.Type {
			t.Errorf("Association type should be %s, got %v", op.TypeName, assoc.Type)
		}
	}
}

func TestGenericsAssociationSlice(t *testing.T) {
	// Test that a slice of Association can be used
	associations := []clause.Association{
		{Association: "Orders", Type: clause.OpDelete},
		{Association: "Profiles", Type: clause.OpUpdate},
	}

	// In practice, each Association would be processed individually
	// since []clause.Association doesn't implement AssociationAssigner directly
	for i, assoc := range associations {
		assigns := assoc.AssociationAssignments()
		if len(assigns) != 1 {
			t.Errorf("Association %d should return one assignment, got %v", i, len(assigns))
		}
		if assigns[0].Association != assoc.Association {
			t.Errorf("Association %d name should be %s, got %v", i, assoc.Association, assigns[0].Association)
		}
	}
}
