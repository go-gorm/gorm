package gorm

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

/*

Understanding Oracle

It's a bit different than the other RDBMS databases and I'll just try to
hightlight a few of the important ones the dialect has to deal with:

1. Oracle upper cases all non-quoted identifiers. That means the dialect
has to decide what to do:
	1. quote all identifiers which would require developers to quote every
		identifer they passed in a string to gorm.
	2. only quote identifers that conflict with reserved words and leave all
		other identifiers unquoted, which means Oracle will automatically
		upper case them.  This would allow developers to pass unquoted
		identifiers in strings they passed to gorm and make the experience
		align better with the other dialects.
We chose option #2.

This design decision has the following side affects:
	a. you must be case insensitive when matching column names, like in
		the Scope.scan function
	b. Devs will have to escape reserved words when they reference them
		in things like: First(&CreditCard{}, `"number" = ?`)


2. Oracle handles last inserted id a bit differently, and requires a sql.Out
parameter to return the value in the oci8 driver.  Since Oracle parameters
are positional, you need to know how many other bind variables there are before
adding the returning clause.  We've implemented the
OraDialect.CreateWithReturningInto(*Scope) to handle this.

3. Oracle doesn't let you specify "AS <tablename> " when selecting a count
	from a dynamic table, so you just omit it. (see Scope.count() )

4. Oracle handles foreign keys a bit differently:
	A. REFERENCES is implicit
	B. ON UPDATE is not supported
	(see scope.addForeignKey() )

5. Searching a blob requires using a function from the dbms_lob package like
	instr() and specifying the offset and number of matches.
	(see oci8.SearchBlob() )

6 Trailing semicolons are not allowed at the end of Oracle sql statements
	(so they were removed in the unit tests)

*/

// OraCommon implements the Dialect and OraDialect interfaces in a common way for all Ora drivers... but many will need
// to override the OraDialect.CreateWithReturningInto.  We've OraCommon public so dialect implementers can use it in modules
// outside of the gorm module/repo
type OraCommon struct {
	db SQLCommon
	DefaultForeignKeyNamer
}

// GetName will panic when called, since ALL driver specific dialects must override GetName() to return the driver name.
func (OraCommon) GetName() string {
	panic("driver specific dialect must override this to return the driver name.")
}

func (*OraCommon) fieldCanAutoIncrement(field *StructField) bool {
	if value, ok := field.TagSettingsGet("AUTO_INCREMENT"); ok {
		return strings.ToLower(value) != "false"
	}
	return field.IsPrimaryKey
}

func (OraCommon) BindVar(i int) string {
	return fmt.Sprintf(":%v", i)
}

// Quote will only quote ora reserved words, which means all other identifiers will be upper cased automatically
func (OraCommon) Quote(key string) string {
	if IsOraReservedWord(key) {
		return fmt.Sprintf(`"%s"`, key)
	}
	return key
}

func (s OraCommon) CurrentDatabase() string {
	var name string
	if err := s.db.QueryRow("SELECT ORA_DATABASE_NAME as \"Current Database\" FROM DUAL").Scan(&name); err != nil {
		return "" // just return "", since the Dialect interface doesn't support returning an error for this func
	}
	return name
}

func (OraCommon) DefaultValueStr() string {
	return "VALUES (DEFAULT)"
}

func (s OraCommon) HasColumn(tableName string, columnName string) bool {
	var count int
	_, tableName = s.oraCurrentDatabaseAndTable(tableName)
	tableName = strings.ToUpper(tableName)
	columnName = strings.ToUpper(columnName)
	if err := s.db.QueryRow("SELECT count(*) FROM ALL_TAB_COLUMNS WHERE TABLE_NAME = :1 AND COLUMN_NAME = :2", tableName, columnName).Scan(&count); err == nil {
		return count > 0
	}
	return false
}

func (s OraCommon) HasForeignKey(tableName string, foreignKeyName string) bool {
	var count int
	tableName = strings.ToUpper(tableName)
	foreignKeyName = strings.ToUpper(foreignKeyName)

	if err := s.db.QueryRow(`SELECT count(*) FROM USER_CONSTRAINTS WHERE CONSTRAINT_NAME = :1 AND constraint_type = 'R' AND table_name = :2`, foreignKeyName, tableName).Scan(&count); err == nil {
		return count > 0
	}
	return false
}

func (s OraCommon) HasIndex(tableName string, indexName string) bool {
	var count int
	tableName = strings.ToUpper(tableName)
	indexName = strings.ToUpper(indexName)
	if err := s.db.QueryRow("SELECT count(*) FROM ALL_INDEXES WHERE INDEX_NAME = :1 AND TABLE_NAME = :2", indexName, tableName).Scan(&count); err == nil {
		return count > 0
	}
	return false
}

func (s OraCommon) HasTable(tableName string) bool {
	var count int
	_, tableName = s.oraCurrentDatabaseAndTable(tableName)
	tableName = strings.ToUpper(tableName)
	if err := s.db.QueryRow("select count(*) from user_tables where table_name = :1", tableName).Scan(&count); err == nil {
		return count > 0
	}
	return false
}

func (OraCommon) LastInsertIDReturningSuffix(tableName, columnName string) string {
	return ""
}

func (OraCommon) LastInsertIDOutputInterstitial(tableName, columnName string, columns []string) string {
	return ""
}

func (s OraCommon) ModifyColumn(tableName string, columnName string, typ string) error {
	_, err := s.db.Exec(fmt.Sprintf("ALTER TABLE %v MODIFY %v %v", tableName, columnName, typ))
	return err
}

func (s OraCommon) RemoveIndex(tableName string, indexName string) error {
	_, err := s.db.Exec(fmt.Sprintf("DROP INDEX %v", indexName))
	return err
}

func (OraCommon) SelectFromDummyTable() string {
	return "FROM DUAL"
}

func (s *OraCommon) SetDB(db SQLCommon) {
	s.db = db
}

func (s OraCommon) oraCurrentDatabaseAndTable(tableName string) (string, string) {
	if strings.Contains(tableName, ".") {
		splitStrings := strings.SplitN(tableName, ".", 2)
		return splitStrings[0], splitStrings[1]
	}
	return s.CurrentDatabase(), tableName
}

func (s *OraCommon) DataTypeOf(field *StructField) string {
	if _, found := field.TagSettingsGet("RESTRICT"); found {
		field.TagSettingsDelete("RESTRICT")
	}
	var dataValue, sqlType, size, additionalType = ParseFieldStructForDialect(field, s)

	if sqlType == "" {
		switch dataValue.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8,
			reflect.Uint16, reflect.Uintptr, reflect.Int64, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			if s.fieldCanAutoIncrement(field) {
				sqlType = "NUMBER GENERATED BY DEFAULT AS IDENTITY"
			} else {
				switch dataValue.Kind() {
				case reflect.Int8,
					reflect.Uint8,
					reflect.Uintptr:
					sqlType = "SHORTINTEGER"
				case reflect.Int, reflect.Int16, reflect.Int32,
					reflect.Uint, reflect.Uint16, reflect.Uint32:
					sqlType = "INTEGER"
				case reflect.Int64,
					reflect.Uint64:
					sqlType = "INTEGER"
				default:
					sqlType = "NUMBER"
				}
			}
		case reflect.Bool:
			sqlType = "INTEGER"
		case reflect.String:
			if _, ok := field.TagSettingsGet("SIZE"); !ok {
				size = 0 // if SIZE haven't been set, use `text` as the default type, as there are no performance different
			}
			switch {
			case size > 0 && size < 4000:
				sqlType = fmt.Sprintf("VARCHAR2(%d)", size)
			case size == 0:
				sqlType = "VARCHAR2 (1000)" // no size specified, so default to something that can be indexed
			default:
				sqlType = "CLOB"
			}

		case reflect.Struct:
			if _, ok := dataValue.Interface().(time.Time); ok {
				sqlType = "TIMESTAMP WITH TIME ZONE"
			}
		default:
			if IsByteArrayOrSlice(dataValue) {
				sqlType = "BLOB"
			}
		}
	}
	if strings.EqualFold(sqlType, "text") {
		sqlType = "CLOB"
	}
	if sqlType == "" {
		panic(fmt.Sprintf("invalid sql type %s (%s) for oracle", dataValue.Type().Name(), dataValue.Kind().String()))
	}

	if strings.TrimSpace(additionalType) == "" {
		return sqlType
	}
	if strings.EqualFold(sqlType, "json") {
		sqlType = "VARCHAR2 (4000)"
	}

	// For oracle, we have to redo the order of the Default type from tag setting
	notNull, _ := field.TagSettingsGet("NOT NULL")
	unique, _ := field.TagSettingsGet("UNIQUE")
	additionalType = notNull + " " + unique
	if value, ok := field.TagSettingsGet("DEFAULT"); ok {
		additionalType = fmt.Sprintf("%s %s %s", "DEFAULT", value, additionalType)
	}

	if value, ok := field.TagSettingsGet("COMMENT"); ok {
		additionalType = additionalType + " COMMENT " + value
	}
	return fmt.Sprintf("%v %v", sqlType, additionalType)
}
func (s OraCommon) LimitAndOffsetSQL(limit, offset interface{}) (sql string, err error) {
	if limit != nil {
		if parsedLimit, err := strconv.ParseInt(fmt.Sprint(limit), 0, 0); err == nil && parsedLimit >= 0 {
			// when only Limit() is called on a query, the offset is set to -1 for some reason
			if offset != nil && offset != -1 {
				if parsedOffset, err := strconv.ParseInt(fmt.Sprint(offset), 0, 0); err == nil && parsedOffset >= 0 {
					sql += fmt.Sprintf(" OFFSET %d ROWS ", parsedOffset)
				} else {
					return "", err
				}
			}
			sql += fmt.Sprintf(" FETCH NEXT %d ROWS ONLY", parsedLimit)
		} else {
			return "", err
		}
	}
	return
}

// NormalizeIndexAndColumn returns argument's index name and column name without doing anything
func (OraCommon) NormalizeIndexAndColumn(indexName, columnName string) (string, string) {
	return indexName, columnName
}

// CreateWithReturningInto to implement the OraDialect interface
func (OraCommon) CreateWithReturningInto(scope *Scope) {
	var stringId string
	var intId uint32
	primaryField := scope.PrimaryField()

	primaryIsString := false
	out := sql.Out{
		Dest: &intId,
	}
	if primaryField.Field.Kind() == reflect.String {
		out = sql.Out{
			Dest: &stringId,
		}
		primaryIsString = true
	}
	scope.SQLVars = append(scope.SQLVars, out)
	scope.SQL = fmt.Sprintf("%s returning %s into :%d", scope.SQL, scope.Quote(primaryField.DBName), len(scope.SQLVars))
	if result, err := scope.SQLDB().Exec(scope.SQL, scope.SQLVars...); scope.Err(err) == nil {
		scope.DB().RowsAffected, _ = result.RowsAffected()
		if primaryIsString {
			scope.Err(primaryField.Set(stringId))
		} else {
			scope.Err(primaryField.Set(intId))
		}
	}
	// this should raise an error, but the gorm.createCallback() which calls it simply doesn't support returning an error
}

// OraSearchBlob returns a where clause substring for searching fieldName and will require you to pass a parameter for the search value
func OraSearchBlob(fieldName string) string {
	// oracle requires some hoop jumping to search []byte stored as BLOB

	const lobSearch = ` dbms_lob.instr (%s, -- the blob
		utl_raw.cast_to_raw (?), -- the search string cast to raw
		1, -- where to start. i.e. offset
		1 -- Which occurrance i.e. 1=first
		 ) > 0 `
	return fmt.Sprintf(lobSearch, fieldName)
}

var setupReserved sync.Once
var reservedWords map[string]struct{}

// IsOraReservedWord is public intentionally so other DEVs can access it for implementation
func IsOraReservedWord(w string) bool {
	setupReserved.Do(
		func() {
			words := strings.Split(reserved, "\n")
			reservedWords = make(map[string]struct{}, len(words))
			for _, s := range words {
				reservedWords[s] = struct{}{}
			}
		},
	)
	_, ok := reservedWords[strings.ToUpper(w)]
	return ok
}

const reserved = `AGGREGATE
AGGREGATES
ALL
ALLOW
ANALYZE
ANCESTOR
AND
ANY
AS
ASC
AT
AVG
BETWEEN
BINARY_DOUBLE
BINARY_FLOAT
BLOB
BRANCH
BUILD
BY
BYTE
CASE
CAST
CHAR
CHILD
CLEAR
CLOB
COMMIT
COMPILE
CONSIDER
COUNT
DATATYPE
DATE
DATE_MEASURE
DAY
DECIMAL
DELETE
DESC
DESCENDANT
DIMENSION
DISALLOW
DIVISION
DML
ELSE
END
ESCAPE
EXECUTE
FIRST
FLOAT
FOR
FROM
HIERARCHIES
HIERARCHY
HOUR
IGNORE
IN
INFINITE
INSERT
INTEGER
INTERVAL
INTO
IS
LAST
LEAF_DESCENDANT
LEAVES
LEVEL
LIKE
LIKEC
LIKE2
LIKE4
LOAD
LOCAL
LOG_SPEC
LONG
MAINTAIN
MAX
MEASURE
MEASURES
MEMBER
MEMBERS
MERGE
MLSLABEL
MIN
MINUTE
MODEL
MONTH
NAN
NCHAR
NCLOB
NO
NONE
NOT
NULL
NULLS
NUMBER
NVARCHAR2
OF
OLAP
OLAP_DML_EXPRESSION
ON
ONLY
OPERATOR
OR
ORDER
OVER
OVERFLOW
PARALLEL
PARENT
PLSQL
PRUNE
RAW
RELATIVE
ROOT_ANCESTOR
ROWID
SCN
SECOND
SELF
SERIAL
SET
SOLVE
SOME
SORT
SPEC
SUM
SYNCH
TEXT_MEASURE
THEN
TIME
TIMESTAMP
TO
UNBRANCH
UPDATE
USING
VALIDATE
VALUES
VARCHAR2
WHEN
WHERE
WITHIN
WITH
YEAR
ZERO
ZONE`
