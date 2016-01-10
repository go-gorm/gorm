package gorm

import (
	"bytes"
	"strings"
	"sync"
)

// Copied from golint
var commonInitialisms = []string{"API", "ASCII", "CPU", "CSS", "DNS", "EOF", "GUID", "HTML", "HTTP", "HTTPS", "ID", "IP", "JSON", "LHS", "QPS", "RAM", "RHS", "RPC", "SLA", "SMTP", "SSH", "TLS", "TTL", "UI", "UID", "UUID", "URI", "URL", "UTF8", "VM", "XML", "XSRF", "XSS"}
var commonInitialismsReplacer *strings.Replacer

func init() {
	var commonInitialismsForReplacer []string
	for _, initialism := range commonInitialisms {
		commonInitialismsForReplacer = append(commonInitialismsForReplacer, initialism, strings.Title(strings.ToLower(initialism)))
	}
	commonInitialismsReplacer = strings.NewReplacer(commonInitialismsForReplacer...)
}

type safeMap struct {
	m map[string]string
	l *sync.RWMutex
}

func (s *safeMap) Set(key string, value string) {
	s.l.Lock()
	defer s.l.Unlock()
	s.m[key] = value
}

func (s *safeMap) Get(key string) string {
	s.l.RLock()
	defer s.l.RUnlock()
	return s.m[key]
}

func newSafeMap() *safeMap {
	return &safeMap{l: new(sync.RWMutex), m: make(map[string]string)}
}

var smap = newSafeMap()

type Case bool

const (
	lower Case = false
	upper Case = true
)

func ToDBName(name string) string {
	if v := smap.Get(name); v != "" {
		return v
	}

	var (
		value                        = commonInitialismsReplacer.Replace(name)
		buf                          = bytes.NewBufferString("")
		lastCase, currCase, nextCase Case
	)

	for i, v := range value[:len(value)-1] {
		nextCase = value[i+1] >= 'A' && value[i+1] <= 'Z'
		if i > 0 {
			if currCase == upper {
				if lastCase == upper && nextCase == upper {
					buf.WriteRune(v)
				} else {
					buf.WriteRune('_')
					buf.WriteRune(v)
				}
			} else {
				buf.WriteRune(v)
			}
		} else {
			currCase = upper
			buf.WriteRune(v)
		}
		lastCase = currCase
		currCase = nextCase
	}

	buf.WriteByte(value[len(value)-1])

	s := strings.Replace(strings.ToLower(buf.String()), "__", "_", -1)

	smap.Set(name, s)
	return s
}

type expr struct {
	expr string
	args []interface{}
}

func Expr(expression string, args ...interface{}) *expr {
	return &expr{expr: expression, args: args}
}
