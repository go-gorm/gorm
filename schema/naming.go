package schema

import (
	"crypto/sha1"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/jinzhu/inflection"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Namer translates Go model and field names into table, column, index, and constraint
// identifiers. Schema parsing caches its results, so use stable rules and a separate
// cache for incompatible strategies. Identifier quoting remains the dialector's
// responsibility.
type Namer interface {
	// TableName returns the database table identifier for a Go model type name.
	TableName(table string) string
	// SchemaName derives a Go-style model name from a table identifier; it need not exactly
	// invert TableName.
	SchemaName(table string) string
	// ColumnName returns the database column identifier for a field in the given table.
	ColumnName(table, column string) string
	// JoinTableName returns the database identifier for a many-to-many join table.
	JoinTableName(joinTable string) string
	// RelationshipFKName returns the foreign-key constraint identifier for an association.
	RelationshipFKName(Relationship) string
	// CheckerName returns the default check-constraint identifier for a table and column.
	CheckerName(table, column string) string
	// IndexName returns the default index identifier for a table and field or composite ID.
	IndexName(table, column string) string
	// UniqueName returns the default unique-constraint identifier for a table and column.
	UniqueName(table, column string) string
}

// Replacer transforms a name before NamingStrategy applies its case and underscore
// rules. strings.Replacer implements this interface.
type Replacer interface {
	// Replace returns the replacement name. NamingStrategy treats an empty replacement as a
	// request to retain the original name without further case conversion.
	Replace(name string) string
}

// This compile-time assertion keeps NamingStrategy compatible with Namer.
var _ Namer = (*NamingStrategy)(nil)

// NamingStrategy implements GORM's default naming conventions: snake_case columns,
// plural table names, and generated constraint/index names. Configure it before parsing
// models, usually through gorm.Config.NamingStrategy; cached schemas do not recompute
// names after configuration changes.
//
// Its zero value uses plural tables, no prefix, standard initialism normalization, and
// a 64-character threshold for generated constraint/index names.
type NamingStrategy struct {
	// TablePrefix is prepended to model and join-table names, for example tenant_. It is
	// not automatically applied to explicit Tabler names.
	TablePrefix string
	// SingularTable disables pluralization of converted model and join-table names.
	SingularTable bool
	// NameReplacer optionally rewrites input names before case conversion, for example to
	// preserve a domain abbreviation.
	NameReplacer Replacer
	// NoLowerCase skips snake_case and lowercase conversion after NameReplacer. Table
	// pluralization still applies unless SingularTable is set.
	NoLowerCase bool
	// IdentifierMaxLength controls shortening of generated constraint/index names,
	// defaulting to 64 when zero. Positive limits must leave room for the eight-byte hash
	// suffix; this does not limit ordinary table or column names.
	IdentifierMaxLength int
}

// TableName applies name conversion, optional pluralization, and TablePrefix to a Go
// model name.
func (ns NamingStrategy) TableName(str string) string {
	if ns.SingularTable {
		return ns.TablePrefix + ns.toDBName(str)
	}
	return ns.TablePrefix + inflection.Plural(ns.toDBName(str))
}

// SchemaName removes TablePrefix, singularizes unless SingularTable is enabled, and
// converts underscore-separated names to Go-style capitalization. It is not guaranteed
// to reverse TableName.
func (ns NamingStrategy) SchemaName(table string) string {
	table = strings.TrimPrefix(table, ns.TablePrefix)

	if ns.SingularTable {
		return ns.toSchemaName(table)
	}
	return ns.toSchemaName(inflection.Singular(table))
}

// ColumnName applies the name replacer and case conversion to column. This default
// strategy ignores table; custom Namers can use it.
func (ns NamingStrategy) ColumnName(table, column string) string {
	return ns.toDBName(column)
}

// JoinTableName preserves an already lowercase name when NoLowerCase is false, adding
// only TablePrefix. Other names follow conversion and optional pluralization; this
// allows tags such as many2many:user_languages to remain literal.
func (ns NamingStrategy) JoinTableName(str string) string {
	if !ns.NoLowerCase && strings.ToLower(str) == str {
		return ns.TablePrefix + str
	}

	if ns.SingularTable {
		return ns.TablePrefix + ns.toDBName(str)
	}
	return ns.TablePrefix + inflection.Plural(ns.toDBName(str))
}

// RelationshipFKName builds an fk-prefixed identifier from the owner table and
// converted relationship name, shortening it through formatName if needed.
func (ns NamingStrategy) RelationshipFKName(rel Relationship) string {
	return ns.formatName("fk", rel.Schema.Table, ns.toDBName(rel.Name))
}

// CheckerName builds a chk-prefixed identifier from table and column, replacing dots
// and applying generated-name shortening.
func (ns NamingStrategy) CheckerName(table, column string) string {
	return ns.formatName("chk", table, column)
}

// IndexName builds an idx-prefixed identifier from table and the converted column or
// composite ID, applying generated-name shortening.
func (ns NamingStrategy) IndexName(table, column string) string {
	return ns.formatName("idx", table, ns.toDBName(column))
}

// UniqueName builds a uni-prefixed identifier from table and converted column, applying
// generated-name shortening.
func (ns NamingStrategy) UniqueName(table, column string) string {
	return ns.formatName("uni", table, ns.toDBName(column))
}

// formatName joins prefix, table, and name with underscores, replaces dots, and
// shortens names exceeding the rune-count limit with an eight-hex-digit SHA-1 suffix.
// The retained prefix is sliced by bytes, so the limit is intended for ordinary ASCII
// identifiers and is not Unicode-safe truncation.
func (ns NamingStrategy) formatName(prefix, table, name string) string {
	formattedName := strings.ReplaceAll(strings.Join([]string{
		prefix, table, name,
	}, "_"), ".", "_")

	if ns.IdentifierMaxLength == 0 {
		ns.IdentifierMaxLength = 64
	}

	if utf8.RuneCountInString(formattedName) > ns.IdentifierMaxLength {
		h := sha1.New()
		h.Write([]byte(formattedName))
		bs := h.Sum(nil)

		formattedName = formattedName[0:ns.IdentifierMaxLength-8] + hex.EncodeToString(bs)[:8]
	}
	return formattedName
}

var (
	// commonInitialisms lists conventional Go abbreviations normalized during name
	// conversion, such as ID, HTTP, and URL.
	commonInitialisms = []string{"API", "ASCII", "CPU", "CSS", "DNS", "EOF", "GUID", "HTML", "HTTP", "HTTPS", "ID", "IP", "JSON", "LHS", "QPS", "RAM", "RHS", "RPC", "SLA", "SMTP", "SSH", "TLS", "TTL", "UID", "UI", "UUID", "URI", "URL", "UTF8", "VM", "XML", "XSRF", "XSS"}
	// commonInitialismsReplacer converts recognized uppercase abbreviations to title case
	// before snake_case splitting.
	commonInitialismsReplacer *strings.Replacer
)

// init builds the initialism replacer once before any NamingStrategy is used.
func init() {
	commonInitialismsForReplacer := make([]string, 0, len(commonInitialisms))
	for _, initialism := range commonInitialisms {
		commonInitialismsForReplacer = append(commonInitialismsForReplacer, initialism, cases.Title(language.Und).String(initialism))
	}
	commonInitialismsReplacer = strings.NewReplacer(commonInitialismsForReplacer...)
}

// toDBName applies optional replacement and ASCII case-boundary conversion to
// snake_case, handling standard Go initialisms. Empty input remains empty; an empty
// replacement preserves the original input.
func (ns NamingStrategy) toDBName(name string) string {
	if name == "" {
		return ""
	}

	if ns.NameReplacer != nil {
		tmpName := ns.NameReplacer.Replace(name)

		if tmpName == "" {
			return name
		}

		name = tmpName
	}

	if ns.NoLowerCase {
		return name
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
	return ret
}

// toSchemaName converts underscores to word boundaries, title-cases them, removes
// spaces, and restores recognized Go initialisms.
func (ns NamingStrategy) toSchemaName(name string) string {
	result := strings.ReplaceAll(cases.Title(language.Und, cases.NoLower).String(strings.ReplaceAll(name, "_", " ")), " ", "")
	for _, initialism := range commonInitialisms {
		result = regexp.MustCompile(cases.Title(language.Und, cases.NoLower).String(strings.ToLower(initialism))+"([A-Z]|$|_)").ReplaceAllString(result, initialism+"$1")
	}
	return result
}
