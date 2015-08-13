package gorm

import "fmt"

type search struct {
	db               *DB
	whereConditions  []map[string]interface{}
	orConditions     []map[string]interface{}
	notConditions    []map[string]interface{}
	havingConditions []map[string]interface{}
	initAttrs        []interface{}
	assignAttrs      []interface{}
	selects          map[string]interface{}
	omits            []string
	orders           []string
	joins            string
	preload          []searchPreload
	offset           string
	limit            string
	group            string
	tableName        string
	raw              bool
	Unscoped         bool
	countingQuery    bool
}

type searchPreload struct {
	schema     string
	conditions []interface{}
}

func (s *search) clone() *search {
	clone := *s
	return &clone
}

func (s *search) Where(query interface{}, values ...interface{}) *search {
	s.whereConditions = append(s.whereConditions, map[string]interface{}{"query": query, "args": values})
	return s
}

func (s *search) Not(query interface{}, values ...interface{}) *search {
	s.notConditions = append(s.notConditions, map[string]interface{}{"query": query, "args": values})
	return s
}

func (s *search) Or(query interface{}, values ...interface{}) *search {
	s.orConditions = append(s.orConditions, map[string]interface{}{"query": query, "args": values})
	return s
}

func (s *search) Attrs(attrs ...interface{}) *search {
	s.initAttrs = append(s.initAttrs, toSearchableMap(attrs...))
	return s
}

func (s *search) Assign(attrs ...interface{}) *search {
	s.assignAttrs = append(s.assignAttrs, toSearchableMap(attrs...))
	return s
}

func (s *search) Order(value string, reorder ...bool) *search {
	if len(reorder) > 0 && reorder[0] {
		if value != "" {
			s.orders = []string{value}
		} else {
			s.orders = []string{}
		}
	} else if value != "" {
		s.orders = append(s.orders, value)
	}
	return s
}

func (s *search) Select(query interface{}, args ...interface{}) *search {
	s.selects = map[string]interface{}{"query": query, "args": args}
	return s
}

func (s *search) Omit(columns ...string) *search {
	s.omits = columns
	return s
}

func (s *search) Limit(value interface{}) *search {
	s.limit = s.getInterfaceAsSql(value)
	return s
}

func (s *search) Offset(value interface{}) *search {
	s.offset = s.getInterfaceAsSql(value)
	return s
}

func (s *search) Group(query string) *search {
	s.group = s.getInterfaceAsSql(query)
	return s
}

func (s *search) Having(query string, values ...interface{}) *search {
	s.havingConditions = append(s.havingConditions, map[string]interface{}{"query": query, "args": values})
	return s
}

func (s *search) Joins(query string) *search {
	s.joins = query
	return s
}

func (s *search) Preload(schema string, values ...interface{}) *search {
	var preloads []searchPreload
	for _, preload := range s.preload {
		if preload.schema != schema {
			preloads = append(preloads, preload)
		}
	}
	preloads = append(preloads, searchPreload{schema, values})
	s.preload = preloads
	return s
}

func (s *search) Raw(b bool) *search {
	s.raw = b
	return s
}

func (s *search) unscoped() *search {
	s.Unscoped = true
	return s
}

func (s *search) Table(name string) *search {
	s.tableName = name
	return s
}

func (s *search) getInterfaceAsSql(value interface{}) (str string) {
	switch value.(type) {
	case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		str = fmt.Sprintf("%v", value)
	default:
		s.db.AddError(InvalidSql)
	}

	if str == "-1" {
		return ""
	}
	return
}
