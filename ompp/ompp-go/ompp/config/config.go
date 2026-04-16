// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

/*
Package config to merge run options: command line arguments and ini-file content.
Command line arguments take precedence over ini-file.
*/
package config

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/stcopenmpp/ompp-go/ompp/helper"
)

// Standard config keys to get values from ini-file or command line arguments
const (
	IniFile      = "OpenM.IniFile" // ini-file path
	IniFileShort = "ini"           // ini-file path (short form)
)

/*
Log config keys.
Log can be enabled/disabled for two independent streams:

	console  => standard output stream
	log file => log file, truncated on every run, (optional) unique "stamped" name

"Stamped" file name produced by adding time-stamp and/or pid-stamp, i.e.:

	exeName.log => exeName.2012_08_17_16_04_59_148.123.log
*/
const (
	LogToConsoleArgKey   = "OpenM.LogToConsole"     // if true then log to standard output
	LogToConsoleShortKey = "v"                      // if true then log to standard output (short form)
	LogToFileArgKey      = "OpenM.LogToFile"        // if true then log to file
	LogFilePathArgKey    = "OpenM.LogFilePath"      // log file path, default = current/dir/exeName.log
	LogUseTsArgKey       = "OpenM.LogUseTimeStamp"  // if true then use time-stamp in log file name
	LogUsePidArgKey      = "OpenM.LogUsePidStamp"   // if true then use pid-stamp in log file name
	LogUseDailyArgKey    = "OpenM.LogUseDailyStamp" // if true then use daily-stamp in log file name
	LogSqlArgKey         = "OpenM.LogSql"           // if true then log sql statements into log file
)

// RunOptions is (key,value) map of command line arguments and ini-file.
// For ini-file options key is combined as section.key
type RunOptions struct {
	KeyValue        map[string]string // (key=>value) from command line arguments and ini-file
	DefaultKeyValue map[string]string // default (key=>value), if non-empty default for command line argument
	Ini             []helper.IniEntry // options as ini (section, key) => value
	iniPath         string            // path to ini-file
}

// LogOptions for console and log file output
type LogOptions struct {
	LogPath   string // path to log file
	IsConsole bool   // if true then log to standard output, default: true
	IsFile    bool   // if true then log to file
	IsLogSql  bool   // if true then log sql statements
	TimeStamp string // log timestamp string, ie: 2012_08_17_16_04_59_148
	IsDaily   bool   // if true then use daily log file names, ie: exeName.20120817.log
}

// FullShort is pair of full option name and short option name
type FullShort struct {
	Full  string // full option name
	Short string // short option name
}

// New combines command-line arguments and ini-file options.
//
// encodingKey, if not empty, is a name of command-line option
// to specify encoding (code page) of source text files,
// for example: -dbcopy.CodePage=windows-1252.
// If encoding value specified then ini-file and csv files converted from such encoding to utf-8.
// If encoding not specified then auto-detection and default values are used (see helper.FileToUtf8()).
// If isExtra=true then allow unknown extra keys in ini-file
// otherwise all iini file keys must be defined as flag keys.
//
// Return
// 1. *RunOptions: is merge of command line key=value and ini-file section.key=value options.
// 2. *LogOptions: openM++ log file settings, also merge of command line and ini-file.
// 3. error or nil on success
func New(encodingKey string, isExtra bool, optFs []FullShort) (*RunOptions, *LogOptions, error) {

	runOpts := &RunOptions{
		KeyValue:        make(map[string]string),
		DefaultKeyValue: make(map[string]string),
		Ini:             []helper.IniEntry{},
	}
	logOpts := &LogOptions{
		IsConsole: true,
		TimeStamp: helper.MakeTimeStamp(time.Now()),
	}

	addStandardFlags(runOpts, logOpts) // add "standard" config options

	// parse command line arguments
	flag.Parse()
	ea := flag.Args()
	if len(ea) > 0 {
		return nil, nil, errors.New("Invalid arguments: " + strings.Join(ea, " "))
	}

	// retrive encoding name from command line
	encName := ""
	if encodingKey != "" {
		if f := flag.Lookup(encodingKey); f != nil {
			encName = f.Value.String()
		}
	}

	// parse ini-file using encoding, if it is not empty
	opts, err := FromIni(runOpts.iniPath, encName)
	if err != nil {
		return nil, nil, err
	}
	runOpts.KeyValue = opts.KeyValue
	runOpts.Ini = opts.Ini

	// validate ini-file flags: all keys should be defined flag names
	if !isExtra {
		for key := range opts.KeyValue {
			if f := flag.Lookup(key); f == nil {
				return nil, nil, errors.New("Invalid ini file section.key: " + key)
			}
		}
	}

	// override ini-file values with command-line arguments
	flag.Visit(func(f *flag.Flag) {
		if f.Name == IniFile || f.Name == IniFileShort {
			runOpts.KeyValue[IniFile] = runOpts.iniPath
			return
		}
		if f.Name == LogToConsoleArgKey || f.Name == LogToConsoleShortKey {
			runOpts.KeyValue[LogToConsoleArgKey] = strconv.FormatBool(logOpts.IsConsole)
			return
		}
		for _, fs := range optFs {
			if f.Name == fs.Full || f.Name == fs.Short {
				runOpts.KeyValue[fs.Full] = f.Value.String()
				return
			}
		}
		runOpts.KeyValue[f.Name] = f.Value.String()
	})

	// set default (key,value) from flag defaults if not empty
	flag.VisitAll(func(f *flag.Flag) {
		if f.DefValue == "" {
			return
		}
		n := f.Name
		if n == IniFileShort {
			n = IniFile
		}
		if n == LogToConsoleShortKey {
			n = LogToConsoleArgKey
		}
		for _, fs := range optFs {
			if n == fs.Short {
				n = fs.Full
			}
		}
		if runOpts.DefaultKeyValue[n] == "" {
			runOpts.DefaultKeyValue[n] = f.DefValue
		}
	})

	// adjust log settings
	adjustLogOptions(runOpts, logOpts)

	// update ini values from new key-value
	for key, val := range runOpts.KeyValue {

		sck := strings.SplitN(key, ".", 2) // expected section.key
		if len(sck) < 2 || sck[0] == "" || sck[1] == "" {
			continue // skip invalid section.key
		}
		runOpts.Ini = helper.MergeIniEntry(runOpts.Ini, sck[0], sck[1], val)
	}

	return runOpts, logOpts, nil
}

// FromIni read ini-file options.
//
// encodingName, if not empty then it is a "code page" to convert source file into utf-8, for example: windows-1252
//
// Return
// 1. *RunOptions: is (key, value) pairs from ini-file: section.key=value
// 2. error or nil on success
func FromIni(iniPath string, encodingName string) (*RunOptions, error) {

	runOpts := &RunOptions{
		KeyValue:        map[string]string{},
		DefaultKeyValue: map[string]string{},
		Ini:             []helper.IniEntry{},
	}

	// parse ini-file using encoding, if it is not empty
	eaIni, err := NewIni(iniPath, encodingName)
	if err != nil {
		return nil, nil
	}
	runOpts.Ini = eaIni

	for _, e := range eaIni {
		runOpts.KeyValue[e.Section+"."+e.Key] = e.Val
	}
	return runOpts, nil
}

// IsExist return true if key is defined as command line argument or ini-file option.
func (opts *RunOptions) IsExist(key string) bool {
	if opts == nil || opts.KeyValue == nil {
		return false
	}
	_, ok := opts.KeyValue[key]
	return ok
}

// String return value by key.
// It can be defined as command line argument or ini-file option or command line default
func (opts *RunOptions) String(key string) string {
	val, _, _ := opts.StringExist(key)
	return val
}

// StringExist return value by key and boolean flags:
// isExist=true if value defined as command line argument or ini-file option,
// isDefault=true if value defined as non-empty default for command line argument.
func (opts *RunOptions) StringExist(key string) (val string, isExist, isDefaultArg bool) {
	if opts == nil || opts.KeyValue == nil {
		return "", false, false
	}
	if val, isExist = opts.KeyValue[key]; isExist {
		return val, isExist, false
	}

	val, isDefaultArg = opts.DefaultKeyValue[key]
	return val, false, isDefaultArg
}

// Bool return boolean value by key.
// If value not defined by command line argument or ini-file option
// or cannot be converted to boolean (see strconv.ParseBool) then return false
func (opts *RunOptions) Bool(key string) bool {
	sVal, isExist, _ := opts.StringExist(key)
	if !isExist || sVal == "" {
		return false
	}
	if val, err := strconv.ParseBool(sVal); err == nil {
		return val
	}
	return false
}

// Int return integer value by key.
// If value not defined by command line argument or ini-file option
// or cannot be converted to integer then default is returned
func (opts *RunOptions) Int(key string, defaultValue int) int {
	sVal, isExist, _ := opts.StringExist(key)
	if !isExist || sVal == "" {
		return defaultValue
	}
	if val, err := strconv.Atoi(sVal); err == nil {
		return val
	}
	return defaultValue
}

// Int64 return 64 bit integer value by key.
// If value not defined by command line argument or ini-file option
// or cannot be converted to int64 then default is returned
func (opts *RunOptions) Int64(key string, defaultValue int64) int64 {
	sVal, isExist, _ := opts.StringExist(key)
	if !isExist || sVal == "" {
		return defaultValue
	}
	if val, err := strconv.ParseInt(sVal, 0, 64); err == nil {
		return val
	}
	return defaultValue
}

// Uint64 return unsigned 64 bit integer value by key.
// If value not defined by command line argument or ini-file option
// or cannot be converted to uint64 then default is returned
func (opts *RunOptions) Uint64(key string, defaultValue uint64) uint64 {
	sVal, isExist, _ := opts.StringExist(key)
	if !isExist || sVal == "" {
		return defaultValue
	}
	if val, err := strconv.ParseUint(sVal, 0, 64); err == nil {
		return val
	}
	return defaultValue
}

// Float return 64 bit float value by key.
// If value not defined by command line argument or ini-file option
// or cannot be converted to float64 then default is returned
func (opts *RunOptions) Float(key string, defaultValue float64) float64 {
	sVal, isExist, _ := opts.StringExist(key)
	if !isExist || sVal == "" {
		return defaultValue
	}
	if val, err := strconv.ParseFloat(sVal, 64); err == nil {
		return val
	}
	return defaultValue
}

// add "standard" config options to command line arguments
func addStandardFlags(runOpts *RunOptions, logOpts *LogOptions) {

	flag.StringVar(&runOpts.iniPath, IniFile, "", "path to `ini-file`")
	flag.StringVar(&runOpts.iniPath, IniFileShort, "", "path to `ini-file` (short of "+IniFile+")")

	// add log options to command line arguments
	flag.BoolVar(&logOpts.IsConsole, LogToConsoleArgKey, true, "if true then log to standard output")
	flag.BoolVar(&logOpts.IsConsole, LogToConsoleShortKey, true, "if true then log to standard output (short of "+LogToConsoleArgKey+")")
	flag.BoolVar(&logOpts.IsFile, LogToFileArgKey, false, "if true then log to file")
	flag.StringVar(&logOpts.LogPath, LogFilePathArgKey, "", "path to log file")
	_ = flag.Bool(LogUseTsArgKey, false, "if true then use time-stamp in log file name")
	_ = flag.Bool(LogUsePidArgKey, false, "if true then use pid-stamp in log file name")
	_ = flag.Bool(LogUseDailyArgKey, false, "if true then use daily-stamp in log file name")
	flag.BoolVar(&logOpts.IsLogSql, LogSqlArgKey, false, "if true then log sql statements into log file")
}

// adjust log settings by merging command line arguments and ini-file options
// make sure if LogToFile then log file path is defined and vice versa
// make "stamped" log file name, if required, by adding time-stamp and/or pid-stamp, i.e.:
//
//	exeName.log => exeName.2012_08_17_16_04_59_148.123.log
func adjustLogOptions(runOpts *RunOptions, logOpts *LogOptions) {

	// if log file path is not empty then LogToFile must be true
	if logOpts.LogPath != "" || logOpts.IsFile || runOpts.Bool(LogToFileArgKey) || runOpts.Bool(LogSqlArgKey) {
		logOpts.IsFile = true
		runOpts.KeyValue[LogToFileArgKey] = strconv.FormatBool(logOpts.IsFile)
	}

	// if LogToFile is true then log file path must not be empty
	if logOpts.IsFile && logOpts.LogPath == "" {

		logOpts.LogPath = runOpts.String(LogFilePathArgKey) // use log file path from ini-file

		// use exeName.log as default
		if logOpts.LogPath == "" {
			_, exeName := filepath.Split(os.Args[0])
			ext := filepath.Ext(exeName)
			if ext != "" {
				exeName = exeName[:len(exeName)-len(ext)]
			}
			logOpts.LogPath = exeName + ".log"
		}
	}

	// update log settings from merged command line arguments and ini-file
	logOpts.IsConsole = !runOpts.IsExist(LogToConsoleArgKey) || runOpts.Bool(LogToConsoleArgKey)
	logOpts.IsLogSql = runOpts.Bool(LogSqlArgKey)

	// update file name with time stamp and pid stamp, if required:
	// exeName.log => exeName.2012_08_17_16_04_59_148.123.log
	isTs := logOpts.IsFile && runOpts.Bool(LogUseTsArgKey)
	isPid := logOpts.IsFile && runOpts.Bool(LogUsePidArgKey)

	if isTs || isPid {

		dir, fName := filepath.Split(logOpts.LogPath)
		ext := filepath.Ext(fName)
		if ext != "" {
			fName = fName[:len(fName)-len(ext)]
		}
		if isTs {
			fName += "." + logOpts.TimeStamp
		}
		if isPid {
			fName += "." + strconv.Itoa(os.Getpid())
		}
		logOpts.LogPath = filepath.Join(dir, fName+ext)
	}
	runOpts.KeyValue[LogFilePathArgKey] = logOpts.LogPath // update value of log file name in run options

	// log daily option: enabled only if file log enabled and no time-stamp
	logOpts.IsDaily = logOpts.IsFile && !isTs && runOpts.Bool(LogUseDailyArgKey)
}
