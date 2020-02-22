package migrator

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
	"github.com/jinzhu/gorm/schema"
)

// Migrator migrator struct
type Migrator struct {
	*Config
}

// Config schema config
type Config struct {
	CheckExistsBeforeDropping bool
	DB                        *gorm.DB
}

func (migrator Migrator) RunWithValue(value interface{}, fc func(*gorm.Statement) error) error {
	stmt := migrator.DB.Statement
	if stmt == nil {
		stmt = &gorm.Statement{DB: migrator.DB}
	}

	if err := stmt.Parse(value); err != nil {
		return err
	}

	return fc(stmt)
}

// AutoMigrate
func (migrator Migrator) AutoMigrate(values ...interface{}) error {
	// TODO smart migrate data type

	for _, value := range values {
		if !migrator.DB.Migrator().HasTable(value) {
			if err := migrator.DB.Migrator().CreateTable(value); err != nil {
				return err
			}
		} else {
			if err := migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
				for _, field := range stmt.Schema.FieldsByDBName {
					if !migrator.DB.Migrator().HasColumn(value, field.DBName) {
						if err := migrator.DB.Migrator().AddColumn(value, field.DBName); err != nil {
							return err
						}
					}
				}

				for _, rel := range stmt.Schema.Relationships.Relations {
					if constraint := rel.ParseConstraint(); constraint != nil {
						if !migrator.DB.Migrator().HasConstraint(value, constraint.Name) {
							if err := migrator.DB.Migrator().CreateConstraint(value, constraint.Name); err != nil {
								return err
							}
						}
					}

					for _, chk := range stmt.Schema.ParseCheckConstraints() {
						if !migrator.DB.Migrator().HasConstraint(value, chk.Name) {
							if err := migrator.DB.Migrator().CreateConstraint(value, chk.Name); err != nil {
								return err
							}
						}
					}

					// create join table
					joinValue := reflect.New(rel.JoinTable.ModelType).Interface()
					if !migrator.DB.Migrator().HasTable(joinValue) {
						defer migrator.DB.Migrator().CreateTable(joinValue)
					}
				}
				return nil
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func (migrator Migrator) CreateTable(values ...interface{}) error {
	for _, value := range values {
		if err := migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
			var (
				createTableSQL          = "CREATE TABLE ? ("
				values                  = []interface{}{clause.Table{Name: stmt.Table}}
				hasPrimaryKeyInDataType bool
			)

			for _, dbName := range stmt.Schema.DBNames {
				field := stmt.Schema.FieldsByDBName[dbName]
				createTableSQL += fmt.Sprintf("? ?")
				hasPrimaryKeyInDataType = hasPrimaryKeyInDataType || strings.Contains(strings.ToUpper(field.DBDataType), "PRIMARY KEY")
				values = append(values, clause.Column{Name: dbName}, clause.Expr{SQL: field.DBDataType})

				if field.AutoIncrement {
					createTableSQL += " AUTO_INCREMENT"
				}

				if field.NotNull {
					createTableSQL += " NOT NULL"
				}

				if field.Unique {
					createTableSQL += " UNIQUE"
				}

				if field.DefaultValue != "" {
					createTableSQL += " DEFAULT ?"
					values = append(values, clause.Expr{SQL: field.DefaultValue})
				}
				createTableSQL += ","
			}

			if !hasPrimaryKeyInDataType {
				createTableSQL += "PRIMARY KEY ?,"
				primaryKeys := []interface{}{}
				for _, field := range stmt.Schema.PrimaryFields {
					primaryKeys = append(primaryKeys, clause.Column{Name: field.DBName})
				}

				values = append(values, primaryKeys)
			}

			for _, idx := range stmt.Schema.ParseIndexes() {
				createTableSQL += "INDEX ? ?,"
				values = append(values, clause.Expr{SQL: idx.Name}, buildIndexOptions(idx.Fields, stmt))
			}

			for _, rel := range stmt.Schema.Relationships.Relations {
				if constraint := rel.ParseConstraint(); constraint != nil {
					sql, vars := buildConstraint(constraint)
					createTableSQL += sql + ","
					values = append(values, vars...)
				}

				// create join table
				joinValue := reflect.New(rel.JoinTable.ModelType).Interface()
				if !migrator.DB.Migrator().HasTable(joinValue) {
					defer migrator.DB.Migrator().CreateTable(joinValue)
				}
			}

			for _, chk := range stmt.Schema.ParseCheckConstraints() {
				createTableSQL += "CONSTRAINT ? CHECK ?,"
				values = append(values, clause.Column{Name: chk.Name}, clause.Expr{SQL: chk.Constraint})
			}

			createTableSQL = strings.TrimSuffix(createTableSQL, ",")

			createTableSQL += ")"
			return migrator.DB.Exec(createTableSQL, values...).Error
		}); err != nil {
			return err
		}
	}
	return nil
}

func (migrator Migrator) DropTable(values ...interface{}) error {
	for _, value := range values {
		if err := migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
			return migrator.DB.Exec("DROP TABLE ?", clause.Table{Name: stmt.Table}).Error
		}); err != nil {
			return err
		}
	}
	return nil
}

func (migrator Migrator) HasTable(values ...interface{}) bool {
	var count int64
	for _, value := range values {
		err := migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
			currentDatabase := migrator.DB.Migrator().CurrentDatabase()
			return migrator.DB.Raw("SELECT count(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = ? AND table_type = ?", currentDatabase, stmt.Table, "BASE TABLE").Scan(&count).Error
		})

		if err != nil || count == 0 {
			return false
		}
	}

	return true
}

func (migrator Migrator) RenameTable(oldName, newName string) error {
	return migrator.DB.Exec("RENAME TABLE ? TO ?", oldName, newName).Error
}

func (migrator Migrator) AddColumn(value interface{}, field string) error {
	return migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		if field := stmt.Schema.LookUpField(field); field != nil {
			return migrator.DB.Exec(
				"ALTER TABLE ? ADD ? ?",
				clause.Table{Name: stmt.Table}, clause.Column{Name: field.DBName}, clause.Expr{SQL: field.DBDataType},
			).Error
		}
		return fmt.Errorf("failed to look up field with name: %s", field)
	})
}

func (migrator Migrator) DropColumn(value interface{}, field string) error {
	return migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		if field := stmt.Schema.LookUpField(field); field != nil {
			return migrator.DB.Exec(
				"ALTER TABLE ? DROP COLUMN ?", clause.Table{Name: stmt.Table}, clause.Column{Name: field.DBName},
			).Error
		}
		return fmt.Errorf("failed to look up field with name: %s", field)
	})
}

func (migrator Migrator) AlterColumn(value interface{}, field string) error {
	return migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		if field := stmt.Schema.LookUpField(field); field != nil {
			return migrator.DB.Exec(
				"ALTER TABLE ? ALTER COLUMN ? TYPE ?",
				clause.Table{Name: stmt.Table}, clause.Column{Name: field.DBName}, clause.Expr{SQL: field.DBDataType},
			).Error
		}
		return fmt.Errorf("failed to look up field with name: %s", field)
	})
}

func (migrator Migrator) HasColumn(value interface{}, field string) bool {
	var count int64
	migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		currentDatabase := migrator.DB.Migrator().CurrentDatabase()
		name := field
		if field := stmt.Schema.LookUpField(field); field != nil {
			name = field.DBName
		}

		return migrator.DB.Raw(
			"SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_schema = ? AND table_name = ? AND column_name = ?",
			currentDatabase, stmt.Table, name,
		).Scan(&count).Error
	})

	if count != 0 {
		return true
	}
	return false
}

func (migrator Migrator) RenameColumn(value interface{}, oldName, field string) error {
	return migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		if field := stmt.Schema.LookUpField(field); field != nil {
			oldName = migrator.DB.NamingStrategy.ColumnName(stmt.Table, oldName)
			return migrator.DB.Exec(
				"ALTER TABLE ? RENAME COLUMN ? TO ?",
				clause.Table{Name: stmt.Table}, clause.Column{Name: oldName}, clause.Column{Name: field.DBName},
			).Error
		}
		return fmt.Errorf("failed to look up field with name: %s", field)
	})
}

func (migrator Migrator) ColumnTypes(value interface{}) ([]*sql.ColumnType, error) {
	return nil, gorm.ErrNotImplemented
}

func (migrator Migrator) CreateView(name string, option gorm.ViewOption) error {
	return gorm.ErrNotImplemented
}

func (migrator Migrator) DropView(name string) error {
	return gorm.ErrNotImplemented
}

func buildConstraint(constraint *schema.Constraint) (sql string, results []interface{}) {
	sql = "CONSTRAINT ? FOREIGN KEY ? REFERENCES ??"
	if constraint.OnDelete != "" {
		sql += " ON DELETE " + constraint.OnDelete
	}

	if constraint.OnUpdate != "" {
		sql += " ON UPDATE  " + constraint.OnUpdate
	}

	var foreignKeys, references []interface{}
	for _, field := range constraint.ForeignKeys {
		foreignKeys = append(foreignKeys, clause.Column{Name: field.DBName})
	}

	for _, field := range constraint.References {
		references = append(references, clause.Column{Name: field.DBName})
	}
	results = append(results, constraint.Name, foreignKeys, clause.Table{Name: constraint.ReferenceSchema.Table}, references)
	return
}

func (migrator Migrator) CreateConstraint(value interface{}, name string) error {
	return migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		checkConstraints := stmt.Schema.ParseCheckConstraints()
		if chk, ok := checkConstraints[name]; ok {
			return migrator.DB.Exec(
				"ALTER TABLE ? ADD CONSTRAINT ? CHECK ?",
				clause.Table{Name: stmt.Table}, clause.Column{Name: chk.Name}, clause.Expr{SQL: chk.Constraint},
			).Error
		}

		for _, rel := range stmt.Schema.Relationships.Relations {
			if constraint := rel.ParseConstraint(); constraint != nil && constraint.Name == name {
				sql, values := buildConstraint(constraint)
				return migrator.DB.Exec("ALTER TABLE ? ADD "+sql, append([]interface{}{clause.Table{Name: stmt.Table}}, values...)...).Error
			}
		}

		err := fmt.Errorf("failed to create constraint with name %v", name)
		if field := stmt.Schema.LookUpField(name); field != nil {
			for _, cc := range checkConstraints {
				if err = migrator.CreateIndex(value, cc.Name); err != nil {
					return err
				}
			}

			for _, rel := range stmt.Schema.Relationships.Relations {
				if constraint := rel.ParseConstraint(); constraint != nil && constraint.Field == field {
					if err = migrator.CreateIndex(value, constraint.Name); err != nil {
						return err
					}
				}
			}
		}

		return err
	})
}

func (migrator Migrator) DropConstraint(value interface{}, name string) error {
	return migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		return migrator.DB.Exec(
			"ALTER TABLE ? DROP CONSTRAINT ?",
			clause.Table{Name: stmt.Table}, clause.Column{Name: name},
		).Error
	})
}

func (migrator Migrator) HasConstraint(value interface{}, name string) bool {
	var count int64
	migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		currentDatabase := migrator.DB.Migrator().CurrentDatabase()
		return migrator.DB.Raw(
			"SELECT count(*) FROM INFORMATION_SCHEMA.referential_constraints WHERE constraint_schema = ? AND table_name = ? AND constraint_name = ?",
			currentDatabase, stmt.Table, name,
		).Scan(&count).Error
	})

	if count != 0 {
		return true
	}
	return false
}

func buildIndexOptions(opts []schema.IndexOption, stmt *gorm.Statement) (results []interface{}) {
	for _, opt := range opts {
		str := stmt.Quote(opt.DBName)
		if opt.Expression != "" {
			str = opt.Expression
		} else if opt.Length > 0 {
			str += fmt.Sprintf("(%d)", opt.Length)
		}

		if opt.Sort != "" {
			str += " " + opt.Sort
		}
		results = append(results, clause.Expr{SQL: str})
	}
	return
}

func (migrator Migrator) CreateIndex(value interface{}, name string) error {
	return migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		err := fmt.Errorf("failed to create index with name %v", name)
		indexes := stmt.Schema.ParseIndexes()

		if idx, ok := indexes[name]; ok {
			opts := buildIndexOptions(idx.Fields, stmt)
			values := []interface{}{clause.Column{Name: idx.Name}, clause.Table{Name: stmt.Table}, opts}

			createIndexSQL := "CREATE "
			if idx.Class != "" {
				createIndexSQL += idx.Class + " "
			}
			createIndexSQL += "INDEX ? ON ??"

			if idx.Comment != "" {
				values = append(values, idx.Comment)
				createIndexSQL += " COMMENT ?"
			}

			if idx.Type != "" {
				createIndexSQL += " USING " + idx.Type
			}

			return migrator.DB.Raw(createIndexSQL, values...).Error
		} else if field := stmt.Schema.LookUpField(name); field != nil {
			for _, idx := range indexes {
				for _, idxOpt := range idx.Fields {
					if idxOpt.Field == field {
						if err = migrator.CreateIndex(value, idx.Name); err != nil {
							return err
						}
					}
				}
			}
		}
		return err
	})
}

func (migrator Migrator) DropIndex(value interface{}, name string) error {
	return migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		return migrator.DB.Raw("DROP INDEX ? ON ?", clause.Column{Name: name}, clause.Table{Name: stmt.Table}).Error
	})
}

func (migrator Migrator) HasIndex(value interface{}, name string) bool {
	var count int64
	migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		currentDatabase := migrator.DB.Migrator().CurrentDatabase()
		return migrator.DB.Raw(
			"SELECT count(*) FROM information_schema.statistics WHERE table_schema = ? AND table_name = ? AND index_name = ?",
			currentDatabase, stmt.Table, name,
		).Scan(&count).Error
	})

	if count != 0 {
		return true
	}
	return false
}

func (migrator Migrator) RenameIndex(value interface{}, oldName, newName string) error {
	return migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		return migrator.DB.Exec(
			"ALTER TABLE ? RENAME INDEX ? TO ?",
			clause.Table{Name: stmt.Table}, clause.Column{Name: oldName}, clause.Column{Name: newName},
		).Error
	})
}

func (migrator Migrator) CurrentDatabase() (name string) {
	migrator.DB.Raw("SELECT DATABASE()").Scan(&name)
	return
}
