// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"os"

	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// Delete directory and log path, return false on delete error.
func dirDeleteAndLog(path string) bool {

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
