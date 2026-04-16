// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package helper

import (
	"bytes"
	"os"
	"testing"
)

// expected result
const expectedUtf8Test = "code_page(1252)\r\n\"grüßEN\";\r\n'FR', 'Français';\r\n\r\ncode_page(1251)\r\n\"О программе Dsm\"\r\n\"Copyright © AMC 1997\"\r\n"

func TestFileToUtf8(t *testing.T) {

	// compare result and report error
	checkString := func(name, val, expected string) {
		if val != expected {
			t.Errorf("%s: INVALID \n:%s:", name, val)
		}
	}

	// test: read file content to UTF-8 string
	s, err := FileToUtf8("testdata/tst_utf8_no_bom.txt", "")
	if err != nil {
		t.Errorf(err.Error())
	}
	checkString("UTF-8 auto detect: tst_utf8_no_bom.txt", s, expectedUtf8Test)

	s, err = FileToUtf8("testdata/tst_utf8_bom.txt", "")
	if err != nil {
		t.Errorf(err.Error())
	}
	checkString("UTF-8 BOM: tst_utf8_bom.txt", s, expectedUtf8Test)

	s, err = FileToUtf8("testdata/tst_utf16_LE.txt", "")
	if err != nil {
		t.Errorf(err.Error())
	}
	checkString("UTF-16LE: tst_utf16_LE.txt", s, expectedUtf8Test)

	s, err = FileToUtf8("testdata/tst_utf16_BE.txt", "")
	if err != nil {
		t.Errorf(err.Error())
	}
	checkString("UTF-16BE: tst_utf16_BE.txt", s, expectedUtf8Test)

	s, err = FileToUtf8("testdata/tst_utf32_LE.txt", "")
	if err != nil {
		t.Errorf(err.Error())
	}
	checkString("UTF-32LE: tst_utf32_LE.txt", s, expectedUtf8Test)

	s, err = FileToUtf8("testdata/tst_utf32_BE.txt", "")
	if err != nil {
		t.Errorf(err.Error())
	}
	checkString("UTF-32BE: tst_utf32_BE.txt", s, expectedUtf8Test)
}

func TestUtf8Reader(t *testing.T) {

	// compare result and report error
	checkString := func(name, val, expected string) {
		if val != expected {
			t.Errorf("%s: INVALID \n:%s:", name, val)
		}
	}

	// test: reader to UTF-8
	f1, err := os.Open("testdata/tst_win1252.txt")
	if err != nil {
		t.Errorf(err.Error())
	}
	defer f1.Close()

	rd1, err := Utf8Reader(f1, "windows-1252")
	if err != nil {
		t.Errorf(err.Error())
	}
	var buf bytes.Buffer
	buf.ReadFrom(rd1)
	buf.WriteString("\r\n")

	f2, err := os.Open("testdata/tst_win1251.txt")
	if err != nil {
		t.Errorf(err.Error())
	}
	defer f2.Close()

	rd2, err := Utf8Reader(f2, "windows-1251")
	if err != nil {
		t.Errorf(err.Error())
	}
	buf.ReadFrom(rd2)

	checkString("tst_win1252.txt + tst_win1251.txt:", buf.String(), expectedUtf8Test)
}
