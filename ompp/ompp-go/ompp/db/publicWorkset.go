// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
	"errors"
	"strconv"
)

// ToPublic convert workset db rows into "public" workset format for json import-export
func (meta *WorksetMeta) ToPublic(dbConn *sql.DB, modelDef *ModelMeta) (*WorksetPub, error) {

	// validate workset model id: workset must belong to the model
	if meta.Set.ModelId != modelDef.Model.ModelId {
		return nil, errors.New("workset: " + strconv.Itoa(meta.Set.SetId) + " " + meta.Set.Name + ", invalid model id " + strconv.Itoa(meta.Set.ModelId) + " expected: " + strconv.Itoa(modelDef.Model.ModelId))
	}

	// workset header
	pub := WorksetPub{
		WorksetHdrPub: WorksetHdrPub{
			ModelName:           modelDef.Model.Name,
			ModelDigest:         modelDef.Model.Digest,
			ModelVersion:        modelDef.Model.Version,
			ModelCreateDateTime: modelDef.Model.CreateDateTime,
			Name:                meta.Set.Name,
			IsReadonly:          meta.Set.IsReadonly,
			UpdateDateTime:      meta.Set.UpdateDateTime,
			Txt:                 make([]DescrNote, len(meta.Txt))},
		Param: make([]ParamRunSetPub, len(meta.Param)),
	}

	// find base run digest by id, if workset based on run then base run id must be positive
	if meta.Set.BaseRunId > 0 {
		runRow, err := GetRun(dbConn, meta.Set.BaseRunId)
		if err != nil {
			return nil, err
		}
		if runRow != nil {
			pub.BaseRunDigest = runRow.RunDigest // base run found
		}
	}

	// workset description and notes by language
	for k := range meta.Txt {
		pub.Txt[k] = DescrNote{
			LangCode: meta.Txt[k].LangCode,
			Descr:    meta.Txt[k].Descr,
			Note:     meta.Txt[k].Note}
	}

	// workset parameters and parameter value notes
	for k := range meta.Param {

		// find model parameter index by name
		idx, ok := modelDef.ParamByHid(meta.Param[k].ParamHid)
		if !ok {
			return nil, errors.New("workset: " + strconv.Itoa(meta.Set.SetId) + " " + meta.Set.Name + ", parameter " + strconv.Itoa(meta.Param[k].ParamHid) + " not found")
		}

		pub.Param[k] = ParamRunSetPub{
			ParamRunSetTxtPub: ParamRunSetTxtPub{
				Name: modelDef.Param[idx].Name,
				Txt:  make([]LangNote, len(meta.Param[k].Txt)),
			},
			SubCount:     meta.Param[k].SubCount,
			DefaultSubId: meta.Param[k].DefaultSubId,
		}
		for j := range meta.Param[k].Txt {
			pub.Param[k].Txt[j] = LangNote{
				LangCode: meta.Param[k].Txt[j].LangCode,
				Note:     meta.Param[k].Txt[j].Note,
			}
		}
	}

	return &pub, nil
}

// FromPublic convert workset metadata from "public" format (coming from json import-export) into db rows.
func (pub *WorksetPub) FromPublic(dbConn *sql.DB, modelDef *ModelMeta) (*WorksetMeta, error) {

	// validate parameters
	if modelDef == nil {
		return nil, errors.New("invalid (empty) model metadata")
	}
	if pub.Name == "" {
		return nil, errors.New("invalid (empty) workset name")
	}
	if pub.ModelName == "" && pub.ModelDigest == "" {
		return nil, errors.New("invalid (empty) model name and digest, workset: " + pub.Name)
	}

	// validate workset model name and/or digest: workset must belong to the model
	if (pub.ModelName != "" && pub.ModelName != modelDef.Model.Name) ||
		(pub.ModelDigest != "" && pub.ModelDigest != modelDef.Model.Digest) {
		return nil, errors.New("invalid workset model name " + pub.ModelName + " or digest " + pub.ModelDigest + " expected: " + modelDef.Model.Name + " " + modelDef.Model.Digest)
	}

	// workset header: workset_lst row with zero default set id
	ws := WorksetMeta{
		Set: WorksetRow{
			SetId:          0, // set id is undefined
			BaseRunId:      0, // if base run id <= 0 then workset is not based on run
			Name:           pub.Name,
			ModelId:        modelDef.Model.ModelId,
			IsReadonly:     pub.IsReadonly,
			UpdateDateTime: pub.UpdateDateTime,
			isNullBaseRun:  pub.IsCleanBaseRun,
		},
		Txt:   make([]WorksetTxtRow, len(pub.Txt)),
		Param: make([]worksetParam, len(pub.Param)),
	}

	// if base run digest not "" empty then find base run for that workset
	if pub.BaseRunDigest != "" {
		runRow, err := GetRunByDigest(dbConn, pub.BaseRunDigest)
		if err != nil {
			return nil, err
		}
		if runRow != nil {
			ws.Set.BaseRunId = runRow.RunId //	base run found
		}
	}

	// workset description and notes: workset_txt rows
	// use set id default zero
	for k := range pub.Txt {
		ws.Txt[k].LangCode = pub.Txt[k].LangCode
		ws.Txt[k].Descr = pub.Txt[k].Descr
		ws.Txt[k].Note = pub.Txt[k].Note
	}

	// workset parameters and parameter value notes: workset_parameter, workset_parameter_txt rows
	// use set id default zero
	for k := range pub.Param {

		// find model parameter index by name
		idx, ok := modelDef.ParamByName(pub.Param[k].Name)
		if !ok {
			return nil, errors.New("workset: " + pub.Name + " parameter " + pub.Param[k].Name + " not found")
		}
		ws.Param[k].ParamHid = modelDef.Param[idx].ParamHid
		ws.Param[k].SubCount = pub.Param[k].SubCount
		ws.Param[k].DefaultSubId = pub.Param[k].DefaultSubId

		// workset parameter value notes, use set id default zero
		if len(pub.Param[k].Txt) > 0 {
			ws.Param[k].Txt = make([]WorksetParamTxtRow, len(pub.Param[k].Txt))

			for j := range pub.Param[k].Txt {
				ws.Param[k].Txt[j].ParamHid = ws.Param[k].ParamHid
				ws.Param[k].Txt[j].LangCode = pub.Param[k].Txt[j].LangCode
				ws.Param[k].Txt[j].Note = pub.Param[k].Txt[j].Note
			}
		}
	}

	return &ws, nil
}
