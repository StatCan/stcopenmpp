// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"crypto/md5"
	"errors"
	"fmt"
	"sort"
	"strconv"
)

// ToPublic convert model run db rows into "public" model run format for json import-export.
func (meta *RunMeta) ToPublic(modelDef *ModelMeta) (*RunPub, error) {

	// validate run model id: run must belong to the model
	if meta.Run.ModelId != modelDef.Model.ModelId {
		return nil, errors.New("model run: " + strconv.Itoa(meta.Run.RunId) + " " + meta.Run.Name + ", invalid model id " + strconv.Itoa(meta.Run.ModelId) + " expected: " + strconv.Itoa(modelDef.Model.ModelId))
	}

	// run header
	pub := RunPub{
		ModelName:           modelDef.Model.Name,
		ModelDigest:         modelDef.Model.Digest,
		ModelVersion:        modelDef.Model.Version,
		ModelCreateDateTime: modelDef.Model.CreateDateTime,
		Name:                meta.Run.Name,
		SubCount:            meta.Run.SubCount,
		SubStarted:          meta.Run.SubStarted,
		SubCompleted:        meta.Run.SubCompleted,
		CreateDateTime:      meta.Run.CreateDateTime,
		Status:              meta.Run.Status,
		UpdateDateTime:      meta.Run.UpdateDateTime,
		RunId:               meta.Run.RunId,
		RunDigest:           meta.Run.RunDigest,
		ValueDigest:         meta.Run.ValueDigest,
		RunStamp:            meta.Run.RunStamp,
		Opts:                make(map[string]string, len(meta.Opts)),
		Txt:                 make([]DescrNote, len(meta.Txt)),
		Param:               make([]ParamRunSetPub, len(meta.Param)),
		Table:               make([]TableRunPub, len(meta.Table)),
		Entity:              make([]EntityRunPub, len(meta.RunEntity)),
		Progress:            make([]RunProgress, len(meta.Progress)),
	}

	// copy run_option rows
	for k, v := range meta.Opts {
		pub.Opts[k] = v
	}

	// run description and notes by language
	for k := range meta.Txt {
		pub.Txt[k] = DescrNote{
			LangCode: meta.Txt[k].LangCode,
			Descr:    meta.Txt[k].Descr,
			Note:     meta.Txt[k].Note}
	}

	// run parameters and parameter value notes
	for k := range meta.Param {

		// find model parameter index by Hid
		idx, ok := modelDef.ParamByHid(meta.Param[k].ParamHid)
		if !ok {
			return nil, errors.New("model run: " + strconv.Itoa(meta.Run.RunId) + " " + meta.Run.Name + ", parameter " + strconv.Itoa(meta.Param[k].ParamHid) + " not found")
		}

		pub.Param[k] = ParamRunSetPub{
			ParamRunSetTxtPub: ParamRunSetTxtPub{
				Name: modelDef.Param[idx].Name,
				Txt:  make([]LangNote, len(meta.Param[k].Txt)),
			},
			SubCount:    meta.Param[k].SubCount,
			ValueDigest: meta.Param[k].ValueDigest,
		}
		for j := range meta.Param[k].Txt {
			pub.Param[k].Txt[j] = LangNote{
				LangCode: meta.Param[k].Txt[j].LangCode,
				Note:     meta.Param[k].Txt[j].Note,
			}
		}
	}

	// output tables included into model run results
	for k := range meta.Table {

		// find model output table index by Hid
		idx, ok := modelDef.OutTableByHid(meta.Table[k].TableHid)
		if !ok {
			return nil, errors.New("model run: " + strconv.Itoa(meta.Run.RunId) + " " + meta.Run.Name + ", table " + strconv.Itoa(meta.Table[k].TableHid) + " not found")
		}
		pub.Table[k] = TableRunPub{
			Name:        modelDef.Table[idx].Name,
			ValueDigest: meta.Table[k].ValueDigest,
		}
	}

	// run entity generations and attributes for each entity generation
	for k := range meta.RunEntity {

		// find entity generation index by Hid
		var entGen *EntityGenMeta = nil

		for j := range meta.EntityGen {

			if meta.EntityGen[j].GenHid == meta.RunEntity[k].GenHid {
				entGen = &meta.EntityGen[j]
				break
			}
		}
		if entGen == nil {
			return nil, errors.New("model run: " + strconv.Itoa(meta.Run.RunId) + " " + meta.Run.Name + ", entity Hid: " + strconv.Itoa(meta.RunEntity[k].GenHid) + " not found")
		}

		// find entity index by entity id (model_entity_id)
		eIdx, ok := modelDef.EntityByKey(entGen.EntityId)
		if !ok {
			return nil, errors.New("model run: " + strconv.Itoa(meta.Run.RunId) + " " + meta.Run.Name + ", entity " + strconv.Itoa(entGen.EntityId) + " not found")
		}
		ent := &modelDef.Entity[eIdx]
		pub.Entity[k].Name = ent.Name
		pub.Entity[k].GenDigest = entGen.GenDigest
		pub.Entity[k].RowCount = meta.RunEntity[k].RowCount
		pub.Entity[k].ValueDigest = meta.RunEntity[k].ValueDigest
		pub.Entity[k].Attr = make([]string, len(entGen.GenAttr))

		// find entity generation attributes
		for j, ga := range entGen.GenAttr {

			aIdx, ok := ent.AttrByKey(ga.AttrId)
			if !ok {
				return nil, errors.New("entity attribute not found by id: " + strconv.Itoa(ga.AttrId) + " " + ent.Name)
			}
			pub.Entity[k].Attr[j] = ent.Attr[aIdx].Name
		}
	}

	// copy run_progress rows
	copy(pub.Progress, meta.Progress)

	return &pub, nil
}

// FromPublic convert model run metadata from "public" format (coming from json import-export) into db rows.
func (pub *RunPub) FromPublic(modelDef *ModelMeta) (*RunMeta, error) {

	// validate parameters
	if modelDef == nil {
		return nil, errors.New("invalid (empty) model metadata")
	}
	if pub.ModelName == "" && pub.ModelDigest == "" {
		return nil, errors.New("invalid (empty) model name and digest, model run: " + pub.Name + " " + pub.CreateDateTime)
	}

	// validate run model name and/or digest: run must belong to the model
	if (pub.ModelName != "" && pub.ModelName != modelDef.Model.Name) ||
		(pub.ModelDigest != "" && pub.ModelDigest != modelDef.Model.Digest) {
		return nil, errors.New("invalid model name " + pub.ModelName + " or digest " + pub.ModelDigest + " expected: " + modelDef.Model.Name + " " + modelDef.Model.Digest)
	}

	// run header: run_lst row with zero default run id
	meta := RunMeta{
		Run: RunRow{
			RunId:          0, // ignore input value
			ModelId:        modelDef.Model.ModelId,
			Name:           pub.Name,
			SubCount:       pub.SubCount,
			SubStarted:     pub.SubStarted,
			SubCompleted:   pub.SubCompleted,
			CreateDateTime: pub.CreateDateTime,
			Status:         pub.Status,
			UpdateDateTime: pub.UpdateDateTime,
			RunDigest:      pub.RunDigest,
			ValueDigest:    pub.ValueDigest,
			RunStamp:       pub.RunStamp,
		},
		Txt:       make([]RunTxtRow, len(pub.Txt)),
		Opts:      make(map[string]string, len(pub.Opts)),
		Param:     make([]runParam, len(pub.Param)),
		Table:     make([]runTable, len(pub.Table)),
		EntityGen: make([]EntityGenMeta, len(pub.Entity)),
		RunEntity: []RunEntityRow{},
		Progress:  make([]RunProgress, len(pub.Progress)),
	}

	// model run description and notes: run_txt rows
	// use run id default zero
	for k := range pub.Txt {
		meta.Txt[k].LangCode = pub.Txt[k].LangCode
		meta.Txt[k].Descr = pub.Txt[k].Descr
		meta.Txt[k].Note = pub.Txt[k].Note
	}

	// run options
	for k, v := range pub.Opts {
		meta.Opts[k] = v
	}

	// run parameters and parameter value notes: run_parameter_txt rows
	// use set id default zero
	for k := range pub.Param {

		// find model parameter index by name
		idx, ok := modelDef.ParamByName(pub.Param[k].Name)
		if !ok {
			return nil, errors.New("model run: " + pub.Name + " parameter " + pub.Param[k].Name + " not found")
		}
		meta.Param[k].ParamHid = modelDef.Param[idx].ParamHid
		meta.Param[k].SubCount = pub.Param[k].SubCount
		// meta.Param[k].ValueDigest = pub.Param[k].ValueDigest // do not use input value digest

		// workset parameter value notes, use set id default zero
		if len(pub.Param[k].Txt) > 0 {
			meta.Param[k].Txt = make([]RunParamTxtRow, len(pub.Param[k].Txt))

			for j := range pub.Param[k].Txt {
				meta.Param[k].Txt[j].ParamHid = meta.Param[k].ParamHid
				meta.Param[k].Txt[j].LangCode = pub.Param[k].Txt[j].LangCode
				meta.Param[k].Txt[j].Note = pub.Param[k].Txt[j].Note
			}
		}
	}

	// output tables included into model run results
	for k := range meta.Table {

		// find model output table index by name
		idx, ok := modelDef.OutTableByName(pub.Table[k].Name)
		if !ok {
			return nil, errors.New("model run: " + pub.Name + " output table " + pub.Param[k].Name + " not found")
		}
		meta.Table[k].TableHid = modelDef.Table[idx].TableHid
		// meta.Table[k].ValueDigest = modelDef.Table[idx].ValueDigest // do not use input value digest
	}

	// run entity generations and attributes for each entity generation
	// use default zero for generation Hid
	// use default empty "" db table name, generation digest and microdata value digest
	for k, ePub := range pub.Entity {

		// find model entity by name
		eIdx, ok := modelDef.EntityByName(ePub.Name)
		if !ok {
			return nil, errors.New("model entity not found: " + ePub.Name)
		}
		ent := &modelDef.Entity[eIdx]

		meta.EntityGen[k].ModelId = modelDef.Model.ModelId
		meta.EntityGen[k].EntityId = ent.EntityId
		meta.EntityGen[k].EntityHid = ent.EntityHid
		meta.EntityGen[k].GenDigest = ePub.GenDigest
		// use default GenHid = 0, DbEntityTable = ""

		// find generation attributes by names, attributes must be ordered by attribute id
		nAttr := len(ePub.Attr)
		if nAttr <= 0 {
			return nil, errors.New("invalid (empty) model run entity generation attribute list: " + ePub.Name)
		}
		ai := make([]int, nAttr)

		for j := range ePub.Attr {

			n, ok := ent.AttrByName(ePub.Attr[j])
			if !ok {
				return nil, errors.New("invalid model run entity generation attribute: " + ePub.Attr[j] + " : " + ePub.Name)
			}
			ai[j] = ent.Attr[n].AttrId
		}

		sort.Ints(ai) // attributes order by id

		// check attribute list for duplicates
		iPrev := ai[0]
		for j := 1; j < nAttr; j++ {
			if ai[j] == iPrev {
				return nil, errors.New("invalid model run entity generation attribute list, it contains duplicates: " + ePub.Name)
			}
			iPrev = ai[j]
		}

		meta.EntityGen[k].GenAttr = make([]entityGenAttrRow, nAttr)

		for j := 0; j < nAttr; j++ {

			meta.EntityGen[k].GenAttr[j].AttrId = ai[j]
			// use default GenHid = 0
		}
	}

	meta.updateEntityGenInternals(modelDef) // set entity generation digest and db table name

	// copy run_progress rows
	copy(meta.Progress, pub.Progress)

	return &meta, nil
}

// Set entity generation internal members: generation digest and db table name.
// It must be called after restoring from json.
func (meta *RunMeta) updateEntityGenInternals(modelDef *ModelMeta) error {

	hMd5 := md5.New()

	for k := range meta.EntityGen {

		eGen := &meta.EntityGen[k]

		// find model entity
		eIdx, ok := modelDef.EntityByKey(eGen.EntityId)
		if !ok {
			return errors.New("model entity not found by id: " + strconv.Itoa(eGen.EntityId))
		}
		ent := &modelDef.Entity[eIdx]

		// entity generation digest header
		hMd5.Reset()

		_, err := hMd5.Write([]byte("entity_digest\n"))
		if err != nil {
			return err
		}
		_, err = hMd5.Write([]byte(
			ent.Digest + "\n"))
		if err != nil {
			return err
		}

		// add attributes: name and attribute type digest
		_, err = hMd5.Write([]byte("attr_name,type_digest\n"))
		if err != nil {
			return err
		}

		for j := range eGen.GenAttr {

			// find entity attribute
			aIdx, ok := ent.AttrByKey(eGen.GenAttr[j].AttrId)
			if !ok {
				return errors.New("model entity attribute not found by id: " + strconv.Itoa(eGen.GenAttr[j].AttrId))
			}
			_, err = hMd5.Write([]byte(
				ent.Attr[aIdx].Name + "," + ent.Attr[aIdx].typeOf.Digest + "\n"))
			if err != nil {
				return err
			}
		}

		eGen.GenDigest = fmt.Sprintf("%x", hMd5.Sum(nil)) // set generation digest string

		// make entity generation table name
		p, s := makeDbTablePrefixSuffix(ent.Name, eGen.GenDigest)

		eGen.DbEntityTable = p + "_g" + s
	}

	return nil
}
