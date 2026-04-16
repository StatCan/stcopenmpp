// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

// +build odbc

package db

import (
	_ "github.com/alexbrainman/odbc"
)

// IsOdbcSupported indicate support of ODBC connections built-in
const IsOdbcSupported = true
