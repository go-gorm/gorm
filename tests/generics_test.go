package tests_test

import (
	"context"
	"testing"

	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

func TestGenericsCreate(t *testing.T) {
	generic := gorm.G[User](DB)
	ctx := context.Background()

	user := User{Name: "TestGenericsCreate"}
	err := generic.Create(ctx, &user)
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

	result := struct {
		ID   int
		Name string
	}{}
	if err := gorm.G[User](DB).Where("name = ?", user.Name).Scan(ctx, &result); err != nil {
		t.Fatalf("failed to scan user, got error: %v", err)
	} else if result.Name != user.Name || uint(result.ID) != user.ID {
		t.Errorf("found invalid user, got %v, expect %v", result, user)
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
}

func TestGenericsExecAndUpdate(t *testing.T) {
	ctx := context.Background()

	name := "GenericsExec"
	if err := gorm.G[User](DB).Exec(ctx, "INSERT INTO users(name) VALUES(?)", name); err != nil {
		t.Fatalf("Exec insert failed: %v", err)
	}

	u, err := gorm.G[User](DB).Where("name = ?", name).First(ctx)
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
