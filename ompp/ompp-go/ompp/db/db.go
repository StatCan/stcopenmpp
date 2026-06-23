// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

/*
Package db to support openM++ model database operations.

Database may contain multiple openM++ models and consists of two parts:
(a) metadata to describe models: data types, input parameters, output tables, etc.
(b) actual modeling data itself: database tables for input parameters and output tables.

To run the model we need to have a set of model input parameters, called "input working set" or "workset".
User can "open" workset in read-write mode and modify model input parameters.
To use that set as model run input user must "close" it by marking workset as read-only.
After model run user can again open workset as read-write and continue input editing.
Each workset has a name (unique inside of the model) and set id (database unique positive int).

Result of model run stored in output tables and also include copy of all input parameters used to run the model.
That pair of input and output data called "run" and identified by run id (database unique positive int).
*/
package db

import (
	"container/list"
	"database/sql"
	"os"
	"strconv"
	"strings"

	"github.com/statcan/stcopenmpp/ompp/ompp-go/ompp/helper"
	"github.com/statcan/stcopenmpp/ompp/ompp-go/ompp/omppLog"

	_ "github.com/mattn/go-sqlite3"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb"
)

// Database connection values
const (
	SQLiteDbDriver     = "SQLite"    // default db driver name
	Sqlite3DbDriver    = "sqlite3"   // actual SQLite db driver name
	OdbcDbDriver       = "odbc"      // ODBC db driver name
	PostgreSqlDbDriver = "postgres"  // PostgreSQL db driver name
	MySqlDbDriver      = "mysql"     // MySQL and MariaDB db driver name
	MsSqlDbDriver      = "sqlserver" // MS SQL Server db driver name
)
const SQLiteTimeout = 86400 // default SQLite busy timeout

// MinSchemaVersion is a minimal compatible db schema version
const MinSchemaVersion = 105

// MaxSchemaVersion is a maximum compatible db schema version
const MaxSchemaVersion = 105

// database connections pool and database engine facet
type Dbc struct {
	*sql.DB
	Dbf Facet // database engine and driver facets
}

// database transaction and database engine facet
type DbTrx struct {
	*sql.Tx
	Dbf Facet // database engine and driver facets
}

// Open database connection.
//
// Default driver name: "SQLite" and connection string is compatible with model connection, e.g:
//
//	driver name: "" empty
//	driver name: SQLite
//	Database=modelName.sqlite; Timeout=86400; ForeignKeys = 1; OpenMode=ReadWrite;
//
// Otherwise it is expected to be driver name and connection string, e.g:
//
//	driver name: odbc
//	DSN=ms2014; UID=sa; PWD=secret;
//
//	driver name: sqlite3
//	file:m1.sqlite?mode=rw&_foreign_keys=on&_busy_timeout=86400000
func Open(dbConnStr, dbDriver string) (Dbc, error) {

	// detect driver name
	if dbDriver == "" && strings.HasPrefix(dbConnStr, "postgresql") {
		dbDriver = PostgreSqlDbDriver
	}
	if dbDriver == "" && (strings.HasPrefix(dbConnStr, "mysql") || strings.HasPrefix(dbConnStr, "mariadb")) {
		dbDriver = MySqlDbDriver
	}
	if dbDriver == "" && strings.HasPrefix(dbConnStr, "sqlserver") {
		dbDriver = MsSqlDbDriver
	}

	// convert default SQLite connection string into sqlite3 format
	// delete existing sqlite file if required
	if dbDriver == "" || dbDriver == SQLiteDbDriver {
		var err error
		if dbConnStr, dbDriver, err = prepareSqlite(dbConnStr); err != nil {
			return Dbc{Dbf: DefaultFacet}, err
		}
	}

	// empty connection string likely produce error message "invalid openM++ database", explain to the user source of the problem
	if dbConnStr == "" {
		omppLog.Log("database connection string is empty, it may be an inavlid parameters")
	}

	// open database connection
	omppLog.LogSql("Connect to " + dbDriver)

	dbConn, err := sql.Open(dbDriver, dbConnStr)
	if err != nil {
		return Dbc{Dbf: DefaultFacet}, err
	}

	// determine db facet if requered and not defined by driver (example: odbc)
	facet := DefaultFacet

	switch strings.ToLower(dbDriver) {
	case Sqlite3DbDriver: // at this point "SQLite" pseudo name replaced by "sqlite3" db-driver name
		facet = SqliteFacet
	case PostgreSqlDbDriver:
		facet = PostgreSqlFacet
	case MySqlDbDriver:
		facet = MySqlFacet
	case MsSqlDbDriver:
		facet = MsSqlFacet
	case OdbcDbDriver: // check if ODBC compiled in, use go install -tags odbc to do this
		if !IsOdbcSupported {
			return Dbc{Dbf: DefaultFacet}, helper.ErrorNew("ODBC database connection not supported (executable build without ODBC library)")
		}
	default: // at this point "" empty driver name replaced by "sqlite3" db-driver name
		return Dbc{Dbf: DefaultFacet}, helper.ErrorNew("Invalid database driver name")
	}

	if facet == DefaultFacet {
		if dbDriver != "odbc" {
			return Dbc{Dbf: DefaultFacet}, helper.ErrorNew("Invalid database driver name", dbDriver)
		}

		engine := detectEngine(dbConn)

		switch engine {
		case SqliteEngine:
			facet = SqliteFacet
		case PostgreSqlEngine:
			facet = PostgreSqlOdbcFacet
		case MySqlEngine:
			facet = MySqlOdbcFacet
		case MsSqlEngine:
			facet = MsSqlOdbcFacet
		case OracleEngine:
			facet = OracleOdbcFacet
		case Db2Engine:
			facet = Db2OdbcFacet
		}
	}
	omppLog.LogSql(facet.String())

	err = connectionSetup(dbDriver, facet.engine(), dbConn) // apply provider-specific connecton settings
	if err != nil {
		return Dbc{Dbf: DefaultFacet}, err
	}

	return Dbc{DB: dbConn, Dbf: facet}, nil
}

// set connection properties using OM_DB_SET_{vendor-or-driver-name} environment variable, for example MySQL:
//
//	set OM_DB_SET_mysql=SET @@SQL_MODE = CONCAT(@@SQL_MODE, ',PIPES_AS_CONCAT,ANSI_QUOTES,NO_BACKSLASH_ESCAPES'), AUTOCOMMIT = 0
//
// names are: driver name or one of: sqlite3, mysql, postgresql, sqlserver, oracle, db2
func connectionSetup(dbDriver string, engine Engine, dbConn *sql.DB) error {

	s := ""
	switch engine {
	case SqliteEngine:
		s = os.Getenv("OM_DB_SET_sqlite3")
	case MySqlEngine:
		s = os.Getenv("OM_DB_SET_mysql")
	case PostgreSqlEngine:
		s = os.Getenv("OM_DB_SET_postgresql")
	case MsSqlEngine:
		s = os.Getenv("OM_DB_SET_sqlserver")
	case OracleEngine:
		s = os.Getenv("OM_DB_SET_oracle")
	case Db2Engine:
		s = os.Getenv("OM_DB_SET_db2")
	default:
		s = os.Getenv("OM_DB_SET_" + dbDriver)
	}

	if s != "" {
		err := Update(dbConn, s)
		if err != nil {
			omppLog.LogSql(err.Error())
			return err
		}
	}
	/*
		// to avoid database lock issues for SQLite with SQLITE_THREADSAFE=1
		if facet == SqliteFacet || dbDriver == Sqlite3DbDriver {
			dbConn.SetMaxOpenConns(1)
		}
	*/

	return nil
}

// return SQLite connection string and driver name based on model name:
//
//	Database=modelName.sqlite; Timeout=86400; ForeignKeys = 1; OpenMode=ReadWrite;
func IfEmptyMakeDefault(modelName, sqlitePath, dbConnStr, dbDriver string) (string, string) {
	if dbDriver == "" {
		dbDriver = SQLiteDbDriver
	}
	if dbDriver == SQLiteDbDriver && dbConnStr == "" {
		p := sqlitePath
		if p == "" && modelName != "" {
			p = modelName + ".sqlite"
		}
		dbConnStr = MakeSqliteDefault(p)
	}
	return dbConnStr, dbDriver
}

// return read-only SQLite connection string and driver name based on model name:
//
//	Database=modelName.sqlite; Timeout=86400; ForeignKeys = 1; OpenMode=ReadWrite;
func IfEmptyMakeDefaultReadOnly(modelName, sqlitePath, dbConnStr, dbDriver string) (string, string) {
	if dbDriver == "" {
		dbDriver = SQLiteDbDriver
	}
	if dbDriver == SQLiteDbDriver && dbConnStr == "" {
		p := sqlitePath
		if p == "" && modelName != "" {
			p = modelName + ".sqlite"
		}
		dbConnStr = MakeSqliteDefaultReadOnly(p)
	}
	return dbConnStr, dbDriver
}

// return default SQLite connection string based on model.sqlite file path:
//
//	Database=model.sqlite; Timeout=86400; ForeignKeys = 1; OpenMode=ReadWrite;
func MakeSqliteDefault(sqlitePath string) string {
	return "Database=" + sqlitePath + "; Timeout=" + strconv.Itoa(SQLiteTimeout) + "; ForeignKeys = 1; OpenMode=ReadWrite;"
}

// return default read-only SQLite connection string based on model.sqlite file path:
//
//	Database=model.sqlite; Timeout=86400; ForeignKeys = 1; OpenMode=ReadOnly;
func MakeSqliteDefaultReadOnly(sqlitePath string) string {
	return "Database=" + sqlitePath + "; Timeout=" + strconv.Itoa(SQLiteTimeout) + "; ForeignKeys = 1; OpenMode=ReadOnly;"
}

// Convert SQLite connection string into "sqlite3" format and delete existing db.slite file if required.
//
// Following parameters allowed for SQLite database connection:
//
//	Database - (required) database file path or URI
//	Timeout - (optional) table lock "busy" timeout in seconds, default=0
//	OpenMode - (optional) database file open mode: ReadOnly, ReadWrite, Create, default=ReadOnly
//	DeleteExisting - (optional) if true then delete existing database file, default: false
//	ForeignKeys - (optional) boolean to control foreign_keys, by default: PRAGMA foreign_keys = on
func prepareSqlite(dbConnStr string) (string, string, error) {

	// parse SQLite connection string
	kv, err := helper.ParseKeyValue(dbConnStr)
	if err != nil {
		return "", "", err
	}

	// check SQLite connection string parts
	dbPath := kv["Database"]
	if dbPath == "" {
		return "", "", helper.ErrorNew("SQLIte database file path cannot be empty")
	}

	m := kv["OpenMode"]
	switch strings.ToLower(m) {
	case "", "readonly":
		m = "ro"
	case "readwrite":
		m = "rw"
	case "create":
		m = "rwc"
	default:
		return "", "", helper.ErrorNew("SQLIte invalid OpenMode", m)
	}

	// check if file exist:
	// sqlite3 driver does create new file if not exist, it should return an error
	if m == "ro" || m == "rw" {
		if _, err := os.Stat(dbPath); err != nil {
			return "", "", helper.ErrorNew("SQLIte file not exist (or not accessible)", dbPath)
		}
	}

	s := kv["Timeout"]
	var t int
	if s != "" {
		if t, err = strconv.Atoi(s); err != nil {
			return "", "", err
		}
	}

	// if foreign key true/false explicitly specified, by default: "PRAGMA foreign_keys = on"
	fk := ""
	s = kv["ForeignKeys"]
	if s != "" {
		isFk, err := strconv.ParseBool(s)
		if err != nil {
			return "", "", err
		}
		fk = "on"
		if !isFk {
			fk = "off"
		}
	}

	// if required delete source file
	s = kv["DeleteExisting"]
	if s != "" {
		var isDel bool
		if isDel, err = strconv.ParseBool(s); err != nil {
			return "", "", err
		}
		if isDel {
			_ = os.Remove(dbPath) // ignore file delete errors, assume file not exist
		}
	}

	// make sqlite3 connection string
	s3Conn := "file:" + dbPath + "?mode=" + m
	if t != 0 {
		s3Conn += "&_busy_timeout=" + strconv.Itoa(1000*t)
	}
	if fk != "" {
		s3Conn += "&_foreign_keys=" + fk
	}

	return s3Conn, Sqlite3DbDriver, nil
}

// SelectFirst select first db row and pass it to cvt() for row.Scan()
func SelectFirst(dbConn *sql.DB, query string, cvt func(row *sql.Row) error) error {
	if dbConn == nil {
		return helper.ErrorNew("invalid database connection")
	}
	omppLog.LogSql(query)
	return cvt(dbConn.QueryRow(query))
}

// SelectRows select db rows and pass each to cvt() for rows.Scan()
func SelectRows(dbConn *sql.DB, query string, cvt func(rows *sql.Rows) error) error {

	if dbConn == nil {
		return helper.ErrorNew("invalid database connection")
	}
	omppLog.LogSql(query)

	rows, err := dbConn.Query(query) // query db rows
	if err != nil {
		return err
	}
	defer rows.Close()

	// process each row
	for rows.Next() {
		if err = cvt(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

// SelectRowsTo select db rows and pass each row to cvt().
// cvt() return true to continue or false to stop rows processing.
func SelectRowsTo(dbConn *sql.DB, query string, cvt func(rows *sql.Rows) (bool, error)) error {

	if dbConn == nil {
		return helper.ErrorNew("invalid database connection")
	}
	omppLog.LogSql(query)

	rows, err := dbConn.Query(query) // query db rows
	if err != nil {
		return err
	}
	defer rows.Close()

	// process each row until the end or until cvt() return false to continue
	for rows.Next() {
		isNext, err := cvt(rows)
		if err != nil {
			return err
		}
		if !isNext {
			break
		}
	}
	return rows.Err()
}

// SelectToList select db rows into list using cvt to convert (scan) each db row into struct.
//
// It selects "page size" number of rows starting from row number == offset (zero based).
// If page size <= 0 then all rows returned.
// If IsFullPage is true then adjust offset to return full last page
func SelectToList(
	dbConn *sql.DB, query string, layout ReadPageLayout, cvt func(rows *sql.Rows) (interface{}, error)) (*list.List, *ReadPageLayout, error) {

	if dbConn == nil {
		return nil, nil, helper.ErrorNew("invalid database connection")
	}

	// query db rows
	omppLog.LogSql(query)

	rows, err := dbConn.Query(query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// adjust page layout: starting offset and page size
	nStart := layout.Offset
	if nStart < 0 {
		nStart = 0
	}
	nSize := layout.Size
	if nSize < 0 {
		nSize = 0
	}
	var nRow int64

	lt := ReadPageLayout{
		Offset:     nStart,
		Size:       0,
		IsLastPage: false,
		IsFullPage: nSize > 0 && layout.IsFullPage,
	}

	// convert each row and append to the result list
	rs := list.New()
	for rows.Next() {
		nRow++
		if nSize > 0 && nRow > nStart+nSize {
			break
		}
		if !lt.IsFullPage && nRow <= nStart {
			continue
		}

		// convert and add row to the page
		r, err := cvt(rows)
		if err != nil {
			return nil, nil, err
		}
		rs.PushBack(r)

		// if this is a full page reading mode: keep page size list of rows
		for nSize > 0 && rs.Len() > int(nSize) {
			rs.Remove(rs.Front())
		}
		lt.Size = int64(rs.Len())
	}
	err = rows.Err()
	if err != nil {
		return nil, nil, err
	}

	// check for the empty result page or last page
	if lt.Size <= 0 {
		lt.Offset = nRow
	}
	lt.IsLastPage = nSize <= 0 || nSize > 0 && nRow <= nStart+nSize

	if lt.IsFullPage { // if this is a full page reading mode then adjust page start

		lt.Offset = nRow - lt.Size
		if !lt.IsLastPage {
			lt.Offset--
		}
		if lt.Offset < 0 {
			lt.Offset = 0
		}
	}

	return rs, &lt, nil
}

// Update execute sql query outside of transaction scope (on different connection)
func Update(dbConn *sql.DB, query string) error {
	if dbConn == nil {
		return helper.ErrorNew("invalid database connection")
	}
	omppLog.LogSql(query)

	_, err := dbConn.Exec(query)
	return err
}

// TrxSelectRows select db rows in transaction scope and pass each to cvt() for rows.Scan()
func TrxSelectRows(dbTrx *sql.Tx, query string, cvt func(rows *sql.Rows) error) error {

	if dbTrx == nil {
		return helper.ErrorNew("invalid database transaction")
	}
	omppLog.LogSql(query)

	rows, err := dbTrx.Query(query) // query db rows
	if err != nil {
		return err
	}
	defer rows.Close()

	// process each row
	for rows.Next() {
		if err = cvt(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

// TrxSelectFirst select first db row in transaction scope and pass it to cvt() for row.Scan()
func TrxSelectFirst(dbTrx *sql.Tx, query string, cvt func(row *sql.Row) error) error {
	if dbTrx == nil {
		return helper.ErrorNew("invalid database transaction")
	}
	omppLog.LogSql(query)
	return cvt(dbTrx.QueryRow(query))
}

// TrxUpdate execute sql query in transaction scope
func TrxUpdate(dbTrx *sql.Tx, query string) error {
	if dbTrx == nil {
		return helper.ErrorNew("invalid database transaction")
	}
	omppLog.LogSql(query)

	_, err := dbTrx.Exec(query)
	return err
}

// TrxUpdateStatement execute sql statement in transaction scope until put() return true
func TrxUpdateStatement(dbTrx *sql.Tx, query string, put func() (bool, []interface{}, error)) error {

	if dbTrx == nil {
		return helper.ErrorNew("invalid database transaction")
	}
	omppLog.LogSql(query)

	// prepare statement in transaction scope
	stmt, err := dbTrx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// until put() return next row values execute statement
	for {
		isNext, r, err := put()
		if err != nil {
			return err
		}
		if !isNext {
			break
		}
		_, err = stmt.Exec(r...)
		if err != nil {
			return err
		}
	}
	return nil
}

// OpenmppSchemaVersion return db schema: select id_value from id_lst where id_key = 'openmpp'
func OpenmppSchemaVersion(dbConn *sql.DB) (int, error) {

	var nVer int

	err := SelectFirst(dbConn,
		"SELECT id_value FROM id_lst WHERE id_key = 'openmpp'",
		func(row *sql.Row) error {
			return row.Scan(&nVer)
		})
	switch {
	case err == sql.ErrNoRows:
		return 0, nil
	case err != nil:
		return -1, err
	}

	return nVer, nil
}

// CheckOpenmppSchemaVersion return error if it is not openM++ db or schema version incompatible
func CheckOpenmppSchemaVersion(dbConn *sql.DB) error {

	nv, err := OpenmppSchemaVersion(dbConn)
	switch {
	case err != nil || nv <= 0:
		return helper.ErrorNew("error: invalid database, likely not an openM++ database")
	case nv < MinSchemaVersion:
		return helper.ErrorFmt("error: incompatible, old version of database: %d, please use earlier version of openM++ tools", nv)
	case nv > MaxSchemaVersion:
		return helper.ErrorFmt("error: incompatible, newer version of database: %d, please use more recent version of openM++ tools", nv)
	}
	return nil
}

// convert boolean to sql value: true=1, false=0
func toBoolSqlConst(isValue bool) string {
	if isValue {
		return "1"
	}
	return "0"
}

// return true if character is can be interpreted as sql ' quote
// MSSQL silently replace following utf-16 chars with 'single' quote:
/*
  &#x2b9;    697  Modifier Letter Prime
  &#x2bc;    700  Modifier Letter Apostrophe
  &#x2c8;    712  Modifier Letter Vertical Line
  &#x2032;  8242  Prime
  &#xff07; 65287  Fullwidth Apostrophe
*/
func IsUnsafeQuote(c rune) bool {
	return c == 0x2b9 || c == 0x2bc || c == 0x2c8 || c == 0x2032 || c == 0xff07
}

// make sql quoted string, ie: 'O”Brien'.
func ToQuoted(src string) string {

	var sb strings.Builder
	sb.WriteRune('\'')

	for _, c := range src {
		if c == '\'' || IsUnsafeQuote(c) {
			sb.WriteString("''")
		} else {
			sb.WriteRune(c)
		}
	}

	sb.WriteRune('\'')
	return sb.String()
}

// return "NULL" if string ” empty or return sql quoted string, ie: 'O”Brien'
func toQuotedOrNull(src string) string {
	if src == "" {
		return "NULL"
	}
	return ToQuoted(src)
}

// make sql quoted string, ie: 'O”Brien'.
// Trim spaces and return up to maxLen bytes from src string.
func toQuotedMax(src string, maxLen int) string {
	return ToQuoted(leftMax(src, maxLen))
}

// return "NULL" if string ” empty or return sql quoted string, ie: 'O”Brien'
// Trim spaces and return up to maxLen bytes from src string.
func toQuotedOrNullMax(src string, maxLen int) string {
	return toQuotedOrNull(leftMax(src, maxLen))
}

// Return up to maxLen bytes from src string.
// It is return bytes (not runes) and last utf-8 rune may be incorrect in result.
func leftMax(src string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(src) > maxLen {
		return src[:maxLen-1]
	}
	return src
}
