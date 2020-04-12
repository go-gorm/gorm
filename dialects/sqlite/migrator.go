package sqlite

import (
	"fmt"

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

func (m Migrator) HasColumn(value interface{}, field string) bool {
	var count int
	m.Migrator.RunWithValue(value, func(stmt *gorm.Statement) error {
		name := field
		if field := stmt.Schema.LookUpField(field); field != nil {
			name = field.DBName
		}

		return m.DB.Raw(
			"SELECT count(*) FROM sqlite_master WHERE type = ? AND tbl_name = ? AND (sql LIKE ? OR sql LIKE ? OR sql LIKE ?)",
			"table", stmt.Table, `%"`+name+`" %`, `%`+name+` %`, "%`"+name+"`%",
		).Row().Scan(&count)
	})
	return count > 0
}

func (m Migrator) HasIndex(value interface{}, name string) bool {
	var count int
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		return m.DB.Raw(
			"SELECT count(*) FROM sqlite_master WHERE type = ? AND tbl_name = ? AND sql LIKE ?",
			"index", stmt.Table, "%INDEX "+name+" ON%",
		).Row().Scan(&count)
	})
	return count > 0
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
		err := fmt.Errorf("failed to create index with name %v", name)
		indexes := stmt.Schema.ParseIndexes()

		if idx, ok := indexes[name]; ok {
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
		} else if field := stmt.Schema.LookUpField(name); field != nil {
			for _, idx := range indexes {
				for _, idxOpt := range idx.Fields {
					if idxOpt.Field == field {
						if err = m.CreateIndex(value, idx.Name); err != nil {
							return err
						}
					}
				}
			}
		}
		return err
	})
}
