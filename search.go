package gorm

import "strconv"

type search struct {
	db          *DB
	whereClause []map[string]interface{}
	orClause    []map[string]interface{}
	notClause   []map[string]interface{}
	initAttrs   []interface{}
	assignAttrs []interface{}
	orders      []string
	selectStr   string
	offsetStr   string
	limitStr    string
	tableName   string
	unscope     bool
}

func (s *search) clone() *search {
	return &search{
		whereClause: s.whereClause,
		orClause:    s.orClause,
		notClause:   s.notClause,
		initAttrs:   s.initAttrs,
		assignAttrs: s.assignAttrs,
		orders:      s.orders,
		selectStr:   s.selectStr,
		offsetStr:   s.offsetStr,
		limitStr:    s.limitStr,
		unscope:     s.unscope,
	}
}

func (s *search) where(query interface{}, values ...interface{}) *search {
	s.whereClause = append(s.whereClause, map[string]interface{}{"query": query, "args": values})
	return s
}

func (s *search) not(query interface{}, values ...interface{}) *search {
	s.notClause = append(s.notClause, map[string]interface{}{"query": query, "args": values})
	return s
}

func (s *search) or(query interface{}, values ...interface{}) *search {
	s.orClause = append(s.orClause, map[string]interface{}{"query": query, "args": values})
	return s
}

func (s *search) attrs(attrs ...interface{}) *search {
	s.initAttrs = append(s.initAttrs, toSearchableMap(attrs...))
	return s
}

func (s *search) assign(attrs ...interface{}) *search {
	s.assignAttrs = append(s.assignAttrs, toSearchableMap(attrs...))
	return s
}

func (s *search) order(value string, reorder ...bool) *search {
	if len(reorder) > 0 && reorder[0] {
		s.orders = []string{value}
	} else {
		s.orders = append(s.orders, value)
	}
	return s
}

func (s *search) selects(value interface{}) *search {
	if str, err := getInterfaceAsString(value); err == nil {
		s.selectStr = str
	}
	return s
}

func (s *search) limit(value interface{}) *search {
	if str, err := getInterfaceAsString(value); err == nil {
		s.limitStr = str
	}
	return s
}

func (s *search) offset(value interface{}) *search {
	if str, err := getInterfaceAsString(value); err == nil {
		s.offsetStr = str
	}
	return s
}

func (s *search) unscoped() *search {
	s.unscope = true
	return s
}

func (s *search) table(name string) *search {
	s.tableName = name
	return s
}

func getInterfaceAsString(value interface{}) (str string, err error) {
	switch value := value.(type) {
	case string:
		str = value
	case int:
		if value < 0 {
			str = ""
		} else {
			str = strconv.Itoa(value)
		}
	default:
		err = InvalidSql
	}
	return
}
