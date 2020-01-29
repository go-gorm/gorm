package model

import (
	"reflect"
)

type Model struct {
	ModelType               reflect.Type
	Table                   string
	PrioritizedPrimaryField *Field
	PrimaryFields           []*Field
	Fields                  []*Field
	FieldsByName            map[string]*Field
	FieldsByDBName          map[string]*Field
	Relationships           Relationships
}

type Field struct {
	Name            string
	DBName          string
	DataType        reflect.Type
	DBDataType      string
	Tag             reflect.StructTag
	TagSettings     map[string]string
	PrimaryKey      bool
	AutoIncrement   bool
	Creatable       bool
	Updatable       bool
	Nullable        bool
	Unique          bool
	Precision       int
	Size            int
	HasDefaultValue bool
	DefaultValue    string
	StructField     reflect.StructField
	Model           *Model
}
