package apaas

import (
	"strings"
)

const (
	TagID      = "apass_engine_lookup_id"
	TagValue   = "apass_engine_lookup_value"
	MaxTagDeep = 4
)

type ApaasFieldType uint8
type FieldType uint8

const (
	_skipFieldType FieldType = iota
	FieldBool
	FieldInt
	FieldInt64
	FieldFloat
	FieldFloat64
	FieldString
	FieldTime
	FieldBit
	FieldBin
	FieldArray
	FieldObject
)

const (
	_skipApaasFieldType ApaasFieldType = iota
	ApaasLookupID
	ApaasLookupValue
	ApaasExtraType
	ApaasFormulaType
)

var FieldMapString = []string{
	"_skipApaasFieldTyle",
	"FieldBool",
	"FieldInt",
	"FieldInt64",
	"FieldFloat",
	"FieldFloat64",
	"FieldString",
	"FieldTime",
	"FieldBit",
	"FieldBin",
	"FieldArray",
	"FieldObject",
}

func (p FieldType) String() string {
	return FieldMapString[p]
}

/*
description: store dynamic field info when is an instance like tag

	`apass_engine_lookup_value`

case:

	type RoomAnchorView struct {
			Room
			Gender string `gorm:"column:gender" json:"gender" apass_engine_lookup_value:"anchor_id.uid.gender"`
			Career string `gorm:"column:career" json:"career" apass_engine_lookup_value:"anchor_id.uid.career"`
			Name   string `gorm:"column:name" json:"name" apass_engine_lookup_value:"anchor_id.uid.name"`
		}

		type RoomAnchorFactionView struct {
			Room
			Gender    string `gorm:"column:gender" json:"gender" apass_engine_lookup_value:"anchor_id.uid.gender"`
			Career    string `gorm:"column:career" json:"career" apass_engine_lookup_value:"anchor_id.uid.career"`
			Name      string `gorm:"column:name" json:"name" apass_engine_lookup_value:"anchor_id.uid.name"`
			OrgName   string `gorm:"column:org_name" json:"org_name" apass_engine_lookup_value:"anchor_id.faction_id.faction_name"`
			UnionInfo string `gorm:"column:union_info" json:"union_info" apass_engine_lookup_value:"anchor_id.faction_id.org_id.union_info"`
		}
*/
type ApaasLookupMeta struct {
	// field gorm column tag name/table field's name. example: union_info
	CName string
	// lookup orgin meta. example: [anchor_id.faction_id.org_id.union_info]
	/*
		    1. anchor_id;       2. faction_id;        3: org_id;
			1. anchor.anchor_id;2. Vction.faction_id;3: union.org_id;
	*/
	LookupMeta []*LookupMeta
	LastField  string // org_name/union_name

	/*
		OrgTag only set value when used in SDK mode
	*/
	// lookup org tag [anchor_id, faction_id, org_id, org_name]
	OrgTag []string
}

// lookup orgin meta. example: [anchor_id.faction_id.org_id.union_info]
type LookupMeta struct {
	FieldName   string
	ForeignMeta ForeignMeta
}

type ApaasTable struct {
	TableName         string
	DBName            string
	Fields            []*ApaasField
	FieldsByName      map[string]*ApaasField
	LookupIDField     *ApaasField   // example: room.room_id, user.uid, faction.faction_id
	ForeignFields     []*ApaasField // foreign key. example: room.anchor_id, named of relookupid
	FormulaFields     []*ApaasField // example: union.title=update_time + org_name
	LookupValueFields []*ApaasField // example: org_id.org_name, anchor_id.org_id.union_id.union_name
	ExtraFields       []*ApaasField // example: anchor.extra, room.extra
}

type ApaasField struct {
	Name        string
	Type        string
	FType       FieldType
	IsUniq      bool
	IsForeign   bool
	foreignMeta *ForeignMeta
	IsApaasType bool
	ApaasMeta   *ApaasMeta
}

func (p *ApaasField) parseFieldType() {
	tp := strings.ToUpper(p.Type)
	ftp := _skipFieldType
	switch tp {
	case "BOOL":
		ftp = FieldBool
	case "INT", "TINYINT", "SMALLINT", "MEDIUMINT":
		ftp = FieldInt
	case "BIGINT":
		ftp = FieldInt64
	case "FLOAT":
		ftp = FieldFloat
	case "DOUBLE", "DECIMAL", "REAL":
		ftp = FieldFloat64
	case "DATE", "DATETIME", "TIMESTAMP", "TIME", "YEAR":
		ftp = FieldTime
	case "VARCHAR", "CHAR", "ENUM", "TEXT", "TINYTEXT", "MEDIUMTEXT", "LONGTEXT", "JSON":
		ftp = FieldString
	case "BLOB", "TINYBLOB", "MEDIUMBLOB", "LONGBLOB", "BINARY", "VARBINARY":
		ftp = FieldBin
	case "BIT":
		ftp = FieldBit
	default:
		ftp = FieldString
	}
	p.FType = ftp
}

func (p *ApaasField) GetApaasMeta() *ApaasMeta {
	return p.ApaasMeta
}

type ForeignMeta struct {
	DBName string
	TName  string
	FName  string
	FTMeta *ApaasTable
}

type ApaasMeta struct {
	ApaasFType  ApaasFieldType
	ExtraMeta   map[string]*ExtraFieldMeta
	FormulaMeta *FormulaMeta
	LookupMeta  *ApaasLookupMeta
	Checker
}

func (p *ApaasMeta) IsApaasFieldType() bool {
	return p.ApaasFType == _skipApaasFieldType
}
func (p *ApaasMeta) IsExtraField() bool {
	return p.ApaasFType == ApaasExtraType
}
func (p *ApaasMeta) IsFormulaField() bool {
	return p.ApaasFType == ApaasFormulaType
}
func (p *ApaasMeta) IsLookupID() bool {
	return p.ApaasFType == ApaasLookupID
}
func (p *ApaasMeta) IsLookupValue() bool {
	return p.ApaasFType == ApaasLookupValue
}

func (p *ApaasMeta) GetApaasFieldType() ApaasFieldType {
	return p.ApaasFType
}
func (p *ApaasMeta) GetExtraMeta() map[string]*ExtraFieldMeta {
	return p.ExtraMeta
}
func (p *ApaasMeta) GetFormulaMeta() *FormulaMeta {
	return p.FormulaMeta
}
func (p *ApaasMeta) Check(extra string) error {
	// step1: extra rule check
	err := ExtraCheck(p.ExtraMeta, extra)
	return err
}

type ExtraFieldMeta struct {
	Key        string
	Type       FieldType                  // Bool/Int/Float/String/Object/Array
	ObjectMeta map[string]*ExtraFieldMeta // if Type is Object, use ObjectMeta
	ArrayMeta  []*ExtraFieldMeta          // if Type is Array, need ArrayMeta
}

type FormulaMeta struct {
	InputFields map[string]*ApaasField
	FormulaRule *FormulaRule
}

type FormulaRule struct {
}

type DBMeta struct {
	tableList    []*ApaasTable
	tableView    map[string]*ApaasTable
	lookupIDView map[string]*ApaasTable // each table has one one and only key to supprort lookup
}

func (p *DBMeta) GetTableByLookupID(lookID string) (*ApaasTable, bool) {
	v, ok := p.lookupIDView[lookID]
	return v, ok
}

func (p *DBMeta) GetTableByName(tableName string) (*ApaasTable, bool) {
	v, ok := p.tableView[tableName]
	return v, ok
}
