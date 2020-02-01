package schema

import (
	"fmt"
	"strings"
	"sync"

	"github.com/jinzhu/inflection"
)

// Namer namer interface
type Namer interface {
	TableName(table string) string
	ColumnName(column string) string
	JoinTableName(table string) string
}

// NamingStrategy tables, columns naming strategy
type NamingStrategy struct {
	TablePrefix   string
	SingularTable bool
}

// TableName convert string to table name
func (ns NamingStrategy) TableName(str string) string {
	if ns.SingularTable {
		return ns.TablePrefix + toDBName(str)
	}
	return ns.TablePrefix + inflection.Plural(toDBName(str))
}

// ColumnName convert string to column name
func (ns NamingStrategy) ColumnName(str string) string {
	return toDBName(str)
}

// JoinTableName convert string to join table name
func (ns NamingStrategy) JoinTableName(str string) string {
	return ns.TablePrefix + toDBName(str)
}

var (
	smap sync.Map
	// https://github.com/golang/lint/blob/master/lint.go#L770
	commonInitialisms         = []string{"API", "ASCII", "CPU", "CSS", "DNS", "EOF", "GUID", "HTML", "HTTP", "HTTPS", "ID", "IP", "JSON", "LHS", "QPS", "RAM", "RHS", "RPC", "SLA", "SMTP", "SSH", "TLS", "TTL", "UID", "UI", "UUID", "URI", "URL", "UTF8", "VM", "XML", "XSRF", "XSS"}
	commonInitialismsReplacer *strings.Replacer
)

func init() {
	var commonInitialismsForReplacer []string
	for _, initialism := range commonInitialisms {
		commonInitialismsForReplacer = append(commonInitialismsForReplacer, initialism, strings.Title(strings.ToLower(initialism)))
	}
	commonInitialismsReplacer = strings.NewReplacer(commonInitialismsForReplacer...)
}

func toDBName(name string) string {
	if name == "" {
		return ""
	} else if v, ok := smap.Load(name); ok {
		return fmt.Sprint(v)
	}

	var (
		value                          = commonInitialismsReplacer.Replace(name)
		buf                            strings.Builder
		lastCase, nextCase, nextNumber bool // upper case == true
		curCase                        = value[0] <= 'Z' && value[0] >= 'A'
	)

	for i, v := range value[:len(value)-1] {
		nextCase = value[i+1] <= 'Z' && value[i+1] >= 'A'
		nextNumber = value[i+1] >= '0' && value[i+1] <= '9'

		if curCase {
			if lastCase && (nextCase || nextNumber) {
				buf.WriteRune(v + 32)
			} else {
				if i > 0 && value[i-1] != '_' && value[i+1] != '_' {
					buf.WriteByte('_')
				}
				buf.WriteRune(v + 32)
			}
		} else {
			buf.WriteRune(v)
		}

		lastCase = curCase
		curCase = nextCase
	}

	if curCase {
		if !lastCase && len(value) > 1 {
			buf.WriteByte('_')
		}
		buf.WriteByte(value[len(value)-1] + 32)
	} else {
		buf.WriteByte(value[len(value)-1])
	}

	return buf.String()
}
