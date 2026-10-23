package schema

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Index describes a database index assembled from one or more model field tags. Fields
// is ordered by priority; SQL options are passed to dialect migration code and are
// trusted model metadata.
type Index struct {
	// Name identifies the index; fields using the same name contribute to a composite
	// index.
	Name string
	// Class is an index class such as UNIQUE, FULLTEXT, or SPATIAL, interpreted by the
	// dialector.
	Class string // UNIQUE | FULLTEXT | SPATIAL
	// Type requests an index access method such as btree or hash, where supported.
	Type string // btree, hash, gist, spgist, gin, and brin
	// Where is the raw predicate for a partial index; support and syntax depend on the
	// database.
	Where string
	// Comment is an optional database index comment.
	Comment string
	// Option holds additional trusted dialect-specific SQL, such as WITH PARSER
	// parser_name.
	Option string // WITH PARSER parser_name
	// Fields lists ordered indexed columns or expressions. A field can appear more than
	// once with different options.
	Fields []IndexOption // Note: IndexOption's Field maybe the same
}

// IndexOption describes one column or expression within an index. It embeds the source
// model Field and records ordering and dialect options.
type IndexOption struct {
	// Field is the model column supplying this index element and its tags.
	*Field
	// Expression replaces the indexed column with trusted SQL when nonempty, for example
	// ABS(age).
	Expression string
	// Sort requests the index element order, typically ASC or DESC.
	Sort string // DESC, ASC
	// Collate requests a database collation for this index element.
	Collate string
	// Length requests an indexed prefix length where supported; zero leaves it unspecified.
	Length int
	// Priority determines ascending element order within a composite index; parsing
	// defaults it to 10. Specify distinct priorities when order must be unambiguous.
	Priority int
}

// ParseIndexes combines index and uniqueIndex tags by name in field-discovery order and
// sorts each index's elements by priority. It also records single-column unique index
// names on Field.UniqueIndex for migration. Parsing failures are stored in schema.err
// and stop collection, so the returned slice can be partial; this method does not
// inspect the database.
func (schema *Schema) ParseIndexes() []*Index {
	indexesByName := map[string]*Index{}
	indexes := []*Index{}

	for _, field := range schema.Fields {
		if field.TagSettings["INDEX"] != "" || field.TagSettings["UNIQUEINDEX"] != "" {
			fieldIndexes, err := parseFieldIndexes(field)
			if err != nil {
				schema.err = err
				break
			}
			for _, index := range fieldIndexes {
				idx := indexesByName[index.Name]
				if idx == nil {
					idx = &Index{Name: index.Name}
					indexesByName[index.Name] = idx
					indexes = append(indexes, idx)
				}
				idx.Name = index.Name
				if idx.Class == "" {
					idx.Class = index.Class
				}
				if idx.Type == "" {
					idx.Type = index.Type
				}
				if idx.Where == "" {
					idx.Where = index.Where
				}
				if idx.Comment == "" {
					idx.Comment = index.Comment
				}
				if idx.Option == "" {
					idx.Option = index.Option
				}

				idx.Fields = append(idx.Fields, index.Fields...)
				sort.Slice(idx.Fields, func(i, j int) bool {
					return idx.Fields[i].Priority < idx.Fields[j].Priority
				})
			}
		}
	}
	for _, index := range indexes {
		if index.Class == "UNIQUE" && len(index.Fields) == 1 {
			index.Fields[0].Field.UniqueIndex = index.Name
		}
	}
	return indexes
}

// LookIndex reparses index metadata and returns the first index whose name or
// constituent Go field name matches name. It returns nil for a nil schema or no match.
// Field-name lookup is ambiguous when a field participates in multiple indexes; use the
// index name for an exact lookup.
func (schema *Schema) LookIndex(name string) *Index {
	if schema != nil {
		indexes := schema.ParseIndexes()
		for _, index := range indexes {
			if index.Name == name {
				return index
			}

			for _, field := range index.Fields {
				if field.Name == name {
					return index
				}
			}
		}
	}

	return nil
}

// parseFieldIndexes parses every index/uniqueIndex tag on a field, preserving repeated
// tags that TagSettings alone would collapse. It supplies generated names and default
// priority 10, handles composite IDs, and returns an error for an empty composite
// identifier.
func parseFieldIndexes(field *Field) (indexes []Index, err error) {
	for _, value := range strings.Split(field.Tag.Get("gorm"), ";") {
		if value != "" {
			v := strings.Split(value, ":")
			k := strings.TrimSpace(strings.ToUpper(v[0]))
			if k == "INDEX" || k == "UNIQUEINDEX" {
				var (
					name       string
					tag        = strings.Join(v[1:], ":")
					idx        = strings.IndexByte(tag, ',')
					tagSetting = strings.Join(strings.Split(tag, ",")[1:], ",")
					settings   = ParseTagSetting(tagSetting, ",")
					length, _  = strconv.Atoi(settings["LENGTH"])
				)

				if idx == -1 {
					idx = len(tag)
				}

				name = tag[0:idx]
				if name == "" {
					subName := field.Name
					const key = "COMPOSITE"
					if composite, found := settings[key]; found {
						if len(composite) == 0 || composite == key {
							err = fmt.Errorf(
								"the composite tag of %s.%s cannot be empty",
								field.Schema.Name,
								field.Name)
							return
						}
						subName = composite
					}
					name = field.Schema.namer.IndexName(
						field.Schema.Table, subName)
				}

				if (k == "UNIQUEINDEX") || settings["UNIQUE"] != "" {
					settings["CLASS"] = "UNIQUE"
				}

				priority, err := strconv.Atoi(settings["PRIORITY"])
				if err != nil {
					priority = 10
				}

				indexes = append(indexes, Index{
					Name:    name,
					Class:   settings["CLASS"],
					Type:    settings["TYPE"],
					Where:   settings["WHERE"],
					Comment: settings["COMMENT"],
					Option:  settings["OPTION"],
					Fields: []IndexOption{{
						Field:      field,
						Expression: settings["EXPRESSION"],
						Sort:       settings["SORT"],
						Collate:    settings["COLLATE"],
						Length:     length,
						Priority:   priority,
					}},
				})
			}
		}
	}

	err = nil
	return
}
