// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
	"strconv"
	"strings"
)

// Facet is type to define database provider and driver facets, ie: name of bigint type
type Facet uint8

// maxTableNameSize return max length of db table or view name.
// Current max name sizes: PostgreSQL=63 MySQL=64 MSSQL=128 DB2=128 Oracle=128 (Oracle antiques not supported)
const maxTableNameSize int = 63

const (
	DefaultFacet Facet = iota // common default db facet
	SqliteFacet               // SQLite db facet
	PgSqlFacet                // PostgreSQL db facet
	MySqlFacet                // MySQL and MariaDB facet
	MsSqlFacet                // MS SQL db facet
	OracleFacet               // Oracle db facet
	Db2Facet                  // DB2 db facet
)

// String is default printable value of db facet, Stringer implementation
func (facet Facet) String() string {
	switch facet {
	case DefaultFacet:
		return "Default db facet"
	case SqliteFacet:
		return "Sqlite db facet"
	case PgSqlFacet:
		return "PostgreSQL db facet"
	case MySqlFacet:
		return "MySQL db facet"
	case MsSqlFacet:
		return "MS SQL db facet"
	case OracleFacet:
		return "Oracle db facet"
	case Db2Facet:
		return "DB2 db facet"
	}
	return "Unknown db facet"
}

// bigintType return type name for BIGINT sql type
func (facet Facet) bigintType() string {
	if facet == OracleFacet {
		return "NUMBER(19)"
	}
	return "BIGINT"
}

// floatType return type name for FLOAT standard sql type
func (facet Facet) floatType() string {
	if facet == OracleFacet {
		return "BINARY_DOUBLE"
	}
	return "FLOAT"
}

// textType return column type DDL for long VARCHAR columns, use it for len > 255.
func (facet Facet) textType(len int) string {
	switch facet {
	case MsSqlFacet:
		if len > 4000 {
			return "TEXT"
		}
	case OracleFacet:
		if len > 2000 {
			return "CLOB"
		}
	}
	return "VARCHAR(" + strconv.Itoa(len) + ")"
}

// createTableIfNotExist return sql statement to create table if not exists
func (facet Facet) createTableIfNotExist(tableName string, bodySql string) string {

	switch facet {
	case SqliteFacet, PgSqlFacet, MySqlFacet:
		return "CREATE TABLE IF NOT EXISTS " + tableName + " " + bodySql
	case MsSqlFacet:
		return "IF NOT EXISTS" +
			" (SELECT * FROM INFORMATION_SCHEMA.TABLES T WHERE T.TABLE_NAME = " + ToQuoted(tableName) + ") " +
			" CREATE TABLE " + tableName + " " + bodySql
	}
	return "CREATE TABLE " + tableName + " " + bodySql
}

// createViewIfNotExist return sql statement to create view if not exists
func (facet Facet) createViewIfNotExist(viewName string, bodySql string) string {

	switch facet {
	case SqliteFacet:
		return "CREATE VIEW IF NOT EXISTS " + viewName + " AS " + bodySql
	case PgSqlFacet, MySqlFacet:
		return "CREATE OR REPLACE VIEW " + viewName + " AS " + bodySql
	case MsSqlFacet:
		return "CREATE VIEW " + viewName + " AS " + bodySql
	case OracleFacet, Db2Facet:
		return "CREATE OR REPLACE VIEW " + viewName + " AS " + bodySql
	}
	return "CREATE VIEW " + viewName + " AS " + bodySql
}

// detectFacet obtains db facet by quiering sql server.
// It may not be always reliable and even not true facet.
// It is better to use driver information to determine db facet.
func detectFacet(dbConn *sql.DB) Facet {

	facet := DefaultFacet

	// check is it PostgreSQL
	// check is it MySQL (not reliable) or MariaDB
	// odbc driver bug (?): PostgreSQL 9.2 + odbc 9.5.400 fails forever after first query failed
	// that means PostgreSQL facet detection must be first
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
					facet = PgSqlFacet
				}
				if facet == DefaultFacet &&
					(strings.Contains(v, "mysql") || strings.Contains(v, "mariadb") || strings.HasPrefix(v, "5.")) {
					facet = MySqlFacet
				}
			}
			return nil
		})
	if facet != DefaultFacet {
		return facet
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
				facet = SqliteFacet
			}
			return nil
		})
	if facet != DefaultFacet {
		return facet
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
				facet = MsSqlFacet
			}
			return nil
		})
	if facet != DefaultFacet {
		return facet
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
				facet = OracleFacet
			}
			return nil
		})
	if facet != DefaultFacet {
		return facet
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
				facet = Db2Facet
			}
			return nil
		})
	return facet
}
