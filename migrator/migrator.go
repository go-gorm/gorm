package migrator

import (
	"database/sql"
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
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
	// if has table
	// not -> create table
	// check columns -> add column, change column type
	// check foreign keys -> create indexes
	// check indexes -> create indexes

	return gorm.ErrNotImplemented
}

func (migrator Migrator) CreateTable(values ...interface{}) error {
	// migrate
	// create join table
	return gorm.ErrNotImplemented
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
				sql := "ALTER TABLE ? ADD CONSTRAINT ? FOREIGN KEY ? REFERENCES ??"
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

				return migrator.DB.Exec(
					sql, clause.Table{Name: stmt.Table}, clause.Column{Name: constraint.Name}, foreignKeys, clause.Table{Name: constraint.ReferenceSchema.Table}, references,
				).Error
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

func (migrator Migrator) CreateIndex(value interface{}, name string) error {
	return migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		err := fmt.Errorf("failed to create index with name %v", name)
		indexes := stmt.Schema.ParseIndexes()

		if idx, ok := indexes[name]; ok {
			fields := []interface{}{}
			for _, field := range idx.Fields {
				str := stmt.Quote(field.DBName)
				if field.Expression != "" {
					str = field.Expression
				} else if field.Length > 0 {
					str += fmt.Sprintf("(%d)", field.Length)
				}

				if field.Sort != "" {
					str += " " + field.Sort
				}
				fields = append(fields, clause.Expr{SQL: str})
			}
			values := []interface{}{clause.Column{Name: idx.Name}, clause.Table{Name: stmt.Table}, fields}

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
