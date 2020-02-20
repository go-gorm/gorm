package migrator

import (
	"database/sql"
	"fmt"

	"github.com/jinzhu/gorm"
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
	return gorm.ErrNotImplemented
}

func (migrator Migrator) CreateTable(values ...interface{}) error {
	return gorm.ErrNotImplemented
}

func (migrator Migrator) DropTable(values ...interface{}) error {
	for _, value := range values {
		if err := migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
			return migrator.DB.Exec("DROP TABLE " + stmt.Quote(stmt.Table)).Error
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
			return migrator.DB.Exec(fmt.Sprintf("ALTER TABLE ? ADD ? %s", field.DBDataType), stmt.Table, field.DBName).Error
		}
		return fmt.Errorf("failed to look up field with name: %s", field)
	})
}

func (migrator Migrator) DropColumn(value interface{}, field string) error {
	return migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		if field := stmt.Schema.LookUpField(field); field != nil {
			return migrator.DB.Exec("ALTER TABLE ? DROP COLUMN ?", stmt.Table, field.DBName).Error
		}
		return fmt.Errorf("failed to look up field with name: %s", field)
	})
}

func (migrator Migrator) AlterColumn(value interface{}, field string) error {
	return migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		if field := stmt.Schema.LookUpField(field); field != nil {
			return migrator.DB.Exec(fmt.Sprintf("ALTER TABLE ? ALTER COLUMN ? TYPE %s", field.DBDataType), stmt.Table, field.DBName).Error
		}
		return fmt.Errorf("failed to look up field with name: %s", field)
	})
}

func (migrator Migrator) RenameColumn(value interface{}, oldName, field string) error {
	return migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		if field := stmt.Schema.LookUpField(field); field != nil {
			oldName = migrator.DB.NamingStrategy.ColumnName(stmt.Table, oldName)
			return migrator.DB.Exec("ALTER TABLE ? RENAME COLUMN ? TO ?", stmt.Table, oldName, field.DBName).Error
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
	return gorm.ErrNotImplemented
}

func (migrator Migrator) DropConstraint(value interface{}, name string) error {
	return migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		return migrator.DB.Raw("ALTER TABLE ? DROP CONSTRAINT ?", stmt.Table, name).Error
	})
}

func (migrator Migrator) CreateIndex(value interface{}, name string) error {
	return gorm.ErrNotImplemented
}

func (migrator Migrator) DropIndex(value interface{}, name string) error {
	return migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		return migrator.DB.Raw("DROP INDEX ? ON ?", name, stmt.Table).Error
	})
}

func (migrator Migrator) HasIndex(value interface{}, name string) bool {
	var count int64
	migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		currentDatabase := migrator.DB.Migrator().CurrentDatabase()
		return migrator.DB.Raw("SELECT count(*) FROM information_schema.statistics WHERE table_schema = ? AND table_name = ? AND index_name = ?", currentDatabase, stmt.Table, name).Scan(&count).Error
	})

	if count != 0 {
		return true
	}
	return false
}

func (migrator Migrator) RenameIndex(value interface{}, oldName, newName string) error {
	return migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		return migrator.DB.Exec("ALTER TABLE ? RENAME INDEX ? TO ?", stmt.Table, oldName, newName).Error
	})
}

func (migrator Migrator) CurrentDatabase() (name string) {
	migrator.DB.Raw("SELECT DATABASE()").Scan(&name)
	return
}
