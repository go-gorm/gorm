package sqlite

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
	"github.com/jinzhu/gorm/migrator"
	"github.com/jinzhu/gorm/schema"
)

type Migrator struct {
	migrator.Migrator
}

func (m Migrator) HasTable(value interface{}) bool {
	var count int
	m.Migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		return m.DB.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", stmt.Table).Row().Scan(&count)
	})
	return count > 0
}

func (m Migrator) HasColumn(value interface{}, name string) bool {
	var count int
	m.Migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		if field := stmt.Schema.LookUpField(name); field != nil {
			name = field.DBName
		}

		return m.DB.Raw(
			"SELECT count(*) FROM sqlite_master WHERE type = ? AND tbl_name = ? AND (sql LIKE ? OR sql LIKE ? OR sql LIKE ?)",
			"table", stmt.Table, `%"`+name+`" %`, `%`+name+` %`, "%`"+name+"`%",
		).Row().Scan(&count)
	})
	return count > 0
}

func (m Migrator) AlterColumn(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if field := stmt.Schema.LookUpField(name); field != nil {
			var (
				createSQL    string
				newTableName = stmt.Table + "__temp"
			)

			m.DB.Raw("SELECT sql FROM sqlite_master WHERE type = ? AND tbl_name = ? AND name = ?", "table", stmt.Table, stmt.Table).Row().Scan(&createSQL)

			if reg, err := regexp.Compile("(`|'|\"| )" + name + "(`|'|\"| ) .*?,"); err == nil {
				tableReg, err := regexp.Compile(" ('|`|\"| )" + stmt.Table + "('|`|\"| ) ")
				if err != nil {
					return err
				}

				createSQL = tableReg.ReplaceAllString(createSQL, fmt.Sprintf(" `%v` ", newTableName))
				createSQL = reg.ReplaceAllString(createSQL, "?")

				var columns []string
				columnTypes, _ := m.DB.Migrator().ColumnTypes(value)
				for _, columnType := range columnTypes {
					columns = append(columns, fmt.Sprintf("`%v`", columnType.Name()))
				}

				createSQL = fmt.Sprintf("PRAGMA foreign_keys=off;BEGIN TRANSACTION;"+createSQL+";INSERT INTO `%v`(%v) SELECT %v FROM `%v`;DROP TABLE `%v`;ALTER TABLE `%v` RENAME TO `%v`;COMMIT;", newTableName, strings.Join(columns, ","), strings.Join(columns, ","), stmt.Table, stmt.Table, newTableName, stmt.Table)
				return m.DB.Exec(createSQL, m.FullDataTypeOf(field)).Error
			} else {
				return err
			}
		} else {
			return fmt.Errorf("failed to alter field with name %v", name)
		}
	})
}

func (m Migrator) DropColumn(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if field := stmt.Schema.LookUpField(name); field != nil {
			name = field.DBName
		}

		var (
			createSQL    string
			newTableName = stmt.Table + "__temp"
		)

		m.DB.Raw("SELECT sql FROM sqlite_master WHERE type = ? AND tbl_name = ? AND name = ?", "table", stmt.Table, stmt.Table).Row().Scan(&createSQL)

		if reg, err := regexp.Compile("(`|'|\"| )" + name + "(`|'|\"| ) .*?,"); err == nil {
			tableReg, err := regexp.Compile(" ('|`|\"| )" + stmt.Table + "('|`|\"| ) ")
			if err != nil {
				return err
			}

			createSQL = tableReg.ReplaceAllString(createSQL, fmt.Sprintf(" `%v` ", newTableName))
			createSQL = reg.ReplaceAllString(createSQL, "")

			var columns []string
			columnTypes, _ := m.DB.Migrator().ColumnTypes(value)
			for _, columnType := range columnTypes {
				if columnType.Name() != name {
					columns = append(columns, fmt.Sprintf("`%v`", columnType.Name()))
				}
			}

			createSQL = fmt.Sprintf("PRAGMA foreign_keys=off;BEGIN TRANSACTION;"+createSQL+";INSERT INTO `%v`(%v) SELECT %v FROM `%v`;DROP TABLE `%v`;ALTER TABLE `%v` RENAME TO `%v`;COMMIT;", newTableName, strings.Join(columns, ","), strings.Join(columns, ","), stmt.Table, stmt.Table, newTableName, stmt.Table)

			return m.DB.Exec(createSQL).Error
		} else {
			return err
		}
	})
}

func (m Migrator) CreateConstraint(interface{}, string) error {
	return gorm.ErrNotImplemented
}

func (m Migrator) DropConstraint(interface{}, string) error {
	return gorm.ErrNotImplemented
}

func (m Migrator) CurrentDatabase() (name string) {
	var null interface{}
	m.DB.Raw("PRAGMA database_list").Row().Scan(&null, &name, &null)
	return
}

func (m Migrator) BuildIndexOptions(opts []schema.IndexOption, stmt *gorm.Statement) (results []interface{}) {
	for _, opt := range opts {
		str := stmt.Quote(opt.DBName)
		if opt.Expression != "" {
			str = opt.Expression
		}

		if opt.Collate != "" {
			str += " COLLATE " + opt.Collate
		}

		if opt.Sort != "" {
			str += " " + opt.Sort
		}
		results = append(results, clause.Expr{SQL: str})
	}
	return
}

func (m Migrator) CreateIndex(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if idx := stmt.Schema.LookIndex(name); idx != nil {
			opts := m.BuildIndexOptions(idx.Fields, stmt)
			values := []interface{}{clause.Column{Name: idx.Name}, clause.Table{Name: stmt.Table}, opts}

			createIndexSQL := "CREATE "
			if idx.Class != "" {
				createIndexSQL += idx.Class + " "
			}
			createIndexSQL += "INDEX ?"

			if idx.Type != "" {
				createIndexSQL += " USING " + idx.Type
			}
			createIndexSQL += " ON ??"

			if idx.Where != "" {
				createIndexSQL += " WHERE " + idx.Where
			}

			return m.DB.Exec(createIndexSQL, values...).Error
		}

		return fmt.Errorf("failed to create index with name %v", name)
	})
}

func (m Migrator) HasIndex(value interface{}, name string) bool {
	var count int
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if idx := stmt.Schema.LookIndex(name); idx != nil {
			name = idx.Name
		}

		m.DB.Raw(
			"SELECT count(*) FROM sqlite_master WHERE type = ? AND tbl_name = ? AND name = ?", "index", stmt.Table, name,
		).Row().Scan(&count)
		return nil
	})
	return count > 0
}

func (m Migrator) RenameIndex(value interface{}, oldName, newName string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		var sql string
		m.DB.Raw("SELECT sql FROM sqlite_master WHERE type = ? AND tbl_name = ? AND name = ?", "index", stmt.Table, oldName).Row().Scan(&sql)
		if sql != "" {
			return m.DB.Exec(strings.Replace(sql, oldName, newName, 1)).Error
		}
		return fmt.Errorf("failed to find index with name %v", oldName)
	})
}

func (m Migrator) DropIndex(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if idx := stmt.Schema.LookIndex(name); idx != nil {
			name = idx.Name
		}

		return m.DB.Exec("DROP INDEX ?", clause.Column{Name: name}).Error
	})
}
