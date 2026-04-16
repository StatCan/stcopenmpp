// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"cmp"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/husobee/vestigo"
	"golang.org/x/text/language"

	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// PathItem contain basic file info after tree walk: relative path, size and modification time
type PathItem struct {
	Path    string // file path in / slash form
	IsDir   bool   // if true then it is a directory
	Size    int64  // file size (may be zero for directories)
	ModTime int64  // file modification time in milliseconds since epoch
}

// logRequest is a middelware to log http request
func logRequest(next http.HandlerFunc) http.HandlerFunc {
	if isLogRequest {
		return func(w http.ResponseWriter, r *http.Request) {
			omppLog.LogNoLT(r.Method, ":", r.Host, r.URL)
			next(w, r)
		}
	} // else
	return next
}

// get value of url parameter ?name or router parameter /:name
func getRequestParam(r *http.Request, name string) string {

	v := r.URL.Query().Get(name)
	if v == "" {
		v = vestigo.Param(r, name)
	}
	return v
}

// get boolean value of url parameter ?name or router parameter /:name
func getBoolRequestParam(r *http.Request, name string) (bool, bool) {

	v := r.URL.Query().Get(name)
	if v == "" {
		v = vestigo.Param(r, name)
	}
	if v == "" {
		return false, true // no such parameter: return = false by default
	}
	if isVal, err := strconv.ParseBool(v); err == nil {
		return isVal, true // return result: value is boolean
	}
	return false, false // value is not boolean
}

// get integer value of url parameter ?name or router parameter /:name
func getIntRequestParam(r *http.Request, name string, defaultVal int) (int, bool) {

	v := r.URL.Query().Get(name)
	if v == "" {
		v = vestigo.Param(r, name)
	}
	if v == "" {
		return defaultVal, true // no such parameter: return defult value
	}
	if nVal, err := strconv.Atoi(v); err == nil {
		return nVal, true // return result: value is integer
	}
	return defaultVal, false // value is not integer
}

// get int64 value of url parameter ?name or router parameter /:name
func getInt64RequestParam(r *http.Request, name string, defaultVal int64) (int64, bool) {

	v := r.URL.Query().Get(name)
	if v == "" {
		v = vestigo.Param(r, name)
	}
	if v == "" {
		return defaultVal, true // no such parameter: return defult value
	}
	if nVal, err := strconv.ParseInt(v, 0, 64); err == nil {
		return nVal, true // return result: value is integer
	}
	return defaultVal, false // value is not integer
}

// match request language with UI supported languages and return prefered language name
// get languages accepted by browser and by optional preferred language request parameter, for example: ..../lang/EN
func preferedRequestLang(r *http.Request, name string) string {
	rqLangTags := getRequestLang(r, name)
	tag, _, _ := uiLangMatcher.Match(rqLangTags...)
	if tag != language.Und {
		return tag.String()
	}
	return ""
}

// get languages accepted by browser and by optional language request parameter, for example: ..../lang/EN
// if language parameter specified then return it as a first element of result (it a preferred language)
func getRequestLang(r *http.Request, name string) []language.Tag {

	// browser languages
	rqLangTags, _, _ := language.ParseAcceptLanguage(r.Header.Get("Accept-Language"))

	// check if optional url parameter ?lang=LN or router parameter /lang/:lang specified
	if name == "" {
		return rqLangTags
	}

	// get lang parameter
	ln := r.URL.Query().Get(name)
	if ln == "" {
		ln = vestigo.Param(r, name)
	}

	// add lang parameter as top language
	if ln != "" {
		if t := language.Make(ln); t != language.Und {
			rqLangTags = append([]language.Tag{t}, rqLangTags...)
		}
	}
	return rqLangTags
}

// set Content-Type header by extension and invoke next handler.
// This function exist to suppress Windows registry content type overrides
func setContentType(next http.Handler) http.Handler {

	var ctDef = map[string]string{
		".css": "text/css; charset=utf-8",
		".js":  "text/javascript",
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if ext := filepath.Ext(r.URL.Path); ext != "" {
			if ct := ctDef[strings.ToLower(ext)]; ct != "" {
				w.Header().Set("Content-Type", ct)
			}
		}
		next.ServeHTTP(w, r) // invoke next handler
	})
}

// set csv response headers: Content-Type: application/csv, Content-Disposition and Cache-Control
func csvSetHeaders(w http.ResponseWriter, name string) {

	// set response headers: no Content-Length result in Transfer-Encoding: chunked
	// todo: ETag instead no-cache and utf-8 file names
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename="+`"`+url.QueryEscape(name)+".csv"+`"`)
	w.Header().Set("Cache-Control", "no-cache")

}

// return list of files by pattern, on error log error message
func filesByPattern(ptrn string, msg string) []string {

	fLst, err := filepath.Glob(ptrn)
	if err != nil {
		omppLog.Log(msg, ":", ptrn)
		return []string{}
	}
	return fLst
}

// Delete file and log path if isLog is true, return false on delete error.
func fileDeleteAndLog(isLog bool, path string) bool {
	if path == "" {
		return true
	}
	if isLog {
		omppLog.Log("Delete:", path)
	}
	if e := os.Remove(path); e != nil && !os.IsNotExist(e) {
		omppLog.LogNoLT(e)
		return false
	}
	return true
}

// Move file to new location and log it if isLog is true, return false on move error.
func fileMoveAndLog(isLog bool, srcPath string, dstPath string) bool {
	if srcPath == "" || dstPath == "" {
		return false
	}
	if isLog {
		omppLog.LogFmt("Move: %s To: %s", srcPath, dstPath)
	}
	if e := os.Rename(srcPath, dstPath); e != nil && !os.IsNotExist(e) {
		omppLog.LogNoLT(e)
		return false
	}
	return true
}

// Create or truncate existing file and log path if isLog is true, return false on create error.
func fileCreateEmpty(isLog bool, fPath string) bool {
	if isLog {
		omppLog.Log("Create:", fPath)
	}
	f, err := os.OpenFile(fPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		omppLog.LogNoLT(err)
		return false
	}
	defer f.Close()

	return true
}

// Copy file and log path if isLog is true, return false on error of if source file not exists
func fileCopy(isLog bool, src, dst string) bool {
	if src == "" || dst == "" || src == dst {
		return false
	}
	if isLog {
		omppLog.LogFmt("Copy: %s To: %s", src, dst)
	}

	inp, err := os.Open(src)
	if err != nil {
		if os.IsNotExist(err) {
			if isLog {
				omppLog.Log("File not found:", src)
			}
		} else {
			omppLog.LogNoLT(err)
		}
		return false
	}
	defer inp.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		omppLog.LogNoLT(err)
		return false
	}
	defer out.Close()

	if _, err = io.Copy(out, inp); err != nil {
		omppLog.LogNoLT(err)
		return false
	}
	return true
}

// append to message to log file
func writeToCmdLog(logPath string, isDoTimestamp bool, msg ...string) bool {

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return false // disable log on error
	}
	defer f.Close()

	tsPrefix := helper.MakeDateTime(time.Now()) + " "

	for _, m := range msg {
		if isDoTimestamp {
			if _, err = f.WriteString(tsPrefix); err != nil {
				return false // disable log on error
			}
		}
		if _, err = f.WriteString(m); err != nil {
			return false // disable log on error
		}
		if runtime.GOOS == "windows" { // adjust newline for windows
			_, err = f.WriteString("\r\n")
		} else {
			_, err = f.WriteString("\n")
		}
		if err != nil {
			return false
		}
	}
	return err == nil // disable log on error
}

// dbcopyPath return path to dbcopy.exe, it is expected to be in the same directory as oms.exe.
func dbcopyPath(binDir string) string {

	p := filepath.Join(binDir, "dbcopy.exe")
	if helper.IsFileExist(p) {
		return p
	}
	p = filepath.Join(binDir, "dbcopy")
	if helper.IsFileExist(p) {
		return p
	}
	return "" // dbcopy not found or not accessible or it is not a regular file
}

// wait for doneC exit signal or sleep, return true on exit signal or return false at the end of sleep interval
func isExitSleep(ms time.Duration, doneC <-chan bool) bool {
	select {
	case <-doneC:
		return true
	case <-time.After(ms * time.Millisecond):
	}
	return false
}

// return files list or file tree under rootDir/folder directory.
// rootDir top removed from the path results.
// if extCsv is not empty then filtered by extensions in comma separated list, for example: csv,tsv
// if isTree is true then return files tree else files path list.
func filesWalk(rootDir, folder string, extCsv string, isTree bool, lang string) ([]PathItem, error) {

	// parse comma separated list of extensions, if it is empty "" string then add all files, do not filter by extension
	eLst := []string{}
	isAll := extCsv == ""

	if !isAll {
		eLst = helper.ParseCsvLine(strings.ToLower(extCsv), ',')

		j := 0
		for _, elc := range eLst {
			if elc == "" {
				continue
			}
			if elc[0] != '.' {
				elc = "." + elc
			}
			eLst[j] = elc
			j++
		}
		eLst = eLst[:j]
	}

	// check if folder path exist under the root dir
	folderPath := filepath.Join(rootDir, folder)
	if !helper.IsDirExist(folderPath) {
		return nil, helper.ErrorNewL(lang, "Folder not found:", folder)
	}
	rDir := filepath.ToSlash(rootDir)
	rsDir := rDir + "/"

	// get list of files under the folder
	treeLst := []PathItem{}
	err := filepath.Walk(folderPath, func(path string, fi fs.FileInfo, err error) error {
		if err != nil {
			omppLog.Log("Error at directory walk:", path, " :", err)
			return err
		}
		p := filepath.ToSlash(path)
		if p == rDir || p == rsDir {
			p = "/"
		} else {
			p = strings.TrimPrefix(p, rsDir)
		}
		elc := strings.ToLower(filepath.Ext(p))

		// if no all files then check if extension is in the list of filter extensions
		isAdd := isAll

		for k := 0; !isAdd && k < len(eLst); k++ {
			isAdd = eLst[k] == elc
		}
		if isAdd {
			treeLst = append(treeLst, PathItem{
				Path:    p,
				IsDir:   fi.IsDir(),
				Size:    fi.Size(),
				ModTime: fi.ModTime().UnixMilli(),
			})
		}
		return nil
	})

	// if required then build files tree from files path list by adding directories into the path list
	if isTree {

		pm := map[string]bool{}
		addLst := []PathItem{}

		for k := 0; k < len(treeLst); k++ {

			d := treeLst[k].Path
			pm[d] = true // mark source path as already processed

			for { // until all directories above that path are processed

				d = path.Dir(d)

				if d == "" || d == "." || d == ".." || d == "/" || d == rDir {
					break // done with that directory and all directories above
				}
				if _, ok := pm[d]; ok {
					continue // directory already processed
				}
				pm[d] = true

				// get directory stat, ignoring error can potentially lead to incorrect tree
				if fi, e := helper.DirStat(filepath.Join(rootDir, filepath.FromSlash(d))); e == nil {
					addLst = append(addLst, PathItem{
						Path:    d,
						IsDir:   fi.IsDir(),
						Size:    fi.Size(),
						ModTime: fi.ModTime().UnixMilli(),
					})
				}
			}
		}

		// merge additional directories into files tree, sort file tree to put files after directories
		treeLst = append(treeLst, addLst...)

		slices.SortStableFunc(treeLst, func(left, right PathItem) int {
			if left.IsDir && !right.IsDir {
				return -1
			}
			if !left.IsDir && right.IsDir {
				return 1
			}
			return cmp.Compare(strings.ToLower(left.Path), strings.ToLower(right.Path))
		})
	}
	return treeLst, err
}
