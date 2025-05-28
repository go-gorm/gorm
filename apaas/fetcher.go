package apaas

import (
	"context"

	"gorm.io/gorm/logger"
)

type DBFetcher interface {
	Fetch() ([]*ApaasTable, error)
	// each fetcher has only uniq name, Name is must DBName
	// if DBName is nil, default fetch all
	DBName() string
}

var gAllFetcher = map[string]DBFetcher{}
var gDeltaFetcher = map[string]DBFetcher{}

func AddDBFetcher(f DBFetcher) {
	gAllFetcher[f.DBName()] = f
}

// Update All DB metas, called by apaas_db_engine SDK
func UpdateAllDBCol() {
	for name, f := range gAllFetcher {
		if f.DBName() == "" {
			tables, err := f.Fetch()
			if err != nil {
				logger.Default.Error(context.Background(), "%s fetcher(name=%s, DBName=%s) Fetch data error=%s", MSG_PREFIX, name, f.DBName(), err.Error())
				continue
			}
			logger.Default.Info(context.Background(), "%s fetcher(name=%s, DBName=%s) Fetch data len=%d", MSG_PREFIX, name, f.DBName(), len(tables))

			dbCol := &DBCollection{
				dbs: make(map[string]*DBMeta, 128),
			}
			for _, table := range tables {
				v, ok := dbCol.dbs[table.DBName]
				if !ok {
					v := &DBMeta{
						tableList:    make([]*ApaasTable, 0, len(tables)>>2),
						tableView:    make(map[string]*ApaasTable, len(tables)>>2),
						lookupIDView: make(map[string]*ApaasTable, len(tables)>>2),
					}
					dbCol.dbs[table.DBName] = v
				}
				v.tableList = append(v.tableList, table)
				v.tableView[table.TableName] = table
				if table.LookupIDField != nil {
					v.lookupIDView[table.LookupIDField.Name] = table
				}
			}
			SetDBCol(dbCol)
		}
	}
	for name, f := range gAllFetcher {
		if f.DBName() == "" {
			continue
		}
		tables, err := f.Fetch()
		if err != nil {
			logger.Default.Error(context.Background(), "%s fetcher(name=%s, DBName=%s) Fetch data error=%s", MSG_PREFIX, name, f.DBName(), err.Error())
			continue
		}
		logger.Default.Info(context.Background(), "%s fetcher(name=%s, DBName=%s) Fetch data len=%d", MSG_PREFIX, name, f.DBName(), len(tables))

		if len(tables) == 0 {
			continue
		}
		if len(f.DBName()) != 0 {
			dbMeta := &DBMeta{
				tableList: tables,
			}
			dbMeta.tableView = make(map[string]*ApaasTable, len(tables))
			dbMeta.lookupIDView = make(map[string]*ApaasTable, len(tables))
			for _, table := range tables {
				dbMeta.tableView[table.TableName] = table
				if table.LookupIDField != nil {
					dbMeta.lookupIDView[table.LookupIDField.Name] = table
				}
			}
			GetDBCol().SetDB(f.DBName(), dbMeta)
			continue
		}

	}
}

/*
// Update DB metas, called by apaas_db_engine SDK
func UpdateDeltaDBCol() {

}
*/
