// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// set json response headers: Content-Type: application/json
func jsonSetHeaders(w http.ResponseWriter, r *http.Request) {

	// if Content-Type not set then use json
	if _, isSet := w.Header()["Content-Type"]; !isSet {
		w.Header().Set("Content-Type", "application/json")
	}

	// if request from localhost then allow response to any protocol or port
	/*
		if strings.HasPrefix(r.Host, "localhost") {
			if _, isSet := w.Header()["Access-Control-Allow-Origin"]; !isSet {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}
		}
	*/
}

// jsonResponse set json response headers and writes src as json into w response writer.
// On error it writes 500 internal server error response.
func jsonResponse(w http.ResponseWriter, r *http.Request, src interface{}) {

	jsonSetHeaders(w, r)

	err := json.NewEncoder(w).Encode(src)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// jsonResponseBytes set json response headers and writes src bytes into w response writer.
// If src bytes empty then "{}" is written.
func jsonResponseBytes(w http.ResponseWriter, r *http.Request, src []byte) {

	jsonSetHeaders(w, r)

	if len(src) <= 0 {
		w.Write([]byte("{}"))
	}
	w.Write(src)
}

// jsonCellWriter writes each row of data page into response as list of comma separated json values.
// It is an append to response and response headers must already be set.
func jsonCellWriter(w http.ResponseWriter, enc *json.Encoder, cvtCell func(interface{}) (interface{}, error)) func(src interface{}) (bool, error) {

	isNext := false

	// write data page into response as list of comma separated json values
	cvtWr := func(src interface{}) (bool, error) {

		if isNext {
			w.Write([]byte{','}) // until the last separate array items with , comma
		}

		val := src // id's cell
		var err error

		// convert cell from id's to code if converter specified
		if cvtCell != nil {
			if val, err = cvtCell(src); err != nil {
				return false, err
			}
		}

		// write actual value
		if err := enc.Encode(val); err != nil {
			return false, err
		}
		isNext = true
		return true, nil
	}
	return cvtWr
}

// jsonRequestDecode validate Content-Type: application/json and decode json body.
// Destination for json decode: dst must be a pointer.
// If isRequired is true then json body is required else it can be empty by default.
// On error it writes error response 400 or 415 and return false.
func jsonRequestDecode(w http.ResponseWriter, r *http.Request, isRequired bool, dst interface{}) bool {

	// json body expected
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "Expected Content-Type: application/json", http.StatusUnsupportedMediaType)
		return false
	}

	// decode json
	err := json.NewDecoder(r.Body).Decode(dst)
	if err != nil {
		if err == io.EOF {
			if !isRequired {
				return true // empty default values
			} else {
				http.Error(w, "Invalid (empty) json at "+r.URL.String(), http.StatusBadRequest)
				return false
			}
		}
		omppLog.Log("Json decode error at " + r.URL.String() + ": " + err.Error())
		http.Error(w, "Json decode error at "+r.URL.String(), http.StatusBadRequest)
		return false
	}
	return true // completed OK
}

// jsonMultipartDecode decode json part of multipart form reader.
// It does move to next part, check part form name and decode json content from part body.
// Destination for json decode: dst must be a pointer.
// On error it writes error response 400 or 415 and return false.
func jsonMultipartDecode(w http.ResponseWriter, mr *multipart.Reader, name string, dst interface{}) bool {

	// open next part and check part name
	part, err := mr.NextPart()
	if err == io.EOF {
		http.Error(w, "Invalid (empty) next part of multipart form "+name, http.StatusBadRequest)
		return false
	}
	if err != nil {
		http.Error(w, "Failed to get next part of multipart form "+name+" : "+err.Error(), http.StatusBadRequest)
		return false
	}
	defer part.Close()

	if part.FormName() != name {
		http.Error(w, "Invalid part of multipart form, expected name: "+name, http.StatusBadRequest)
		return false
	}

	// decode json
	err = json.NewDecoder(part).Decode(dst)
	if err != nil {
		if err == io.EOF {
			http.Error(w, "Invalid (empty) json part of multipart form "+name, http.StatusBadRequest)
			return false
		}
		http.Error(w, "Json decode error at part of multipart form "+name, http.StatusBadRequest)
		return false
	}
	return true // completed OK
}

// jsonRequestToFile validate Content-Type: application/json and copy request body into output file as is.
// If output file already exists then it is truncated.
// On error it writes error response 400 or 500 and return false.
func jsonRequestToFile(w http.ResponseWriter, r *http.Request, outPath string) bool {

	// json body expected
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "Expected Content-Type: application/json", http.StatusUnsupportedMediaType)
		return false
	}

	// create or truncate output file
	fName := filepath.Base(outPath)

	err := helper.SaveTo(outPath, r.Body)
	if err != nil {
		omppLog.Log("Error: unable to write into ", outPath, err)
		http.Error(w, "Error: unable to write into "+fName, http.StatusInternalServerError)
		return false
	}
	return true // completed OK
}
