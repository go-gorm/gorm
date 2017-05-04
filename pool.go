package gorm

import (
"fmt"
"time"
"sync"

)


// database driver constant int.
type DriverType int

const (
_           DriverType = iota // int enum type
DR_MySQL                      // mysql
DR_Sqlite                     // sqlite
DR_Oracle                     // oracle
DR_Postgres                   // pgsql
)

// database driver string.
type driverDB string

// get type constant int of current driver..
func (d driverDB) Type() DriverType {
a, _ := dataBaseCache.get(string(d))
return a.Driver
}

// get name of current driver
func (d driverDB) Name() string {
return string(d)
}



var (
dataBaseCache = &_dbCache{cache: make(map[string]*alias)}
drivers = map[string]DriverType{
"mysql":    DR_MySQL,
"postgres": DR_Postgres,
"sqlite3":  DR_Sqlite,
}

)

// database alias cacher.
type _dbCache struct {
mux   sync.RWMutex
cache map[string]*alias
}

// add database alias with original name.
func (ac *_dbCache) add(name string, al *alias) (added bool) {
ac.mux.Lock()
defer ac.mux.Unlock()
if _, ok := ac.cache[name]; ok == false {
ac.cache[name] = al
added = true
}
return
}

// get database alias if cached.
func (ac *_dbCache) get(name string) (al *alias, ok bool) {
ac.mux.RLock()
defer ac.mux.RUnlock()
al, ok = ac.cache[name]
return
}

// get default alias.
func (ac *_dbCache) getDefault() (al *alias) {
al, _ = ac.get("default")
return
}

type alias struct {
Name         string
Driver       DriverType
DriverName   string
DataSource   string
MaxIdleConns int
MaxOpenConns int
DB           *DB
TZ           *time.Location
Engine       string
}


func addAliasWthDB(aliasName, driverName string, db *DB) (*alias, error) {
al := new(alias)
al.Name = aliasName
al.DriverName = driverName
al.DB = db

if dr, ok := drivers[driverName]; ok {
al.Driver = dr
} else {
return nil, fmt.Errorf("driver name `%s` have not registered", driverName)
}


if dataBaseCache.add(aliasName, al) == false {
return nil, fmt.Errorf("DataBase alias name `%s` already registered, cannot reuse", aliasName)
}

return al, nil
}

func AddAliasWthDB(aliasName, driverName string, db *DB) error {
_, err := addAliasWthDB(aliasName, driverName, db)
return err
}

// Setting the database connect params. Use the database driver self dataSource args.
func RegisterDataBase(aliasName, driverName, dataSource string, params ...int) error {
var (
db *DB
al  *alias
)

//	db, err = sql.Open(driverName, dataSource)
_db, err := Open(driverName, dataSource)
db=&_db


if err != nil {
err = fmt.Errorf("register db `%s`, %s", aliasName, err.Error())
goto end
}

al, err = addAliasWthDB(aliasName, driverName, db)
if err != nil {
goto end
}

al.DataSource = dataSource


for i, v := range params {
switch i {
case 0:
SetMaxIdleConns(al.Name, v)
case 1:
SetMaxOpenConns(al.Name, v)
}
}

end:
if err != nil {
if db != nil {
db.Close()
}
}

return err
}



// Register a database driver use specify driver name, this can be definition the driver is which database type.
func RegisterDriver(driverName string, typ DriverType) error {
if t, ok := drivers[driverName]; ok == false {
drivers[driverName] = typ
} else {
if t != typ {
return fmt.Errorf("driverName `%s` db driver already registered and is other type\n", driverName)
}
}
return nil
}


// Change the database default used timezone
func SetDataBaseTZ(aliasName string, tz *time.Location) error {
if al, ok := dataBaseCache.get(aliasName); ok {
al.TZ = tz
} else {
return fmt.Errorf("DataBase alias name `%s` not registered\n", aliasName)
}
return nil
}
func detectTZ(al *alias) {
// orm timezone system match database
// default use Local
al.TZ = time.Local

if al.DriverName == "sphinx" {
return
}

switch al.Driver {
case DR_MySQL:
row := al.DB.Raw("SELECT TIMEDIFF(NOW(), UTC_TIMESTAMP)")
var tz string
row.Scan(&tz)
if len(tz) >= 8 {
if tz[0] != '-' {
tz = "+" + tz
}
t, err := time.Parse("-07:00:00", tz)
if err == nil {
al.TZ = t.Location()
} else {
fmt.Printf("Detect DB timezone: %s %s\n", tz, err.Error())
}
}

// get default engine from current database
row = al.DB.Raw("SELECT ENGINE, TRANSACTIONS FROM information_schema.engines WHERE SUPPORT = 'DEFAULT'")

type Egtx struct {
engine string
tx bool
}
var eg Egtx
row.Scan(&eg)

if eg.engine != "" {
al.Engine = eg.engine
} else {
al.Engine = "INNODB"
}

case DR_Sqlite:
al.TZ = time.UTC

case DR_Postgres:
row := al.DB.Raw("SELECT current_setting('TIMEZONE')")
var tz string
row.Scan(&tz)
loc, err := time.LoadLocation(tz)
if err == nil {
al.TZ = loc
} else {
fmt.Printf("Detect DB timezone: %s %s\n", tz, err.Error())
}
}
}


// get table alias.
func getDbAlias(name string) *alias {
if al, ok := dataBaseCache.get(name); ok {
return al
} else {
panic(fmt.Errorf("unknown DataBase alias name %s", name))
}
}

// Change the max idle conns for *sql.DB, use specify database alias name
func SetMaxIdleConns(aliasName string, maxIdleConns int) {
al := getDbAlias(aliasName)
al.MaxIdleConns = maxIdleConns
al.DB.DB().SetMaxIdleConns(maxIdleConns)
}

// Change the max open conns for *sql.DB, use specify database alias name
func SetMaxOpenConns(aliasName string, maxOpenConns int) {
al := getDbAlias(aliasName)
al.MaxOpenConns = maxOpenConns
al.DB.DB().SetMaxOpenConns(maxOpenConns)

}

// Get *sql.DB from registered database by db alias name.
// Use "default" as alias name if you not set.
func GetDB(aliasNames ...string) (*DB, error) {
var name string
if len(aliasNames) > 0 {
name = aliasNames[0]
} else {
name = "default"
}
if al, ok := dataBaseCache.get(name); ok {
return al.DB, nil
} else {
return nil, fmt.Errorf("DataBase of alias name `%s` not found\n", name)
}
}




