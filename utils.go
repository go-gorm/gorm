package gorm

import (
	"bytes"
	"strings"
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

var smap = map[string]string{}

func ToDBName(name string) string {
	if v, ok := smap[name]; ok {
		return v
	}

	value := commonInitialismsReplacer.Replace(name)
	buf := bytes.NewBufferString("")
	for i, v := range value {
		if i > 0 && v >= 'A' && v <= 'Z' {
			buf.WriteRune('_')
		}
		buf.WriteRune(v)
	}

	s := strings.ToLower(buf.String())
	smap[name] = s
	return s
}

type expr struct {
	expr string
	args []interface{}
}

func Expr(expression string, args ...interface{}) *expr {
	return &expr{expr: expression, args: args}
}
