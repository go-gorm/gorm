package gorm

type Association struct {
	Scope  *Scope
	Column string
}

func (*Association) Find(value interface{}) {
}

func (*Association) Append(values interface{}) {
}

func (*Association) Delete(value interface{}) {
}

func (*Association) Clear(value interface{}) {
}

func (*Association) Replace(values interface{}) {
}

func (*Association) Count(values interface{}) int {
	return 0
}
