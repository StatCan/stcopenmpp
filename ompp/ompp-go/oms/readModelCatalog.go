// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"

	"github.com/stcopenmpp/ompp-go/ompp/db"
	"golang.org/x/text/language"
)

// get "public" configuration of model catalog
func (mc *ModelCatalog) toPublicConfig() ModelCatalogConfig {
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	return ModelCatalogConfig{
		ModelDir:        mc.modelDir,
		ModelLogDir:     mc.modelLogDir,
		IsLogDirEnabled: mc.isLogDirEnabled,
		LastTimeStamp:   mc.lastTimeStamp,
	}
}

// getModelDir return model directory
func (mc *ModelCatalog) getModelDir() (string, bool) {
	mc.theLock.Lock()
	defer mc.theLock.Unlock()
	return mc.modelDir, mc.isDirEnabled
}

// getModelLogDir return default model directory and true if that directory exist
func (mc *ModelCatalog) getModelLogDir() (string, bool) {
	mc.theLock.Lock()
	defer mc.theLock.Unlock()
	return mc.modelLogDir, mc.isLogDirEnabled
}

// allModelDigests return digests for all models.
func (mc *ModelCatalog) allModelDigests() []string {
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	ds := make([]string, len(mc.modelLst))
	for idx := range mc.modelLst {
		ds[idx] = mc.modelLst[idx].meta.Model.Digest
	}
	return ds
}

// allModels return basic info from catalog about all models: name, digest, files location.
func (mc *ModelCatalog) allModels() []modelBasic {
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	mbs := make([]modelBasic, len(mc.modelLst))
	for idx := range mc.modelLst {
		mbs[idx] = modelBasic{
			model:    mc.modelLst[idx].meta.Model,
			binDir:   mc.modelLst[idx].binDir,
			dbPath:   mc.modelLst[idx].dbPath,
			relPath:  mc.modelLst[idx].relPath,
			logDir:   mc.modelLst[idx].logDir,
			isLogDir: mc.modelLst[idx].isLogDir,
			isIni:    mc.modelLst[idx].isIni,
			extra:    mc.modelLst[idx].extra,
		}

	}
	return mbs
}

// modelBasicByDigestOrName return basic info from catalog about model by digest or model name.
func (mc *ModelCatalog) modelBasicByDigestOrName(dn string) (modelBasic, bool) {
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	idx, ok := mc.indexByDigestOrName(dn)
	if !ok {
		return modelBasic{}, false // model not found, empty result
	}
	return modelBasic{
			model:    mc.modelLst[idx].meta.Model,
			binDir:   mc.modelLst[idx].binDir,
			dbPath:   mc.modelLst[idx].dbPath,
			relPath:  mc.modelLst[idx].relPath,
			logDir:   mc.modelLst[idx].logDir,
			isLogDir: mc.modelLst[idx].isLogDir,
			isIni:    mc.modelLst[idx].isIni,
			extra:    mc.modelLst[idx].extra,
		},
		true
}

// modelMeta return model metadata and database connection by digest or model name.
// It is also return boolean success flag if both model metadata pointer and connection pointer are not nil.
func (mc *ModelCatalog) modelMeta(dn string) (*db.ModelMeta, *sql.DB, bool) {
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	idx, ok := mc.indexByDigestOrName(dn)
	if !ok {
		return nil, nil, false // model not found, empty result
	}
	return mc.modelLst[idx].meta, mc.modelLst[idx].dbConn, (mc.modelLst[idx].meta != nil && mc.modelLst[idx].dbConn != nil)
}

// modelTextMeta return model text metadata:
// boolean flag to indicate if text metadata fully loaded from database and ModelTxtMeta itself.
func (mc *ModelCatalog) modelTextMeta(digest string) (bool, *db.ModelTxtMeta) {
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	idx, ok := mc.indexByDigest(digest)
	if !ok {
		return false, nil // model not found, empty result
	}
	return (mc.modelLst[idx].isTxtMetaFull && mc.modelLst[idx].txtMeta != nil), mc.modelLst[idx].txtMeta
}

// modelLangs return
// model default language, language codes (first is default language) and model found flag (false if model not found)
func (mc *ModelCatalog) modelLangs(dn string) (string, []string, bool) {
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	if idx, ok := mc.indexByDigestOrName(dn); ok {
		return mc.modelLst[idx].meta.Model.DefaultLangCode, mc.modelLst[idx].langCodes, mc.modelLst[idx].meta.Model.DefaultLangCode != ""
	}
	return "", []string{}, false
}

// modelLangMeta return model LangMeta: lang_lst rows and lang_word rows for each language
func (mc *ModelCatalog) modelLangMeta(dn string) *db.LangMeta {
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	if idx, ok := mc.indexByDigestOrName(dn); ok {
		return mc.modelLst[idx].langMeta
	}
	return nil
}

// modelEntityAttrs return model entities and attributes from catalog about model by digest.
func (mc *ModelCatalog) entityAttrsByDigest(digest string) []db.EntityMeta {
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	idx, ok := mc.indexByDigest(digest)
	if !ok {
		return []db.EntityMeta{} // model not found, empty result
	}

	// copy model entity db rows
	em := make([]db.EntityMeta, len(mc.modelLst[idx].meta.Entity))

	for k := range mc.modelLst[idx].meta.Entity {

		em[k] = mc.modelLst[idx].meta.Entity[k]
		em[k].Attr = make([]db.EntityAttrRow, len(mc.modelLst[idx].meta.Entity[k].Attr))
		copy(em[k].Attr, mc.modelLst[idx].meta.Entity[k].Attr)
	}
	return em
}

// languageTagMatch return model language code matched to request languages.
// if there is no such match then return is empty "" language code.
func (mc *ModelCatalog) languageTagMatch(dn string, preferredLang []language.Tag) string {
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	idx, ok := mc.indexByDigestOrName(dn)
	if !ok {
		return ""
	}

	_, np, _ := mc.modelLst[idx].matcher.Match(preferredLang...)

	if np >= 0 && np < len(mc.modelLst[idx].langCodes) {
		return mc.modelLst[idx].langCodes[np]
	}
	return mc.modelLst[idx].meta.Model.DefaultLangCode
}

// languageCodeMatch return model language matched to request language, e.g. EN from en-CA.
// if there is no such match then return is empty "" language code.
func (mc *ModelCatalog) languageCodeMatch(dn string, langCode string) string {
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	idx, ok := mc.indexByDigestOrName(dn)
	if !ok {
		return ""
	}

	_, np, _ := mc.modelLst[idx].matcher.Match(language.Make(langCode))

	if np >= 0 && np < len(mc.modelLst[idx].langCodes) {
		return mc.modelLst[idx].langCodes[np]
	}
	return ""
}

// indexByDigest return index of model by digest.
//
// It can be used only inside the lock.
func (mc *ModelCatalog) indexByDigest(digest string) (int, bool) {
	for k := range mc.modelLst {
		if mc.modelLst[k].meta.Model.Digest == digest {
			return k, true
		}
	}
	return 0, false
}

// indexByDigestOrName return index of model by digest or by name.
//
// It can be used only inside the lock.
// If digest exist in model list then return index by digest else first index of name.
// If not found then return false flag.
func (mc *ModelCatalog) indexByDigestOrName(dn string) (int, bool) {
	n := -1
	for k := range mc.modelLst {
		if mc.modelLst[k].meta.Model.Digest == dn {
			return k, true // return: digest found
		}
		if n < 0 && mc.modelLst[k].meta.Model.Name == dn {
			n = k
		}
	}
	if n >= 0 {
		return n, true // return: name found
	}
	return 0, false // not found
}
