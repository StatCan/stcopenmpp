// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

/*
Package helper is a set common helper functions
*/
package helper

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
)

const TimeStampLength = len("2012_08_17_16_04_59_123") // length of timestap string

// UnQuote trim spaces and remove "double" or 'single' quotes around string
func UnQuote(src string) string {
	s := strings.TrimSpace(src)
	if len(s) > 1 && (s[0] == '"' || s[0] == '\'') && s[0] == s[len(s)-1] {
		return s[1 : len(s)-1]
	}
	return s
}

// MakeDateTime return date-time string, ie: 2012-08-17 16:04:59.148
func MakeDateTime(t time.Time) string {
	y, mm, dd := t.Date()
	h, mi, s := t.Clock()
	ms := int(time.Duration(t.Nanosecond()) / time.Millisecond)

	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d.%03d", y, mm, dd, h, mi, s, ms)
}

// MakeTimeStamp return timestamp string as: 2012_08_17_16_04_59_148
func MakeTimeStamp(t time.Time) string {
	y, mm, dd := t.Date()
	h, mi, s := t.Clock()
	ms := int(time.Duration(t.Nanosecond()) / time.Millisecond)

	return fmt.Sprintf("%04d_%02d_%02d_%02d_%02d_%02d_%03d", y, mm, dd, h, mi, s, ms)
}

// IsUnderscoreTimeStamp return true if src is underscore timestamp string: 2021_07_16_13_40_53_882
func IsUnderscoreTimeStamp(src string) bool {

	if len(src) != TimeStampLength {
		return false
	}
	for k, r := range src {
		switch {
		case k == 4 || k == 7 || k == 10 || k == 13 || k == 16 || k == 19:
			if r != '_' {
				return false
			}
		default:
			if !unicode.IsDigit(r) {
				return false
			}
		}
	}
	return true
}

// IsTimeStamp return true if src is  timestamp string: 2021-07-16 13:40:53.882
func IsTimeStamp(src string) bool {

	if len(src) != TimeStampLength {
		return false
	}

	for k, r := range src {
		switch {
		case k == 4 || k == 7:
			if r != '-' {
				return false
			}
		case k == 10:
			if r != '\x20' {
				return false
			}
		case k == 13 || k == 16:
			if r != ':' {
				return false
			}
		case k == 19:
			if r != '.' {
				return false
			}
		default:
			if !unicode.IsDigit(r) {
				return false
			}
		}
	}
	return true
}

// ToUnderscoreTimeStamp converts date-time string to timestamp string: 2021-07-16 13:40:53.882 into 2021_07_16_13_40_53_882
// If source string is not a date-time string then return empty "" string
func ToUnderscoreTimeStamp(src string) string {

	if len(src) != TimeStampLength {
		return ""
	}

	dst := ""
	for k, r := range src {
		switch {
		case k == 4 || k == 7:
			if r != '-' {
				return ""
			}
			dst += "_"
		case k == 10:
			if r != '\x20' {
				return ""
			}
			dst += "_"
		case k == 13 || k == 16:
			if r != ':' {
				return ""
			}
			dst += "_"
		case k == 19:
			if r != '.' {
				return ""
			}
			dst += "_"
		default:
			if !unicode.IsDigit(r) {
				return ""
			}
			dst += string(r)
		}
	}
	return dst
}

// FromUnderscoreTimeStamp converts timestamp string to date-time string: 2021_07_16_13_40_53_882 into 2021-07-16 13:40:53.882
// If source string is not a date-time string then return empty "" string
func FromUnderscoreTimeStamp(src string) string {

	if len(src) != TimeStampLength {
		return ""
	}

	dst := ""
	for k, r := range src {

		switch {
		case k == 4 || k == 7:
			if r != '_' {
				return ""
			}
			dst += "-"
		case k == 10:
			if r != '_' {
				return ""
			}
			dst += "\x20"
		case k == 13 || k == 16:
			if r != '_' {
				return ""
			}
			dst += ":"
		case k == 19:
			if r != '_' {
				return ""
			}
			dst += "."
		default:
			if !unicode.IsDigit(r) {
				return ""
			}
			dst += string(r)
		}
	}
	return dst
}

// ToAlphaNumeric replace all non [A-Z,a-z,0-9] by _ underscore and remove repetitive underscores
func ToAlphaNumeric(src string) string {

	var sb strings.Builder
	isPrevUnder := false

	for _, r := range src {
		if '0' <= r && r <= '9' || 'A' <= r && r <= 'Z' || 'a' <= r && r <= 'z' {
			sb.WriteRune(r)
			isPrevUnder = false
		} else {
			if isPrevUnder {
				continue // skip repetitive underscore
			}
			sb.WriteRune('_')
			isPrevUnder = true
		}
	}
	return sb.String()
}

// ToIntValue cast src to int if it not nil and type is one of integer or float types.
// Return int value and true on success or 0 and false if src is nil or invalid type.
func ToIntValue(src interface{}) (int, bool) {

	if src == nil {
		return 0, false
	}

	var iv int
	switch e := src.(type) {
	case int:
		iv = e
	case uint:
		iv = int(e)
	case int64:
		iv = int(e)
	case uint64:
		iv = int(e)
	case int32:
		iv = int(e)
	case uint32:
		iv = int(e)
	case int16:
		iv = int(e)
	case uint16:
		iv = int(e)
	case int8:
		iv = int(e)
	case uint8:
		iv = int(e)
	case float64: // from json or oracle (often)
		iv = int(e)
	case float32: // from json or oracle (unlikely)
		iv = int(e)
	default:
		return 0, false
	}

	return iv, true
}

// greatest common divisor of two integers
func Gcd2(a, b int) int {
	for b != 0 {
		t := b
		b = a % b
		a = t
	}
	return a
}

// return greatest common divisor of all src[]
func Gcd(src []int) int {

	if len(src) <= 0 {
		return 1
	}

	d := src[0]
	for k := 1; k < len(src); k++ {
		d = Gcd2(src[k], d)
	}
	return d
	/*
	   src d:  [12 5 7] 1
	   src d:  [12 2 2 4 8] 2
	   src d:  [44 16 20 12 16 4 8] 4
	*/
}

// DeepCopy using gob to make a deep copy from src into dst, both src and dst expected to be a pointers
func DeepCopy(src interface{}, dst interface{}) error {
	var bt bytes.Buffer
	enc := gob.NewEncoder(&bt)
	dec := gob.NewDecoder(&bt)

	err := enc.Encode(src)
	if err != nil {
		return errors.New("deep copy encode failed: " + err.Error())
	}

	err = dec.Decode(dst)
	if err != nil {
		return errors.New("deep copy decode failed: " + err.Error())
	}
	return nil
}

// ParseKeyValue string of multiple key = value; pairs separated by semicolon.
// Key cannot be empty, value can be.
// Value can be escaped with "double" or 'single' quotes
func ParseKeyValue(src string) (map[string]string, error) {

	kv := make(map[string]string)
	var key string
	var isKey = true

	for src != "" {

		// split key= and value
		if isKey {
			// skip ; semicolon(s) and spaces in front of the key
			src = strings.TrimLeftFunc(src, func(c rune) bool {
				return c == ';' || unicode.IsSpace(c)
			})
			if src == "" {
				break // empty end of string, no more key=...
			}

			nEq := strings.IndexRune(src, '=')

			if nEq <= 0 {
				return nil, errors.New("expected key=... inside of key=value; string")
			}

			key = strings.TrimSpace(src[:nEq])
			if key == "" {
				return nil, errors.New("invalid (empty) key inside of key=value; string")
			}
			isKey = false
			src = src[nEq+1:] // key is found, skip =
			//continue
		}
		// expected begin of the value position

		// search for end of value ; semicolon, skip quoted part of value
		isQuote := false
		var cQuote rune
		for nPos, chr := range src {

			// if end of value as ; semicolon found
			if !isQuote && chr == ';' {

				// append result to the map, unquote "value" if quotes balanced
				kv[key] = UnQuote(src[:nPos])

				// value is found, skip ; semicolon and reset state
				src = src[nPos+1:]
				key = ""
				isKey = true
				break
			}

			// open or close quotes
			if !isQuote && (chr == '"' || chr == '\'') || isQuote && chr == cQuote {
				isQuote = !isQuote
				if isQuote {
					cQuote = chr // opening quote
				} else {
					cQuote = 0 // quote closed
				}
				continue
			}
		}
		// last key=value without ; semicolon at the end of line
		if !isKey && key != "" {
			kv[key] = UnQuote(src)
			break
		}
	}

	return kv, nil
}

// ParseCsvLine comma separated string: " value ", value, ' value '.
// Value can be empty and can be escaped with "double" or 'single' quotes.
// If comma is zero rune then , comma used by default.
func ParseCsvLine(src string, comma rune) []string {

	if comma == 0 {
		comma = ','
	}
	vLst := []string{}

	for src != "" {

		// skip spaces in front of the next value
		src = strings.TrimLeftFunc(src, func(c rune) bool {
			return unicode.IsSpace(c)
		})
		if src == "" {
			break // empty end of string, no more values
		}

		// search for end of value , comma and skip quoted part of value
		isNext := false
		isQuote := false
		var cQuote rune

		for nPos, chr := range src {

			// if end of value as , comma found
			if !isQuote && chr == comma {

				// append to result result to the map, unquote "value" if quotes balanced
				vLst = append(vLst, UnQuote(src[:nPos]))

				// value is found: skip , comma and process next value
				src = src[nPos+1:]
				isNext = true
				break
			}

			// open or close quotes
			if !isQuote && (chr == '"' || chr == '\'') || isQuote && chr == cQuote {
				isQuote = !isQuote
				if isQuote {
					cQuote = chr // opening quote
				} else {
					cQuote = 0 // quote closed
				}
				continue
			}
		}
		// last value without comma at the end of line
		if !isNext && src != "" {
			vLst = append(vLst, UnQuote(src))
			break
		}
	}

	return vLst
}

// Escape src value for ini-file writing: add 'single' or "double" quotes around if src contains ; semicolon or # hash.
func QuoteForIni(src string) string {

	if src == "" {
		return src // source value empty or already 'quoted' or "double" quoted
	}
	if (src[0] == '"' || src[0] == '\'') && src[len(src)-1] == src[0] {
		return src // source value already 'quoted' or "double" quoted
	}
	if strings.IndexAny(src, ";#") < 0 {
		return src // source value does not contain ; semicolon or # hash: "quotes" are not required
	}

	// use "double" quotes if there are no double quotes inside of the source value else use 'single' quotes
	if strings.IndexRune(src, '"') < 0 {
		return "\"" + src + "\""
	}
	return "'" + src + "'"
}
