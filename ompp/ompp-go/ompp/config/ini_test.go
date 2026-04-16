// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package config

import (
	"bufio"
	"io"
	"os"
	"strings"
	"testing"
	"unicode"

	"github.com/stcopenmpp/ompp-go/ompp/helper"
)

func TestIni(t *testing.T) {

	// load ini-file and compare content
	eaIni, err := NewIni("testdata/test.ompp.config.ini", "")
	if err != nil {
		t.Fatal(err)
	}
	// ini content loaded
	for k, e := range eaIni {
		t.Logf("[%d]: [%s]:%s=%s|", k, e.Section, e.Key, e.Val)
	}

	checkString := func(section, key, expected string) {

		for _, e := range eaIni {
			if e.Section == section && e.Key == key {
				if e.Val != expected {
					t.Errorf("[%s]%s=%s: NOT :%s:", section, key, expected, e.Val)
				}
				return // found section and key
			}
		}
		// not found section and key
		t.Errorf("not found [%s]:%s:", section, key)
	}

	checkString(`Test`, `non`, ``)
	checkString(`Test`, `rem`, ``)
	checkString(`Test`, `val`, `new value of no comments`)
	checkString(`Test`, `dsn`, `new value of UID='user'; PWD='secret';`)
	checkString(`Test`, `lst`, `new value of "the # quick" fox 'jumps # over'`)
	checkString(`Test`, `unb`, `"unbalanced quote                           ; this is not a comment: it is a value started from " quote`)

	checkString(`General`, `StartingSeed`, `16807`)
	checkString(`General`, `Subsamples`, `8`)
	checkString(`General`, `Cases`, `5000`)
	checkString(`General`, `SimulationEnd`, `100`)
	checkString(`General`, `UseSparse`, `true`)

	checkString(`multi`, `trim`, `Aname,Bname,Cname,DName`)
	checkString(`multi`, `keep`, `Multi line   text with spaces`)
	checkString(`multi`, `same`, `Multi line   text with spaces`)
	checkString(`multi`, `multi1`, `DSN='server'; UID='user'; PWD='secret';`)
	checkString(`multi`, `multi2`, `new value of "the # quick" fox "jumps # over"`)
	checkString(`multi`, `c-prog`, `C:\Program Files \Windows`)
	checkString(`multi`, `c-prog-win`, `C:\Program Files \Windows`)

	checkTheSame := func(section, key, keySame string) {

		v := ""
		ok := false
		for _, e := range eaIni {
			if ok = e.Section == section && e.Key == key; ok {
				v = e.Val
				break // found section and key
			}
		}
		if !ok {
			t.Errorf("not found [%s]:%s:", section, key)
		}

		vSame := ""
		ok = false
		for _, e := range eaIni {
			if ok = e.Section == section && e.Key == keySame; ok {
				vSame = e.Val
				break // found section and key
			}
		}
		if !ok {
			t.Errorf("not found [%s]:%s:", section, keySame)
		}
		if v != vSame {
			t.Errorf("NOT equal: [%s].%s and [%s].%s :%s: :%s:", section, key, section, keySame, v, vSame)
		}
	}
	checkTheSame(`multi`, `keep`, `same`)
	checkTheSame(`multi`, `c-prog`, `c-prog-win`)

	checkString(`replace`, `k`, `4`)

	checkString(`escape`, `dsn`, `DSN='server'; UID='user'; PWD='pas#word';`)
	checkString(`escape`, `t w`, `the "# quick #" brown 'fox ; jumps' over`)
	checkString(`escape`, ` key "" 'quoted' here `, `some value`)
	checkString(`escape`, `qts`, `" allow ' unbalanced quotes                 ; with comment`)

	checkString(`end`, `end`, ``)

	// merge KeyValue to ini file content
	eaIni = helper.MergeSectionKeyIniEntry(eaIni, "OpenM.IniFile", "OpenM/to/IniFile")
	eaIni = helper.MergeSectionKeyIniEntry(eaIni, "openm.IniFile", "openm/to/IniFile")
	eaIni = helper.MergeSectionKeyIniEntry(eaIni, "Log.Sql", "TRUE")
	eaIni = helper.MergeSectionKeyIniEntry(eaIni, "openm.inifile", "openm/to/inifile")
	eaIni = helper.MergeSectionKeyIniEntry(eaIni, "Log.Sql", "false")
	eaIni = helper.InsertIniEntry(eaIni, "Log", "Sql", "ERROR")

	t.Log("==== After MergeSectionKey ====")
	for k, e := range eaIni {
		t.Logf("[%d]: [%s]:%s=%s|", k, e.Section, e.Key, e.Val)
	}

	// verify ini file content
	// each line formatted as: [section]:key=value|

	fct, err := os.Open("testdata/test.ompp.config.ini-content.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer fct.Close()
	crd := bufio.NewReader(fct)

	// read log file, trim lines and skip empty lines
	fctLines := []string{}

	for {
		ln, e := crd.ReadString('\n')
		if e != nil {
			if e != io.EOF {
				t.Fatal(err)
			}
			break
		}
		ln = strings.TrimRightFunc(ln, func(r rune) bool { return unicode.IsSpace(r) || !unicode.IsPrint(r) })
		if ln != "" {

			n := len(fctLines)
			fctLines = append(fctLines, ln)

			// check if content is equal to ini file
			if n < 0 || n >= len(eaIni) {
				t.Errorf("Line %d not exists in ini file: %s", n, ln)
			} else {
				c := "[" + eaIni[n].Section + "]:" + eaIni[n].Key + "=" + eaIni[n].Val + "|"
				if ln != c {
					t.Errorf("%d: %s MUST: %s", n, c, ln)

				}
			}
		}
	}

	for k := len(fctLines); k < len(eaIni); k++ {
		c := "[" + eaIni[k].Section + "]:" + eaIni[k].Key + "=" + eaIni[k].Val + "|"
		t.Logf("Extra %d: %s", k, c)
	}
	if len(fctLines) != len(eaIni) {
		t.Errorf("Extra ini count %d", len(eaIni)-len(fctLines))
	}

	// verify ini sections
	// it must be in the same order

	scLst := helper.IniSectionList(eaIni)

	t.Log("==== Ini sections ====")

	for k := range scLst {
		t.Logf("[%d]: |%s|", k, scLst[k])
	}

	fsc, err := os.Open("testdata/test.ompp.config.ini-sections.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer fsc.Close()
	srd := bufio.NewReader(fsc)

	// read log file, trim lines and skip empty lines
	sctLines := []string{}

	for {
		ln, e := srd.ReadString('\n')
		if e != nil {
			if e != io.EOF {
				t.Fatal(err)
			}
			break
		}
		ln = strings.TrimRightFunc(ln, func(r rune) bool { return unicode.IsSpace(r) || !unicode.IsPrint(r) })
		if ln != "" {

			n := len(sctLines)
			sctLines = append(sctLines, ln)

			// check if content is equal to ini file section list
			if n < 0 || n >= len(scLst) {
				t.Errorf("Section %d not exists in ini file: %s", n, ln)
			} else {
				if ln != scLst[n] {
					t.Errorf("%d: |%s| MUST: |%s|", n, scLst[n], ln)

				}
			}
		}
	}

	for k := len(sctLines); k < len(scLst); k++ {
		t.Logf("Extra %d: |%s|", k, scLst[k])
	}
	if len(sctLines) < len(scLst) {
		t.Errorf("Extra sections count %d", len(scLst)-len(sctLines))
	}
	if len(sctLines) > len(scLst) {
		t.Errorf("Missing setions count %d", len(sctLines)-len(scLst))
	}
}
