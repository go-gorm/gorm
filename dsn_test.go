package gorm

import "testing"

func TestDsn(t *testing.T) {
	dsn := DSN{
		Host: "127.0.0.1",
		Port: 3306,
		User: "root",
		Pass: "password",
		DB:   "gorm",
	}

	t.Run("dsn string", func(t *testing.T) {
		if dsn.String() != "root:password@tcp(127.0.0.1:3306)/gorm" {
			t.Error("dsn string error")
		}
	})

	t.Run("dsn string with options", func(t *testing.T) {
		dsn.Options = map[string]string{
			"charset":   "utf8mb4",
			"parseTime": "True",
		}

		if dsn.String() != "root:password@tcp(127.0.0.1:3306)/gorm?charset=utf8mb4&parseTime=True" {
			t.Error("dsn string with options error")
		}
	})
}
