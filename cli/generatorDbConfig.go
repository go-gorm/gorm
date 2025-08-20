package cli

import (
	"fmt"
	"os"
	"strings"
)

// GenerateDBConfig membuat file configs/db.go untuk berbagai DB
func GenerateDBConfig(baseFolder, dbType string) error {
	configsFolder := fmt.Sprintf("%s/configs", baseFolder)
	if err := os.MkdirAll(configsFolder, os.ModePerm); err != nil {
		return err
	}

	dbFile := fmt.Sprintf("%s/db.go", configsFolder)

	dbType = strings.ToLower(dbType)
	var importLine, openLine string

	switch dbType {
	case "postgres":
		importLine = `"gorm.io/driver/postgres"`
		openLine = `gorm.Open(postgres.Open(dsn), &gorm.Config{})`
	case "mysql":
		importLine = `"gorm.io/driver/mysql"`
		openLine = `gorm.Open(mysql.Open(dsn), &gorm.Config{})`
	case "sqlite":
		importLine = `"gorm.io/driver/sqlite"`
		openLine = `gorm.Open(sqlite.Open(dsn), &gorm.Config{})`
	case "sqlserver":
		importLine = `"gorm.io/driver/sqlserver"`
		openLine = `gorm.Open(sqlserver.Open(dsn), &gorm.Config{})`
	default:
		return fmt.Errorf("Unsupported DB type: %s", dbType)
	}

	content := fmt.Sprintf(`package configs

import (
	"fmt"
	"log"
	"gorm.io/gorm"
	%s
)

var DB *gorm.DB

func InitDB(dsn string) {
	var err error
	DB, err = %s
	if err != nil {
		log.Fatalf("Failed to connect to database: %%v", err)
	}
	fmt.Println("Database connected via GORM")
}
`, importLine, openLine)

	return os.WriteFile(dbFile, []byte(content), 0644)
}
