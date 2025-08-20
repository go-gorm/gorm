package cli

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type FieldInfo struct {
	Name string
	Type string
}

// GenerateModelEntity membuat GORM-ready model, entity, dan file migrasi
func GenerateModelEntity(modelName string, fields []FieldInfo, baseFolder string) error {
	if modelName == "" || len(fields) == 0 {
		return fmt.Errorf("modelName and fields must be provided")
	}

	modelsFolder := fmt.Sprintf("%s/internal/models", baseFolder)
	entityFolder := fmt.Sprintf("%s/internal/entity", baseFolder)
	migrationsFolder := fmt.Sprintf("%s/internal/migrations", baseFolder)

	if err := os.MkdirAll(modelsFolder, os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(entityFolder, os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(migrationsFolder, os.ModePerm); err != nil {
		return err
	}

	// Tulis model
	modelFile := fmt.Sprintf("%s/%s.go", modelsFolder, strings.ToLower(modelName))
	if err := writeFile(modelFile, modelName, fields, true); err != nil {
		return err
	}

	// Tulis entity
	entityFile := fmt.Sprintf("%s/%s.go", entityFolder, strings.ToLower(modelName))
	if err := writeFile(entityFile, modelName, fields, false); err != nil {
		return err
	}

	moduleName, err := getModuleName(baseFolder)
	if err != nil {
		return err
	}
	if err := createMigrationFile(modelName, migrationsFolder, moduleName); err != nil {
		return err
	}

	if err:= updateMasterMigration(migrationsFolder, modelName);err !=nil{
		return err
	}

	return nil
}

func writeFile(filename, modelName string, fields []FieldInfo, isModel bool) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	var lines []string
	for _, field := range fields {
		lines = append(lines, fmt.Sprintf("\t%s %s `gorm:\"column:%s\"`", capitalize(field.Name), mapType(field.Type), field.Name))
	}

	var content string
	if isModel {
		content = fmt.Sprintf(`package models

import (
	"time"
	"gorm.io/gorm"
)

type %s struct {
	ID        uint           `+"`gorm:\"primaryKey\"`"+`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `+"`gorm:\"index\"`"+`
%s
}
`, modelName, joinLines(lines))
	} else {
		content = fmt.Sprintf(`package entity

type %s struct {
	ID uint
%s
}
`, modelName, joinLines(lines))
	}

	_, err = f.WriteString(content)
	if err != nil {
		return err
	}

	fmt.Println("Create file:", filename)
	return nil
}

func createMigrationFile(modelName, migrationsFolder, moduleName string) error {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s/%s_create_%s.go", migrationsFolder, timestamp, strings.ToLower(modelName))

	content := fmt.Sprintf(`package migrations

import (
	"%s/configs"
	"%s/internal/models"
)

// Up migrates table %s
func Up%s() {
	configs.DB.AutoMigrate(&models.%s{})
}

// Down rolls back table %s
func Down%s() {
	configs.DB.Migrator().DropTable(&models.%s{})
}
`, moduleName, moduleName, modelName, modelName, modelName, modelName, modelName, modelName)

	return os.WriteFile(filename, []byte(content), 0644)
}

func updateMasterMigration(migrationsFolder, modelName string) error {
	masterFile := fmt.Sprintf("%s/migrate.go", migrationsFolder)

	// Jika belum ada, buat file baru
	if _, err := os.Stat(masterFile); os.IsNotExist(err) {
		content := `package migrations

import "fmt"

func MigrateAll() {
	fmt.Println("Running migrations...")
	Up` + modelName + `()
	// Add other migrations here
	fmt.Println("Migrations completed!")
}

func RollbackAll() {
	fmt.Println("Rolling back migrations...")
	Down` + modelName + `()
	// Add other rollbacks here
	fmt.Println("Rollback completed!")
}
`
		return os.WriteFile(masterFile, []byte(content), 0644)
	}

	// Jika sudah ada, append import Up/Down baru jika belum ada
	data, err := os.ReadFile(masterFile)
	if err != nil {
		return err
	}

	text := string(data)
	if !strings.Contains(text, "Up"+modelName+"()") {
		text = strings.Replace(text, "// Add other migrations here", "Up"+modelName+"()\n\t// Add other migrations here", 1)
	}
	if !strings.Contains(text, "Down"+modelName+"()") {
		text = strings.Replace(text, "// Add other rollbacks here", "Down"+modelName+"()\n\t// Add other rollbacks here", 1)
	}

	return os.WriteFile(masterFile, []byte(text), 0644)
}

func mapType(t string) string {
	switch t {
	case "string":
		return "string"
	case "int":
		return "int"
	case "float":
		return "float64"
	case "bool":
		return "bool"
	default:
		return "string"
	}
}

func capitalize(s string) string {
	if len(s) == 0 {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return "\n" + strings.Join(lines, "\n")
}
