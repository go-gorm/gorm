package schema

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Index struct {
	Name    string
	Class   string // UNIQUE | FULLTEXT | SPATIAL
	Type    string // btree, hash, gist, spgist, gin, and brin
	Where   string
	Comment string
	Option  string        // WITH PARSER parser_name
	Fields  []IndexOption // Note: IndexOption's Field maybe the same
}

type IndexOption struct {
	*Field
	Expression string
	Sort       string // DESC, ASC
	Collate    string
	Length     int
	Priority   int
}

// ParseIndexes parse schema indexes
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
