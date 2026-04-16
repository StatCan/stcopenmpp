// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

/*
Package helper is a set common helper functions
*/
package helper

import (
	"slices"
	"strings"
)

// Ini-file entry: section, key and value
type IniEntry struct {
	Section string // section: in lower case, without [], space trimed
	Key     string // key: in lower case, without =, space trimed
	Val     string // value: space trimed and unquoted ("quotes" or 'apostrophes' removed)
}

// Insert new or update existing ini file entry:
// search by (section, key) in source []IniEntry
// if not found then insert new entry
// if found and isUpdate true then update existing entry with new value
func AddIniEntry(isUpdate bool, eaIni []IniEntry, section, key, val string) []IniEntry {

	scIdx := -1
	for k := 0; scIdx < 0 && k < len(eaIni); k++ {
		if strings.EqualFold(eaIni[k].Section, section) {
			scIdx = k
		}
	}
	if scIdx < 0 { // section not found: append new section, key value

		return append(eaIni, IniEntry{Section: section, Key: key, Val: val})

	} // else section found: search key inside of the section

	for ; scIdx < len(eaIni) && strings.EqualFold(eaIni[scIdx].Section, section); scIdx++ {
		if strings.EqualFold(eaIni[scIdx].Key, key) {
			if isUpdate {
				eaIni[scIdx].Val = val // section and key found: update existing section.key with new value
			}
			return eaIni // key found: return []IniEnty array with updated value or original value
		}
	}
	// else key not found in the section: insert new key at the end of section

	return slices.Insert(eaIni, scIdx, IniEntry{Section: section, Key: key, Val: val})
}

// Update existing or insert new ini file entry:
// search by (section, key) in source []IniEntry
// if not found then insert new entry
// if found then update existing entry with new value
func MergeIniEntry(eaIni []IniEntry, section, key, val string) []IniEntry {
	return AddIniEntry(true, eaIni, section, key, val)
}

// Insert new ini file entry if not already exists:
// search by (section, key) in source []IniEntry
// if not found then insert new entry
func InsertIniEntry(eaIni []IniEntry, section, key, val string) []IniEntry {
	return AddIniEntry(false, eaIni, section, key, val)
}

// Update existing or insert new ini file entry:
// sectionKey expected to be: section.key if not then exit and do nothing.
// find section and key in source []IniEntry
// if not found then insert new entry
// if found then update existing entry with new value
func MergeSectionKeyIniEntry(eaIni []IniEntry, sectionKey, val string) []IniEntry {

	sck := strings.SplitN(sectionKey, ".", 2) // expected section.key
	if len(sck) < 2 || sck[0] == "" || sck[1] == "" {
		return eaIni // skip invalid section.key
	}
	return MergeIniEntry(eaIni, sck[0], sck[1], val)
}

// return ini-file sections
func IniSectionList(eaIni []IniEntry) []string {

	scLst := []string{}
	sc := ""

	for _, e := range eaIni {
		if e.Section != sc {
			scLst = append(scLst, e.Section)
		}
		sc = e.Section
	}
	return scLst
}
