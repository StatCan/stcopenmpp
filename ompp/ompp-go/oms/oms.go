// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

/*
oms is openM++ JSON web-service which is also used as simple web-server for openM++ UI html pages.

Web-service allow to view and update model database(s) and run openM++ models from models/bin subdirectory.
Web-server allow to serve static html (css, images, javascipt) content from html subdirectory.

Arguments for oms can be specified on command line or through .ini file:

oms -ini my-oms.ini
oms -OpenM.IniFile my-oms.ini

	Command line arguments take precedence over ini-file options.
	There are some arguments which can be specified through ini-file and not on command line,
	please see oms.ini file in our source code or our wiki for more details.

Following arguments supporetd by oms:

-l localhost:4040
-oms.Listen localhost:4040

	address to listen, default: localhost:4040.
	Use -l :4040 if you need to access oms web-service from other computer (make sure firewall configured properly).

-oms.UrlSaveTo some/dir/oms.url.txt

	file path to save oms URL which can be used to open web UI in browser.
	If paths is relative then must be relative to oms root directory.
	Default: empty value, URL is not saved in a file by default, example of URL file content: http://localhost:4040

-oms.PidSaveTo some/dir/oms.pid.txt

	file path to save oms process id which can be used to control or kill oms from scripts.
	If paths is relative then must be relative to oms root directory.
	Default: empty value, PID is not saved in a file by default.

-oms.RootDir ompp/root

	oms root directory, default: current directory.
	Recommended to have log/ subdirectory to store oms web-service log files.

-oms.ModelDir models/bin

	models executable and model.sqlite database files directory, default: models/bin,
	If relative then must be relative to oms root directory.

-oms.ModelLogDir models/log

	models log directory, default: models/log, if relative then must be relative to oms root directory.

-oms.ModelDocDir models/doc

	models documentation directory, default: models/doc, if relative then must be relative to oms root directory

-oms.HtmlDir html

	front-end UI directory, default: html.
	If relative then must be relative to oms root directory.
	It is not used if -oms.ApiOnly specified.

-oms.EtcDir etc

	configuration files directory, default: etc.
	If relative then must be relative to oms root directory.
	It is an optional directory, it may contain configuration files,for example, templates to run models on MPI cluster.

-oms.JobDir job

	jobs control directory.
	If relative then must be relative to oms root directory.
	Jobs control allow to manage computational resources (e.g. CPUs) and organize model run queue.
	Default value is empty "" string and it is disable jobs control.

-oms.Name someName

	instance name which used for job control.

-oms.HomeDir models/home

	user personal home directory to store files and settings.
	If relative then must be relative to oms root directory.
	Default value is empty "" string and it is disable use of home directory.

-oms.AllowDownload false

	if true then allow download from user home/io/download directory.

-oms.AllowUpload false

	if true then allow upload to user home/io/upload directory.

-oms.FilesDir

	user files directory, where user can store, upload and download ini-files or CSV files.
	If relative then must be relative to oms root directory.
	If user home directory specified then user files directory by default is home/io.

-oms.AllowMicrodata

	if true then allow model runs microdata usage else model microdata API disabled.

-oms.ApiOnly false

	if true then API only web-service, it is false by default and oms also act as http server for openM++ UI.

-oms.LogRequest false

	if true then log HTTP requests on console and/or log file.

-oms.NoAdmin

	if true then disable administrative routes: /admin/ and /admin-all/

-oms.NoShutdown

	if true then disable shutdown route: /shutdown/

-oms.AdminAll

	if true then allow global administrative routes: /admin-all/

-oms.Languages en

	comma-separated list of supported languages, default: current user OS language.
	That list is matched with request language list and model language list in order to return proper text results.

-OpenM.MessageLanguage fr-CA

	prefered oms output messages language, default: current user OS language.

-oms.DoubleFormat %.15g

	format to convert float or double value to string, default: %.15g.
	OpenM++ is using hash digest to compare models, input parameters and output values.
	By default float and double values converted into text with "%.15g" format.

-oms.CodePage

	"code page" to convert source file into utf-8, for example: windows-1252.
	It is used only for compatibility with old Windows files.

Also oms support OpenM++ standard log settings (described in openM++ wiki):

	-OpenM.LogToConsole:     if true then log to standard output, default: true
	-v:                      short form of: -OpenM.LogToConsole
	-OpenM.LogToFile:        if true then log to file
	-OpenM.LogFilePath:      path to log file, default = current/dir/exeName.log
	-OpenM.LogUseDailyStamp: if true then use dayily stamp in log file name (rotate log files dayily)
	-OpenM.LogUseTs:         if true then use time-stamp in log file name
	-OpenM.LogUsePid:        if true then use pid-stamp in log file name
	-OpenM.LogSql:           if true then log sql statements into log file
*/
package main

import (
	"context"
	"errors"
	"flag"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/husobee/vestigo"
	"github.com/jeandeaual/go-locale"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/text/language"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// config keys to get values from ini-file or command line arguments.
const (
	listenArgKey       = "oms.Listen"            // address to listen, default: localhost:4040
	listenShortKey     = "l"                     // address to listen (short form)
	omsNameArgKey      = "oms.Name"              // oms instance name, if empty then derived from address to listen
	urlFileArgKey      = "oms.UrlSaveTo"         // file path to save oms URL in form of: http://localhost:4040, if relative then must be relative to oms root directory
	pidFileArgKey      = "oms.PidSaveTo"         // file path to save oms processs ID, if relative then must be relative to oms root directory
	rootDirArgKey      = "oms.RootDir"           // oms root directory, expected to contain log subfolder
	modelDirArgKey     = "oms.ModelDir"          // models executable and model.sqlite directory, if relative then must be relative to oms root directory
	modelLogDirArgKey  = "oms.ModelLogDir"       // models log directory, if relative then must be relative to oms root directory
	modelDocDirArgKey  = "oms.ModelDocDir"       // models documentation directory, if relative then must be relative to oms root directory
	etcDirArgKey       = "oms.EtcDir"            // configuration files directory, if relative then must be relative to oms root directory
	htmlDirArgKey      = "oms.HtmlDir"           // front-end UI directory, if relative then must be relative to oms root directory
	jobDirArgKey       = "oms.JobDir"            // job control directory, if relative then must be relative to oms root directory
	homeDirArgKey      = "oms.HomeDir"           // user personal home directory, if relative then must be relative to oms root directory
	isDownloadArgKey   = "oms.AllowDownload"     // if true then allow download from user home sub-directory: home/io/download
	isUploadArgKey     = "oms.AllowUpload"       // if true then allow upload to user home sub-directory: home/io/upload
	filesDirArgKey     = "oms.FilesDir"          // user files directory, if relative then must be relative to oms root directory, if user home exists then: home/io
	isMicrodataArgKey  = "oms.AllowMicrodata"    // if true then allow model run microdata
	logRequestArgKey   = "oms.LogRequest"        // if true then log http request
	apiOnlyArgKey      = "oms.ApiOnly"           // if true then API only web-service, no web UI
	adminAllArgKey     = "oms.AdminAll"          // if true then allow global administrative routes: /admin-all/
	noAdminArgKey      = "oms.NoAdmin"           // if true then disable administrative routes: /admin/ and /admin-all/
	noShutdownArgKey   = "oms.NoShutdown"        // if true then disable shutdown route: /shutdown/
	uiLangsArgKey      = "oms.Languages"         // comma-separated list of supported languages
	msgLangArgKey      = "OpenM.MessageLanguage" // oms prefered output messages language, e.g. fr-CA
	encodingArgKey     = "oms.CodePage"          // code page for converting source files, e.g. windows-1252
	doubleFormatArgKey = "oms.DoubleFormat"      // format to convert float or double value to string, e.g. %.15g
)

// server run configuration
var theCfg = struct {
	rootDir      string            // server root directory
	htmlDir      string            // front-end UI directory with html and javascript
	etcDir       string            // configuration files directory
	omsBinDir    string            // path to bin directory where oms.exe and dbcopy.exe are located
	isHome       bool              // if true then it is a single user mode
	homeDir      string            // user home directory
	downloadDir  string            // if download allowed then it is home/io/download directory
	uploadDir    string            // if upload allowed then it is home/io/upload directory
	inOutDir     string            // if download or upload or user files allowed then it is home/io directory
	filesDir     string            // user files directory, if home directory specified then default is: home/io
	isMicrodata  bool              // if true then allow model run microdata
	docDir       string            // if not empty then models documentation directory, default: models/doc
	isJobControl bool              // if true then do job control: model run queue and resource allocation
	isJobPast    bool              // if true then do job history shadow copy
	isDiskUse    bool              // if true then control disk space usage, it enabled if etc/disk.ini exists
	isAdminAll   bool              // if true then allow global administrative routes: /admin-all
	jobDir       string            // job control directory
	omsName      string            // oms instance name, if empty then derived from address to listen
	dbcopyPath   string            // if download or upload allowed then it is path to dbcopy.exe
	doubleFmt    string            // format to convert float or double value to string
	encodingName string            // "code page" to convert source file into utf-8, for example: windows-1252
	env          map[string]string // server config environmemt variables to control UI
	uiExtra      string            // UI extra configuration from etc/ui.extra.json
}{
	htmlDir:      "html",
	etcDir:       "etc",
	isHome:       false,
	homeDir:      "",
	downloadDir:  "",
	uploadDir:    "",
	filesDir:     "",
	docDir:       "",
	isJobControl: false,
	jobDir:       "",
	omsName:      "",
	doubleFmt:    "%.15g",
	env:          map[string]string{},
}

// if true then log http requests
var isLogRequest bool

// matcher to find UI supported language corresponding to request
var uiLangMatcher language.Matcher

var refreshDiskScanC chan bool

// main entry point: wrapper to handle errors
func main() {
	defer exitOnPanic() // fatal error handler: log and exit

	err := mainBody(os.Args)
	if err != nil {
		omppLog.LogNoLT(err)
		os.Exit(1)
	}
	omppLog.Log("Done.") // compeleted OK
}

// actual main body
func mainBody(args []string) error {

	// set command line argument keys and ini-file keys
	_ = flag.String(listenArgKey, "localhost:4040", "address to listen")
	_ = flag.String(listenShortKey, "localhost:4040", "address to listen (short form of "+listenArgKey+")")
	_ = flag.String(urlFileArgKey, "", "file path to save oms URL, if relative then must be relative to root directory")
	_ = flag.String(rootDirArgKey, "", "root directory, default: current directory")
	_ = flag.String(modelDirArgKey, "models/bin", "models directory, if relative then must be relative to root directory")
	_ = flag.String(modelLogDirArgKey, "models/log", "models log directory, if relative then must be relative to root directory")
	_ = flag.String(modelDocDirArgKey, "models/doc", "models documentation directory, if relative then must be relative to root directory")
	_ = flag.String(etcDirArgKey, theCfg.etcDir, "configuration files directory, if relative then must be relative to root directory")
	_ = flag.String(htmlDirArgKey, theCfg.htmlDir, "front-end UI directory, if relative then must be relative to root directory")
	_ = flag.String(homeDirArgKey, "", "user personal home directory, if relative then must be relative to root directory")
	_ = flag.Bool(isDownloadArgKey, false, "if true then allow download from user home/io/download directory")
	_ = flag.Bool(isUploadArgKey, false, "if true then allow upload to user home/io/upload directory")
	_ = flag.String(filesDirArgKey, "", "user files directory, if home directory path specified then files directory is home/io")
	_ = flag.Bool(isMicrodataArgKey, false, "if true then allow model run microdata")
	_ = flag.String(jobDirArgKey, "", "job control directory, if relative then must be relative to root directory")
	_ = flag.String(omsNameArgKey, "", "instance name, automatically generated if empty")
	_ = flag.Bool(logRequestArgKey, false, "if true then log HTTP requests")
	_ = flag.Bool(apiOnlyArgKey, false, "if true then API only web-service, no web UI")
	_ = flag.Bool(adminAllArgKey, false, "if true then allow global administrative routes: /admin-all/")
	_ = flag.Bool(noAdminArgKey, false, "if true then disable administrative routes: /admin/ and /admin-all/")
	_ = flag.Bool(noShutdownArgKey, false, "if true then disable shutdown route: /shutdown/")
	_ = flag.String(uiLangsArgKey, "", "comma-separated list of supported languages")
	_ = flag.String(msgLangArgKey, "", "oms output messages language, e.g.: fr-CA, default: current user OS language")
	_ = flag.String(encodingArgKey, "", "code page to convert source file into utf-8, e.g.: windows-1252")
	_ = flag.String(doubleFormatArgKey, theCfg.doubleFmt, "format to convert float or double value to string")
	_ = flag.String(pidFileArgKey, "", "file path to save OMS process ID")

	// pairs of full and short argument names to map short name to full name
	optFs := []config.FullShort{
		{Full: listenArgKey, Short: listenShortKey},
	}

	// parse command line arguments and ini-file
	runOpts, logOpts, err := config.New(encodingArgKey, false, optFs)
	if err != nil {
		return helper.ErrorNew("Invalid arguments:", err)
	}
	isLogRequest = runOpts.Bool(logRequestArgKey)
	isApiOnly := runOpts.Bool(apiOnlyArgKey)
	theCfg.isMicrodata = runOpts.Bool(isMicrodataArgKey)
	isAdmin := !runOpts.Bool(noAdminArgKey)
	theCfg.isAdminAll = isAdmin && runOpts.Bool(adminAllArgKey)
	isShutdown := !runOpts.Bool(noShutdownArgKey)
	theCfg.doubleFmt = runOpts.String(doubleFormatArgKey)
	theCfg.encodingName = runOpts.String(encodingArgKey)

	// get server config environmemt variables and pass it to UI
	env := os.Environ()
	for _, e := range env {
		if strings.HasPrefix(e, "OM_CFG_") {
			if kv := strings.SplitN(e, "=", 2); len(kv) == 2 {
				theCfg.env[kv[0]] = kv[1]
			}
		}
	}

	// change to root directory
	selfPath, err := filepath.Abs(args[0])
	if err != nil {
		return helper.ErrorNew("Error: unable to make absolute path to oms:", err)
	}
	theCfg.omsBinDir = filepath.Dir(selfPath)

	// get prefered oms language
	mLang := runOpts.String(msgLangArgKey)

	if mLang == "" { // get default user language

		mLang, err = locale.GetLocale()
		if err != nil {
			omppLog.Log("Warning: unable to get user default language")
		}
	}

	// load translated strings from oms.message.ini
	_, iniLangs, err := omppLog.LoadMessageIni("oms", theCfg.omsBinDir, mLang, theCfg.encodingName)
	if err != nil {
		omppLog.LogNoLT("Error at loading oms.message.ini or go-common.message.ini:", err)
	}

	// set UI languages to find model text in browser language
	// check if oms.Languages option specified
	lNames := helper.ParseCsvLine(runOpts.String(uiLangsArgKey), ',')

	for _, sl := range iniLangs {
		if slices.IndexFunc(lNames, func(ln string) bool { return strings.EqualFold(ln, sl) }) < 0 {
			lNames = append(lNames, sl)
		}
	}
	var lt []language.Tag
	for _, ln := range lNames {
		if ln != "" {
			lt = append(lt, language.Make(ln))
		}
	}
	// add prefered language, it may OS language and not a user preferd language if oms running on the server
	if len(lt) <= 0 && mLang != "" {
		if pt := language.Make(mLang); pt != language.Und {
			lt = []language.Tag{pt}
		}

	}

	if len(lt) <= 0 {
		lt = []language.Tag{language.English} // use English if no other options
	}
	uiLangMatcher = language.NewMatcher(lt)

	// change to OM_ROOT directory
	theCfg.rootDir = filepath.Clean(runOpts.String(rootDirArgKey)) // OM_ROOT directory

	if theCfg.rootDir != "" && theCfg.rootDir != "." {
		if err := os.Chdir(theCfg.rootDir); err != nil {
			return helper.ErrorNew("Error: unable to change directory:", err)
		}
	}
	omppLog.New(logOpts) // adjust log options, log path can be relative to root directory

	if theCfg.rootDir != "" && theCfg.rootDir != "." {
		omppLog.Log("Change directory to:  ", theCfg.rootDir)
	}

	// model directory required to build initial list of model sqlite files
	modelDir := filepath.Clean(runOpts.String(modelDirArgKey))
	if modelDir == "" || modelDir == "." {
		modelDir = ""
		return helper.ErrorNew("Error: model directory argument cannot be empty or . dot")
	}
	omppLog.Log("Models directory:     ", modelDir)

	modelLogDir := filepath.Clean(runOpts.String(modelLogDirArgKey))
	modelLogDir = strings.TrimSuffix(modelLogDir, string(filepath.Separator))
	if modelLogDir != "" && modelLogDir != "." && helper.IsDirExist(modelLogDir) {
		omppLog.Log("Models log directory: ", modelLogDir)
	} else {
		modelLogDir = "" // dot . log directory does not allowed
	}

	if err := theCatalog.refreshSqlite(modelDir, modelLogDir); err != nil {
		return err
	}

	// check if model documentation directory exists
	theCfg.docDir = filepath.Clean(runOpts.String(modelDocDirArgKey))
	theCfg.docDir = strings.TrimSuffix(theCfg.docDir, string(filepath.Separator))

	if theCfg.docDir == "." || !helper.IsDirExist(theCfg.docDir) {
		omppLog.Log("Warning: model documentation directory not found or invalid:", theCfg.docDir)
		theCfg.docDir = ""
	} else {
		omppLog.Log("Models documentation: ", theCfg.docDir)
	}

	// check if it is single user run mode and use of home directory enabled
	if runOpts.IsExist(homeDirArgKey) {

		theCfg.homeDir = filepath.Clean(runOpts.String(homeDirArgKey))
		theCfg.homeDir = strings.TrimSuffix(theCfg.homeDir, string(filepath.Separator))

		if theCfg.homeDir == "." || !helper.IsDirExist(theCfg.homeDir) {
			omppLog.Log("Warning: user home directory not found or invalid:", theCfg.homeDir)
			theCfg.homeDir = ""
		}
		theCfg.isHome = theCfg.homeDir != ""

		if theCfg.isHome {
			omppLog.Log("User Home directory:  ", theCfg.homeDir)

			theCfg.inOutDir = filepath.Join(theCfg.homeDir, "io") // download and upload directory for web-server, to serve static content

			if theCfg.inOutDir == "." || !helper.IsDirExist(theCfg.inOutDir) {
				theCfg.inOutDir = ""
			}
		}
	}

	// check download and upload options:
	// home/io/download or home/io/upload directory must exist and dbcopy.exe must exist
	isDownload := false
	isUpload := false

	theCfg.dbcopyPath = dbcopyPath(theCfg.omsBinDir)

	if runOpts.Bool(isDownloadArgKey) {
		if theCfg.inOutDir != "" && theCfg.dbcopyPath != "" {

			theCfg.downloadDir = filepath.Join(theCfg.inOutDir, "download") // download directory UI

			if !helper.IsDirExist(theCfg.downloadDir) {
				theCfg.downloadDir = ""
			}
		}
		isDownload = theCfg.downloadDir != ""
		if !isDownload {
			theCfg.downloadDir = ""
			omppLog.Log("Warning: user home download directory not found or dbcopy not found, download disabled")
		} else {
			omppLog.Log("Download directory:   ", theCfg.downloadDir)
		}
	}
	if runOpts.Bool(isUploadArgKey) {
		if theCfg.inOutDir != "" && theCfg.dbcopyPath != "" {

			theCfg.uploadDir = filepath.Join(theCfg.inOutDir, "upload") // upload directory UI

			if !helper.IsDirExist(theCfg.uploadDir) {
				theCfg.uploadDir = ""
			}
		}
		isUpload = theCfg.uploadDir != ""
		if !isUpload {
			theCfg.uploadDir = ""
			omppLog.Log("Warning: user home upload directory not found or dbcopy not found, upload disabled")
		} else {
			omppLog.Log("Upload directory:     ", theCfg.uploadDir)
		}
	}

	// user files directory can be explicitly specified or be the home/io
	if runOpts.IsExist(filesDirArgKey) {

		theCfg.filesDir = filepath.Clean(runOpts.String(filesDirArgKey))
		theCfg.filesDir = strings.TrimSuffix(theCfg.filesDir, string(filepath.Separator))

		if theCfg.filesDir == "." || !helper.IsDirExist(theCfg.filesDir) {
			omppLog.Log("Warning: user files directory not found or invalid:", theCfg.filesDir)
			theCfg.filesDir = ""
		}
	} else {
		if theCfg.inOutDir != "" && (isDownload || isUpload) {
			theCfg.filesDir = theCfg.inOutDir
		}
	}
	if theCfg.filesDir != "" {
		omppLog.Log("User Files directory: ", theCfg.filesDir)
	}

	// if UI required then server root directory must have html subdir
	if !isApiOnly {
		theCfg.htmlDir = runOpts.String(htmlDirArgKey)
		if !helper.IsDirExist(theCfg.htmlDir) {
			isApiOnly = true
			omppLog.Log("Warning: serving API only because UI directory not found:", theCfg.htmlDir)
		} else {
			omppLog.Log("HTML UI directory:    ", theCfg.htmlDir)
		}
	}

	// etc subdirectory required to run MPI models
	theCfg.etcDir = runOpts.String(etcDirArgKey)
	if !helper.IsDirExist(theCfg.etcDir) {
		omppLog.Log("Warning: configuration files directory not found, it is required to run models on MPI cluster:", filepath.Join(theCfg.etcDir))
	} else {
		omppLog.Log("Etc directory:        ", theCfg.etcDir)
	}

	// read UI extra configuration from etc/ui.extra.json
	if bt, err := os.ReadFile(filepath.Join(theCfg.etcDir, "ui.extra.json")); err == nil {
		theCfg.uiExtra = string(bt)
	}

	// check if storage control enabled by presence of etc/disk.ini
	dini := filepath.Join(theCfg.etcDir, "disk.ini")
	theCfg.isDiskUse = helper.IsFileExist(dini)
	if theCfg.isDiskUse {
		omppLog.Log("Storage control:      ", dini)
	}

	// check if job control is required:
	theCfg.jobDir = runOpts.String(jobDirArgKey)
	theCfg.isJobControl, theCfg.isJobPast, err = jobDirValid(theCfg.jobDir)
	if err != nil {
		return helper.ErrorNew("Error: invalid job control directory:", err)
	}
	if !theCfg.isJobControl && theCfg.jobDir != "" {
		return helper.ErrorNew("Error: invalid job control directory:" + theCfg.jobDir)
	}
	if theCfg.isJobControl {
		omppLog.Log("Jobs directory:       ", theCfg.jobDir)
	}

	// make instance name, use address to listen if name not specified
	theCfg.omsName = runOpts.String(omsNameArgKey)
	if theCfg.omsName == "" {
		theCfg.omsName = runOpts.String(listenArgKey)
	}
	theCfg.omsName = helper.CleanFileName(theCfg.omsName)
	omppLog.Log("Oms instance name:    ", theCfg.omsName)

	// refresh run state catalog and start scanning model log files
	jsc, _ := jobStateRead()
	if err := theRunCatalog.refreshCatalog(theCfg.etcDir, jsc); err != nil {
		return err
	}

	doneModelRunScanC := make(chan bool)
	go scanModelRuns(doneModelRunScanC)

	// start scanning for model run jobs
	doneOuterJobScanC := make(chan bool)
	go scanOuterJobs(doneOuterJobScanC)

	doneStateJobScanC := make(chan bool)
	go scanStateJobs(doneStateJobScanC)

	doneRunJobScanC := make(chan bool)
	go scanRunJobs(doneRunJobScanC)

	doneDiskScanC := make(chan bool)
	refreshDiskScanC = make(chan bool)
	go scanDisk(doneDiskScanC, refreshDiskScanC)

	// setup router and start server
	router := vestigo.NewRouter()

	router.SetGlobalCors(&vestigo.CorsAccessControl{
		AllowOrigin:      []string{"*"},
		AllowCredentials: true,
		AllowHeaders:     []string{"Content-Type"},
		ExposeHeaders:    []string{"Content-Type", "Content-Location"},
	})

	apiGetRoutes(router)     // web-service /api routes to get metadata
	apiReadRoutes(router)    // web-service /api routes to read values
	apiReadCsvRoutes(router) // web-service /api routes to read values into csv stream
	if isDownload {
		apiDownloadRoutes(router) // web-service /api routes to download and manage files at home/io/download folder
	}
	if isUpload {
		apiUploadRoutes(router) // web-service /api routes to upload and manage files at home/io/upload folder
	}
	// disable user files downloads from home/io if download disabled
	if theCfg.filesDir != "" && (isDownload || theCfg.filesDir != theCfg.inOutDir) {

		router.Get("/files/*", filesHandler, logRequest) // serve static content at /files/ url from user files folders, default: home/io
		apiFilesRoutes(router)                           // web-service /api routes to upload and manage files at home/io/upload folder
	}
	apiUpdateRoutes(router)   // web-service /api routes to update metadata
	apiRunModelRoutes(router) // web-service /api routes to run the model
	apiUserRoutes(router)     // web-service /api routes for user-specific requests
	apiServiceRoutes(router)  // web-service /api routes for service state
	if isAdmin {
		apiAdminRoutes(router) // web-service /api routes for oms instance administrative tasks
	}

	// serve static content from home/io/download, home/io/upload, models/doc and user files folders
	if isDownload {
		router.Get("/download/*", downloadHandler, logRequest)
	}
	// serve static content from home/io/upload folder
	if isUpload {
		router.Get("/upload/*", downloadHandler, logRequest)
	}
	// serve static content from models/doc folder
	if theCfg.docDir != "" {
		router.Get("/doc/*", modelDocHandler, logRequest)
	}

	// set web root handler: UI web pages or "not found" if this is web-service mode
	if !isApiOnly {
		uiModelRoutes(router)                     // serve UI routes /model/*
		router.Get("/*", homeHandler, logRequest) // serve UI web pages
	} else {
		router.Get("/*", http.NotFound) // only /api, any other pages not found
	}

	// initialize server
	addr := runOpts.String(listenArgKey)
	srv := http.Server{Addr: addr, Handler: router}

	// add shutdown handler, it does not wait for requests, it does reset connections and exit
	// PUT /shutdown
	ctx, cancel := context.WithCancel((context.Background()))
	defer cancel()

	shutdownHandler := func(w http.ResponseWriter, r *http.Request) {

		// close models catalog
		omppLog.Log("Shutdown server...")
		if err := theCatalog.closeAll(); err != nil {
			omppLog.Log(err)
		}

		// send response: confirm shutdown
		srv.SetKeepAlivesEnabled(false)
		w.Write([]byte("Shutdown completed"))

		cancel() // send shutdown completed to the main
	}
	if isShutdown {
		router.Put("/shutdown", shutdownHandler, logRequest)
	}

	// start to listen at specified TCP address
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		omppLog.Log("Error at start of TCP listen:", addr)
		return err
	}
	ta, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		return errors.New("Error: unable to find TCP port of:" + addr)
	}
	localUrl := "http://localhost:" + strconv.Itoa(ta.Port)

	// if url file path specified then write oms url into that url file
	if urlFile := runOpts.String(urlFileArgKey); urlFile != "" {
		if err = os.WriteFile(urlFile, []byte(localUrl), 0644); err != nil {
			omppLog.Log("Error at writing into:", urlFile)
			return err
		}
	}

	// save oms process id into file
	if pidFile := runOpts.String(pidFileArgKey); pidFile != "" {
		pid := os.Getpid()
		if err = os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
			omppLog.Log("Error writing PID to file:", pidFile)
			return err
		}
	}

	// initialization completed, notify user and start the server
	omppLog.Log("Listen at ", addr)
	if !isApiOnly {
		omppLog.Log("To start open in your browser:", localUrl)
	}
	omppLog.Log("To finish press Ctrl+C")

	go func() {
		if err = srv.Serve(ln); err != nil {
			// send completed by error to the main
			// error may be http.ErrServerClosed by shutdown which is not an actual error
			cancel()
		}
	}()

	// wait for shutdown or Ctrl+C interupt signal
	<-ctx.Done()
	if e := srv.Shutdown(context.Background()); e != nil && e != http.ErrServerClosed {
		omppLog.LogNoLT("Shutdown error:", e)
	} else {
		// shutdown completed without error: clean main error code
		if err == http.ErrServerClosed {
			err = nil
		}
	}

	doneDiskScanC <- true
	doneRunJobScanC <- true
	doneStateJobScanC <- true
	doneOuterJobScanC <- true
	doneModelRunScanC <- true
	return err
}

// exitOnPanic log error message and exit with return = 2
func exitOnPanic() {
	r := recover()
	if r == nil {
		return // not in panic
	}
	switch e := r.(type) {
	case error:
		omppLog.LogNoLT(e)
	case string:
		omppLog.LogNoLT(e)
	default:
		omppLog.Log("FAILED")
	}
	os.Exit(2) // final exit
}
