package gorm

type tidb struct {
	mysql
}

func init() {
	RegisterDialect("tidb", &tidb{})
}

func (tidb) GetName() string {
	return "tidb"
}
