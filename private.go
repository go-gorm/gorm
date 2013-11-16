package gorm

import (
	"fmt"
	"time"
)

func (s *DB) clone() *DB {
	db := &DB{db: s.db, parent: s.parent, search: s.parent.search.clone()}
	db.search.db = db
	return db
}

func (s *DB) do(data interface{}) *Do {
	s.data = data
	return &Do{db: s}
}

func (s *DB) err(err error) error {
	if err != nil {
		s.Error = err
		s.warn(err)
	}
	return err
}

func (s *DB) hasError() bool {
	return s.Error != nil
}

func (s *DB) print(level string, v ...interface{}) {
	if s.d.logMode || s.debug_mode || level == "debug" {
		if _, ok := s.d.logger.(Logger); !ok {
			fmt.Println("logger haven't been set, using os.Stdout")
			s.d.logger = default_logger
		}
		args := []interface{}{level}
		s.d.logger.(Logger).Print(append(args, v...)...)
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
