package schema

import (
	"strconv"
	"strings"
)

type Index struct {
	Name   string
	Class  string // UNIQUE | FULLTEXT | SPATIAL
	Fields []IndexOption
}

type IndexOption struct {
	*Field
	Expression string
	Sort       string // DESC, ASC
	Collate    string
	Length     int
	Type       string // btree, hash, gist, spgist, gin, and brin
	Where      string
	Comment    string
}

// ParseIndexes parse schema indexes
func (schema *Schema) ParseIndexes() map[string]Index {
	var indexes = map[string]Index{}

	for _, field := range schema.FieldsByDBName {
		if field.TagSettings["INDEX"] != "" || field.TagSettings["UNIQUE_INDEX"] != "" {
			for _, index := range parseFieldIndexes(field) {
				idx := indexes[index.Name]
				idx.Name = index.Name
				if idx.Class == "" {
					idx.Class = index.Class
				}
				idx.Fields = append(idx.Fields, index.Fields...)
				indexes[index.Name] = idx
			}
		}
	}

	return indexes
}

func parseFieldIndexes(field *Field) (indexes []Index) {
	for _, value := range strings.Split(field.Tag.Get("gorm"), ";") {
		if value != "" {
			v := strings.Split(value, ":")
			k := strings.TrimSpace(strings.ToUpper(v[0]))
			if k == "INDEX" || k == "UNIQUE_INDEX" {
				var (
					name     string
					tag      = strings.Join(v[1:], ":")
					settings = map[string]string{}
				)

				names := strings.Split(tag, ",")
				for i := 0; i < len(names); i++ {
					if len(names[i]) > 0 {
						j := i
						for {
							if names[j][len(names[j])-1] == '\\' {
								i++
								names[j] = names[j][0:len(names[j])-1] + names[i]
								names[i] = ""
							} else {
								break
							}
						}
					}

					if i == 0 {
						name = names[0]
					}

					values := strings.Split(names[i], ":")
					k := strings.TrimSpace(strings.ToUpper(values[0]))

					if len(values) >= 2 {
						settings[k] = strings.Join(values[1:], ":")
					} else if k != "" {
						settings[k] = k
					}
				}

				if name == "" {
					name = field.Schema.namer.IndexName(field.Schema.Table, field.Name)
				}

				length, _ := strconv.Atoi(settings["LENGTH"])

				if (k == "UNIQUE_INDEX") || settings["UNIQUE"] != "" {
					settings["CLASS"] = "UNIQUE"
				}

				indexes = append(indexes, Index{
					Name:  name,
					Class: settings["CLASS"],
					Fields: []IndexOption{{
						Field:      field,
						Expression: settings["EXPRESSION"],
						Sort:       settings["SORT"],
						Collate:    settings["COLLATE"],
						Type:       settings["TYPE"],
						Length:     length,
						Where:      settings["WHERE"],
						Comment:    settings["COMMENT"],
					}},
				})
			}
		}
	}

	return
}
