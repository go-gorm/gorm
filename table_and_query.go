package gorm

import (
	"fmt"
	"strings"
)

// TableAndQuery returns the table name and the query already formatted as a string
func (scope Scope) TableAndQuery() (string, string) {
	scope.InstanceSet("skip_bindvar", true)
	//qs := LogFormatter("sql", "q", time.Duration(1), scope.SQL, scope.SQLVars, int64(1))
	//t, q := scope.TableName(), qs[3].(string)

	fmt.Println("#####", scope.SQL)
	t, q := scope.TableName(), scope.SQL
	if t == "" {
		qsplit := strings.Fields(strings.ToLower(q))
		for i := range qsplit {
			if qsplit[i] == "from" && i < len(qsplit)-2 {
				t = qsplit[i+1]
				break
			}
		}
	}

	return t, q
}
