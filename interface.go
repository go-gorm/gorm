package gorm

type sqlTx interface {
	Commit() error
	Rollback() error
}
