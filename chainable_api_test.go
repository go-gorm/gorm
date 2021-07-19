package gorm_test

import (
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/utils/tests"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestDB_Raw_Limit(t *testing.T) {
	var (
		t1 *gorm.DB
		t2 *gorm.DB
		t3 *gorm.DB
	)
	tx, _ := gorm.Open(
		tests.DummyDialector{},
		//&gorm.Config{Logger: logger.Default.LogMode(logger.Info)},
	)
	t1 = tx.Table("pod_events").Limit(1)
	t2 = tx.Raw("SELECT * FROM `pod_events`").Limit(1)
	t3 = tx.Raw("SELECT * FROM `pod_events` LIMIT 1").Limit(1)
	t1.Statement.BuildClauses = []string{"SELECT", "FROM", "WHERE", "GROUP BY", "ORDER BY", "LIMIT", "FOR"}
	callbacks.BuildQuerySQL(t1)
	callbacks.BuildQuerySQL(t2)
	callbacks.BuildQuerySQL(t3)
	s1 := t1.Statement.SQL.String()
	s2 := t2.Statement.SQL.String()
	s3 := t3.Statement.SQL.String()
	if !strings.EqualFold(s1, s2) || !strings.EqualFold(s1, s3) {
		t.Errorf("s1 != s2 != s3\ns1 = %v\ns2 = %v\ns3 = %v\n", s1, s2, s3)
	}
}
