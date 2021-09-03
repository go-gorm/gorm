package gorm

import (
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type DeletedFlag uint

// // Scan implements the Scanner interface.
// func (n *DeletedFlag) Scan(value interface{}) error {
// 	return (*sql.NullTime)(n).Scan(value)
// }

// // Value implements the driver Valuer interface.
// func (n DeletedFlag) Value() (driver.Value, error) {
// 	if !n.Valid {
// 		return nil, nil
// 	}
// 	return n.Time, nil
// }

// func (n DeletedFlag) MarshalJSON() ([]byte, error) {
// 	if n.Valid {
// 		return json.Marshal(n.Time)
// 	}
// 	return json.Marshal(nil)
// }

// func (n *DeletedFlag) UnmarshalJSON(b []byte) error {
// 	if string(b) == "null" {
// 		n.Valid = false
// 		return nil
// 	}
// 	err := json.Unmarshal(b, &n.Time)
// 	if err == nil {
// 		n.Valid = true
// 	}
// 	return err
// }

func (DeletedFlag) QueryClauses(f *schema.Field) []clause.Interface {
	return []clause.Interface{SoftDeleteQueryClause{Field: f}}
}

func (DeletedFlag) DeleteClauses(f *schema.Field) []clause.Interface {
	return []clause.Interface{SoftDeleteUniqueDeleteClause{Field: f}}
}

type SoftDeleteUniqueDeleteClause struct {
	Field *schema.Field
}

func (sd SoftDeleteUniqueDeleteClause) Name() string {
	return ""
}

func (sd SoftDeleteUniqueDeleteClause) Build(clause.Builder) {
}

func (sd SoftDeleteUniqueDeleteClause) MergeClause(*clause.Clause) {
}

func (sd SoftDeleteUniqueDeleteClause) ModifyStatement(stmt *Statement) {
	re := regexp.MustCompile(`UPDATE (.*) WHERE `)
	if sql := stmt.SQL.String(); sql != "" {
		setClause := re.FindStringSubmatch(sql)[1]
		if setClause == "" {
			return
		}

		newSetClause := fmt.Sprintf("%s, %s = `%s`.`id`", setClause, sd.Field.DBName, stmt.Table)
		stmt.SQL.Reset()
		stmt.SQL.WriteString(strings.Replace(sql, setClause, newSetClause, 1))
	}
}
