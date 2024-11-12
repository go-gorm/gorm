package clause_test

import (
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Test_ExprCase(t *testing.T) {
	type exampleUser struct {
		ID   string
		Name string
	}

	inputUsers := []*exampleUser{
		{
			ID:   "user-001",
			Name: "user-name-001",
		},
		{
			ID:   "user-002",
			Name: "user-name-002",
		},
	}

	userIDs := make([]string, len(inputUsers))
	userNameCases := make([]*clause.ExprCaseCondition, len(inputUsers))
	for idx, user := range inputUsers {
		userIDs[idx] = user.ID
		userNameCases[idx] = &clause.ExprCaseCondition{
			When: "user_id=?",
			Then: "?",
			Vars: []any{
				user.ID,
				user.Name,
			},
		}
	}

	sqlQuery := db.ToSQL(func(db *gorm.DB) *gorm.DB {
		return db.
			Table("users").
			Where("user_id IN (?)", userIDs).
			UpdateColumns(map[string]any{
				"user_name": clause.ExprCase{
					Cases: userNameCases,
					Else: &clause.ExprCaseElse{
						Then: "user_name",
						Vars: nil,
					},
				},
			})
	})

	expectedSQLQuery := "UPDATE `users` SET `user_name`=CASE WHEN user_id=\"user-001\" THEN \"user-name-001\" WHEN user_id=\"user-002\" THEN \"user-name-002\" ELSE user_name END WHERE user_id IN (\"user-001\",\"user-002\")"
	if sqlQuery != expectedSQLQuery {
		t.Errorf("SQLQuery is mismatch actual: %v expected:%v\n", sqlQuery, expectedSQLQuery)
	}
}
