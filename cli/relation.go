package cli

import (
	"fmt"
	"os"
	"strings"
)

type RelationType string

const (
	One2Many  RelationType = "one2many"
	Many2Many RelationType = "many2many"
)

type RelationInfo struct {
	FieldName string
	Target    string
	Type      RelationType
}

// AddRelation menambahkan relasi ke file model
func AddRelation(modelFile string, relations []RelationInfo, modelName string) error {
	content, err := os.ReadFile(modelFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "}" {
			for _, r := range relations {
				var relLine string
				if r.Type == One2Many {
					relLine = fmt.Sprintf("\t%s []%s `gorm:\"foreignKey:%sID\"`", r.FieldName, r.Target, modelName)
				} else if r.Type == Many2Many {
					relLine = fmt.Sprintf("\t%s []%s `gorm:\"many2many:%s_%s\"`", r.FieldName, r.Target, strings.ToLower(modelName), strings.ToLower(r.Target))
				}
				lines = append(lines[:i], append([]string{relLine}, lines[i:]...)...)
				i++
			}
			break
		}
	}

	return os.WriteFile(modelFile, []byte(strings.Join(lines, "\n")), 0644)
}
