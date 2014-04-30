package gorm

import "strconv"

type search struct {
	db              *DB
	WhereConditions []map[string]interface{}
	OrConditions    []map[string]interface{}
	NotConditions   []map[string]interface{}
	InitAttrs       []interface{}
	AssignAttrs     []interface{}
	HavingCondition map[string]interface{}
	Orders          []string
	Joins           string
	Select          string
	Offset          string
	Limit           string
	Group           string
	TableName       string
	Unscope         bool
	Raw             bool
}

func (s *search) clone() *search {
	return &search{
		WhereConditions: s.WhereConditions,
		OrConditions:    s.OrConditions,
		NotConditions:   s.NotConditions,
		InitAttrs:       s.InitAttrs,
		AssignAttrs:     s.AssignAttrs,
		HavingCondition: s.HavingCondition,
		Orders:          s.Orders,
		Select:          s.Select,
		Offset:          s.Offset,
		Limit:           s.Limit,
		Unscope:         s.Unscope,
		Group:           s.Group,
		Joins:           s.Joins,
		TableName:       s.TableName,
		Raw:             s.Raw,
	}
}

func (s *search) where(query interface{}, values ...interface{}) *search {
	s.WhereConditions = append(s.WhereConditions, map[string]interface{}{"query": query, "args": values})
	return s
}

func (s *search) not(query interface{}, values ...interface{}) *search {
	s.NotConditions = append(s.NotConditions, map[string]interface{}{"query": query, "args": values})
	return s
}

func (s *search) or(query interface{}, values ...interface{}) *search {
	s.OrConditions = append(s.OrConditions, map[string]interface{}{"query": query, "args": values})
	return s
}

func (s *search) attrs(attrs ...interface{}) *search {
	s.InitAttrs = append(s.InitAttrs, toSearchableMap(attrs...))
	return s
}

func (s *search) assign(attrs ...interface{}) *search {
	s.AssignAttrs = append(s.AssignAttrs, toSearchableMap(attrs...))
	return s
}

func (s *search) order(value string, reorder ...bool) *search {
	if len(reorder) > 0 && reorder[0] {
		s.Orders = []string{value}
	} else {
		s.Orders = append(s.Orders, value)
	}
	return s
}

func (s *search) selects(value interface{}) *search {
	s.Select = s.getInterfaceAsSql(value)
	return s
}

func (s *search) limit(value interface{}) *search {
	s.Limit = s.getInterfaceAsSql(value)
	return s
}

func (s *search) offset(value interface{}) *search {
	s.Offset = s.getInterfaceAsSql(value)
	return s
}

func (s *search) group(query string) *search {
	s.Group = s.getInterfaceAsSql(query)
	return s
}

func (s *search) having(query string, values ...interface{}) *search {
	s.HavingCondition = map[string]interface{}{"query": query, "args": values}
	return s
}

func (s *search) includes(value interface{}) *search {
	return s
}

func (s *search) joins(query string) *search {
	s.Joins = query
	return s
}

func (s *search) raw(b bool) *search {
	s.Raw = b
	return s
}

func (s *search) unscoped() *search {
	s.Unscope = true
	return s
}

func (s *search) table(name string) *search {
	s.TableName = name
	return s
}

func (s *search) getInterfaceAsSql(value interface{}) (str string) {
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
		s.db.err(InvalidSql)
	}
	return
}
