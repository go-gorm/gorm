package gorm

import (
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/utils"
)

// Model specify the model you would like to run db operations
//    // update all users's name to `hello`
//    db.Model(&User{}).Update("name", "hello")
//    // if user's primary key is non-blank, will use it as condition, then will only update the user's name to `hello`
//    db.Model(&user).Update("name", "hello")
func (db *DB) Model(value interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Model = value
	return
}

// Clauses Add clauses
func (db *DB) Clauses(conds ...clause.Expression) (tx *DB) {
	tx = db.getInstance()
	var whereConds []interface{}

	for _, cond := range conds {
		if c, ok := cond.(clause.Interface); ok {
			tx.Statement.AddClause(c)
		} else if optimizer, ok := cond.(StatementModifier); ok {
			optimizer.ModifyStatement(tx.Statement)
		} else {
			whereConds = append(whereConds, cond)
		}
	}

	if len(whereConds) > 0 {
		tx.Statement.AddClause(clause.Where{Exprs: tx.Statement.BuildCondition(whereConds[0], whereConds[1:]...)})
	}
	return
}

var tableRegexp = regexp.MustCompile(`(?i).+? AS (\w+)\s*(?:$|,)`)

// Table specify the table you would like to run db operations
func (db *DB) Table(name string, args ...interface{}) (tx *DB) {
	tx = db.getInstance()
	if strings.Contains(name, " ") || strings.Contains(name, "`") || len(args) > 0 {
		tx.Statement.TableExpr = &clause.Expr{SQL: name, Vars: args}
		if results := tableRegexp.FindStringSubmatch(name); len(results) == 2 {
			tx.Statement.Table = results[1]
			return
		}
	} else if tables := strings.Split(name, "."); len(tables) == 2 {
		tx.Statement.TableExpr = &clause.Expr{SQL: tx.Statement.Quote(name)}
		tx.Statement.Table = tables[1]
		return
	}

	tx.Statement.Table = name
	return
}

// Distinct specify distinct fields that you want querying
func (db *DB) Distinct(args ...interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Distinct = true
	if len(args) > 0 {
		tx = tx.Select(args[0], args[1:]...)
	}
	return
}

// Select specify fields that you want when querying, creating, updating
func (db *DB) Select(query interface{}, args ...interface{}) (tx *DB) {
	tx = db.getInstance()

	switch v := query.(type) {
	case []string:
		tx.Statement.Selects = v

		for _, arg := range args {
			switch arg := arg.(type) {
			case string:
				tx.Statement.Selects = append(tx.Statement.Selects, arg)
			case []string:
				tx.Statement.Selects = append(tx.Statement.Selects, arg...)
			default:
				tx.AddError(fmt.Errorf("unsupported select args %v %v", query, args))
				return
			}
		}
		delete(tx.Statement.Clauses, "SELECT")
	case string:
		fields := strings.FieldsFunc(v, utils.IsValidDBNameChar)

		// normal field names
		if len(fields) == 1 || (len(fields) == 3 && strings.ToUpper(fields[1]) == "AS") {
			tx.Statement.Selects = []string{v}

			for _, arg := range args {
				switch arg := arg.(type) {
				case string:
					tx.Statement.Selects = append(tx.Statement.Selects, arg)
				case []string:
					tx.Statement.Selects = append(tx.Statement.Selects, arg...)
				default:
					tx.Statement.AddClause(clause.Select{
						Distinct:   db.Statement.Distinct,
						Expression: clause.Expr{SQL: v, Vars: args},
					})
					return
				}
			}

			delete(tx.Statement.Clauses, "SELECT")
		} else {
			tx.Statement.AddClause(clause.Select{
				Distinct:   db.Statement.Distinct,
				Expression: clause.Expr{SQL: v, Vars: args},
			})
		}
	default:
		tx.AddError(fmt.Errorf("unsupported select args %v %v", query, args))
	}

	return
}

// Omit specify fields that you want to ignore when creating, updating and querying
func (db *DB) Omit(columns ...string) (tx *DB) {
	tx = db.getInstance()

	if len(columns) == 1 && strings.ContainsRune(columns[0], ',') {
		tx.Statement.Omits = strings.FieldsFunc(columns[0], utils.IsValidDBNameChar)
	} else {
		tx.Statement.Omits = columns
	}
	return
}

// Where add conditions
func (db *DB) Where(query interface{}, args ...interface{}) (tx *DB) {
	tx = db.getInstance()
	if conds := tx.Statement.BuildCondition(query, args...); len(conds) > 0 {
		tx.Statement.AddClause(clause.Where{Exprs: conds})
	}
	return
}

// Not add NOT conditions
func (db *DB) Not(query interface{}, args ...interface{}) (tx *DB) {
	tx = db.getInstance()
	if conds := tx.Statement.BuildCondition(query, args...); len(conds) > 0 {
		tx.Statement.AddClause(clause.Where{Exprs: []clause.Expression{clause.Not(conds...)}})
	}
	return
}

// Or add OR conditions
func (db *DB) Or(query interface{}, args ...interface{}) (tx *DB) {
	tx = db.getInstance()
	if conds := tx.Statement.BuildCondition(query, args...); len(conds) > 0 {
		tx.Statement.AddClause(clause.Where{Exprs: []clause.Expression{clause.Or(clause.And(conds...))}})
	}
	return
}

// Joins specify Joins conditions
//     db.Joins("Account").Find(&user)
//     db.Joins("JOIN emails ON emails.user_id = users.id AND emails.email = ?", "jinzhu@example.org").Find(&user)
func (db *DB) Joins(query string, args ...interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Joins = append(tx.Statement.Joins, join{Name: query, Conds: args})
	return
}

// Group specify the group method on the find
func (db *DB) Group(name string) (tx *DB) {
	tx = db.getInstance()

	fields := strings.FieldsFunc(name, utils.IsValidDBNameChar)
	tx.Statement.AddClause(clause.GroupBy{
		Columns: []clause.Column{{Name: name, Raw: len(fields) != 1}},
	})
	return
}

// Having specify HAVING conditions for GROUP BY
func (db *DB) Having(query interface{}, args ...interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.AddClause(clause.GroupBy{
		Having: tx.Statement.BuildCondition(query, args...),
	})
	return
}

// Order specify order when retrieve records from database
//     db.Order("name DESC")
//     db.Order(clause.OrderByColumn{Column: clause.Column{Name: "name"}, Desc: true})
func (db *DB) Order(value interface{}) (tx *DB) {
	tx = db.getInstance()

	switch v := value.(type) {
	case clause.OrderByColumn:
		tx.Statement.AddClause(clause.OrderBy{
			Columns: []clause.OrderByColumn{v},
		})
	default:
		tx.Statement.AddClause(clause.OrderBy{
			Columns: []clause.OrderByColumn{{
				Column: clause.Column{Name: fmt.Sprint(value), Raw: true},
			}},
		})
	}
	return
}

// Limit specify the number of records to be retrieved
func (db *DB) Limit(limit int) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.AddClause(clause.Limit{Limit: limit})
	return
}

// Offset specify the number of records to skip before starting to return the records
func (db *DB) Offset(offset int) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.AddClause(clause.Limit{Offset: offset})
	return
}

// Scopes pass current database connection to arguments `func(DB) DB`, which could be used to add conditions dynamically
//     func AmountGreaterThan1000(db *gorm.DB) *gorm.DB {
//         return db.Where("amount > ?", 1000)
//     }
//
//     func OrderStatus(status []string) func (db *gorm.DB) *gorm.DB {
//         return func (db *gorm.DB) *gorm.DB {
//             return db.Scopes(AmountGreaterThan1000).Where("status in (?)", status)
//         }
//     }
//
//     db.Scopes(AmountGreaterThan1000, OrderStatus([]string{"paid", "shipped"})).Find(&orders)
func (db *DB) Scopes(funcs ...func(*DB) *DB) *DB {
	for _, f := range funcs {
		db = f(db)
	}
	return db
}

// Preload preload associations with given conditions
//    db.Preload("Orders", "state NOT IN (?)", "cancelled").Find(&users)
func (db *DB) Preload(query string, args ...interface{}) (tx *DB) {
	tx = db.getInstance()
	if tx.Statement.Preloads == nil {
		tx.Statement.Preloads = map[string][]interface{}{}
	}
	tx.Statement.Preloads[query] = args
	return
}

func (db *DB) Attrs(attrs ...interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.attrs = attrs
	return
}

func (db *DB) Assign(attrs ...interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.assigns = attrs
	return
}

func (db *DB) Unscoped() (tx *DB) {
	tx = db.getInstance()
	tx.Statement.Unscoped = true
	return
}

func (db *DB) Raw(sql string, values ...interface{}) (tx *DB) {
	tx = db.getInstance()
	tx.Statement.SQL = strings.Builder{}

	if strings.Contains(sql, "@") {
		clause.NamedExpr{SQL: sql, Vars: values}.Build(tx.Statement)
	} else {
		clause.Expr{SQL: sql, Vars: values}.Build(tx.Statement)
	}
	return
}
