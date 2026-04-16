// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

const diskScanDefaultInterval = 383 * 1000 // timeout in msec, default value of sleep interval between scanning storage use
const minDiskScanInterval = 11 * 1000      // timeout in msec, minimum value to sleep between scanning storage use

// storage space use state
type diskUseState struct {
	IsOver        bool  // if true then storage use reach the limit
	diskUseConfig       // storage use settings
	AllSize       int64 // all oms instances size
	TotalSize     int64 // total size: models/bin size + download + upload
	BinSize       int64 // total models/bin size
	DbSize        int64 // total size of all db files
	DownSize      int64 // download total size
	UpSize        int64 // upload total size
	UpdateTs      int64 // info update time (unix milliseconds)
}

// model and database file info
type dbDiskUse struct {
	DbPath string // path to model.sqlite, relative to model root and slashed: dir/sub/model.sqlite
	Digest string // model digest
	Size   int64  // bytes, if non zero then model database file size
	ModTs  int64  // db file mod time info update time (unix milliseconds)
}

// storage usage control settings
type diskUseConfig struct {
	DiskScanMs   int64  // timeout in msec, sleep interval between scanning storage
	Limit        int64  // bytes, this instance storage limit
	AllLimit     int64  // bytes, total storage limit for all oms instances
	dbCleanupCmd string // path to database cleanup script
}

/*
scan all models directories, download and upload directories to collect storage space usage.

Other file statistics also collected, e.g. files count, SQLite db file size for each model, etc.
*/
func scanDisk(doneC <-chan bool, refreshC <-chan bool) {

	if !theCfg.isDiskUse {
		deleteDiskUseStateFiles() // storage use control disabled: delete disk usage state file and exit
		return
	}

	// if disk use file does not updated more than 3 times of scan interval (and minimum 1 minute) then oms instance is dead
	// disk use file: disk-#-_4040-#-size-#-100-#-ok-#-120-#-2022_07_08_23_45_12_123-#-125678.json
	diskUsePtrn := filepath.Join(theCfg.jobDir, "state", "disk-#-*-#-size-#-*-#-*-#-*-#-*-#-*.json")

	diskIniPath := filepath.Join(theCfg.etcDir, "disk.ini") // path to disk.ini: storage quotas and configuration
	var nOtherSize int64                                    // all other oms instances disk use size

	duState := diskUseState{
		diskUseConfig: diskUseConfig{
			DiskScanMs: diskScanDefaultInterval,
		},
	}
	dbUse := []dbDiskUse{}

	for {
		isOk, cfg := initDiskState(diskIniPath)
		if !isOk {
			if isExitSleep(diskScanDefaultInterval, doneC) { // wait for doneC or sleep
				return
			}
			continue
		}
		// clean previous state
		nowTime := time.Now()
		nowTs := nowTime.UnixMilli()

		duState = diskUseState{
			diskUseConfig: cfg,
			UpdateTs:      nowTs,
		}

		// find all disk use files
		// if disk use file does not updated more than 3 times of scan interval (and minimum 1 minute) then oms instance is dead
		minuteTs := nowTime.Add(-1 * time.Minute).UnixMilli()
		minTs := nowTime.Add(-1 * 3 * time.Duration(cfg.DiskScanMs) * time.Millisecond).UnixMilli()
		if minTs > minuteTs {
			minTs = minuteTs
		}

		// calculate total disk usage by all other oms instances
		nOtherSize = 0
		if theCfg.isJobControl {

			diskUseFiles := filesByPattern(diskUsePtrn, "Error at disk use files search")

			for _, fp := range diskUseFiles {

				oms, _, _, _, _, ts, ut, _ := parseDiskUseStatePath(fp)

				if oms == "" || oms == theCfg.omsName {
					continue // skip: invalid disk use state file path or it is current instance
				}
				if ts > minTs {
					nOtherSize += ut // oms instance is alive
				}
			}
		}

		// scan models/bin directory and for each .sqlite or .db file get database file size and mod time
		mDir, _ := theCatalog.getModelDir()

		duState.BinSize = 0
		dbUse = dbUse[:0]

		err := filepath.Walk(mDir, func(path string, fi fs.FileInfo, err error) error {
			if err != nil {
				omppLog.Log("Error at directory walk:", path, ":", err)
				return err
			}
			if fi.IsDir() {
				return nil
			}
			duState.BinSize = duState.BinSize + fi.Size() // total models/bin size

			isSqlite := strings.EqualFold(filepath.Ext(path), ".sqlite")
			if !isSqlite && !strings.EqualFold(filepath.Ext(path), ".db") {
				return nil
			}

			// it is a database file: get path, size and mod time
			rp, e := filepath.Rel(mDir, path)
			if e != nil {
				omppLog.Log("Error at directory walk:", path, ":", e)
				return e
			}
			nSize := fi.Size()

			dbUse = append(dbUse,
				dbDiskUse{
					DbPath: filepath.ToSlash(rp),
					Size:   nSize,
					ModTs:  fi.ModTime().UnixMilli(),
				})
			if isSqlite {
				duState.DbSize = duState.DbSize + nSize // total size of all model.sqlite files
			}
			return nil
		})
		if err != nil {
			// omppLog.Log("Error at directory walk:", folderPath, " :", err)
		}

		// for currently open model.sqlite databases set model digest
		mbs := theCatalog.allModels()

		for k := 0; k < len(mbs); k++ {
			j := slices.IndexFunc(
				dbUse, func(du dbDiskUse) bool { return du.DbPath == mbs[k].relPath },
			)
			if j >= 0 && j < len(dbUse) {
				dbUse[j].Digest = mbs[k].model.Digest
			}
		}

		// get total size of all files in the folder and sub-folders
		doTotalSize := func(folderPath string) int64 {

			var nTotal int64

			err := filepath.Walk(folderPath, func(path string, fi fs.FileInfo, err error) error {
				if err != nil {
					omppLog.Log("Error at directory walk:", path, ":", err)
					return err
				}
				if !fi.IsDir() {
					nTotal = nTotal + fi.Size()
				}
				return nil
			})
			if err != nil {
				// omppLog.Log("Error at directory walk:", folderPath, ":", err)
			}
			return nTotal
		}

		// total size of downlod and upload directories
		if theCfg.downloadDir != "" {
			duState.DownSize = doTotalSize(theCfg.downloadDir)
		}
		if theCfg.uploadDir != "" {
			duState.UpSize = doTotalSize(theCfg.uploadDir)
		}
		duState.TotalSize = duState.BinSize + duState.DownSize + duState.UpSize
		duState.AllSize = duState.TotalSize + nOtherSize

		// check if current disk usage reach the limit
		duState.IsOver = duState.Limit > 0 && duState.TotalSize >= cfg.Limit ||
			cfg.AllLimit > 0 && duState.AllSize >= cfg.AllLimit

		// update run catalog with current storage use state and save persistent part of the state
		theRunCatalog.updateDiskUse(&duState, dbUse)
		diskUseStateWrite(&duState, dbUse)

		// wait for doneC or sleep
		isExit := false
		select {
		case <-doneC:
			isExit = true
		case <-refreshC:
		case <-time.After(time.Duration(cfg.DiskScanMs) * time.Millisecond):
		}
		if isExit {
			break
		}
	}
}

// update run catalog with current disk use state
func (rsc *RunCatalog) updateDiskUse(duState *diskUseState, dbUse []dbDiskUse) {

	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	rsc.DiskUse = *duState

	// copy all db files disk usage, it can be not only model.sqlite but also model.db files
	rsc.DbDiskUse = rsc.DbDiskUse[:0]

	for _, du := range dbUse {
		rsc.DbDiskUse = append(rsc.DbDiskUse, du)
	}
}

// return disk use status: flag is disk use over limit and disk use config
func (rsc *RunCatalog) getDiskUseStatus() (bool, diskUseConfig) {

	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	return rsc.DiskUse.IsOver, rsc.DiskUse.diskUseConfig
}

// return copy of current disk use state
func (rsc *RunCatalog) getDiskUse() (diskUseState, []dbDiskUse) {

	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	duState := rsc.DiskUse

	dbUse := make([]dbDiskUse, len(rsc.DbDiskUse))
	copy(dbUse, rsc.DbDiskUse)

	return duState, dbUse
}

// read disk usage settings from disk.ini
func initDiskState(diskIniPath string) (bool, diskUseConfig) {

	cfg := diskUseConfig{DiskScanMs: diskScanDefaultInterval}

	// exit if disk.ini does not exists: return empty default configuration
	if diskIniPath == "" || !helper.IsFileExist(diskIniPath) {
		return true, cfg
	}

	opts, err := config.FromIni(diskIniPath, theCfg.encodingName)
	if err != nil {
		omppLog.LogNoLT(err)
		return false, cfg
	}

	cfg.DiskScanMs = 1000 * opts.Int64("Common.ScanInterval", 0)
	if cfg.DiskScanMs < minDiskScanInterval {
		cfg.DiskScanMs = diskScanDefaultInterval // if too low then use default
	}
	cfg.AllLimit = 1024 * 1024 * 1024 * opts.Int64("Common.AllUsersLimit", 0) // total limit in bytes for all oms instances
	if cfg.AllLimit < 0 {
		cfg.AllLimit = 0 // unlimited
	}
	cfg.dbCleanupCmd = opts.String("Common.DbCleanup")

	// find oms instance limit defined by name
	var uGb int64

	isOk := opts.IsExist(theCfg.omsName + ".UserLimit")
	if isOk {
		uGb = opts.Int64(theCfg.omsName+".UserLimit", 0) // limit defined for current instance name
	}

	if !isOk && opts.IsExist("Common.Groups") {

		gLst := helper.ParseCsvLine(opts.String("Common.Groups"), ',')

		for k := 0; !isOk && k < len(gLst); k++ {

			if !opts.IsExist(gLst[k]+".Users") || !opts.IsExist(gLst[k]+".UserLimit") {
				continue // skip: no users in that group or no user limit defined
			}
			uLst := helper.ParseCsvLine(opts.String(gLst[k]+".Users"), ',')

			for j := 0; !isOk && j < len(uLst); j++ {
				isOk = uLst[j] == theCfg.omsName // check if instance name exists in that group
			}
			if isOk {
				uGb = opts.Int64(gLst[k]+".UserLimit", 0) // group limit applied to the current instance
			}
		}
	}

	if !isOk {
		uGb = opts.Int64("Common.UserLimit", 0) // apply limit common for any user
	}
	if uGb < 0 {
		uGb = 0
	}
	cfg.Limit = 1024 * 1024 * 1024 * uGb // bytes, storage limit for current instance name

	return true, cfg
}

// Return db cleanup log file name and file path.
// Example of db cleanup file name: db-cleanup.2022_07_08_23_03_27_555.RiskPaths.console.txt
func dbCleanupLogNamePath(baseName, logDir string) (string, string) {

	ts, _ := theCatalog.getNewTimeStamp()
	fn := "db-cleanup." + ts + "." + baseName + ".console.txt"

	return fn, filepath.Join(logDir, fn)
}

// parse db cleanup log path:
// remove directory, remove db-cleanup. prefix, remove .console.txt extension.
// Return date-time stamp and db file name and log file name without directory.
func parseDbCleanupLogPath(srcPath string) (string, string, string) {

	_, fn := filepath.Split(srcPath)

	if !strings.HasPrefix(fn, "db-cleanup.") || !strings.HasSuffix(fn, ".console.txt") {
		return "", "", ""
	}
	p := fn[:len(fn)-len(".console.txt")]
	p = p[len("db-cleanup."):]

	// check result: it must 2 non-empty parts and first must be a time stamp
	sp := strings.SplitN(p, ".", 2)

	if len(sp) < 2 || !helper.IsUnderscoreTimeStamp(sp[0]) || sp[1] == "" {
		return "", "", "" // source file path is not db cleanup log file
	}
	return sp[0], sp[1], fn
}
