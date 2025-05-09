package tests_test

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"testing"

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
	} else if u.Name != "" || u.Age != u.Age {
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
		fmt.Println(found)
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

	row := gorm.G[User](DB).Raw("SELECT name FROM users WHERE id = ?", user.ID).Row(ctx)
	var name string
	if err := row.Scan(&name); err != nil {
		t.Fatalf("Row scan failed: %v", err)
	}
	if name != user.Name {
		t.Errorf("expected %s, got %s", user.Name, name)
	}

	user2 := User{Name: "GenericsRow2"}
	if err := gorm.G[User](DB).Create(ctx, &user2); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	rows, err := gorm.G[User](DB).Raw("SELECT name FROM users WHERE id IN ?", []uint{user.ID, user2.ID}).Rows(ctx)
	if err != nil {
		t.Fatalf("Rows failed: %v", err)
	}

	count := 0
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("rows.Scan failed: %v", err)
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

func TestGenericsJoinsAndPreload(t *testing.T) {
	ctx := context.Background()
	db := gorm.G[User](DB)

	u := User{Name: "GenericsJoins", Company: Company{Name: "GenericsCompany"}}
	db.Create(ctx, &u)

	// LEFT JOIN + WHERE
	result, err := db.Joins(clause.Has("Company"), func(db gorm.ChainInterface[any], joinTable clause.Table, curTable clause.Table) gorm.ChainInterface[any] {
		return db.Where("?.name = ?", joinTable, u.Company.Name)
	}).First(ctx)
	if err != nil {
		t.Fatalf("Joins failed: %v", err)
	}
	if result.Name != u.Name || result.Company.Name != u.Company.Name {
		t.Fatalf("Joins expected %s, got %+v", u.Name, result)
	}

	// JOIN
	result, err = db.Joins(clause.Has("Company"), func(db gorm.ChainInterface[any], joinTable clause.Table, curTable clause.Table) gorm.ChainInterface[any] {
		return nil
	}).First(ctx)
	if err != nil {
		t.Fatalf("Joins failed: %v", err)
	}
	if result.Name != u.Name || result.Company.Name != u.Company.Name {
		t.Fatalf("Joins expected %s, got %+v", u.Name, result)
	}

	// Left JOIN
	result, err = db.Joins(clause.LeftJoin.Association("Company").As("t"), func(db gorm.ChainInterface[any], joinTable clause.Table, curTable clause.Table) gorm.ChainInterface[any] {
		return nil
	}).First(ctx)
	if err != nil {
		t.Fatalf("Joins failed: %v", err)
	}
	if result.Name != u.Name || result.Company.Name != u.Company.Name {
		t.Fatalf("Joins expected %s, got %+v", u.Name, result)
	}

	// Preload
	result3, err := db.Preload("Company").Where("name = ?", u.Name).First(ctx)
	if err != nil {
		t.Fatalf("Joins failed: %v", err)
	}
	if result3.Name != u.Name || result3.Company.Name != u.Company.Name {
		t.Fatalf("Joins expected %s, got %+v", u.Name, result)
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
