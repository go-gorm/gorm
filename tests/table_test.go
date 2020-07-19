package tests_test

import (
	"regexp"
	"testing"

	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

type UserWithTable struct {
	gorm.Model
	Name string
}

func (UserWithTable) TableName() string {
	return "gorm.user"
}

func TestTable(t *testing.T) {
	dryDB := DB.Session(&gorm.Session{DryRun: true})

	r := dryDB.Table("`user`").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT \\* FROM `user`").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("user as u").Select("name").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT .name. FROM user as u WHERE .u.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("gorm.user").Select("name").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT .name. FROM .gorm.\\..user. WHERE .user.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Select("name").Find(&UserWithTable{}).Statement
	if !regexp.MustCompile("SELECT .name. FROM .gorm.\\..user. WHERE .user.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("(?) as u", DB.Model(&User{}).Select("name")).Find(&User{}).Statement
	if !regexp.MustCompile("SELECT \\* FROM \\(SELECT .name. FROM .users. WHERE .users.\\..deleted_at. IS NULL\\) as u WHERE .u.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("(?) as u, (?) as p", DB.Model(&User{}).Select("name"), DB.Model(&Pet{}).Select("name")).Find(&User{}).Statement
	if !regexp.MustCompile("SELECT \\* FROM \\(SELECT .name. FROM .users. WHERE .users.\\..deleted_at. IS NULL\\) as u, \\(SELECT .name. FROM .pets. WHERE .pets.\\..deleted_at. IS NULL\\) as p WHERE .u.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}
}
