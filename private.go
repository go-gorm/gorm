package gorm

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"
)

func (s *DB) clone() *DB {
	db := DB{db: s.db, parent: s.parent, logMode: s.logMode, Value: s.Value, Error: s.Error}

	if s.search == nil {
		db.search = &search{}
	} else {
		db.search = s.search.clone()
	}

	db.search.db = &db
	return &db
}

func (s *DB) new() *DB {
	s.search = nil
	return s.clone()
}

func (s *DB) do(data interface{}) *Do {
	s.Value = data
	do := Do{db: s}
	do.setModel(data)
	return &do
}

func (s *DB) err(err error) error {
	if err != nil {
		if s.logMode == 0 {
			if err != RecordNotFound {
				go s.print(fileWithLineNum(), err)
				if regexp.MustCompile(`^sql: Scan error on column index`).MatchString(err.Error()) {
					return nil
				}
			}
		} else {
			s.log(err)
		}
		s.Error = err
	}
	return err
}

func (s *DB) hasError() bool {
	return s.Error != nil
}

func fileWithLineNum() string {
	for i := 5; i < 15; i++ {
		_, file, line, ok := runtime.Caller(i)
		if ok && (!regexp.MustCompile(`jinzhu/gorm/.*.go`).MatchString(file) || regexp.MustCompile(`jinzhu/gorm/.*test.go`).MatchString(file)) {
			return fmt.Sprintf("%v:%v", strings.TrimPrefix(file, os.Getenv("GOPATH")+"src/"), line)
		}
	}
	return ""
}

func (s *DB) print(v ...interface{}) {
	go s.parent.logger.(logger).Print(v...)
}

func (s *DB) log(v ...interface{}) {
	if s.logMode == 2 {
		s.print(append([]interface{}{"log", fileWithLineNum()}, v...)...)
	}
}

func (s *DB) slog(sql string, t time.Time, vars ...interface{}) {
	if s.logMode == 2 {
		s.print("sql", fileWithLineNum(), time.Now().Sub(t), sql, vars)
	}
}
