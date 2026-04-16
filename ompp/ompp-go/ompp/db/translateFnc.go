// Copyright (c) 2021 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

// left side name delimiters
const leftDelims = ",(+-*/%^|&~!=<>"

// right side name delimiters
const rightDelims = ")+-*/%^|&~!=<>["

// simple non-aggregation functions
var simpleFncLst = []string{"OM_IF", "OM_DIV_BY"}

// aggregation functions
var aggrFncLst = []string{"OM_AVG", "OM_SUM", "OM_COUNT", "OM_COUNT_IF", "OM_AVG", "OM_MIN", "OM_MAX", "OM_VAR", "OM_SD", "OM_SE", "OM_CV"}

// translate (substitute) all non-aggregation functions: OM_DIV_BY OM_IF...
func translateAllSimpleFnc(expr string) (string, error) {

	// do substitution of all non-aggregation functions by sql
	for {

		// find most left (top level) non-aggregation function
		fncName, namePos, arg, nAfter, err := findFirstFnc(expr, simpleFncLst)
		if err != nil {
			return "", err
		}
		if fncName == "" { // all done: no any functions found
			break
		}

		// translate (substitute) non-aggregation function by sql fragment
		t, err := translateSimpleFnc(fncName, arg, expr)
		if err != nil {
			return "", err
		}

		// replace source
		if nAfter >= len(expr) {
			expr = expr[:namePos] + t
		} else {
			expr = expr[:namePos] + t + expr[nAfter:]
		}
	}
	return expr, nil
}

// find first (left most) function in source expression.
// return function name, name position, argument and position after argumnet end.
func findFirstFnc(src string, fncNameLst []string) (string, int, string, int, error) {

	// find left most function name in source expression
	nFnc, namePos, err := findFirstNameFnc(src, fncNameLst)
	if err != nil {
		return "", 0, "", 0, err
	}
	if nFnc < 0 {
		return "", 0, "", 0, nil // not found
	}

	// find open and closing bracket at the same level
	nAfter := namePos + len(fncNameLst[nFnc])
	if nAfter >= len(src) {
		return "", 0, "", 0, errors.New("Error in expression, missing brackets after: " + fncNameLst[nFnc] + ": " + src)
	}
	level := 0
	nOpen := -1
	nClose := -1
	isInside := false

	for n, c := range src[nAfter:] {

		if !isInside && c == '(' { // open bracket: down to next level
			if level == 0 {
				nOpen = nAfter + n
			}
			level++
			continue
		}

		if !isInside && level <= 0 && !unicode.IsSpace(c) {
			return "", 0, "", 0, errors.New("Error in expression, missing brackets after: " + fncNameLst[nFnc] + ": " + src)
		}

		if !isInside && c == ')' { // close bracket: up to previous level
			level--
			if level < 0 {
				return "", 0, "", 0, errors.New("Error in expression, unbalanced brackets after: " + fncNameLst[nFnc] + ": " + src)
			}
			if level == 0 {
				nClose = nAfter + n
				break // done: close bracket found
			}
			continue
		}

		if c == '\'' {
			isInside = !isInside // begin or end of 'quoted' sql
		}
	}
	if level != 0 {
		return "", 0, "", 0, errors.New("Error in expression, unbalanced brackets after: " + fncNameLst[nFnc] + ": " + src)
	}
	if nOpen < nAfter || nClose < nAfter || nOpen >= nClose || nClose >= len(src) {
		return "", 0, "", 0, errors.New("Error in expression, missing brackets after: " + fncNameLst[nFnc] + ": " + src)
	}

	return fncNameLst[nFnc], namePos, src[nOpen+1 : nClose], nClose + 1, nil
}

// find first (left most) function name in src source expression from the fncNameLst name list.
// return index of function and name position.
func findFirstNameFnc(src string, fncNameLst []string) (int, int, error) {

	nFnc := -1
	namePos := len(src)

	for nf := 0; nf < len(fncNameLst); nf++ {

		// for each 'unquoted' part of source expression search for left most position of function name
		nFirst := 0
		var err error = nil

		for nLast := 0; nFirst >= 0 && nLast >= 0; {

			if nFirst, nLast, err = nextUnquoted(src, nFirst); err != nil {
				return -1, 0, err
			}
			if nFirst < 0 || nLast < 0 { // end of source sql string
				break
			}

			// find function name and delimited by space or open bracket
			nc := nFirst
			for n := nc; n < nLast; {

				n = strings.Index(src[nc:nLast], fncNameLst[nf])
				if n < 0 || n+nc >= namePos {
					break // skip: function name not found or it is not a left most
				}
				n += nc
				nl := len(fncNameLst[nf])

				// check if before function name is a space or left delimiter or name is at the beginning
				isOk := true
				if n > 0 {
					r, _ := utf8.DecodeRuneInString(src[n-1:])
					isOk = unicode.IsSpace(r) || strings.ContainsRune(leftDelims, r)
				}
				if !isOk {
					nc += nl
					continue // skip: it is not a function name
				}

				// if function name is at the end then skip it: it must be at least open ( bracket after
				if n+nl >= nLast {
					nc += nl
					continue // skip
				}

				// check if after function name is a space or right delimiter or name is at the end
				isOk = src[n+nl] == '('
				if !isOk {
					r, _ := utf8.DecodeRuneInString(src[n+nl:])
					isOk = unicode.IsSpace(r)
				}
				if !isOk {
					nc += nl
					continue // skip: it is not a function name
				}

				// found function name
				namePos = n
				nFnc = nf
				break
			}
			// found function name
			if nFnc >= 0 {
				break
			}

			nFirst = nLast // to the next 'unquoted' part of source expression
		}
	}
	return nFnc, namePos, nil
}

// translate (substitute) non-aggregation function:
//
//	OM_DIV_BY(acc1)
//	=>
//	CASE WHEN ABS(acc1) > 1.0e-37 THEN acc1 ELSE NULL END
//
//	OM_IF(acc1 > 1.5 THEN acc1 ELSE 1.5)
//	=>
//	CASE WHEN acc1 > 1.5 THEN acc1 ELSE 1.5 END
func translateSimpleFnc(name, arg string, src string) (string, error) {

	if len(arg) <= 0 {
		return "", errors.New("invalid (empty) function argument: " + name + " : " + src)
	}

	switch name {
	case "OM_IF":
		return "CASE WHEN " + arg + " END", nil

	case "OM_DIV_BY":
		return "CASE WHEN ABS(" + arg + ") > 1.0e-37 THEN " + arg + " ELSE NULL END", nil
	}
	return "", errors.New("unknown non-aggregation function: " + name + " : " + src)
}
