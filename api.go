package gorm

import (
	"github.com/jinzhu/gorm/builder"
	"github.com/jinzhu/gorm/dialects"
)

// Where add condition
func (s *DB) Where(query interface{}, args ...interface{}) *DB {
	tx := s.init()
	tx.Statement.AddConditions(tx.Statement.BuildCondition(query, args...))
	return tx
}

// Not add NOT condition
func (s *DB) Not(query interface{}, args ...interface{}) *DB {
	tx := s.init()
	tx.Statement.AddConditions(builder.Not(tx.Statement.BuildCondition(query, args...)))
	return tx
}

// And add AND conditions
func (s *DB) And(conds ...builder.Condition) *DB {
	tx := s.init()
	tx.Statement.AddConditions(builder.And(conds))
	return tx
}

// Or add OR conditions
func (s *DB) Or(conds ...builder.Condition) *DB {
	tx := s.init()
	tx.Statement.AddConditions(builder.Or(conds))
	return tx
}

// Joins specify Joins conditions
//     db.Joins("JOIN emails ON emails.user_id = users.id AND emails.email = ?", "jinzhu@example.org").Find(&user)
func (s *DB) Joins(query string, args ...interface{}) *DB {
	tx := s.init()
	// FIXME
	tx.Statement.Joins = append(tx.Statement.Joins, builder.Join{Conditions: []builder.Condition{tx.Statement.BuildCondition(query, args...)}})
	return tx
}

// Group specify the group method on the find
func (s *DB) Group(column string) *DB {
	tx := s.init()
	tx.Statement.GroupBy.GroupByColumns = append(tx.Statement.GroupBy.GroupByColumns, column)
	return tx
}

// Having specify HAVING conditions for GROUP BY
func (s *DB) Having(query interface{}, args ...interface{}) *DB {
	tx := s.init()
	tx.Statement.GroupBy.Having = append(tx.Statement.GroupBy.Having, tx.Statement.BuildCondition(query, args...))
	return tx
}

// Order specify order when retrieve records from database
//     db.Order("name DESC")
//     db.Order(gorm.Expr("name = ? DESC", "first")) // sql expression
func (s *DB) Order(value interface{}) *DB {
	tx := s.init()
	tx.Statement.OrderBy = append(tx.Statement.OrderBy, value)
	return tx
}

// Reorder works like Order, but will overwrite current order information
func (s *DB) Reorder(value interface{}) *DB {
	tx := s.init()
	tx.Statement.OrderBy = []builder.OrderCondition{value}
	return tx
}

// Limit specify the number of records to be retrieved
func (s *DB) Limit(limit int64) *DB {
	tx := s.init()
	if limit < 0 {
		tx.Statement.Limit.Limit = nil
	} else {
		tx.Statement.Limit.Limit = &limit
	}
	return tx
}

// Offset specify the number of records to skip before starting to return the records
func (s *DB) Offset(offset int64) *DB {
	tx := s.init()
	if offset < 0 {
		tx.Statement.Limit.Offset = nil
	} else {
		tx.Statement.Limit.Offset = &offset
	}
	return tx
}

// Select specify fields that you want when querying, creating, updating
func (s *DB) Select(query interface{}, args ...interface{}) *DB {
	tx := s.init()

	switch value := query.(type) {
	case string:
		tx.Statement.Select.Columns = []string{value}
	case []string:
		tx.Statement.Select.Columns = value
	default:
		tx.AddError(ErrUnsupportedSelect)
	}
	tx.Statement.Select.Args = args
	return tx
}

// Omit specify fields that you want to ignore when creating, updating and querying
func (s *DB) Omit(columns ...string) *DB {
	tx := s.init()
	tx.Statement.Omit = columns
	return tx
}

// First find first record that match given conditions, order by primary key
func (s *DB) First(out interface{}, where ...interface{}) *DB {
	conds := []interface{}{builder.Limit{Limit: &one}, builder.Settings{"gorm:order_by_primary_key": "ASC"}}
	if len(where) > 0 {
		conds = append(conds, s.Statement.BuildCondition(where[0], where[1:]...))
	}
	return s.Find(out, conds...)
}

// Take return a record that match given conditions, the order will depend on the database implementation
func (s *DB) Take(out interface{}, where ...interface{}) *DB {
	conds := []interface{}{builder.Limit{Limit: &one}}
	if len(where) > 0 {
		conds = append(conds, s.Statement.BuildCondition(where[0], where[1:]...))
	}
	return s.Find(out, conds...)
}

// Last find last record that match given conditions, order by primary key
func (s *DB) Last(out interface{}, where ...interface{}) *DB {
	conds := []interface{}{builder.Limit{Limit: &one}, builder.Settings{"gorm:order_by_primary_key": "DESC"}}
	if len(where) > 0 {
		conds = append(conds, s.Statement.BuildCondition(where[0], where[1:]...))
	}
	return s.Find(out, conds...)
}

// Find find records that match given conditions
func (s *DB) Find(out interface{}, where ...interface{}) *DB {
	tx := s.init()
	stmt := tx.Statement
	stmt.Dest = out

	if len(where) > 0 {
		stmt = s.Statement.Clone()
		stmt.Conditions = append(stmt.Conditions, s.Statement.BuildCondition(where[0], where[1:]...))
	}

	tx.AddError(tx.Dialect().Query(stmt))
	return tx
}

// Scan scan value to a struct
func (s *DB) Scan(dest interface{}) *DB {
	var (
		tx   = s.init()
		stmt = tx.Statement.Clone()
	)

	stmt.Table = stmt.Dest
	stmt.Dest = dest
	tx.AddError(tx.Dialect().Query(stmt))
	return tx
}

// Create insert the value into database
func (s *DB) Create(value interface{}) *DB {
	tx := s.init()
	tx.Statement.Dest = value
	tx.AddError(tx.Dialect().Insert(tx.Statement))
	return tx
}

// Save update value in database, if the value doesn't have primary key, will insert it
func (s *DB) Save(value interface{}) *DB {
	tx := s.init()
	tx.Statement.Dest = value
	// FIXME check primary key has value or not
	tx.AddError(tx.Dialect().Update(tx.Statement))
	return tx
}

// Update update attributes with callbacks, refer: https://jinzhu.github.io/gorm/crud.html#update
func (s *DB) Update(column string, value interface{}) *DB {
	tx := s.init()
	tx.Statement.Assignments = append(tx.Statement.Assignments, builder.Assignment{Column: column, Value: value})
	tx.AddError(tx.Dialect().Update(tx.Statement))
	return tx
}

// Updates update attributes with callbacks, refer: https://jinzhu.github.io/gorm/crud.html#update
func (s *DB) Updates(values interface{}) *DB {
	tx := s.init()
	tx.Statement.Assignments = append(tx.Statement.Assignments, builder.Assignment{Value: values})
	tx.AddError(tx.Dialect().Update(tx.Statement))
	return tx
}

// Delete delete value match given conditions, if the value has primary key, then will including the primary key as condition
func (s *DB) Delete(value interface{}, where ...interface{}) *DB {
	tx := s.init()
	stmt := tx.Statement
	stmt.Dest = value

	if len(where) > 0 {
		stmt = s.Statement.Clone()
		stmt.Conditions = append(stmt.Conditions, s.Statement.BuildCondition(where[0], where[1:]...))
	}

	tx.AddError(tx.Dialect().Update(stmt))
	return tx
}

// Model specify the model you would like to run db operations
//    // update all users's name to `hello`
//    db.Model(&User{}).Update("name", "hello")
//    // if user's primary key is non-blank, will use it as condition, then will only update the user's name to `hello`
//    db.Model(&user).Update("name", "hello")
func (s *DB) Model(value interface{}) *DB {
	tx := s.init()
	tx.Statement.Dest = value
	return tx
}

// Table specify the table you would like to run db operations
func (s *DB) Table(name string) *DB {
	tx := s.init()
	tx.Statement.Table = name
	return tx
}

// AddError add error to the db
func (s *DB) AddError(err error) {
	if err != nil {
		if err != ErrRecordNotFound {
			s.Config.Logger.Error(err)
		}

		if errs := s.GetErrors(); len(errs) == 0 {
			s.Error = err
		} else {
			s.Error = Errors(errs).Add(err)
		}
	}
}

// GetErrors get happened errors from the db
func (s *DB) GetErrors() []error {
	if errs, ok := s.Error.(Errors); ok {
		return errs
	} else if s.Error != nil {
		return []error{s.Error}
	}
	return []error{}
}

// Dialect return DB dialect
func (s *DB) Dialect() dialects.Dialect {
	if s.TxDialect != nil {
		return s.TxDialect
	}
	return s.Config.Dialect
}

// Scopes pass current database connection to arguments `func(*DB) *DB`, which could be used to add conditions dynamically
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
// Refer https://jinzhu.github.io/gorm/crud.html#scopes
func (s *DB) Scopes(funcs ...func(*DB) *DB) *DB {
	for _, f := range funcs {
		s = f(s)
	}
	return s
}

// init init DB
func (s *DB) init() *DB {
	if s.Statement == nil {
		return &DB{
			TxDialect: s.TxDialect,
			Statement: &builder.Statement{},
			Config:    s.Config,
		}
	}
	return s
}
