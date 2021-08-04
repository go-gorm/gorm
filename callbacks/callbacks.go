package callbacks

import (
	"gorm.io/gorm"
)

var (
	createClauses = []string{"INSERT", "VALUES", "ON CONFLICT"}
	queryClauses  = []string{"SELECT", "FROM", "WHERE", "GROUP BY", "ORDER BY", "LIMIT", "FOR"}
	updateClauses = []string{"UPDATE", "SET", "WHERE"}
	deleteClauses = []string{"DELETE", "FROM", "WHERE"}
)

type Config struct {
	LastInsertIDReversed bool
	WithReturning        bool
	CreateClauses        []string
	QueryClauses         []string
	UpdateClauses        []string
	DeleteClauses        []string
}

func RegisterDefaultCallbacks(db *gorm.DB, config *Config) {
	enableTransaction := func(db *gorm.DB) bool {
		return !db.SkipDefaultTransaction
	}

	createCallback := db.Callback().Create()
	createCallback.Match(enableTransaction).Register(gorm.BeforeTransactionCk, BeginTransaction)
	createCallback.Register(gorm.BeforeCreateCk, BeforeCreate)
	createCallback.Register(gorm.SaveBeforeAssociationsCk, SaveBeforeAssociations(true))
	createCallback.Register(gorm.CreateCk, Create(config))
	createCallback.Register(gorm.SaveAfterAssociationsCk, SaveAfterAssociations(true))
	createCallback.Register(gorm.AfterCreateCk, AfterCreate)
	createCallback.Match(enableTransaction).Register(gorm.CommitOrRollbackCk, CommitOrRollbackTransaction)
	if len(config.CreateClauses) == 0 {
		config.CreateClauses = createClauses
	}
	createCallback.Clauses = config.CreateClauses

	queryCallback := db.Callback().Query()
	queryCallback.Register(gorm.QueryCk, Query)
	queryCallback.Register(gorm.PreloadCk, Preload)
	queryCallback.Register(gorm.AfterQueryCk, AfterQuery)
	if len(config.QueryClauses) == 0 {
		config.QueryClauses = queryClauses
	}
	queryCallback.Clauses = config.QueryClauses

	deleteCallback := db.Callback().Delete()
	deleteCallback.Match(enableTransaction).Register(gorm.BeforeTransactionCk, BeginTransaction)
	deleteCallback.Register(gorm.BeforeDeleteCk, BeforeDelete)
	deleteCallback.Register(gorm.DeleteBeforeAssociationsCk, DeleteBeforeAssociations)
	deleteCallback.Register(gorm.DeleteCk, Delete)
	deleteCallback.Register(gorm.AfterDeleteCk, AfterDelete)
	deleteCallback.Match(enableTransaction).Register(gorm.CommitOrRollbackCk, CommitOrRollbackTransaction)
	if len(config.DeleteClauses) == 0 {
		config.DeleteClauses = deleteClauses
	}
	deleteCallback.Clauses = config.DeleteClauses

	updateCallback := db.Callback().Update()
	updateCallback.Match(enableTransaction).Register(gorm.BeforeTransactionCk, BeginTransaction)
	updateCallback.Register(gorm.SetUpReflectValueCk, SetupUpdateReflectValue)
	updateCallback.Register(gorm.BeforeUpdateCk, BeforeUpdate)
	updateCallback.Register(gorm.SaveBeforeAssociationsCk, SaveBeforeAssociations(false))
	updateCallback.Register(gorm.UpdateCk, Update)
	updateCallback.Register(gorm.SaveAfterAssociationsCk, SaveAfterAssociations(false))
	updateCallback.Register(gorm.AfterUpdateCk, AfterUpdate)
	updateCallback.Match(enableTransaction).Register(gorm.CommitOrRollbackCk, CommitOrRollbackTransaction)
	if len(config.UpdateClauses) == 0 {
		config.UpdateClauses = updateClauses
	}
	updateCallback.Clauses = config.UpdateClauses

	rowCallback := db.Callback().Row()
	rowCallback.Register(gorm.RowCk, RowQuery)
	rowCallback.Clauses = config.QueryClauses

	rawCallback := db.Callback().Raw()
	rawCallback.Register(gorm.RawCk, RawExec)
	rawCallback.Clauses = config.QueryClauses
}
