// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package helper

import (
	"archive/zip"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// PackZip create new (overwrite) zip archive from specified file or directory and all subdirs.
// If dstDir is "" empty then result located in source base directory.
func PackZip(srcPath string, isCleanDstDir bool, dstDir string) (string, error) {

	// create output directory if not exist and make archive name as base.zip
	cleanPath := filepath.Clean(srcPath)
	baseDir, base := filepath.Split(cleanPath)

	var zipPath string
	if dstDir == "" {
		zipPath = filepath.Join(baseDir, base+".zip")
	} else {

		zipPath = filepath.Join(dstDir, base+".zip")

		if isCleanDstDir {
			if e := os.RemoveAll(dstDir); e != nil && !os.IsNotExist(e) {
				return "", errors.New("Error: unable to delete: " + dstDir + " : " + e.Error())
			}
		}
		if err := os.MkdirAll(dstDir, 0750); err != nil {
			return "", errors.New("make directory failed at pack to zip: " + err.Error())
		}
	}

	// create zip file
	zf, err := os.OpenFile(zipPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return "", errors.New("create file failed at pack to zip: " + err.Error())
	}
	defer zf.Close()

	zwr := zip.NewWriter(zf)
	defer zwr.Close()

	// walk in source directory and compress files and subdirs
	err = filepath.WalkDir(cleanPath, func(src string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// make archive name relative to source base directory
		rel, err := filepath.Rel(baseDir, src)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		// if this is directory add it to header to store empty dirs
		if de.IsDir() {
			_, err := zwr.Create(rel + "/")
			return err
		}
		// else: add file to archive
		w, err := zwr.Create(rel)
		if err != nil {
			return err
		}
		f, err := os.Open(src)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(w, f) // do compression
		return err
	})

	if err != nil {
		return "", errors.New("failed pack to zip: " + err.Error())
	}
	return zipPath, nil
}

// UnpackZip unpack zip archive into specified directory, creating it if not exist.
// If dstDir is "" empty then result located in source base directory.
func UnpackZip(zipPath string, isCleanDstDir bool, dstDir string) error {

	// create output directory if not exist
	var baseDir string
	if dstDir == "" || dstDir == "." {
		baseDir = filepath.Dir(zipPath)
	} else {

		baseDir = filepath.Clean(dstDir)

		if isCleanDstDir {
			if baseDir == filepath.Dir(zipPath) {
				return errors.New("Error: output directory the same as input, unable cleanup: " + baseDir)
			}
			if e := os.RemoveAll(baseDir); e != nil && !os.IsNotExist(e) {
				return errors.New("Error: unable to delete: " + baseDir + " : " + e.Error())
			}
		}
		if err := os.MkdirAll(baseDir, 0750); err != nil {
			return errors.New("make directory failed at unpack from zip: " + err.Error())
		}
	}

	// open zip archive
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return errors.New("open zip file failed at unpack from zip: " + err.Error())
	}
	defer zr.Close()

	// for all zipped files and directories
	for _, znext := range zr.File {

		err := func(zf *zip.File) error {

			// if this is empty directory then create it
			info := zf.FileInfo()
			p := filepath.Join(baseDir, zf.Name)

			if info.IsDir() {
				return os.MkdirAll(p, 0750)
			}
			// else unpack file

			f, err := os.OpenFile(p, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
			if err != nil {
				return err
			}
			defer f.Close()

			r, err := zf.Open()
			if err != nil {
				return err
			}
			defer r.Close()

			_, err = io.Copy(f, r) // do unpack
			return err
		}(znext)

		if err != nil {
			return errors.New("unpack from zip failed: " + err.Error())
		}
	}
	return nil
}
