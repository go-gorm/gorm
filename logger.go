package gorm

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"time"
)

type Logger interface {
	Print(v ...interface{})
}

func (s *Chain) print(level string, v ...interface{}) {
	if s.d.log_mode || s.debug_mode || level == "debug" {
		if _, ok := s.d.logger.(Logger); !ok {
			fmt.Println("logger haven't been set, using os.Stdout")
			s.d.logger = log.New(os.Stdout, "", 0)
		}
		args := []interface{}{level}
		s.d.logger.(Logger).Print(append(args, v...))
	}
}

func (s *Chain) warn(v ...interface{}) {
	go s.print("warn", v...)
}

func (s *Chain) slog(sql string, t time.Time, vars ...interface{}) {
	go s.print("sql", time.Now().Sub(t), fmt.Sprintf(regexp.MustCompile(`\$\d|\?`).ReplaceAllString(sql, "'%v'"), vars...))
}

func (s *Chain) debug(v ...interface{}) {
	go s.print("debug", v...)
}
