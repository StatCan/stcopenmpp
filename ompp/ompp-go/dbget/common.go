// Copyright OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// return row []string or isEof = true
type rowConverter func() (isEof bool, row []string, err error)

// write into outputDir/file.json if jsonPath is "" empty then write into stdout
func toJsonOutput(jsonPath string, src interface{}) error {

	if jsonPath != "" {
		return helper.ToJsonIndentFile(jsonPath, src)
	}
	// else output to console
	ce := json.NewEncoder(os.Stdout)
	ce.SetIndent("", "  ")
	if err := ce.Encode(src); err != nil {
		return helper.ErrorNew("json encode error:", err)
	}
	return nil
}

// write into outputDir/file.csv if csvPath is "" empty then write into stdout
func toCsvOutput(csvPath string, columnNames []string, lineCvt rowConverter) error {

	// create csv file
	f, wr, err := createCsvWriter(csvPath)
	if err != nil {
		return err
	}
	isFile := f != nil

	defer func() {
		if isFile {
			f.Close()
		}
	}()

	// write header line: column names, if provided
	if len(columnNames) > 0 {
		if err = wr.Write(columnNames); err != nil {
			return err
		}
	}

	// write csv lines until eof
	for {
		isEof, row, err := lineCvt()
		if err != nil {
			return err
		}
		if isEof {
			break
		}
		if err = wr.Write(row); err != nil {
			return err
		}
	}

	// flush and return error, if any
	wr.Flush()
	return wr.Error()
}

// create csv or tsv output writer
func createCsvWriter(csvPath string) (*os.File, *csv.Writer, error) {

	// create csv file
	isFile := csvPath != ""
	var f *os.File
	var err error
	isClose := false

	if isFile {
		f, err = os.OpenFile(csvPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return nil, nil, err
		}
		isClose = true
	}
	defer func() {
		if isClose {
			f.Close()
		}
	}()

	if isFile && theCfg.isWriteUtf8Bom { // if required then write utf-8 bom
		if _, err = f.Write(helper.Utf8bom); err != nil {
			return nil, nil, err
		}
	}

	// create csv writes to file and/or to console
	var csvWr *csv.Writer
	if isFile {
		csvWr = csv.NewWriter(f)
	} else {
		csvWr = csv.NewWriter(os.Stdout)
		if runtime.GOOS == "windows" {
			csvWr.UseCRLF = true
		}
	}
	if theCfg.kind == asTsv {
		csvWr.Comma = '\t'
	}

	isClose = false // return open file to upper level

	return f, csvWr, nil
}

// if directory path not empty then create output directory if not already exists, remove existing directory if required
func makeOutputDir(path string, isKeep bool) error {

	if path != "" {
		if !isKeep {
			if isOk := dirDeleteAndLog(path); !isOk {
				return helper.ErrorNew("Error: unable to delete:", path)
			}
		}
		if err := os.MkdirAll(path, 0750); err != nil {
			return err
		}
	}
	return nil
}

// Delete directory and log path, return false on delete error.
func dirDeleteAndLog(path string) bool {

	if path == "" {
		return true // OK: nothing to delete
	}
	isExist, err := helper.IsDir(path)
	if err != nil {
		return false // error: path not accessible or it is not a directory
	}
	if !isExist {
		return true // OK: nothing to delete
	}

	omppLog.Log("Delete:", path)

	if e := os.RemoveAll(path); e != nil && !os.IsNotExist(e) {
		omppLog.Log(e)
		return false // error: delete failed
	}
	return true // OK: deleted successfully
}

// return file extension by output kind: .csv .tsv or .json
func extByKind() string {
	switch theCfg.kind {
	case asTsv:
		return ".tsv"
	case asJson:
		return ".json"
	}
	return ".csv" // by default
}

// return kind of by file extension: .csv .tsv or .json,
// if file path is empty or extension is unknown then return csv by default
func kindByExt(path string) outputAs {
	if path != "" {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".tsv":
			return asTsv
		case ".json":
			return asJson
		}
	}
	return asCsv // csv by default
}
