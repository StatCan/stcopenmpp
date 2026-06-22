// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
	"strconv"
	"strings"
)

// Facet is type to define database engine and driver facets, e.g.: name of bigint type
type Facet uint16

// database provider engine, for example: MySQL engine of MariaDB or MySQL provider
type Engine uint16

const eShift = 8 // shift of engine part of the facet

const (
	DefaultFacet = Facet(DefaultEngine<<eShift + Engine(DefaultPhs)) // common default db facet

	SqliteFacet     = Facet(SqliteEngine<<eShift + Engine(QmarkPhs))      // SQLite db facet
	PostgreSqlFacet = Facet(PostgreSqlEngine<<eShift + Engine(DollarPhs)) // PostgreSQL db facet
	MySqlFacet      = Facet(MySqlEngine<<eShift + Engine(QmarkPhs))       // MySQL and MariaDB facet
	MsSqlFacet      = Facet(MsSqlEngine<<eShift + Engine(MsSqlPhs))       // MS SQL db facet
	OracleFacet     = Facet(OracleEngine<<eShift + Engine(ColonPhs))      // Oracle db facet

	PostgreSqlOdbcFacet = Facet(PostgreSqlEngine<<eShift + Engine(OdbcPhs)) // PostgreSQL db ODBC facet
	MySqlOdbcFacet      = Facet(MySqlEngine<<eShift + Engine(OdbcPhs))      // MySQL and MariaDB ODBC facet
	MsSqlOdbcFacet      = Facet(MsSqlEngine<<eShift + Engine(OdbcPhs))      // MS SQL ODBC db facet
	OracleOdbcFacet     = Facet(OracleEngine<<eShift + Engine(OdbcPhs))     // Oracle ODBC db facet
	Db2OdbcFacet        = Facet(Db2Engine<<eShift + Engine(OdbcPhs))        // DB2 ODBC db facet
)

const (
	DefaultEngine    Engine = iota // common default db engine
	SqliteEngine                   // SQLite db engine
	PostgreSqlEngine               // PostgreSQL db engine
	MySqlEngine                    // MySQL and MariaDB engine
	MsSqlEngine                    // MS SQL db engine
	OracleEngine                   // Oracle db engine
	Db2Engine                      // DB2 db engine
)

type placeHolderStyle uint8

const (
	DefaultPhs placeHolderStyle = iota // by default return empty "" string as invalid query parameter placeholder
	QmarkPhs                           // use ? question mark positional parameter placeholder
	DollarPhs                          // use $N dollar sign and index PostgreSQL parameter placeholder
	MsSqlPhs                           // use @pN MS SQL style of positional parameter placeholder
	ColonPhs                           // use :N colon and index sign positional parameter placeholder
)
const OdbcPhs = QmarkPhs // use ? as odbc positional parameter placeholder

// return database provider engine
func (facet Facet) engine() Engine {
	return Engine(facet >> eShift)
}

// return style of query positional parameter placeholder, example: ? for ODBC or $N for PostgreSQL
func (facet Facet) holderStyle() placeHolderStyle {
	return placeHolderStyle(facet & 0xFF)
}

// maxTableNameSize return max length of db table or view name.
// Current max name sizes: PostgreSQL=63 MySQL=64 MSSQL=128 DB2=128 Oracle=128 (Oracle antiques not supported)
const maxTableNameSize int = 63

// String is default printable value of db facet, Stringer implementation
func (facet Facet) String() string {
	switch facet {
	case DefaultFacet:
		return "Default db facet"
	case SqliteFacet:
		return "Sqlite db facet"
	case PostgreSqlFacet:
		return "PostgreSQL db facet"
	case MySqlFacet:
		return "MySQL db facet"
	case MsSqlFacet:
		return "MS SQL db facet"
	case OracleFacet:
		return "Oracle db facet"
	case PostgreSqlOdbcFacet:
		return "PostgreSQL ODBC facet"
	case MsSqlOdbcFacet:
		return "MS SQL ODBC facet"
	case OracleOdbcFacet:
		return "Oracle ODBC facet"
	case Db2OdbcFacet:
		return "DB2 ODBC facet"
	}
	return "Unknown db facet"
}

// bigintType return type name for BIGINT sql type
func (facet Facet) bigintType() string {
	if facet.engine() == OracleEngine {
		return "NUMBER(19)"
	}
	return "BIGINT"
}

// floatType return type name for FLOAT standard sql type
func (facet Facet) floatType() string {
	if facet.engine() == OracleEngine {
		return "BINARY_DOUBLE"
	}
	return "FLOAT"
}

// textType return column type DDL for long VARCHAR columns, use it for len > 255.
func (facet Facet) textType(len int) string {
	switch facet.engine() {
	case MsSqlEngine:
		if len > 4000 {
			return "TEXT"
		}
	case OracleEngine:
		if len > 2000 {
			return "CLOB"
		}
	}
	return "VARCHAR(" + strconv.Itoa(len) + ")"
}

// createTableIfNotExist return sql statement to create table if not exists
func (facet Facet) createTableIfNotExist(tableName string, bodySql string) string {

	switch facet.engine() {
	case SqliteEngine, PostgreSqlEngine, MySqlEngine:
		return "CREATE TABLE IF NOT EXISTS " + tableName + " " + bodySql
	case MsSqlEngine:
		return "IF NOT EXISTS" +
			" (SELECT * FROM INFORMATION_SCHEMA.TABLES T WHERE T.TABLE_NAME = " + ToQuoted(tableName) + ") " +
			" CREATE TABLE " + tableName + " " + bodySql
	}
	return "CREATE TABLE " + tableName + " " + bodySql
}

// createViewIfNotExist return sql statement to create view if not exists
func (facet Facet) createViewIfNotExist(viewName string, bodySql string) string {

	switch facet.engine() {
	case SqliteEngine:
		return "CREATE VIEW IF NOT EXISTS " + viewName + " AS " + bodySql
	case PostgreSqlEngine, MySqlEngine:
		return "CREATE OR REPLACE VIEW " + viewName + " AS " + bodySql
	case MsSqlEngine:
		return "CREATE VIEW " + viewName + " AS " + bodySql
	case OracleEngine, Db2Engine:
		return "CREATE OR REPLACE VIEW " + viewName + " AS " + bodySql
	}
	return "CREATE VIEW " + viewName + " AS " + bodySql
}

// return query positional parameter placeholder
func (facet Facet) PlaceHolder(pos int) string {

	if pos <= 0 {
		return "" // error: positive paramter position expected
	}
	phs := facet.holderStyle()

	if phs == OdbcPhs {
		return "?"
	}
	switch phs {
	case QmarkPhs:
		return "?"
	case DollarPhs:
		return "$" + strconv.Itoa(pos)
	case MsSqlPhs:
		return "@p" + strconv.Itoa(pos)
	case ColonPhs:
		return ":" + strconv.Itoa(pos)
	}
	return "" // return empty "" string as invalid parameter palceholder
}

// Detect db provider engine by quiering sql server.
// It may not be always reliable and even not true engine.
// It is better to use driver information to determine db provider.
func detectEngine(dbConn *sql.DB) Engine {

	eng := DefaultEngine

	// check is it PostgreSQL
	// check is it MySQL (not reliable) or MariaDB
	// odbc driver bug (?): PostgreSQL 9.2 + odbc 9.5.400 fails forever after first query failed
	// that means PostgreSQL engine detection must be first
	_ = SelectRows(dbConn,
		"SELECT LOWER(VERSION())",
		func(rows *sql.Rows) error {
			var s sql.NullString
			if err := rows.Scan(&s); err != nil {
				return err
			}
			if s.Valid {
				v := s.String
				if strings.Contains(v, "postgresql") {
					eng = PostgreSqlEngine
				}
				if eng == DefaultEngine &&
					(strings.Contains(v, "mysql") || strings.Contains(v, "mariadb") || strings.HasPrefix(v, "5.")) {
					eng = MySqlEngine
				}
			}
			return nil
		})
	if eng != DefaultEngine {
		return eng
	}

	// check is it SQLite
	_ = SelectRows(dbConn,
		"SELECT COUNT(*) FROM sqlite_master",
		func(rows *sql.Rows) error {
			var n sql.NullInt64
			if err := rows.Scan(&n); err != nil {
				return err
			}
			if n.Valid {
				eng = SqliteEngine
			}
			return nil
		})
	if eng != DefaultEngine {
		return eng
	}

	// check is it MS SQL
	_ = SelectRows(dbConn,
		"SELECT LOWER(@@VERSION)",
		func(rows *sql.Rows) error {
			var s sql.NullString
			if err := rows.Scan(&s); err != nil {
				return err
			}
			if s.Valid {
				eng = MsSqlEngine
			}
			return nil
		})
	if eng != DefaultEngine {
		return eng
	}

	// check is it Oracle
	_ = SelectRows(dbConn,
		"SELECT LOWER(product) FROM product_component_version",
		func(rows *sql.Rows) error {
			var s sql.NullString
			if err := rows.Scan(&s); err != nil {
				return err
			}
			if s.Valid {
				eng = OracleEngine
			}
			return nil
		})
	if eng != DefaultEngine {
		return eng
	}

	// check is it IBM DB2
	_ = SelectRows(dbConn,
		"SELECT COUNT(*) FROM SYSIBMADM.ENV_PROD_INFO",
		func(rows *sql.Rows) error {
			var n sql.NullInt64
			if err := rows.Scan(&n); err != nil {
				return err
			}
			if n.Valid {
				eng = Db2Engine
			}
			return nil
		})
	return eng
}
