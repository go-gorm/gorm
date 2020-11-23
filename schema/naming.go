package schema

import (
	"crypto/sha1"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/jinzhu/inflection"
)

// Namer namer interface
type Namer interface {
	TableName(table string) string
	ColumnName(table, column string) string
	JoinTableName(joinTable string) string
	RelationshipFKName(Relationship) string
	CheckerName(table, column string) string
	IndexName(table, column string) string
}

// NamingStrategy tables, columns naming strategy
type NamingStrategy struct {
	TablePrefix   string
	SingularTable bool
	NameReplacer  *strings.Replacer
}

// TableName convert string to table name
func (ns NamingStrategy) TableName(str string) string {
	if ns.SingularTable {
		return ns.TablePrefix + ns.toDBName(str)
	}
	return ns.TablePrefix + inflection.Plural(ns.toDBName(str))
}

// ColumnName convert string to column name
func (ns NamingStrategy) ColumnName(table, column string) string {
	return ns.toDBName(column)
}

// JoinTableName convert string to join table name
func (ns NamingStrategy) JoinTableName(str string) string {
	if strings.ToLower(str) == str {
		return ns.TablePrefix + str
	}

	if ns.SingularTable {
		return ns.TablePrefix + ns.toDBName(str)
	}
	return ns.TablePrefix + inflection.Plural(ns.toDBName(str))
}

// RelationshipFKName generate fk name for relation
func (ns NamingStrategy) RelationshipFKName(rel Relationship) string {
	return strings.Replace(fmt.Sprintf("fk_%s_%s", rel.Schema.Table, ns.toDBName(rel.Name)), ".", "_", -1)
}

// CheckerName generate checker name
func (ns NamingStrategy) CheckerName(table, column string) string {
	return strings.Replace(fmt.Sprintf("chk_%s_%s", table, column), ".", "_", -1)
}

// IndexName generate index name
func (ns NamingStrategy) IndexName(table, column string) string {
	idxName := fmt.Sprintf("idx_%v_%v", table, ns.toDBName(column))
	idxName = strings.Replace(idxName, ".", "_", -1)

	if utf8.RuneCountInString(idxName) > 64 {
		h := sha1.New()
		h.Write([]byte(idxName))
		bs := h.Sum(nil)

		idxName = fmt.Sprintf("idx%v%v", table, column)[0:56] + string(bs)[:8]
	}
	return idxName
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

func (ns NamingStrategy) toDBName(name string) string {
	if name == "" {
		return ""
	} else if v, ok := smap.Load(name); ok {
		return v.(string)
	}

	if ns.NameReplacer != nil {
		name = ns.NameReplacer.Replace(name)
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
	ret := buf.String()
	smap.Store(name, ret)
	return ret
}
