package sqlite

import (
	"bytes"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/builder"
	"github.com/jinzhu/gorm/dialects/common/utils"

	// import sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

// Dialect Sqlite3 Dialect for GORM
type Dialect struct {
}

// Insert insert
func (*Dialect) Insert(stmt *builder.Statement) error {
	var (
		args                []interface{}
		defaultValueColumns []string
		errs                = gorm.Errors{}
		assignmentsChan     = utils.GetCreatingAssignments(stmt, &errs)
		tableNameChan       = utils.GetTable(stmt, &errs)
	)

	s := bytes.NewBufferString("INSERT INTO ")
	s.WriteString(<-tableNameChan)

	if assignments := <-assignmentsChan; len(assignments) > 0 {
		for column, value := range assignments {
			args = append(args, value...)
		}
	} else {
		// assign default value
	}

	return nil
}

// Query query
func (*Dialect) Query(*builder.Statement) error {
	return nil
}

// Update update
func (*Dialect) Update(*builder.Statement) error {
	return nil
}

// Delete delete
func (*Dialect) Delete(*builder.Statement) error {
	return nil
}
