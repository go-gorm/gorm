package callbacks

import (
	"fmt"
	"reflect"

	"gorm.io/gorm"
	"gorm.io/gorm/apaas"
)

var dbNameCaller func(*gorm.DB) (string, error)

func SetDBNameCaller(fn func(*gorm.DB) (string, error)) {
	dbNameCaller = fn
}

func ExtraCheckerCallBack(stage string) func(db *gorm.DB) {
	return func(db *gorm.DB) {
		if db.Error == nil && db.Statement.Schema != nil {
			if db.Config.DBName == "" {
				if dbNameCaller != nil {
					db.Config.DBName, _ = dbNameCaller(db)
				} else {
					db.Config.DBName, _ = db.GetDBName()
				}
			}
			dbName := db.Config.DBName
			if dbName == "" {
				//db.Error = db.AddError(GenError(fmt.Sprintf("%s ExtraCheckerCallBack(stage=%s) GetDBName nil", MSG_PREFIX, stage)))
				return
			}
			/*
				db.Logger.Info(db.Statement.Context, "===schema: %#v\n", db.Statement.Schema.Fields)
				for i, s := range db.Statement.Schema.Fields {
					db.Logger.Info(db.Statement.Context, "===schema[i=%d]: %#v\n", i, *s)
				}
			*/
			dbCol := apaas.GetDBCol()
			if dbCol == nil {
				//db.Error = db.AddError(GenError(fmt.Sprintf("%s ExtraCheckerCallBack(stage=%s) GetDBCollection nil ", MSG_PREFIX, stage)))
				return
			}
			dbMeta, ok := dbCol.GetDB(dbName)
			if !ok {
				//db.Error = db.AddError(GenError(fmt.Sprintf("%s ExtraCheckerCallBack(stage=%s) GetDB(db=%s) nil", dbName, MSG_PREFIX, stage)))
				return
			}
			tableMeta, ok := dbMeta.GetTableByName(db.Statement.Table)
			if !ok {
				//db.Error = db.AddError(GenError(fmt.Sprintf("%s ExtraCheckerCallBack(stage=%s) GetTable(db=%s,table=%s) GetDB nil", MSG_PREFIX, stage, dbName, db.Statement.Table)))
				return
			}
			db.Logger.Info(db.Statement.Context, "%s ExtraCheckerCallBack(stage=%s) db_name=%s, table=%s", apaas.MSG_PREFIX, stage, dbName, tableMeta.TableName)
			val := reflect.ValueOf(db.Statement.Dest)
			for _, extraField := range tableMeta.ExtraFields {
				db.Logger.Info(db.Statement.Context, "%s ExtraCheckerCallBack(stage=%s) db_name=%s, table=%s, check extra field=%s begin", apaas.MSG_PREFIX, stage, dbName, db.Statement.Table, extraField.Name)
				field, ok := db.Statement.Schema.FieldsByDBName[extraField.Name]
				if !ok {
					db.Error = db.AddError(apaas.GenError(fmt.Sprintf("ExtraCheckerCallBack(stage=%s) Extra Field(db=%s,table=%s,field=%s) not found value in DestValue", stage, dbName, db.Statement.Table, field.DBName)))
					return
				}
				v, _ := field.ValueOf(db.Statement.Context, val)
				strV, ok := v.(string)
				pStrV, ok1 := v.(*string)
				if !ok && !ok1 {
					db.Error = db.AddError(apaas.GenError(fmt.Sprintf("ExtraCheckerCallBack(stage=%s) Extra Field(db=%s,table=%s,field=%s) is not string/*string value in DestValue", stage, dbName, db.Statement.Table, field.DBName)))
					return
				}
				if ok1 {
					strV = *pStrV
				}
				if err := extraField.GetApaasMeta().Check(strV); err != nil {
					db.Error = db.AddError(apaas.GenError(fmt.Sprintf("ExtraCheckerCallBack(stage=%s) Extra Field(db=%s,table=%s,field=%s) check error=%s", stage, dbName, db.Statement.Table, field.DBName, err.Error())))
					return
				}
			}
		}
	}
}
