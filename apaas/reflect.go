package apaas

import (
	"fmt"
	"reflect"
	"strings"
)

var _unknown_field = reflect.StructField{}

func GetFieldTypeByColumnNameV2(value reflect.Value, fieldColumnName string) (reflect.StructField, bool) {
	// check obj is pointer or not
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := value.Field(i)
		if field.Anonymous {
			return GetFieldTypeByColumnNameV2(fieldValue, fieldColumnName)
		}
		tag := field.Tag.Get("gorm")
		if tag == "" {
			continue
		}
		if cname, ok := getColumnNameByColumnTag(tag); ok && cname == fieldColumnName {
			return field, ok
		}
	}
	return _unknown_field, false
}

/*
case：

	type Faction struct {
			ID          int64  `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
			LiveID      int64  `gorm:"column:live_id" json:"live_id"`
			BizID       int64  `gorm:"column:biz_id" json:"biz_id"`
			FactionID   int64  `gorm:"column:faction_id;not null" json:"faction_id"`
			FactionName string `gorm:"column:faction_name" json:"faction_name"`
			OrgID       int64  `gorm:"column:org_id;comment:union外键" json:"org_id" apass_engine_lookup_id:"webcast.union.org_id"` // union外键
		}

fieldColumnName: faction_id
*/
func GetFieldTypeByColumnName(obj any, fieldColumnName string) (reflect.StructField, bool) {
	return GetFieldTypeByColumnNameV2(reflect.ValueOf(obj), fieldColumnName)
}

/*
case: `gorm:"column:id;primaryKey;autoIncrement:true"
*/
func getColumnNameByColumnTag(tag string) (string, bool) {
	fs := strings.Split(tag, ";")
	if len(fs) >= 1 {
		sfs := strings.Split(fs[0], ":")
		if sfs[0] == "column" {
			return sfs[1], true
		}
	}
	return "", false
}

func GetFieldTypeByColumnNameV3(typ reflect.Type, fieldColumnName string) (reflect.StructField, bool) {
	// check obj is pointer or not
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Anonymous {
			return GetFieldTypeByColumnNameV3(field.Type, fieldColumnName)
		}
		tag := field.Tag.Get("gorm")
		if tag == "" {
			continue
		}
		if cname, ok := getColumnNameByColumnTag(tag); ok && cname == fieldColumnName {
			return field, ok
		}
	}
	return _unknown_field, false
}

func ParseLookupTagMeta(tag, columnName, dbName, tableName string) (*ApaasLookupMeta, error) {
	fmt.Printf("[apaas_engine] tag: %s, columnName: %s, dbName: %s, tableName: %s\n", tag, columnName, dbName, tableName)
	if tag == "" {
		return nil, nil
	}
	fs := strings.Split(tag, ".")
	if len(fs) > MaxTagDeep {
		return nil, GenError(fmt.Sprintf("apass_engine_lookup_value=%s, lookup deep=%d>%d", tag, len(fs), MaxTagDeep))
	}
	meta := &ApaasLookupMeta{
		CName:      columnName,
		LookupMeta: make([]*LookupMeta, len(fs)-1),
		LastField:  fs[len(fs)-1],
		OrgTag:     fs,
	}
	dbCol := GetDBCol()
	if dbCol == nil {
		return nil, GenError(fmt.Sprintf("apass_engine_lookup_value=%s, cann't get db collection", tag))
	}
	dbMeta, ok := dbCol.GetDB(dbName)
	if !ok || dbMeta == nil {
		return nil, GenError(fmt.Sprintf("apass_engine_lookup_value=%s, cann't get db(name=%s) meta", tag, dbName))
	}

	tableMeta, ok := dbMeta.tableView[tableName]
	if !ok {
		return nil, GenError(fmt.Sprintf("apass_engine_lookup_value=%s, cann't get table(name=%s) meta", tag, tableName))
	}
	var idx int = 0
	for idx < len(fs)-1 {
		lp := &LookupMeta{
			FieldName: fs[idx],
			ForeignMeta: ForeignMeta{
				DBName: dbName,
				FName:  fs[idx+1],
			},
		}
		found := false
		for _, ff := range tableMeta.ForeignFields {
			if ff.Name == fs[idx] {
				if fs[idx+1] != ff.foreignMeta.FName {
					return nil, GenError(fmt.Sprintf("apass_engine_lookup_value=%s, db=%s, table=%s, field=%s, foreign(table=%s), foreign field=%s not equal to lookup field=%s)", tag, dbName, tableName, ff.Name, ff.foreignMeta.TName, fs[idx+1]))
				}
				if dbName != ff.foreignMeta.DBName {
					return nil, GenError(fmt.Sprintf("apass_engine_lookup_value=%s, db=%s, table=%s, field=%s, foreign(table=%s), foreign db=%s not equal to lookup db=%s)", tag, tag, dbName, tableName, ff.Name, ff.foreignMeta.DBName, dbName))
				}
				lp.ForeignMeta.TName = ff.foreignMeta.TName
				lp.ForeignMeta.FTMeta = ff.foreignMeta.FTMeta
				found = true
				break
			}
		}
		if !found {
			return nil, GenError(fmt.Sprintf("apass_engine_lookup_value=%s, db=%s, table=%s, field=%s is not foreign key", tag, tag, dbName, tableName, fs[idx]))
		}
		meta.LookupMeta[idx] = lp
		tableName = lp.ForeignMeta.TName
		tableMeta, ok = dbMeta.tableView[tableName]
		if !ok {
			return nil, GenError(fmt.Sprintf("apass_engine_lookup_value=%s, cann't get table(name=%s) meta", tag, tableName))
		}
	}
	return meta, nil
}
