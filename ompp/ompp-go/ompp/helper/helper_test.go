// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package helper

import "testing"

func TestParseKeyValue(t *testing.T) {

	// parse key=value string compare content
	kvStr := `DSN=server;UID=user;PWD=password`

	kv, err := ParseKeyValue(kvStr)
	if err != nil {
		t.Errorf(err.Error())
	}

	checkString := func(key, expected string) {
		val, ok := kv[key]
		if !ok {
			t.Errorf("not found: %s", key)
		}
		if val != expected {
			t.Errorf("%s=%s: NOT :%s:", key, expected, val)
		}
	}
	checkString("DSN", "server")
	checkString("UID", `user`)
	checkString("PWD", `password`)

	// extra semicolons and spaces and complex 'single' and "double" quotes
	kvStr = ` ; ;DSN='server' ; ;;UID='us""''er'; PWD=' pas;word ';`

	kv, err = ParseKeyValue(kvStr)
	if err != nil {
		t.Errorf(err.Error())
	}
	checkString("DSN", "server")
	checkString("UID", `us""''er`)
	checkString("PWD", ` pas;word `)

	// empty key= at the end of line
	kvStr = `abc=`

	kv, err = ParseKeyValue(kvStr)
	if err != nil {
		t.Errorf(err.Error())
	}
	checkString("abc", "")

	// unbalanced quotes at the end of line
	kvStr = `abc="unbalanced quoutes ;    `

	kv, err = ParseKeyValue(kvStr)
	if err != nil {
		t.Errorf(err.Error())
	}
	checkString("abc", `"unbalanced quoutes ;`)
}

func TestParseCvsLine(t *testing.T) {

	checkString := func(vals []string, idx int, expected string) {
		if idx < 0 || idx >= len(vals) {
			t.Errorf("value index: %d outside of range: %d", idx, len(vals))
		} else {
			if vals[idx] != expected {
				t.Errorf(":%s: NOT :%s:", vals[idx], expected)
			}
		}
	}
	checkSize := func(vLst []string, size int) {
		if len(vLst) != size {
			t.Errorf("Expected CSV values count: %d, actual: %d", size, len(vLst))
		}
	}

	// parse comma separated string and compare content
	src := `a,b,c`
	vLst := ParseCsvLine(src, 0)

	checkString(vLst, 0, "a")
	checkString(vLst, 1, "b")
	checkString(vLst, 2, "c")
	checkSize(vLst, 3)

	// quotted values and empty values
	src = `" 1 value , ", 2 value ,,   , ' 3 value, '`
	vLst = ParseCsvLine(src, 0)

	checkString(vLst, 0, " 1 value , ")
	checkString(vLst, 1, "2 value")
	checkString(vLst, 2, "")
	checkString(vLst, 3, "")
	checkString(vLst, 4, " 3 value, ")
	checkSize(vLst, 5)

	// empty csv line
	src = "  "
	vLst = ParseCsvLine(src, 0)

	checkSize(vLst, 0)

	// use ; semicolon separator and optional ; semicolon at the end of line
	src = ` 1 value ;  `
	vLst = ParseCsvLine(src, ';')

	checkString(vLst, 0, "1 value")
	checkSize(vLst, 1)

	// unbalanced quotes at the end of line
	src = ` "unbalanced quoutes , `
	vLst = ParseCsvLine(src, 0)

	checkString(vLst, 0, `"unbalanced quoutes ,`)
	checkSize(vLst, 1)
}
