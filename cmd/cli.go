package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"gorm.io/gorm/cli"
)

func main() {
	// --- Flags ---
	modelName := flag.String("name", "", "Model name, e.g.: User")
	attributes := flag.String("attributes", "", "Model attributes, e.g.: name:string,email:string")
	baseFolder := flag.String("folder", ".", "Base folder of the project")
	relations := flag.String("relations", "", "Relations, e.g.: Products:Product:one2many,Tags:Tag:many2many")
	initDB := flag.Bool("init", false, "Generate configs/db.go for supported databases")
	dbType := flag.String("db", "postgres", "Database type: postgres, mysql, sqlite, sqlserver")

	flag.Parse()

	if *initDB {
		if err := cli.GenerateDBConfig(*baseFolder, *dbType); err != nil {
			log.Fatal("Failed to create db.go:", err)
		}
		fmt.Println("configs/db.go created successfully for", *dbType)
		return
	}

	if *modelName == "" || *attributes == "" {
		fmt.Println("Use : go run main.go --name User --attributes name:string,email:string")
		return
	}

	fields := parseFields(*attributes)

	if err := cli.GenerateModelEntity(*modelName, fields, *baseFolder); err != nil {
		log.Fatal(err)
	}

	if *relations != "" {
		rels := parseRelations(*relations)
		modelFile := fmt.Sprintf("%s/internal/models/%s.go", *baseFolder, strings.ToLower(*modelName))
		if err := cli.AddRelation(modelFile, rels, *modelName); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("Done!")
}

// --- Helpers ---
func parseFields(attr string) []cli.FieldInfo {
	var fields []cli.FieldInfo
	for _, a := range strings.Split(attr, ",") {
		parts := strings.Split(a, ":")
		if len(parts) != 2 {
			log.Fatalf("Attribute format is invalid: %s", a)
		}
		fields = append(fields, cli.FieldInfo{Name: parts[0], Type: parts[1]})
	}
	return fields
}

func parseRelations(rel string) []cli.RelationInfo {
	var rels []cli.RelationInfo
	for _, r := range strings.Split(rel, ",") {
		parts := strings.Split(r, ":")
		if len(parts) != 3 {
			log.Fatalf("Relation format is invalid: %s", r)
		}
		rt := cli.RelationType(parts[2])
		rels = append(rels, cli.RelationInfo{FieldName: parts[0], Target: parts[1], Type: rt})
	}
	return rels
}
