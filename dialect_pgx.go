package gorm

type pgx struct {
	postgres
}

func init() {
	RegisterDialect("pgx", &pgx{})
}

func (pgx) GetName() string {
	return "pgx"
}