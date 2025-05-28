package apaas

type ApaasQueryType int8

const (
	_skipQueryType ApaasQueryType = iota
	SelectType
	CreateType
	InsertType
	UpdateType
	DeleteType
	RawType
)

type ApaasQueryArgs struct {
	Select    any
	From      any
	Where     any
	Join      any
	Group     any
	Order     any
	Limit     any
	Offset    any
	Update    any
	Delete    any
	create    any
	RawSql    string
	QueryType ApaasQueryType
}
