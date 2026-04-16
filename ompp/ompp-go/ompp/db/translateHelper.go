// Copyright (c) 2021 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// clean source comparison expression:
// replace cr or lf with space and
// replace all special sql ' quote characters with normal ' sql quote
func cleanSourceExpr(src string) string {

	var sb strings.Builder

	for _, c := range src {
		if c == '\r' || c == '\n' {
			c = '\x20'
		}
		if IsUnsafeQuote(c) {
			c = '\''
		}
		sb.WriteRune(c)
	}
	return sb.String()
}

// find name position in source expression or return -1 if not found.
// name must be delimited by space or left and right delimeters
func findNamePos(expr string, name string) int {

	n := strings.Index(expr, name)

	isOk := n >= 0
	if isOk && n > 0 {
		r, _ := utf8.DecodeRuneInString(expr[n-1:])
		isOk = unicode.IsSpace(r) || strings.ContainsRune(leftDelims, r)
	}
	if isOk && n+len(name) < len(expr) {
		r, _ := utf8.DecodeRuneInString(expr[n+len(name):])
		isOk = r == ',' || unicode.IsSpace(r) || strings.ContainsRune(rightDelims, r)
	}

	if !isOk {
		return -1
	}
	return n
}

// return error if sql contains unsafe sql keyword, semicolon ;  or -- comment or \ escape.
// unsafe sql keyword outside of 'quotes', for example:
// DELETE INSERT UPDATE CREATE DROP ALTER MERGE EXEC EXECUTE CALL GO
func errorIfUnsafeSqlOrComment(sql string) error {

	// until the end of source sql check if part of sql outside of 'quotes' is safe
	nStart := 0
	var err error = nil

	for nEnd := 0; nStart >= 0 && nEnd >= 0; {

		if nStart, nEnd, err = nextUnquoted(sql, nStart); err != nil {
			return err
		}
		if nStart < 0 || nEnd < 0 { // end of source sql string
			break
		}

		// check if there contains semicolon or comment
		if strings.Contains(sql[nStart:nEnd], ";") {
			return errors.New("Error in expression, semicolon found: " + sql)
		}
		if strings.Contains(sql[nStart:nEnd], "--") {
			return errors.New("Error in expression, SQL -- comment found: " + sql)
		}

		if strings.Contains(sql[nStart:nEnd], "\\") {
			return errors.New("Error in expression, SQL \\ escape sequence found: " + sql)
		}

		// check if there are any of sql keywords, which are not allowed
		if err = errorIfUnsafeSqlKeyword(sql[nStart:nEnd]); err != nil {
			return err
		}

		nStart = nEnd // to the next part of sql string
	}
	return nil
}

// return error if sql contains unsafe sql keyword outside of 'quotes', for example:
// DELETE INSERT UPDATE CREATE DROP ALTER MERGE EXEC EXECUTE CALL GO
func errorIfUnsafeSqlKeyword(sql string) error {

	unsafeSqlKeywords := [...]string{ // list is incomplete by nature
		"ABORT",
		"ALTER",
		"ATTACH",
		"CALL",
		"COMMIT",
		"CREATE",
		"DATABASE",
		"DELETE",
		"DETACH",
		"DISABLE",
		"DO",
		"DROP",
		"ENABLE",
		"EXEC",
		"EXECUTE",
		"GO",
		"GRANT",
		"IGNORE",
		"INDEX",
		"INSERT",
		"MERGE",
		"PROCEDURE",
		"QUERY",
		"RECURSIVE",
		"REFERENCES",
		"REINDEX",
		"RELEASE",
		"RENAME",
		"REPLACE",
		"RETURNING",
		"REVOKE",
		"ROLLBACK",
		"TABLE",
		"TRANSACTION",
		"TRIGGER",
		"TRUNCATE",
		"UPDATE",
		"VACUUM",
		"VIEW",
	}

	// check if there are any of sql keywords, which are not allowed
	s := strings.ToUpper(sql)

	for _, w := range unsafeSqlKeywords {

		nc := 0
		for n := nc; n < len(sql); {

			j := strings.Index(s[nc:], w)
			if j < 0 {
				break // keyword not found in string
			}
			n = nc + j + len(w) // next char position after keyword

			if n >= len(s) {
				return errors.New("Error in expression, unsafe SQL keyword: " + w + " : " + sql)
			}
			// else: it is not the end of string, check if next char is delimeter: space, math symbol, etc.

			ce, _ := utf8.DecodeRuneInString(s[n:])
			if unicode.IsSpace(ce) || unicode.IsPunct(ce) || unicode.IsControl(ce) || unicode.IsSymbol(ce) || unicode.IsMark(ce) {
				return errors.New("Error in expression, unsafe SQL keyword: " + w + " : " + sql)
			}

			nc = n // skip: it is not a keyword but a prefix
		}
	}
	return nil
}

// skip sql 'quoted' part of source string at start position, return position of first 'unquoted' sql.
func skipIfQuoted(src string, startPos int) (int, error) {

	if startPos >= len(src) {
		return startPos, nil
	}
	if startPos < 0 {
		return -1, errors.New("Internal error in expression parse: invalid string position: " + strconv.Itoa(startPos) + ": " + src)
	}

	isInside := false

	for k, c := range src[startPos:] {

		if !isInside && c != '\'' {
			return startPos + k, nil // done: return position of first char outside of 'quotes'
		}
		if c != '\'' {
			continue // skip: character inside 'quotes'
		}

		// else: this is begin or end of 'quoted' sql
		isInside = !isInside
	}

	// sql 'quotes' must be closed (paired)
	if isInside {
		return -1, errors.New("Error in expression, unbalanced SQL 'quotes' in: " + src)
	}

	// empty return: nothing after last closing 'quotes'
	return len(src), nil
}

// find next part of source string outside of sql 'quotes' at start position, return start and end position of 'unquoted' sql.
func nextUnquoted(src string, startPos int) (int, int, error) {

	if startPos >= len(src) {
		return startPos, -1, nil
	}
	if startPos < 0 {
		return -1, -1, errors.New("Internal error in expression parse: invalid string position: " + strconv.Itoa(startPos) + ": " + src)
	}

	nPos := startPos
	isInside := false

	for k, c := range src[startPos:] {

		if c != '\'' {
			continue
		}
		// else: this is sql ' quote: end or begin of quoted constant

		isInside = !isInside
		if !isInside {
			nPos = startPos + k + 1 // next char position after closing ' quote
			continue
		}
		// else start of 'quotes'

		if startPos+k > nPos { // found part of source string outside of sql 'quotes'
			return nPos, startPos + k, nil
		}
	}

	// sql 'quotes' must be closed (paired)
	if isInside {
		return -1, -1, errors.New("Error in expression, unbalanced SQL 'quotes' in: " + src)
	}

	// if there is any part of the string after last closing 'quote' then return it as result
	if nPos < len(src) {
		return nPos, len(src), nil
	}
	// else empty return: nothing after last closing 'quotes'
	return startPos, -1, nil
}
