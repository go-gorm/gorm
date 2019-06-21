package gorm

import "testing"

func TestRandName(t *testing.T) {
	names := map[string]bool{}
	max := 1000
	for i := 0; i < max; i++ {
		name := randName()
		if l := len(name); l >= 16 {
			t.Fatal(l, name)
		}
		names[name] = true
	}
	if len(names) != max {
		t.Fatal()
	}
}

func TestSqlSavepoint(t *testing.T) {
	tcs := []struct {
		dialect string
		sql     string
	}{
		{"mssql", "SAVE TRAN nced368066575bc;"},
		{"sqlite3", "SAVEPOINT nced368066575bc;"},
		{"mysql", "SAVEPOINT nced368066575bc;"},
		{"postgres", "SAVEPOINT nced368066575bc;"},
	}
	savepoint := "nced368066575bc"
	for _, tc := range tcs {
		sql := sqlSavepoint(tc.dialect, savepoint)
		if sql != tc.sql {
			t.Fatal(sql, tc.sql)
		}
	}
}

func TestSqlRollback(t *testing.T) {
	tcs := []struct {
		dialect string
		sql     string
	}{
		{"mssql", "ROLLBACK TRAN nced368066575bc;"},
		{"sqlite3", "ROLLBACK TO SAVEPOINT nced368066575bc;"},
		{"mysql", "ROLLBACK TO SAVEPOINT nced368066575bc;"},
		{"postgres", "ROLLBACK TO SAVEPOINT nced368066575bc;"},
	}
	savepoint := "nced368066575bc"
	for _, tc := range tcs {
		sql := sqlRollback(tc.dialect, savepoint)
		if sql != tc.sql {
			t.Fatal(sql, tc.sql)
		}
	}
}
