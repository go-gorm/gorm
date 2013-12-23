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
	db := DB{db: s.db, parent: s.parent, logMode: s.logMode, data: s.data, Error: s.Error}

	if s.search == nil {
		db.search = &search{}
	} else {
		db.search = s.search.clone()
	}

	db.search.db = &db
	return &db
}

func (s *DB) new() *DB {
	db := DB{db: s.db, parent: s.parent, logMode: s.logMode, data: s.data, Error: s.Error, search: &search{}}
	db.search.db = &db
	return &db
}

func (s *DB) do(data interface{}) *Do {
	s.data = data
	do := Do{db: s}
	do.setModel(data)
	return &do
}

func (s *DB) fileWithLineNum() string {
	for i := 5; i < 15; i++ {
		_, file, line, ok := runtime.Caller(i)
		if ok && (!regexp.MustCompile(`jinzhu/gorm/.*.go`).MatchString(file) || regexp.MustCompile(`jinzhu/gorm/.*test.go`).MatchString(file)) {
			return fmt.Sprintf("%v:%v", strings.TrimPrefix(file, os.Getenv("GOPATH")+"src/"), line)
		}
	}
	return ""
}

func (s *DB) err(err error) error {
	if err != nil {
		s.Error = err
		if s.logMode == 0 {
			if err != RecordNotFound {
				go fmt.Println(s.fileWithLineNum(), err)
			}
		} else {
			s.warn(err)
		}
	}
	return err
}

func (s *DB) hasError() bool {
	return s.Error != nil
}

func (s *DB) print(level string, v ...interface{}) {
	if s.logMode == 2 || level == "debug" {
		if _, ok := s.parent.logger.(Logger); !ok {
			fmt.Println("logger haven't been set, using os.Stdout")
			s.parent.logger = default_logger
		}
		args := []interface{}{level}
		s.parent.logger.(Logger).Print(append(args, v...)...)
	}
}

func (s *DB) warn(v ...interface{}) {
	go s.print("warn", v...)
}

func (s *DB) info(v ...interface{}) {
	go s.print("info", v...)
}

func (s *DB) slog(sql string, t time.Time, vars ...interface{}) {
	go s.print("sql", time.Now().Sub(t), sql, vars)
}

func (s *DB) debug(v ...interface{}) {
	go s.print("debug", v...)
}
