// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
	"errors"
	"strconv"
)

// UpdateLanguage insert new or update existing language and words in lang_lst and lang_word tables.
// Language ids updated with actual id's from database
func UpdateLanguage(dbConn *sql.DB, langDef *LangMeta) error {

	// source is empty: nothing to do, exit
	if langDef == nil || len(langDef.Lang) <= 0 {
		return nil
	}

	// do update in transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return err
	}

	if err = doUpdateLanguage(trx, langDef); err != nil {
		trx.Rollback()
		return err
	}

	// rebuild language id index
	langDef.idIndex = make(map[int]int, len(langDef.Lang))
	for k := range langDef.Lang {
		langDef.idIndex[langDef.Lang[k].langId] = k
	}

	trx.Commit()
	return nil
}

// UpdateModelWord insert new or update existing model language-specific strings in model_word table.
func UpdateModelWord(dbConn *sql.DB, modelDef *ModelMeta, langDef *LangMeta, mwDef *ModelWordMeta) error {

	// source is empty: nothing to do, exit
	if mwDef == nil || len(mwDef.ModelWord) <= 0 {
		return nil
	}

	// validate parameters
	if modelDef == nil {
		return errors.New("invalid (empty) model metadata")
	}
	if langDef == nil {
		return errors.New("invalid (empty) language list")
	}
	if mwDef.ModelName != modelDef.Model.Name || mwDef.ModelDigest != modelDef.Model.Digest {
		return errors.New("invalid model name " + mwDef.ModelName + " or digest " + mwDef.ModelDigest + " expected: " + modelDef.Model.Name + " " + modelDef.Model.Digest)
	}

	// do update in transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	if err = doUpdateModelWord(trx, modelDef.Model.ModelId, langDef, mwDef); err != nil {
		trx.Rollback()
		return err
	}
	trx.Commit()
	return nil
}

// doUpdateLanguage insert new or update existing language and words in lang_lst and lang_word tables.
// It does update as part of transaction
// Language ids updated with actual id's from database
func doUpdateLanguage(trx *sql.Tx, langDef *LangMeta) error {

	// for each language:
	// if language code exist then update else insert into lang_lst
	for idx := range langDef.Lang {

		// check if this language already exist
		langDef.Lang[idx].langId = -1
		err := TrxSelectFirst(trx,
			"SELECT lang_id FROM lang_lst WHERE lang_code = "+ToQuoted(langDef.Lang[idx].LangCode),
			func(row *sql.Row) error {
				return row.Scan(&langDef.Lang[idx].langId)
			})
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		// if language exist then update else insert into lang_lst
		if langDef.Lang[idx].langId >= 0 {

			// UPDATE lang_lst SET lang_name = 'English' WHERE lang_id = 0
			err = TrxUpdate(trx,
				"UPDATE lang_lst"+
					" SET lang_name = "+toQuotedMax(langDef.Lang[idx].Name, nameDbMax)+
					" WHERE lang_id = "+strconv.Itoa(langDef.Lang[idx].langId))
			if err != nil {
				return err
			}

		} else { // insert into lang_lst

			// get new language id
			err = TrxUpdate(trx, "UPDATE id_lst SET id_value = id_value + 1 WHERE id_key = 'lang_id'")
			if err != nil {
				return err
			}
			err = TrxSelectFirst(trx,
				"SELECT id_value FROM id_lst WHERE id_key = 'lang_id'",
				func(row *sql.Row) error {
					return row.Scan(&langDef.Lang[idx].langId)
				})
			switch {
			case err == sql.ErrNoRows:
				return errors.New("invalid destination database, likely not an openM++ database")
			case err != nil:
				return err
			}

			// INSERT INTO lang_lst (lang_id, lang_code, lang_name) VALUES (0, 'EN', 'English')
			err = TrxUpdate(trx,
				"INSERT INTO lang_lst (lang_id, lang_code, lang_name)"+
					" VALUES ("+
					strconv.Itoa(langDef.Lang[idx].langId)+", "+
					toQuotedMax(langDef.Lang[idx].LangCode, codeDbMax)+", "+
					toQuotedMax(langDef.Lang[idx].Name, nameDbMax)+")")
			if err != nil {
				return err
			}
		}

		// update lang_word for that language
		if err = doUpdateWord(trx, langDef.Lang[idx].langId, langDef.Lang[idx].Words); err != nil {
			return err
		}
	}

	return nil
}

// doUpdateWord insert new or update existing language words in lang_word table.
// It does update as part of transaction
func doUpdateWord(trx *sql.Tx, langId int, wordRs map[string]string) error {

	// source is empty: nothing to do, exit
	if len(wordRs) <= 0 {
		return nil
	}

	// for each word:
	// if language id and word code exist then update else insert into lang_word
	for code, val := range wordRs {

		// skip word if code is "" empty
		if code == "" {
			continue
		}

		// check if this language word already exist
		// "SELECT COUNT(*) FROM lang_word WHERE lang_id = 0 AND word_code = 'EN'
		cnt := 0
		err := TrxSelectFirst(trx,
			"SELECT COUNT(*) FROM lang_word"+
				" WHERE lang_id = "+strconv.Itoa(langId)+
				" AND word_code = "+ToQuoted(code),
			func(row *sql.Row) error {
				return row.Scan(&cnt)
			})
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		// if language word exist and new value not empty then update else insert into lang_word
		if cnt > 0 && val != "" {

			// UPDATE lang_word SET word_value = 'Max' WHERE lang_id = 0 AND word_code = 'max'
			err = TrxUpdate(trx,
				"UPDATE lang_word"+
					" SET word_value = "+toQuotedMax(val, wordDbMax)+
					" WHERE lang_id = "+strconv.Itoa(langId)+
					" AND word_code = "+ToQuoted(code))
			if err != nil {
				return err
			}

		} else { // insert into lang_word

			// INSERT INTO lang_word (lang_id, word_code, word_value) VALUES (0, 'Max', 'max')
			err = TrxUpdate(trx,
				"INSERT INTO lang_word (lang_id, word_code, word_value)"+
					" VALUES ("+
					strconv.Itoa(langId)+", "+
					toQuotedMax(code, wordDbMax)+", "+
					toQuotedMax(val, wordDbMax)+")")
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// doUpdateModelWord insert new or update existing model language-specific strings in model_word table.
// It does update as part of transaction
func doUpdateModelWord(trx *sql.Tx, modelId int, langDef *LangMeta, mwDef *ModelWordMeta) error {

	// update model_word and ids
	smId := strconv.Itoa(modelId)
	for idx := range mwDef.ModelWord {

		// if language code valid then delete and insert into model_word
		if lId, ok := langDef.IdByCode(mwDef.ModelWord[idx].LangCode); ok {

			err := TrxUpdate(trx,
				"DELETE FROM model_word WHERE model_id = "+smId+" AND lang_id = "+strconv.Itoa(lId))
			if err != nil {
				return err
			}

			for code, val := range mwDef.ModelWord[idx].Words {
				if code == "" {
					continue // skip word if code is "" empty
				}
				err = TrxUpdate(trx,
					"INSERT INTO model_word (model_id, lang_id, word_code, word_value) VALUES ("+
						smId+", "+
						strconv.Itoa(lId)+", "+
						toQuotedMax(code, wordDbMax)+", "+
						toQuotedOrNullMax(val, wordDbMax)+")")
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
