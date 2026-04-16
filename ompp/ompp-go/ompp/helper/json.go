// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package helper

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
)

// FromJsonFile reads read from json file and convert to destination pointer.
func FromJsonFile(jsonPath string, dst interface{}) (bool, error) {

	if jsonPath == "" || jsonPath == "." {
		return false, nil // file path is empty
	}

	// open file and convert to utf-8
	f, err := os.Open(jsonPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // return: json file not exist
		}
		return false, ErrorNew("json file open error:", err)
	}
	defer f.Close()

	// make utf-8 converter:
	// assume utf-8 as default encoding on any OS because json file must be unicode and cannot be "windows code page"
	rd, err := Utf8Reader(f, "utf-8")
	if err != nil {
		return false, ErrorNew("json file read error:", err)
	}

	// decode json
	err = json.NewDecoder(rd).Decode(dst)
	if err != nil {
		if err == io.EOF {
			return false, nil // return "not exist" if json file empty
		}
		return false, ErrorNew("json decode error:", err)
	}
	return true, nil
}

// FromJson restore from json string bytes and convert to destination pointer.
func FromJson(srcJson []byte, dst interface{}) (bool, error) {

	err := json.NewDecoder(bytes.NewReader(srcJson)).Decode(dst)
	if err != nil {
		if err == io.EOF {
			return false, nil // return "not exist" if json empty
		}
		return false, ErrorNew("json decode error:", err)
	}
	return true, nil
}

// ToJsonFile convert source to json and write into jsonPath file.
func ToJsonFile(jsonPath string, src interface{}) error {

	f, err := os.OpenFile(jsonPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return ErrorNew("json file create error:", err)
	}
	defer f.Close()

	err = json.NewEncoder(f).Encode(src)
	if err != nil {
		return ErrorNew("json encode error:", err)
	}
	return nil
}

// ToJsonIndent return source conveted to json indeneted string.
func ToJsonIndent(src interface{}) (string, error) {

	srcJson, err := json.MarshalIndent(src, "", "  ")
	if err != nil {
		return "", ErrorNew("json marshal indent error:", err)
	}
	return string(srcJson), nil
}

// ToJsonIndentFile convert source to json and write into jsonPath file.
func ToJsonIndentFile(jsonPath string, src interface{}) error {

	f, err := os.OpenFile(jsonPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return ErrorNew("json file create error:", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	err = enc.Encode(src)
	if err != nil {
		return ErrorNew("json encode error:", err)
	}
	return nil
}
