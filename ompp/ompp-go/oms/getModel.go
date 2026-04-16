// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
	"golang.org/x/text/language"
)

// ModelDicByDigest return model_dic db row by model digest.
func (mc *ModelCatalog) ModelDicByDigest(digest string) (db.ModelDicRow, bool) {

	// if model digest is empty then return empty results
	if digest == "" {
		omppLog.Log("Warning: invalid (empty) model digest")
		return db.ModelDicRow{}, false
	}

	// find model index by digest
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	idx, ok := mc.indexByDigest(digest)
	if !ok {
		return db.ModelDicRow{}, false // model not found, empty result
	}

	return mc.modelLst[idx].meta.Model, true
}

// ModelDicByDigestOrName return model_dic db row by model digest or name.
func (mc *ModelCatalog) ModelDicByDigestOrName(dn string) (db.ModelDicRow, bool) {

	// if model digest is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return db.ModelDicRow{}, false
	}

	// find model index by digest
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	idx, ok := mc.indexByDigestOrName(dn)
	if !ok {
		return db.ModelDicRow{}, false // model not found, empty result
	}

	return mc.modelLst[idx].meta.Model, true
}

// find model metadata by digest or name
func (mc *ModelCatalog) ModelMetaByDigestOrName(dn string) (*db.ModelMeta, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return &db.ModelMeta{}, false
	}

	// lock model catalog and return copy of model metadata
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	idx, ok := mc.indexByDigestOrName(dn)
	if !ok {
		return &db.ModelMeta{}, false // return empty result: model not found or error
	}

	return mc.modelLst[idx].meta, true
}

// ModelTextByDigest return model_dic_txt db row by model digest and preferred language tags.
// It can be in preferred language, default model language or empty if no model_dic_txt rows exist.
func (mc *ModelCatalog) ModelTextByDigest(digest string, preferredLang []language.Tag) (*db.ModelDicDescrNote, bool) {

	// if model digest is empty then return empty results
	if digest == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return &db.ModelDicDescrNote{}, false
	}

	// get model_dic row
	mdRow, ok := mc.ModelDicByDigest(digest)
	if !ok {
		omppLog.Log("Warning: model digest not found:", digest)
		return &db.ModelDicDescrNote{}, false // return empty result: model not found or error
	}

	// get model_dic_txt rows from catalog: it is loaded at catalog initialization
	_, txt := mc.modelTextMeta(digest)
	if txt == nil {
		return &db.ModelDicDescrNote{}, false // return empty result: model not found or error
	}

	// match preferred languages and model languages
	lc := mc.languageTagMatch(digest, preferredLang)
	lcd, _, _ := mc.modelLangs(digest)
	if lc == "" && lcd == "" {
		omppLog.Log("Error: invalid (empty) model default language:", digest)
		return &db.ModelDicDescrNote{}, false
	}

	// if model_dic_txt rows not empty then find row by matched language or by default language
	t := db.ModelDicDescrNote{Model: mdRow}

	if len(txt.ModelTxt) > 0 {

		var nd, i int
		for ; i < len(txt.ModelTxt); i++ {
			if txt.ModelTxt[i].LangCode == lc {
				break // language match
			}
			if txt.ModelTxt[i].LangCode == lcd {
				nd = i // index of default language
			}
		}
		if i >= len(txt.ModelTxt) {
			i = nd // use default language or zero index row
		}

		t.DescrNote = db.DescrNote{
			LangCode: txt.ModelTxt[i].LangCode,
			Descr:    txt.ModelTxt[i].Descr,
			Note:     txt.ModelTxt[i].Note}
	}
	return &t, true
}

// ModelMetaAllTextByDigest return language-specific model metadata by model digest or name in all languages.
func (mc *ModelCatalog) ModelMetaAllTextByDigestOrName(dn string) (*db.ModelTxtMeta, error) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return &db.ModelTxtMeta{}, nil
	}

	// if language-specific model metadata not loaded then read it from database
	if ok := mc.loadModelText(dn); !ok {
		return &db.ModelTxtMeta{}, nil // return empty result: model not found or error
	}

	// lock model catalog and return copy of model metadata
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	idx, ok := mc.indexByDigestOrName(dn)
	if !ok {
		return &db.ModelTxtMeta{}, nil // return empty result: model not found or error
	}

	return mc.modelLst[idx].txtMeta, nil
}

// loadModelText reads language-specific model metadata from db by digest or name.
// If metadata already loaded then skip db reading and return success.
func (mc *ModelCatalog) loadModelText(dn string) bool {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return false
	}

	// get model_dic row
	mdRow, ok := mc.ModelDicByDigestOrName(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return false // model not found or error
	}

	// check if model text metadata already fully loaded from database
	if isFull, _ := mc.modelTextMeta(mdRow.Digest); isFull {
		return true
	}
	// else: no model text in catalog: read from database and update catalog

	// get database connection
	_, dbConn, ok := mc.modelMeta(mdRow.Digest)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return false // model not found or error
	}

	// read model text metadata from database and update catalog
	txt, err := db.GetModelText(dbConn, mdRow.ModelId, "", true)
	if err != nil {
		omppLog.Log("Error at get model text metadata:", dn, ":", err)
		return false
	}

	ok = mc.setModelTextMeta(mdRow.Digest, true, txt)
	if !ok {
		omppLog.Log("Error: model digest not found:", mdRow.Digest)
		return false // model not found or error
	}
	return true
}
