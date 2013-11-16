package gorm

import "strconv"

type search struct {
	conditions         map[string][]interface{}
	orders             []string
	selectStr          string
	offsetStr          string
	limitStr           string
	specifiedTableName string
	unscope            bool
}

func (s *search) addToCondition(typ string, value interface{}) {
	s.conditions[typ] = append(s.conditions[typ], value)
}

func (s *search) where(query string, values ...interface{}) {
	s.addToCondition("where", map[string]interface{}{"query": query, "args": values})
}

func (s *search) not(query string, values ...interface{}) {
	s.addToCondition("not", map[string]interface{}{"query": query, "args": values})
}

func (s *search) or(query string, values ...interface{}) {
	s.addToCondition("or", map[string]interface{}{"query": query, "args": values})
}

func (s *search) attrs(attrs ...interface{}) {
	s.addToCondition("attrs", toSearchableMap(attrs...))
}

func (s *search) assign(attrs ...interface{}) {
	s.addToCondition("assign", toSearchableMap(attrs...))
}

func (s *search) order(value string, reorder ...bool) {
	if len(reorder) > 0 && reorder[0] {
		s.orders = []string{value}
	} else {
		s.orders = append(s.orders, value)
	}
}

func (s *search) selects(value interface{}) {
	if str, err := getInterfaceAsString(value); err == nil {
		s.selectStr = str
	}
}

func (s *search) limit(value interface{}) {
	if str, err := getInterfaceAsString(value); err == nil {
		s.limitStr = str
	}
}

func (s *search) offset(value interface{}) {
	if str, err := getInterfaceAsString(value); err == nil {
		s.offsetStr = str
	}
}

func (s *search) unscoped() {
	s.unscope = true
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
