// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package helper

import (
	"errors"
	"io"
	"os"
	"runtime"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/encoding/unicode/utf32"
	"golang.org/x/text/transform"
)

// byte order mark bytes
var (
	Utf8bom    = []byte{0xEF, 0xBB, 0xBF}
	Utf16LEbom = []byte{0xFF, 0xFE}
	Utf16BEbom = []byte{0xFE, 0xFF}
	Utf32LEbom = []byte{0xFF, 0xFE, 0x00, 0x00}
	Utf32BEbom = []byte{0x00, 0x00, 0xFE, 0xFF}
)

const utf8ProbeLen = 4 * 32 * 1024 // probe length: if this length utf8 then the rest of the file is utf8

// FileToUtf8 read file content and convert it to UTF-8 string.
//
// If file starts with BOM (utf-8 utf-16LE utf-16BE utf-32LE utf-32BE) then BOM is used.
// If no BOM and encodingName is "" empty then file content probed to see is it already utf-8.
// If encodingName explicitly specified then it is used to convert file content to string.
// If none of above then assume default encoding: "windows-1252" on Windows and "utf-8" on Linux.
func FileToUtf8(filePath string, encodingName string) (string, error) {

	// open file and create utf-8 transform reader
	f, err := os.Open(filePath)
	if err != nil {
		return "", errors.New("file open error: " + err.Error())
	}
	defer f.Close()

	rd, err := Utf8Reader(f, encodingName)
	if err != nil {
		return "", errors.New("failed to create utf-8 reader " + encodingName + " : " + err.Error())
	}

	// read and convert into utf-8
	bt, err := io.ReadAll(rd)
	if err != nil {
		return "", errors.New("read to utf-8 error: " + err.Error())
	}
	return string(bt), nil
}

// Utf8Reader return a reader to transform file content to utf-8.
//
// If file starts with BOM (utf-8 utf-16LE utf-16BE utf-32LE utf-32BE) then BOM is used.
// If no BOM and encodingName is "" empty then file content probed to see is it already utf-8.
// If encodingName explicitly specified then it is used to convert file content to string.
// If none of above then assume default encoding: "windows-1252" on Windows and "utf-8" on Linux.
func Utf8Reader(f *os.File, encodingName string) (io.Reader, error) {

	// validate parameters
	if f == nil {
		return nil, errors.New("invalid (nil) source file")
	}

	// detect BOM
	bom := make([]byte, utf8.UTFMax)

	nBom, err := f.Read(bom)
	if err != nil {
		if nBom == 0 && err == io.EOF { // empty file: return source file as is
			return f, nil
		}
		return nil, errors.New("file read error: " + err.Error())
	}

	// if utf-8 BOM then skip it and return source file
	if nBom >= len(Utf8bom) && bom[0] == Utf8bom[0] && bom[1] == Utf8bom[1] && bom[2] == Utf8bom[2] {
		if _, err := f.Seek(int64(len(Utf8bom)), 0); err != nil {
			return nil, errors.New("file seek error: " + err.Error())
		}
		return f, nil
	}

	// move back to the file begining to use BOM, if present
	if _, err := f.Seek(0, 0); err != nil {
		return nil, errors.New("file seek error: " + err.Error())
	}

	// ambiguos utf-16LE and utf32-LE detection: assume utf-32LE because 00 00 is very unlikely in text file
	if nBom >= len(Utf32LEbom) && bom[0] == Utf32LEbom[0] && bom[1] == Utf32LEbom[1] && bom[2] == Utf32LEbom[2] && bom[3] == Utf32LEbom[3] {
		return transform.NewReader(f, utf32.UTF32(utf32.LittleEndian, utf32.UseBOM).NewDecoder()), nil
	}
	if nBom >= len(Utf32BEbom) && bom[0] == Utf32BEbom[0] && bom[1] == Utf32BEbom[1] && bom[2] == Utf32BEbom[2] && bom[3] == Utf32BEbom[3] {
		return transform.NewReader(f, utf32.UTF32(utf32.BigEndian, utf32.UseBOM).NewDecoder()), nil
	}
	if nBom >= len(Utf16LEbom) && bom[0] == Utf16LEbom[0] && bom[1] == Utf16LEbom[1] {
		return transform.NewReader(f, unicode.BOMOverride(encoding.Nop.NewDecoder())), nil
	}
	if nBom >= len(Utf16BEbom) && bom[0] == Utf16BEbom[0] && bom[1] == Utf16BEbom[1] {
		return transform.NewReader(f, unicode.BOMOverride(encoding.Nop.NewDecoder())), nil
	}
	// no BOM detected

	// encoding not specified then probe file to check is it utf-8
	if encodingName == "" {

		// read probe bytes from the file
		buf := make([]byte, utf8ProbeLen)
		nProbe, err := f.Read(buf)
		if err != nil {
			if nProbe == 0 && err == io.EOF { // empty file: return source file as is
				return f, nil
			}
			return nil, errors.New("file read error: " + err.Error())
		}

		// check if all runes are utf-8
		nPos := 0
		for nPos < nProbe {
			r, n := utf8.DecodeRune(buf)
			if n <= 0 || r == utf8.RuneError { // if eof or not utf-8 rune
				break
			}
			nPos += n
			buf = buf[n:]
		}

		// move back to the file begining
		if _, err := f.Seek(0, 0); err != nil {
			return nil, errors.New("file seek error: " + err.Error())
		}

		// file is utf-8 if:
		// all runes are utf-8 and file size less than max probe size or file size exceeds probe size
		if nPos >= nProbe || nPos >= utf8ProbeLen-utf8.UTFMax {
			return f, nil // utf-8 file: return source file reader
		}
	}

	// if encoding is not explicitly specified then use OS default
	if encodingName == "" {
		if runtime.GOOS == "windows" {
			encodingName = "windows-1252"
		} else {
			encodingName = "utf-8"
		}
	}

	// get encoding by name
	enc, err := htmlindex.Get(encodingName)
	if err != nil {
		return nil, errors.New("invalid encoding: " + encodingName + " " + err.Error())
	}

	return transform.NewReader(f, unicode.BOMOverride(enc.NewDecoder())), nil
}
