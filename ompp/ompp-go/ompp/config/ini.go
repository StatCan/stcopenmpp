// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package config

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/stcopenmpp/ompp-go/ompp/helper"
)

/*
NewIni read ini-file content into  IniEntry[] of { Section, Key Value }

It is very light and able to parse:

	dsn = "DSN='server'; UID='user'; PWD='pas#word';"   ; comments are # here

Section and key are trimmed and cannot contain comments ; or # chars inside.
Key and values trimmed and "unquoted".
Key or value escaped with "double" or 'single' quotes can include spaces or ; or # chars

Example:

	; comments can start from ; or
	# from # and empty lines are skipped

	[section]  ; section comment
	val = no comment
	rem = ; comment only and empty value
	nul =
	dsn = "DSN='server'; UID='user'; PWD='pas#word';" ; quoted value
	t w = the "# quick #" brown 'fox ; jumps' over    ; escaped: ; and # chars
	" key "" 'quoted' here " = some value

	[multi-line]
	trim = Aname, \    ; multi-line value joined with spaces trimmed
	Bname, \    ; result is:
	CName       ; Aname,Bname,Cname

	; multi-line value started with " quote or ' apostrophe
	; right spaces before \ is not trimmed
	; result is:
	; Multi line   text with spaces
	;
	keep = "\
	       Multi line   \
	       text with spaces\
	       "
*/
func NewIni(iniPath string, encodingName string) ([]helper.IniEntry, error) {

	if iniPath == "" {
		return []helper.IniEntry{}, nil // no ini-file
	}

	// read ini-file and convert to utf-8
	s, err := helper.FileToUtf8(iniPath, encodingName)
	if err != nil {
		return []helper.IniEntry{}, errors.New("reading ini-file to utf-8 failed: " + err.Error())
	}

	// parse ini-file into strings map of (section.key)=>value
	eaIni, err := loadIni(s)
	if err != nil {
		return []helper.IniEntry{}, errors.New("reading ini-file failed: " + err.Error())
	}

	return eaIni, nil
}

// Parse ini-file content into strings map of (section.key)=>value
func loadIni(iniContent string) ([]helper.IniEntry, error) {

	eaIni := []helper.IniEntry{}

	var section, key, val, line string
	var isContinue, isQuote bool
	var cQuote rune

	for nLine, nStart := 0, 0; nStart < len(iniContent); {

		// get current line and move to next
		nextPos := strings.IndexAny(iniContent[nStart:], "\r\n")
		if nextPos < 0 {
			nextPos = len(iniContent)
		}
		if nStart+nextPos < len(iniContent)-1 {
			if iniContent[nStart+nextPos] == '\r' && iniContent[nStart+nextPos+1] == '\n' {
				nextPos++
			}
		}
		nextPos += 1 + nStart
		if nextPos > len(iniContent) {
			nextPos = len(iniContent)
		}

		line = strings.TrimSpace(iniContent[nStart:nextPos])
		nStart = nextPos
		nLine++

		// skip empty line and if it is end of continuation \ value then push it into ini content
		if len(line) <= 0 {

			if key != "" {
				eaIni = helper.MergeIniEntry(eaIni, section, key, helper.UnQuote(val))

				key, val, isContinue, isQuote, cQuote = "", "", false, false, 0 // reset state
			}
			continue // skip empty line
		}

		// remove ; comments or # Linux comments:
		//   comment starts with ; or # outside of "quote" or 'single quote'
		// get the key:
		//   find first = outside of "quote" or 'single quote'
		// get value:
		//    value can be after key= or entire line if it is a continuation \ value
		nEq := len(line) + 1
		nRem := len(line) + 1

		for k, c := range line {

			if !isQuote && (c == '"' || c == '\'') || isQuote && c == cQuote { // open or close quotes
				isQuote = !isQuote
				if isQuote {
					cQuote = c // opening quote
				} else {
					cQuote = 0 // quote closed
				}
				continue
			}
			if !isQuote && c == '=' && (nEq < 0 || nEq >= len(line)) { // positions of first key= outside of quote
				nEq = k
			}
			if !isQuote && (c == ';' || c == '#') { // start of comment outside of quotes
				nRem = k
				break
			}
		}

		// remove comment from the end of the line
		if nRem < len(line) {
			line = strings.TrimSpace(line[:nRem])
		}
		// skip line: it is a comment only line
		// if it is end of continuation \ value then push it into ini content
		if len(line) <= 0 {

			if key != "" {
				eaIni = helper.MergeIniEntry(eaIni, section, key, helper.UnQuote(val))
				key, val, isContinue, isQuote, cQuote = "", "", false, false, 0 // reset state
			}
			continue // skip line: it is a comment only line
		}

		// check for the [section]
		// section is not allowed after previous line \ continuation
		// section cannot have empty [] name
		if !isQuote {

			if line[0] == '[' {

				if len(line) < 3 || line[len(line)-1] != ']' {
					return nil, errors.New("line " + strconv.Itoa(nLine) + ": " + "invalid section name")
				}
				if isContinue {
					return nil, errors.New("line " + strconv.Itoa(nLine) + ": " + "invalid section name as line continuation")
				}

				section = strings.TrimSpace(line[1 : len(line)-1]) // [section] found
				continue
			}
		}
		if section == "" {
			return nil, errors.New("line " + strconv.Itoa(nLine) + ": " + "only comments or empty lines can be before first section")
		}

		// line is not empty: it must start with key= or it can be continuation of previous line value
		if key == "" {

			if nEq < 1 || nEq >= len(line) {
				return nil, errors.New("line " + strconv.Itoa(nLine) + ": " + "expected key=value")
			}
			key = helper.UnQuote(line[:nEq])
			line = strings.TrimSpace(line[nEq+1:])
		}
		if key == "" {
			return nil, errors.New("line " + strconv.Itoa(nLine) + ": " + "expected key=value")
		}

		// if line end with continuation \ then append line to value
		// else push key and value into ini content
		isContinue = len(line) > 0 && line[len(line)-1] == '\\'

		if isContinue {
			if isQuote {
				val = val + strings.TrimLeftFunc(line[:len(line)-1], unicode.IsSpace)
			} else {
				val = val + strings.TrimSpace(line[:len(line)-1])
			}
		} else {

			val = val + line
			eaIni = helper.MergeIniEntry(eaIni, section, key, helper.UnQuote(val))
			key, val, isContinue, isQuote, cQuote = "", "", false, false, 0 // reset state
		}
	}

	// last line: continuation at last line without cr-lf
	if isContinue && section != "" && key != "" {
		eaIni = helper.MergeIniEntry(eaIni, section, key, helper.UnQuote(val))
	}

	return eaIni, nil
}

// read common.message.ini file if exists in one of:
//
//	path/to/exe/common.message.ini
//	OM_ROOT/common.message.ini
//	OM_ROOT/models/common.message.ini
func ReadCommonMessageIni(exeDir string, encodingName string) ([]helper.IniEntry, error) {
	return ReadSharedMessageIni("common.message.ini", exeDir, encodingName)
}

// read commonName.message.ini file if exists in one of:
//
//	path/to/exe/commonName.message.ini
//	OM_ROOT/commonName.message.ini
//	OM_ROOT/models/commonName.message.ini
//
// if commonName is empty then return empty result
func ReadSharedMessageIni(sharedName, exeDir string, encodingName string) ([]helper.IniEntry, error) {

	if sharedName == "" {
		return []helper.IniEntry{}, nil
	}
	if cmIni, e := ReadMessageIni(sharedName, exeDir, encodingName); e == nil && len(cmIni) > 0 {
		return cmIni, e
	}
	if omroot := os.Getenv("OM_ROOT"); omroot != "" {
		if cmIni, e := ReadMessageIni(sharedName, omroot, encodingName); e == nil && len(cmIni) > 0 {
			return cmIni, e
		}
		return ReadMessageIni(sharedName, filepath.Join(omroot, "models"), encodingName)
	}
	return []helper.IniEntry{}, nil
}

// read path/to/exe/name.message.ini,
// it does not return error if message.ini not exist
func ReadMessageIni(name, dir string, encodingName string) ([]helper.IniEntry, error) {

	if name == "" {
		return []helper.IniEntry{}, nil // file name is empty
	}
	p := filepath.Join(dir, name+".message.ini")

	if !helper.IsFileExist(p) {
		return []helper.IniEntry{}, nil // message.ini not found
	}

	return NewIni(p, encodingName)
}
