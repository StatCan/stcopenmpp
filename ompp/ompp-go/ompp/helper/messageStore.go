// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

/*
Package helper is a set common helper functions
*/
package helper

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// translated messages map and language specific printers, first element is translations in user prefered language
type msgPrt struct {
	lang   string            // language
	msgMap map[string]string // messages translation map for the language
	prt    *message.Printer  // language specific printer, if not null then use it instead of fmt
}

var (
	msgLock  sync.Mutex         // mutex to lock for message string search or update
	msgLang  string             // user prefered language for the messages, it can be empty "" string, means no translations
	langIdx  = map[string]int{} // map language to index in msgTrans slice, index of prefered language is always 0 zero
	msgTrans = []msgPrt{}       // translated messages map and language specific printers, first element is translations in user prefered language
)

// short form of: errors.New(helper.Msg(msg))
func ErrorNew(msg ...any) error { return errors.New(Msg(msg...)) }

// short form of: errors.New(helper.MsgL(lang, msg))
func ErrorNewL(lang string, msg ...any) error { return errors.New(MsgL(lang, msg...)) }

// short form of: errors.New(helper.Fmt(format, msg))
func ErrorFmt(format string, msg ...any) error { return errors.New(Fmt(format, msg...)) }

// short form of: errors.New(helper.FmtL(lang, format, msg))
func ErrorFmtL(lang, format string, msg ...any) error { return errors.New(FmtL(lang, format, msg...)) }

// return translated string, if translation not found then retrun source string
func LT(src string) string {
	return LTL(msgLang, src)
}

// return translated string, if translation not found then retrun source string
func LTL(lang, src string) string {
	if src == "" {
		return src
	}
	msgLock.Lock()
	defer msgLock.Unlock()

	return doLT(lang, src)
}

// for internal use only: inside the lock.
// return translated string, if translation not found then retrun source string
func doLT(lang, src string) string {

	if src == "" || lang == "" || lang == "und" {
		return src
	}
	li := langIdx[lang]

	if li < 0 || li >= len(msgTrans) {
		return src
	}
	if t, ok := msgTrans[li].msgMap[src]; ok {
		return t
	}
	return src
}

// for internal use only: inside the lock.
// return language-specific printer or null if language not found
func getPrt(lang string) *message.Printer {
	if lang == "" || lang == "und" {
		return nil
	}
	li, ok := langIdx[lang]

	if !ok || li < 0 || li >= len(msgTrans) {
		return nil
	}
	return msgTrans[li].prt
}

// Join items by space, use Sprint for each item,
// translate first msg[0] item using message.ini content,
// use language specific printer instead of fmt by default
func Msg(msg ...any) string { return MsgL(msgLang, msg...) }

// Join items by space, use Sprint for each item,
// translate first msg[0] item using message.ini content,
// use language specific printer instead of fmt by default
func MsgL(lang string, msg ...any) string {

	if len(msg) <= 0 {
		return ""
	}
	msgLock.Lock()
	defer msgLock.Unlock()

	return doMsg(lang, msg...)
}

// for internal use only: inside the lock.
// Join items by space, use Sprint for each item,
// translate first msg[0] item using message.ini content,
// use language specific printer instead of fmt by default
func doMsg(lang string, msg ...any) string {

	if len(msg) <= 0 {
		return ""
	}
	prt := getPrt(lang)

	// translate first item of msg[] if it is a string
	m := ""
	if m0, ok := msg[0].(string); ok {
		m = doLT(lang, m0)
	} else {
		if prt != nil {
			m = prt.Sprint(msg[0])
		} else {
			m = fmt.Sprint(msg[0])
		}
	}

	// append the rest of message items
	for _, v := range msg[1:] {
		if m != "" {
			m += " "
		}
		if prt != nil {
			m += prt.Sprint(v)
		} else {
			m += fmt.Sprint(v)
		}
	}
	return m
}

// Join msg items by space
// this function should NOT use anything from message store
func MsgNoLT(msg ...any) string {

	// this function should NOT use anything from message store
	// msgLock.Lock()
	// defer msgLock.Unlock()

	m := ""
	for _, v := range msg {
		if m != "" {
			m += " "
		}
		m += fmt.Sprint(v)
	}
	return m
}

// return string formatted by Sprintf
// translate format string using message.ini content,
// use language specific printer instead of fmt by default
func Fmt(format string, msg ...any) string { return FmtL(msgLang, format, msg...) }

// return string formatted by Sprintf
// translate format string using message.ini content,
// use language specific printer instead of fmt by default
func FmtL(lang, format string, msg ...any) string {

	msgLock.Lock()
	defer msgLock.Unlock()

	return doFmt(lang, format, msg...)
}

// for internal use only: inside the lock.
// return string formatted by Sprintf
// translate format string using message.ini content,
// use language specific printer instead of fmt by default
func doFmt(lang, format string, msg ...any) string {

	prt := getPrt(lang)

	if format != "" {
		if prt != nil {
			return prt.Sprintf(doLT(lang, format), msg...)
		} // else
		return fmt.Sprintf(format, msg...)
	}
	if prt != nil {
		return prt.Sprint(msg...)
	}
	return fmt.Sprint(msg...)
}

// update user language name and translation map
func SetMsg(userLang string, msgIni []IniEntry) (string, []string) {

	// match message.ini languages to user language to build message.ini sectioans list ordered by language match confidence
	// examples: en_CA => en-US, fr_CA => fr
	// known issue: it does not understand .UTF-8 suffix

	mMap := map[string]string{}
	uLang := ""
	uTag := language.Und

	scLst := IniSectionList(msgIni)        // languages from message.ini
	lcLst := make([]string, 0, len(scLst)) // languages with non-zero confidence match to prefered language

	if userLang != "" {
		uTag = language.Make(userLang)
		uLang = uTag.String()
		if uLang == "und" {
			uLang = "" // undefined user language
		}
		cLst := make([]language.Confidence, 0, len(scLst))

		for _, sn := range scLst {

			lt := []language.Tag{language.Make(sn)}
			_, _, c := language.NewMatcher(lt).Match(uTag)
			if c > language.No {
				lcLst = append(lcLst, sn)
				cLst = append(cLst, c)
			}
		}

		// sort language list in confidence ascending order
		sort.SliceStable(lcLst, func(i, j int) bool { return cLst[i] < cLst[j] })

		// include most confident translations into message map for prefered language
		for _, ln := range lcLst {
			for k := range msgIni {
				if msgIni[k].Section == ln {
					mMap[msgIni[k].Key] = msgIni[k].Val
				}
			}
		}
	}

	// update user prefered language, set preferd language translation map and printer to replace default fmt
	msgLock.Lock()
	defer msgLock.Unlock()

	// clean translation maps and printers
	msgTrans = []msgPrt{}

	// include prefered language translations and printer as first elementin translations array
	msgLang = uLang
	langLst := make([]string, 0, len(scLst)) // languages with non-zero confidence match to prefered language

	if msgLang != "" {

		langIdx[msgLang] = len(msgTrans)
		langLst = append(langLst, msgLang)

		msgTrans = []msgPrt{
			{
				lang:   msgLang,
				msgMap: mMap,
				prt:    nil,
			},
		}
		if uTag != language.Und {
			msgTrans[0].prt = message.NewPrinter(uTag)
		}
	}

	// include language from message.ini
	addLn := func(ln string) {

		if ln == "" || ln == "und" {
			return
		}
		if _, ok := langIdx[ln]; ok {
			return // that language already included in result
		}

		mt := map[string]string{}

		for k := range msgIni {
			if msgIni[k].Section == ln {
				mt[msgIni[k].Key] = msgIni[k].Val
			}
		}
		if len(mt) < 1 {
			return // no translations for that language
		}
		// else add translations and create language specific printer

		n := len(msgTrans)
		langIdx[ln] = n
		langLst = append(langLst, ln)

		msgTrans = append(msgTrans,
			msgPrt{
				lang:   ln,
				msgMap: mt,
				prt:    nil,
			},
		)

		lt := language.Make(ln)
		if lt != language.Und {
			msgTrans[n].prt = message.NewPrinter(lt)
		}
	}

	// include languages from message.ini where translation for user prefered langauges has non-zero confidence
	for _, ln := range lcLst {
		addLn(ln)
	}

	// include all other languages from message.ini
	for _, ln := range scLst {
		addLn(ln)
	}

	// append English if not already on the list
	// English implicitly exists because it does not required message translation
	if slices.IndexFunc(langLst, func(ln string) bool {
		return strings.EqualFold(ln, "en") || strings.HasPrefix(ln, "en-") || strings.HasPrefix(ln, "en_")
	}) < 0 {

		n := len(msgTrans)
		langIdx["en"] = n
		langLst = append(langLst, "en")

		msgTrans = append(msgTrans,
			msgPrt{
				lang:   "en",
				msgMap: map[string]string{},
				prt:    message.NewPrinter(language.Make("en")),
			},
		)
	}

	return msgLang, langLst
}
