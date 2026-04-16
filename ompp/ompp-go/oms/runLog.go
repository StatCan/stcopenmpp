// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"bufio"
	"io"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// get current run status and page of log file lines
func (rsc *RunCatalog) readModelRunLog(digest, stamp string, start, count int) (*RunStateLogPage, error) {

	// search for run status and log text by model digest and run stamp or submit stamp
	lrp, isFound := rsc.getRunStateLogPage(digest, stamp, start, count)
	if !isFound {
		return lrp, nil // model run not found in catalog
	}

	// check run status: if not completed then read most recent status from database
	isUpdated := false

	rl, isOk := theCatalog.RunRowList(digest, lrp.RunStamp)
	if isOk && len(rl) > 0 {

		// use most recent run with that run stamp
		r := rl[len(rl)-1]

		isUpdated = lrp.RunState.UpdateDateTime != r.UpdateDateTime || lrp.RunState.IsFinal != db.IsRunCompleted(r.Status)
		if isUpdated {
			lrp.RunState.UpdateDateTime = r.UpdateDateTime
			lrp.RunState.IsFinal = db.IsRunCompleted(r.Status)
		}
	}

	// if run updated or log lines are not in memory
	if isUpdated || lrp.RunState.IsLog && len(lrp.Lines) <= 0 {

		// read log file and select log page to return
		lines := []string{}
		if lrp.RunState.IsLog {

			ok := false
			if lines, ok = readLogFile(lrp.logPath); ok && len(lines) > 0 {
				lrp.Offset, lrp.Size, lrp.Lines = getLinesPage(start, count, lines) // make log page to return
				lrp.TotalSize = len(lines)
			}
		}
		// update model run catalog
		rsc.postRunStateLog(digest, lrp.RunStamp, lrp.IsFinal, lrp.UpdateDateTime, lines)
	}
	return lrp, nil
}

// get current run status and page of log lines and return true if run status found in catalog.
func (rsc *RunCatalog) getRunStateLogPage(digest, stamp string, start, count int) (*RunStateLogPage, bool) {

	lrp := &RunStateLogPage{
		Lines: []string{},
	}
	if digest == "" || stamp == "" {
		return lrp, false // empty model digest or stamp (run-or-submit stamp): exit
	}

	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	// find model run state by digest and run-or-submit stamp
	rsl := rsc.findRunStateLog(digest, stamp)
	if rsl == nil {
		return lrp, false // not found by run stamp or submit stamp: return empty result
	}

	// model run found: update log lines last used time, return run state and copy log lines
	rsl.logUsedTs = time.Now().Unix()

	lrp.RunState = rsl.RunState
	lrp.TotalSize = len(rsl.logLineLst)
	lrp.Offset, lrp.Size, lrp.Lines = getLinesPage(start, count, rsl.logLineLst)

	return lrp, true
}

// update model run catalog with new run state and log file lines.
func (rsc *RunCatalog) postRunStateLog(digest string, runStamp string, isFinal bool, updateDt string, logLines []string) {

	if digest == "" || runStamp == "" {
		return // model run undefined
	}

	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	// find model run state by digest and run stamp
	rsl := rsc.findRunStateLog(digest, runStamp)
	if rsl == nil {
		return // model digest or run stamp not found
	}

	// update model run state and append log message
	if !rsl.IsFinal {
		rsl.IsFinal = isFinal
	}
	if rsl.UpdateDateTime < updateDt {
		rsl.UpdateDateTime = updateDt
	}
	if len(logLines) > len(rsl.logLineLst) {
		rsl.logLineLst = logLines
		rsl.logUsedTs = time.Now().Unix()
	}
}

// read all non-empty text lines from log file.
func readLogFile(logPath string) ([]string, bool) {

	if logPath == "" || logPath == "." {
		return []string{}, false // empty log file path: exit
	}

	f, err := os.Open(logPath)
	if err != nil {
		return []string{}, false // cannot open log file for reading
	}
	defer f.Close()
	rd := bufio.NewReader(f)

	// read log file, trim lines and skip empty lines
	lines := []string{}

	for {
		ln, e := rd.ReadString('\n')
		if e != nil {
			if e != io.EOF {
				omppLog.Log("Error at reading log file: ", logPath)
			}
			break
		}
		ln = strings.TrimRightFunc(ln, func(r rune) bool { return unicode.IsSpace(r) || !unicode.IsPrint(r) })
		if ln != "" {
			lines = append(lines, ln)
		}
	}

	return lines, true
}

// getLinesPage return count pages from logLines starting at offset.
// if count <= 0 then return all lines starting at offset
func getLinesPage(offset, count int, logLines []string) (int, int, []string) {

	nTotal := len(logLines)
	if nTotal <= 0 {
		return 0, 0, []string{}
	}

	if offset < 0 {
		offset = 0
	}
	if offset >= nTotal {
		return offset, 0, []string{} // log offset (first line to read) past last log line
	}
	if count <= 0 || offset+count > nTotal {
		count = nTotal - offset
	}

	// copy log lines into result
	lines := make([]string, count)
	copy(lines, logLines[offset:offset+count])

	return offset, count, lines
}
