package gorm_test

import (
	"context"
	"strings"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type joinTestDialector struct{}

func (joinTestDialector) Name() string { return "join-test" }
func (joinTestDialector) Initialize(db *gorm.DB) error {
	callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{})
	return nil
}
func (joinTestDialector) Migrator(*gorm.DB) gorm.Migrator { return nil }
func (joinTestDialector) DataTypeOf(*schema.Field) string { return "" }
func (joinTestDialector) DefaultValueOf(*schema.Field) clause.Expression {
	return clause.Expr{}
}
func (joinTestDialector) BindVarTo(writer clause.Writer, _ *gorm.Statement, _ interface{}) {
	writer.WriteByte('?')
}
func (joinTestDialector) QuoteTo(writer clause.Writer, value string) {
	writer.WriteString(value)
}
func (joinTestDialector) Explain(sql string, _ ...interface{}) string { return sql }

type joinTestCompany struct {
	ID int
}

type joinTestUser struct {
	ID        int
	CompanyID int
	Company   joinTestCompany
	ManagerID *int
	Manager   *joinTestUser
}

func TestGenericsJoinPhysicalTableOverride(t *testing.T) {
	db, err := gorm.Open(joinTestDialector{}, &gorm.Config{
		DisableAutomaticPing: true,
		DryRun:               true,
	})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	tests := []struct {
		name           string
		target         clause.JoinTarget
		wantJoinTable  string
		wantSQL        []string
		doNotWantSQL   []string
		wantTableCount int
	}{
		{
			name:          "default association table",
			target:        clause.InnerJoin.Association("Company"),
			wantJoinTable: "Company",
			wantSQL:       []string{"INNER JOIN join_test_companies Company"},
		},
		{
			name: "physical table override",
			target: clause.JoinTarget{
				Type:        clause.InnerJoin,
				Association: "Company",
				From:        clause.Table{Name: "tenant_1.companies"},
			},
			wantJoinTable: "Company",
			wantSQL:       []string{"INNER JOIN tenant_1.companies Company"},
		},
		{
			name: "association alias overrides From alias",
			target: func() clause.JoinTarget {
				target := clause.InnerJoin.Association("Company").As("Employer")
				target.From = clause.Table{Name: "tenant_1.companies", Alias: "ignored"}
				return target
			}(),
			wantJoinTable: "Employer",
			wantSQL:       []string{"INNER JOIN tenant_1.companies Employer"},
			doNotWantSQL:  []string{"ignored"},
		},
		{
			name: "nested override applies to final association",
			target: clause.JoinTarget{
				Type:        clause.InnerJoin,
				Association: "Manager.Company",
				From:        clause.Table{Name: "tenant_1.companies"},
			},
			wantJoinTable:  "Manager__Company",
			wantSQL:        []string{"INNER JOIN join_test_users Manager", "INNER JOIN tenant_1.companies Manager__Company"},
			wantTableCount: 1,
		},
		{
			name: "subquery takes precedence",
			target: func() clause.JoinTarget {
				target := clause.InnerJoin.AssociationFrom("Company", clause.Expr{SQL: "SELECT * FROM company_view"})
				target.From = clause.Table{Name: "tenant_1.companies"}
				return target
			}(),
			wantJoinTable: "Company",
			wantSQL:       []string{"INNER JOIN (SELECT * FROM company_view) AS Company"},
			doNotWantSQL:  []string{"tenant_1.companies"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			joinTableName := ""
			var findErr error
			sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
				_, findErr = gorm.G[joinTestUser](tx).Joins(tt.target, func(_ gorm.JoinBuilder, joinTable clause.Table, _ clause.Table) error {
					joinTableName = joinTable.Name
					return nil
				}).Find(context.Background())
				tx.AddError(findErr)
				return tx
			})

			if findErr != nil {
				t.Fatalf("build join query: %v", findErr)
			}
			if joinTableName != tt.wantJoinTable {
				t.Fatalf("join builder table = %q, want %q", joinTableName, tt.wantJoinTable)
			}
			for _, want := range tt.wantSQL {
				if !strings.Contains(sql, want) {
					t.Errorf("join SQL %q does not contain %q", sql, want)
				}
			}
			for _, unwanted := range tt.doNotWantSQL {
				if strings.Contains(sql, unwanted) {
					t.Errorf("join SQL %q unexpectedly contains %q", sql, unwanted)
				}
			}
			if tt.wantTableCount > 0 && strings.Count(sql, tt.target.From.Name) != tt.wantTableCount {
				t.Errorf("join SQL %q contains physical table %q %d times, want %d", sql, tt.target.From.Name, strings.Count(sql, tt.target.From.Name), tt.wantTableCount)
			}
		})
	}
}
