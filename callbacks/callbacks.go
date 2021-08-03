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

var (
	//transaction callback names
	BeforeTransactionCk = "gorm:begin_transaction"
	CommitOrRollbackCk  = "gorm:commit_or_rollback_transaction"

	// create callback names
	BeforeCreateCk           = "gorm:before_create"
	SaveBeforeAssociationsCk = "gorm:save_before_associations"
	CreateCk                 = "gorm:create"
	SaveAfterAssociationsCk  = "gorm:save_after_associations"
	AfterCreateCk            = "gorm:after_create"

	// query callback names
	QueryCk      = "gorm:query"
	PreloadCk    = "gorm:preload"
	AfterQueryCk = "gorm:after_query"

	// delete callback names
	BeforeDeleteCk             = "gorm:before_delete"
	DeleteBeforeAssociationsCk = "gorm:delete_before_associations"
	DeleteCk                   = "gorm:delete"
	AfterDeleteCk              = "gorm:after_delete"

	// update callback names
	SetUpReflectValueCk = "gorm:setup_reflect_value"
	BeforeUpdateCk      = "gorm:before_update"
	UpdateCk            = "gorm:update"
	AfterUpdateCk       = "gorm:after_update"

	// row callback names
	RowCk = "gorm:row"

	// raw callback names
	RawCk = "gorm:raw"

	CoreCallbackNames = [...]string{BeforeTransactionCk, CommitOrRollbackCk,
		SaveBeforeAssociationsCk, SaveAfterAssociationsCk,
		CreateCk, QueryCk, PreloadCk,
		DeleteBeforeAssociationsCk, DeleteCk,
		SetUpReflectValueCk, UpdateCk,
		RowCk, RawCk}
)

func RegisterDefaultCallbacks(db *gorm.DB, config *Config) {
	enableTransaction := func(db *gorm.DB) bool {
		return !db.SkipDefaultTransaction
	}

	createCallback := db.Callback().Create()
	createCallback.Match(enableTransaction).Register(BeforeTransactionCk, BeginTransaction)
	createCallback.Register(BeforeCreateCk, BeforeCreate)
	createCallback.Register(SaveBeforeAssociationsCk, SaveBeforeAssociations(true))
	createCallback.Register(CreateCk, Create(config))
	createCallback.Register(SaveAfterAssociationsCk, SaveAfterAssociations(true))
	createCallback.Register(AfterCreateCk, AfterCreate)
	createCallback.Match(enableTransaction).Register(CommitOrRollbackCk, CommitOrRollbackTransaction)
	if len(config.CreateClauses) == 0 {
		config.CreateClauses = createClauses
	}
	createCallback.Clauses = config.CreateClauses

	queryCallback := db.Callback().Query()
	queryCallback.Register(QueryCk, Query)
	queryCallback.Register(PreloadCk, Preload)
	queryCallback.Register(AfterQueryCk, AfterQuery)
	if len(config.QueryClauses) == 0 {
		config.QueryClauses = queryClauses
	}
	queryCallback.Clauses = config.QueryClauses

	deleteCallback := db.Callback().Delete()
	deleteCallback.Match(enableTransaction).Register(BeforeTransactionCk, BeginTransaction)
	deleteCallback.Register(BeforeDeleteCk, BeforeDelete)
	deleteCallback.Register(DeleteBeforeAssociationsCk, DeleteBeforeAssociations)
	deleteCallback.Register(DeleteCk, Delete)
	deleteCallback.Register(AfterDeleteCk, AfterDelete)
	deleteCallback.Match(enableTransaction).Register(CommitOrRollbackCk, CommitOrRollbackTransaction)
	if len(config.DeleteClauses) == 0 {
		config.DeleteClauses = deleteClauses
	}
	deleteCallback.Clauses = config.DeleteClauses

	updateCallback := db.Callback().Update()
	updateCallback.Match(enableTransaction).Register(BeforeTransactionCk, BeginTransaction)
	updateCallback.Register(SetUpReflectValueCk, SetupUpdateReflectValue)
	updateCallback.Register(BeforeUpdateCk, BeforeUpdate)
	updateCallback.Register(SaveBeforeAssociationsCk, SaveBeforeAssociations(false))
	updateCallback.Register(UpdateCk, Update)
	updateCallback.Register(SaveAfterAssociationsCk, SaveAfterAssociations(false))
	updateCallback.Register(AfterUpdateCk, AfterUpdate)
	updateCallback.Match(enableTransaction).Register(CommitOrRollbackCk, CommitOrRollbackTransaction)
	if len(config.UpdateClauses) == 0 {
		config.UpdateClauses = updateClauses
	}
	updateCallback.Clauses = config.UpdateClauses

	rowCallback := db.Callback().Row()
	rowCallback.Register(RowCk, RowQuery)
	rowCallback.Clauses = config.QueryClauses

	rawCallback := db.Callback().Raw()
	rawCallback.Register(RawCk, RawExec)
	rawCallback.Clauses = config.QueryClauses
}
