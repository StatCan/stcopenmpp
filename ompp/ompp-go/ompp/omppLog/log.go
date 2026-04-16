// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

/*
Package omppLog to println messages to standard output and log file.
It is intended for progress or error logging and should not be used for profiling (it is slow).

Log can be enabled/disabled for two independent streams:

	console  => standard output stream
	log file => log file, truncated on every run, (optional) unique "stamped" name

"Stamped" file name produced by adding time-stamp and/or pid-stamp, i.e.:

	exeName.log => exeName.2012_08_17_16_04_59_148.123.log
*/
package omppLog

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
)

// log control state
var (
	theLock       sync.Mutex                           // mutex to lock log control state
	isFileEnabled bool                                 // if true then log to file enabled
	isFileCreated bool                                 // if true then log file created
	logPath       string                               // log file path
	lastYear      int                                  // if daily log then year of current daily stamp
	lastMonth     time.Month                           // if daily log then month current daily stamp
	lastDay       int                                  // if daily log then day current daily stamp
	logOpts       = config.LogOptions{IsConsole: true} // log options, default is log to console
)

// LogIfTime do Log(msg) not more often then every nSeconds.
// It does nothing if time now < lastT + nSeconds. Time is a Unix time, seconds since epoch.
func LogIfTime(lastT int64, nSeconds int64, msg ...any) int64 {

	now := time.Now().Unix()
	if now < lastT+nSeconds {
		return lastT // exit, period is not expired yet
	}
	Log(msg...) // log the message
	return now
}

// New log settings
func New(opts *config.LogOptions) {
	theLock.Lock()
	defer theLock.Unlock()

	if opts != nil {
		logOpts = *opts
	}
	isFileEnabled = logOpts.IsFile // file may be enabled but not created
	isFileCreated = false
}

// Log message to console and log file,
// put space between msg items,
// translate first msg[0] item using message.ini content,
// use language specific msgPrt printer instead of fmt by default
func Log(msg ...any) {
	now := time.Now()
	writeToLog(
		now,
		helper.MakeDateTime(now)+" "+helper.Msg(msg...),
	)
}

// Log message to console and log file,
// put space between msg items
func LogNoLT(msg ...any) {
	now := time.Now()
	writeToLog(
		now,
		helper.MakeDateTime(now)+" "+helper.MsgNoLT(msg...),
	)
}

// log formattted message to console and log file,
// translate format string using message.ini content,
// use language specific msgPrt printer instead of fmt by default
func LogFmt(format string, msg ...any) {
	if format == "" {
		Log(msg...) // ignore empty format to avoid ugly output
	} else {
		now := time.Now()
		writeToLog(
			now,
			helper.MakeDateTime(now)+" "+helper.Fmt(format, msg...),
		)
	}
}

// log formattted message to console and log file,
func LogFmtNoLT(format string, msg ...any) {
	if format == "" {
		LogNoLT(msg...) // ignore empty format to avoid ugly output
	} else {
		now := time.Now()
		writeToLog(
			now,
			helper.MakeDateTime(now)+" "+fmt.Sprintf(format, msg...),
		)
	}
}

// LogSql sql query to file
func LogSql(sql string) {
	theLock.Lock()
	defer theLock.Unlock()

	if !logOpts.IsLogSql { // exit if log sql not enabled
		return
	}

	// create log file if required log to file if file log enabled
	now := time.Now()
	if isFileEnabled &&
		(!isFileCreated ||
			logOpts.IsDaily && (now.Year() != lastYear || now.Month() != lastMonth || now.Day() != lastDay)) {
		isFileCreated = createLogFile(now)
		isFileEnabled = isFileCreated
	}
	if isFileEnabled {
		isFileEnabled = writeToLogFile(helper.MakeDateTime(now) + " " + sql)
	}
}

// create log file or truncate if already exist, return false on errors to disable file log
func createLogFile(nowTime time.Time) bool {

	// make log file path as log settings file path with daily-stamp, if daily stamp required
	logPath = logOpts.LogPath

	if logOpts.IsDaily {
		dir, fName := filepath.Split(logPath)
		ext := filepath.Ext(fName)
		if ext != "" {
			fName = fName[:len(fName)-len(ext)]
		}
		lastYear = nowTime.Year()
		lastMonth = nowTime.Month()
		lastDay = nowTime.Day()
		logPath = filepath.Join(dir, fName+"."+fmt.Sprintf("%04d%02d%02d", lastYear, lastMonth, lastDay)+ext)
	}

	// create log file or truncate existing
	f, err := os.Create(logPath)
	if err != nil {
		return false
	}
	defer f.Close()
	return true
}

// write log message to console and log file
func writeToLog(now time.Time, m string) {
	// log message to console
	theLock.Lock()
	defer theLock.Unlock()

	if logOpts.IsConsole {
		fmt.Println(m)
	}

	// create log file if required log to file if file log enabled
	if isFileEnabled &&
		(!isFileCreated ||
			logOpts.IsDaily && (now.Year() != lastYear || now.Month() != lastMonth || now.Day() != lastDay)) {
		isFileCreated = createLogFile(now)
		isFileEnabled = isFileCreated
	}
	if isFileEnabled {
		isFileEnabled = writeToLogFile(m)
	}
}

// write message to log file, return false on errors to disable file log
func writeToLogFile(msg string) bool {

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return false
	}
	defer f.Close()

	_, err = f.WriteString(msg)
	if err == nil {
		if runtime.GOOS == "windows" { // adjust newline for windows
			_, err = f.WriteString("\r\n")
		} else {
			_, err = f.WriteString("\n")
		}
	}
	return err == nil
}

// Load translated messages from language sections of exeName.message.ini go-common.message.ini files.
// It is a merge of exeName.message.ini and go.common.message.ini.
// If exeName is empty or exeName.message.ini file not exist then do nothing
func LoadMessageIni(exeName, exeDir string, userLang string, encodingName string) (string, []string, error) {

	// read message.ini file if exeName is not empty
	if exeName == "" {
		return userLang, []string{}, nil
	}

	msgIni := []helper.IniEntry{}
	var err error

	// read exeName.message.ini
	if helper.IsFileExist(filepath.Join(exeDir, exeName+".message.ini")) {
		msgIni, err = config.ReadMessageIni(exeName, exeDir, encodingName)
		if err != nil {
			return userLang, []string{}, err // errror at reading or parsing existing message.ini
		}
	}

	// read go-common.message.ini and merge with exe message.ini
	if cmIni, e := config.ReadSharedMessageIni("go-common", exeDir, encodingName); e == nil {
		for _, ea := range cmIni {
			msgIni = helper.InsertIniEntry(msgIni, ea.Section, ea.Key, ea.Val)
		}
	}
	if len(msgIni) <= 0 {
		return userLang, []string{}, nil // message.ini is empty
	}

	// update user language name and translation map
	uLang, langLst := helper.SetMsg(userLang, msgIni)
	return uLang, langLst, nil
}
