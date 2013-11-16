package gorm

import (
	"fmt"
	"time"
)

func (s *DB) clone() *DB {
	db := &DB{db: s.db, parent: s.parent}
	if s.parent.search == nil {
		db.search = &search{}
	} else {
		db.search = s.parent.search.clone()
	}
	db.search.db = db
	return db
}

func (s *DB) do(data interface{}) *Do {
	s.data = data
	do := Do{db: s}
	do.setModel(data)
	return &do
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
	if s.logMode || level == "debug" {
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
