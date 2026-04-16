// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"strings"

	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
	"golang.org/x/text/language"
)

// LangListByDigestOrName return lang_lst db rows by model digest or name.
// First language is model default language.
// It may contain more languages than model actually have.
func (mc *ModelCatalog) LangListByDigestOrName(dn string) ([]db.LangLstRow, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return []db.LangLstRow{}, false
	}

	// lock model catalog and return copy of langauge list
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	idx, ok := mc.indexByDigestOrName(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return []db.LangLstRow{}, false
	}

	ls := make([]db.LangLstRow, len(mc.modelLst[idx].langMeta.Lang))
	for k := range mc.modelLst[idx].langMeta.Lang {
		ls[k] = mc.modelLst[idx].langMeta.Lang[k].LangLstRow
	}

	return ls, true
}

// ModelProfileByName return model profile db rows by model digest-or-name and by profile name.
func (mc *ModelCatalog) ModelProfileByName(dn, profile string) (*db.ProfileMeta, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return &db.ProfileMeta{}, false
	}
	if profile == "" {
		omppLog.Log("Warning: invalid (empty) profile name")
		return &db.ProfileMeta{}, false
	}

	// get database connection
	_, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return &db.ProfileMeta{}, false
	}

	// read profile from database
	p, err := db.GetProfile(dbConn, profile)
	if err != nil {
		omppLog.Log("Error at get profile:", dn, ": ", profile, ": ", err)
		return &db.ProfileMeta{}, false // return empty result: model not found or error
	}

	return p, true
}

// ProfileNamesByDigestOrName return list of profile(s) names by model digest-or-name.
// This is a list of profiles from model database, it is not a "model" profile(s).
// There is no explicit link between profile and model, profile can be applicable to multiple models.
func (mc *ModelCatalog) ProfileNamesByDigestOrName(dn string) ([]string, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return []string{}, false
	}

	// get database connection
	_, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return []string{}, false // return empty result: model not found or error
	}

	// read profile names from database
	nameLst, err := db.GetProfileList(dbConn)
	if err != nil {
		omppLog.Log("Error at get profile list from model database:", dn, ":", err)
		return []string{}, false
	}

	return nameLst, true
}

// WordListByDigestOrName return model "words" by model digest and preferred language tags.
// Model "words" are arrays of rows from lang_word and model_word db tables.
// It can be in preferred language, default model language or empty if no lang_word or model_word rows exist.
func (mc *ModelCatalog) WordListByDigestOrName(dn string, preferredLang []language.Tag) (*ModelWordLabel, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return &ModelWordLabel{}, false
	}

	// match preferred languages and model languages
	lc := mc.languageTagMatch(dn, preferredLang)
	lcd, _, _ := mc.modelLangs(dn)
	if lc == "" && lcd == "" {
		omppLog.Log("Error: invalid (empty) model default language:", dn)
		return &ModelWordLabel{}, false
	}

	// lock model catalog
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	idx, ok := mc.indexByDigestOrName(dn)
	if !ok {
		return &ModelWordLabel{}, false // return empty result: model not found or error
	}

	// find lang_word rows in preferred or model default language
	mlw := ModelWordLabel{
		ModelName:   mc.modelLst[idx].meta.Model.Name,
		ModelDigest: mc.modelLst[idx].meta.Model.Digest}

	var nd, i int
	for i = 0; i < len(mc.modelLst[idx].langMeta.Lang); i++ {
		if mc.modelLst[idx].langMeta.Lang[i].LangCode == lc {
			break // language match
		}
		if mc.modelLst[idx].langMeta.Lang[i].LangCode == lcd {
			nd = i // index of default language
		}
	}
	if i >= len(mc.modelLst[idx].langMeta.Lang) {
		i = nd // use default language or zero index row
	}

	// copy lang_word rows, if exist for that language index
	if i < len(mc.modelLst[idx].langMeta.Lang) {
		mlw.LangCode = mc.modelLst[idx].langMeta.Lang[i].LangCode // actual language of result
		for c, v := range mc.modelLst[idx].langMeta.Lang[i].Words {
			mlw.LangWords = append(mlw.LangWords, codeLabel{Code: c, Label: v})
		}
	}

	// find model_word rows in preferred or model default language
	nd = 0
	for i = 0; i < len(mc.modelLst[idx].modelWord.ModelWord); i++ {
		if mc.modelLst[idx].modelWord.ModelWord[i].LangCode == lc {
			break // language match
		}
		if mc.modelLst[idx].modelWord.ModelWord[i].LangCode == lcd {
			nd = i // index of default language
		}
	}
	if i >= len(mc.modelLst[idx].modelWord.ModelWord) {
		i = nd // use default language or zero index row
	}

	// copy model_word rows, if exist for that language index
	if i < len(mc.modelLst[idx].modelWord.ModelWord) {
		mlw.ModelLangCode = mc.modelLst[idx].modelWord.ModelWord[i].LangCode // actual language of result
		for c, v := range mc.modelLst[idx].modelWord.ModelWord[i].Words {
			mlw.ModelWords = append(mlw.ModelWords, codeLabel{Code: c, Label: v})
		}
	}

	// find model translated strings in preferred or model default language
	nd = 0
	for i = 0; i < len(mc.modelLst[idx].msg); i++ {
		if strings.EqualFold(mc.modelLst[idx].msg[i].LangCode, lc) {
			break // language match
		}
		if strings.EqualFold(mc.modelLst[idx].msg[i].LangCode, lcd) {
			nd = i // index of default language
		}
	}
	if i >= len(mc.modelLst[idx].msg) {
		i = nd // use default language or zero index row
	}

	// copy model translated strings, if exist for that language index
	if i < len(mc.modelLst[idx].msg) {
		mlw.MsgLangCode = mc.modelLst[idx].msg[i].LangCode // actual language of result
		for c, v := range mc.modelLst[idx].msg[i].Msg {
			mlw.Msg = append(mlw.Msg, codeLabel{Code: c, Label: v})
		}
	}

	return &mlw, true
}

// EntityGenByName select entity_gen and entity_gen_attr db row by entity name, run id and model digest or name.
func (mc *ModelCatalog) EntityGenByName(dn string, runId int, entityName string) (*db.EntityMeta, *db.EntityGenMeta, bool) {

	if dn == "" {
		omppLog.Log("Error: invalid (empty) model digest and name")
		return nil, nil, false
	}
	if entityName == "" {
		omppLog.Log("Error: invalid (empty) entity name")
		return nil, nil, false
	}
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Error: model digest or name not found:", dn)
		return nil, nil, false // return empty result: model not found or error
	}

	// find model entity by entity name
	eIdx, ok := meta.EntityByName(entityName)
	if !ok {
		omppLog.Log("Error: model entity not found:", entityName)
		return nil, nil, false
	}
	ent := &meta.Entity[eIdx]

	// get list of entity generations for that model run
	egLst, err := db.GetEntityGenList(dbConn, runId)
	if err != nil {
		omppLog.Log("Error at get run entities:", entityName, ": ", runId, ":", err)
		return nil, nil, false
	}

	// find entity generation by entity id, as it is today model run has only one entity generation for each entity
	gIdx := -1
	for k := range egLst {

		if egLst[k].EntityId == ent.EntityId {
			gIdx = k
			break
		}
	}
	if gIdx < 0 {
		omppLog.Log("Error: model run entity generation not found:", entityName, ":", runId)
		return nil, nil, false
	}

	return ent, &egLst[gIdx], true
}

// EntityGenAttrsRunList select entity_gen and entity_gen_attr db rows and run_entity rows by entity name, run id and model digest or name.
func (mc *ModelCatalog) EntityGenAttrsRunList(dn string, runId int, entityName string) (*db.EntityMeta, *db.EntityGenMeta, []db.EntityAttrRow, []db.RunEntityRow, error) {

	ent, entGen, ok := mc.EntityGenByName(dn, runId, entityName)
	if !ok {
		return nil, nil, []db.EntityAttrRow{}, []db.RunEntityRow{}, helper.ErrorNew("Error: model run entity generation not found:", dn, ":", entityName+":", runId)
	}

	_, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		return nil, nil, []db.EntityAttrRow{}, []db.RunEntityRow{}, helper.ErrorNew("Error: model digest or name not found:", dn)
	}

	// collect generation attribues
	attrs := make([]db.EntityAttrRow, len(entGen.GenAttr))

	for k, ga := range entGen.GenAttr {

		aIdx, ok := ent.AttrByKey(ga.AttrId)
		if !ok {
			return nil, nil, []db.EntityAttrRow{}, []db.RunEntityRow{}, helper.ErrorNew("entity attribute not found by id:", ga.AttrId, entityName)
		}
		attrs[k] = ent.Attr[aIdx]
	}

	// find all run_entity rows for that entity generation
	runEnt, err := db.GetRunEntityGenByModel(dbConn, entGen.ModelId)
	if err != nil {
		return nil, nil, []db.EntityAttrRow{}, []db.RunEntityRow{}, helper.ErrorNew("Error at get run entities by model id:", entGen.ModelId, ":", err)
	}

	n := 0
	for k := 0; k < len(runEnt); k++ {
		if runEnt[k].GenHid == entGen.GenHid {
			runEnt[n] = runEnt[k]
			n++
		}
	}
	runEnt = runEnt[:n]

	return ent, entGen, attrs, runEnt, nil
}
