// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"crypto/md5"
	"database/sql"
	"errors"
	"fmt"
	"hash"
	"strconv"
)

// WriteMicrodataFrom insert microdata values into model run until from() return not nil CellMicro value.
//
// Model run should not already contain microdata values: microdata can be inserted only once in model run and cannot be updated after.
// Entity generation digest is used to find existing entity generation, it cannot be empty "" string otherwise error returned.
// Entity generation metadata updated with actual database values of generation Hid
//
// Double format is used for float model types digest calculation, if non-empty format supplied.
//
// Return run entity metadata rows.
func WriteMicrodataFrom(
	dbConn *sql.DB, dbFacet Facet, modelDef *ModelMeta, runMeta *RunMeta, layout *WriteMicroLayout, from func() (interface{}, error),
) error {

	// validate parameters
	if modelDef == nil {
		return errors.New("invalid (empty) model metadata, look like model not found")
	}
	if runMeta == nil {
		return errors.New("invalid (empty) model run metadata, look like model run not found")
	}
	if layout == nil {
		return errors.New("invalid (empty) write layout")
	}
	if layout.Name == "" {
		return errors.New("invalid (empty) entity microdata name")
	}
	if layout.ToId <= 0 {
		return errors.New("invalid destination run id: " + strconv.Itoa(layout.ToId))
	}
	if from == nil {
		return errors.New("invalid (empty) microdata values")
	}

	// do insert or update microdata in transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return err
	}

	reRows, err := doWriteMicrodataFrom(trx, dbFacet, modelDef, runMeta, layout.Name, layout.ToId, from, layout.DoubleFmt)
	if err != nil {
		trx.Rollback()
		return err
	}
	trx.Commit()

	runMeta.RunEntity = reRows // update run entity db rows with actual values
	return nil
}

// doWriteMicrodataFrom insert microdata values into model run.
// It does insert as part of transaction.
// Model run must exist and be in completed state (i.e. success or error state).
// Model run should not already contain microdata values: microdata can be inserted only once in model run and cannot be updated after.
// Double format is used for float model types digest calculation, if non-empty format supplied.
// Return run entity metadata rows.
func doWriteMicrodataFrom(
	trx *sql.Tx, dbFacet Facet, modelDef *ModelMeta, runMeta *RunMeta, entityName string, runId int, from func() (interface{}, error), doubleFmt string,
) ([]RunEntityRow, error) {

	// find entity by name
	eIdx, ok := modelDef.EntityByName(entityName)
	if !ok {
		return []RunEntityRow{}, errors.New("entity not found: " + entityName)
	}
	entity := &modelDef.Entity[eIdx]

	// find entity generation by name and check: generation digest and db table name are not empty
	gIdx, ok := runMeta.EntityGenByEntityId(modelDef.Entity[eIdx].EntityId)
	if !ok {
		return []RunEntityRow{}, errors.New("model entity generation not found by entity id: " + strconv.Itoa(modelDef.Entity[eIdx].EntityId) + " " + entityName)
	}
	entityGen := &runMeta.EntityGen[gIdx]

	if entityGen == nil {
		return []RunEntityRow{}, errors.New("invalid (empty) entity generation metadata")
	}
	if entityGen.GenDigest == "" {
		return []RunEntityRow{}, errors.New("invalid (empty) microdata entity generation digest, entity: " + entityName)
	}
	if entityGen.DbEntityTable == "" {
		return []RunEntityRow{}, errors.New("invalid (empty) microdata entity database table name, entity: " + entityName)
	}

	// find entity generation attributes
	entAttr := make([]EntityAttrRow, len(entityGen.GenAttr))

	for k, ga := range entityGen.GenAttr {

		aIdx, isOk := entity.AttrByKey(ga.AttrId)
		if !isOk {
			return []RunEntityRow{}, errors.New("entity attribute not found, id: " + strconv.Itoa(ga.AttrId) + " " + entityName)
		}
		entAttr[k] = entity.Attr[aIdx]
	}

	//
	// update model run master record to prevent run use
	//
	sRunId := strconv.Itoa(runId)
	err := TrxUpdate(trx,
		"UPDATE run_lst SET sub_restart = sub_restart - 1 WHERE run_id = "+sRunId)
	if err != nil {
		return []RunEntityRow{}, errors.New("insert microdata failed: " + entityName + ": " + err.Error())
	}

	// check if model run exist and status is completed
	st := ""
	err = TrxSelectFirst(trx,
		"SELECT status FROM run_lst WHERE run_id = "+sRunId,
		func(row *sql.Row) error {
			if err := row.Scan(&st); err != nil {
				return err
			}
			return nil
		})
	switch {
	case err == sql.ErrNoRows:
		return []RunEntityRow{}, errors.New("model run not found, id: " + sRunId)
	case err != nil:
		return []RunEntityRow{}, errors.New("insert microdata failed: " + entityName + ": " + err.Error())
	}
	if !IsRunCompleted(st) {
		return []RunEntityRow{}, errors.New("model run not completed, id: " + sRunId)
	}

	// check if microdata values not already exist for that run
	sEntHid := strconv.Itoa(entity.EntityHid)
	n := 0
	err = TrxSelectFirst(trx,
		"SELECT COUNT(*)"+
			" FROM run_entity RE"+
			" INNER JOIN entity_gen EG ON (EG.entity_gen_hid = RE.entity_gen_hid)"+
			" WHERE RE.run_id = "+sRunId+
			" AND EG.entity_hid = "+sEntHid,
		func(row *sql.Row) error {
			if err := row.Scan(&n); err != nil {
				return err
			}
			return nil
		})
	switch {
	case err != nil && err != sql.ErrNoRows:
		return []RunEntityRow{}, errors.New("insert microdata failed: " + entityName + ": " + err.Error())
	}
	if n > 0 {
		return []RunEntityRow{}, errors.New("model run with id: " + sRunId + " already contain microdata values " + entityName)
	}

	// UPDATE id_lst SET id_value =
	//   CASE
	//     WHEN 0 = (SELECT COUNT(*) FROM entity_gen WHERE gen_digest = 'abdcd9785')
	//       THEN id_value + 1
	//     ELSE id_value
	//   END
	// WHERE id_key = 'entity_hid'
	err = TrxUpdate(trx,
		"UPDATE id_lst SET id_value ="+
			" CASE"+
			" WHEN 0 = (SELECT COUNT(*) FROM entity_gen WHERE gen_digest = "+ToQuoted(entityGen.GenDigest)+")"+
			" THEN id_value + 1"+
			" ELSE id_value"+
			" END"+
			" WHERE id_key = 'entity_hid'")
	if err != nil {
		return []RunEntityRow{}, errors.New("insert microdata failed: " + entityName + ": " + err.Error())
	}

	// check if entity generation exist by generation digest
	genHid := 0
	err = TrxSelectFirst(trx,
		"SELECT entity_gen_hid FROM entity_gen WHERE gen_digest = "+ToQuoted(entityGen.GenDigest),
		func(row *sql.Row) error {
			if err := row.Scan(&genHid); err != nil {
				return err
			}
			return nil
		})
	switch {
	case err != nil && err != sql.ErrNoRows:
		return []RunEntityRow{}, errors.New("insert microdata failed: " + entityName + ": " + err.Error())
	}
	sGenHid := strconv.Itoa(genHid)

	// create new entity generation
	// insert into entity_gen and entity_gen_attr metadata tables
	// create microdata values db table, if not exist
	if genHid <= 0 {

		// get new entity generation Hid
		err = TrxSelectFirst(trx,
			"SELECT id_value FROM id_lst WHERE id_key = 'entity_hid'",
			func(row *sql.Row) error {
				return row.Scan(&genHid)
			})
		switch {
		case err == sql.ErrNoRows:
			return []RunEntityRow{}, errors.New("invalid destination database, likely not an openM++ database")
		case err != nil:
			return []RunEntityRow{}, errors.New("insert microdata failed: " + entityName + ": " + err.Error())
		}

		// insert entity generation metadata and generation attributes metadata
		sGenHid = strconv.Itoa(genHid)

		err = TrxUpdate(trx,
			"INSERT INTO entity_gen (entity_gen_hid, entity_hid, db_entity_table, gen_digest)"+
				" VALUES ("+
				sGenHid+", "+
				sEntHid+", "+
				ToQuoted(entityGen.DbEntityTable)+", "+
				toQuotedMax(entityGen.GenDigest, codeDbMax)+")",
		)
		if err != nil {
			return []RunEntityRow{}, errors.New("insert microdata failed: " + entityName + ": " + err.Error())
		}

		for _, a := range entAttr {

			err = TrxUpdate(trx,
				"INSERT INTO entity_gen_attr (entity_gen_hid, attr_id, entity_hid)"+
					" VALUES ("+
					sGenHid+", "+
					strconv.Itoa(a.AttrId)+", "+
					sEntHid+")",
			)
			if err != nil {
				return []RunEntityRow{}, errors.New("insert microdata failed: " + entityName + ": " + err.Error())
			}
		}

		// CREATE TABLE Person_g87abcdef
		// (
		//   run_id     INT    NOT NULL,
		//   entity_key BIGINT NOT NULL,
		//   attr4      INT    NOT NULL,
		//   attr7      FLOAT,          -- float attribute value NaN is NULL
		//   PRIMARY KEY (run_id, entity_key)
		// )
		//
		attrSql := ""

		for _, a := range entAttr {

			sqlType, err := a.typeOf.sqlColumnType(dbFacet)
			if err != nil {
				return []RunEntityRow{}, errors.New("insert microdata failed: " + entityName + ": " + err.Error())
			}
			if a.typeOf.IsFloat() {
				attrSql += a.colName + " " + sqlType + " NULL, "
			} else {
				attrSql += a.colName + " " + sqlType + " NOT NULL, "
			}
		}

		tSql := dbFacet.createTableIfNotExist(
			entityGen.DbEntityTable,
			"("+
				"run_id INT NOT NULL, "+
				"entity_key BIGINT NOT NULL, "+
				attrSql+
				"PRIMARY KEY (run_id, entity_key)"+
				")",
		)
		err = TrxUpdate(trx, tSql)
		if err != nil {
			return []RunEntityRow{}, errors.New("insert microdata failed: " + entityName + ": " + err.Error())
		}
	}

	// update run metadata with actual entity generation Hid
	entityGen.GenHid = genHid

	for k := range entityGen.GenAttr {
		entityGen.GenAttr[k].GenHid = genHid
	}

	// insert into run_entity with current run id as base run id and NULL digest
	err = TrxUpdate(trx,
		"INSERT INTO run_entity (run_id, entity_gen_hid, base_run_id, row_count, value_digest)"+
			" VALUES ("+
			sRunId+", "+sGenHid+", "+sRunId+", 0, NULL)",
	)
	if err != nil {
		return []RunEntityRow{}, errors.New("insert microdata failed: " + entityName + ": " + err.Error())
	}

	// create microdata digest calculator
	var rowCount int
	hMd5, digestFrom, err := digestMicrodataFrom(modelDef, entityName, entityGen, &rowCount, doubleFmt)
	if err != nil {
		return []RunEntityRow{}, errors.New("insert microdata failed: " + entityName + ": " + err.Error())
	}

	// make sql to insert microdata values into model run
	// prepare put() closure to convert each cell into parameters of insert sql statement
	q := makeSqlInsertMicroValue(entityGen.DbEntityTable, entAttr, runId)
	put := putInsertMicroFrom(entityName, entAttr, from, digestFrom)

	// execute sql insert using put() above for each row
	if err = TrxUpdateStatement(trx, q, put); err != nil {
		return []RunEntityRow{}, errors.New("insert microdata failed: " + entityName + ": " + err.Error())
	}

	// update microdata digest with actual value
	dgst := fmt.Sprintf("%x", hMd5.Sum(nil))

	err = TrxUpdate(trx,
		"UPDATE run_entity"+
			" SET value_digest = "+ToQuoted(dgst)+","+
			" row_count = "+strconv.Itoa(rowCount)+
			" WHERE run_id = "+sRunId+
			" AND entity_gen_hid ="+sGenHid)
	if err != nil {
		return []RunEntityRow{}, errors.New("insert microdata failed: " + entityName + ": " + err.Error())
	}

	// find base run by digest, it must exist
	nBase := 0
	err = TrxSelectFirst(trx,
		"SELECT MIN(run_id) FROM run_entity"+
			" WHERE entity_gen_hid = "+sGenHid+
			" AND value_digest = "+ToQuoted(dgst),
		func(row *sql.Row) error {
			if err := row.Scan(&nBase); err != nil {
				return err
			}
			return nil
		})
	switch {
	// case err == sql.ErrNoRows: it must exist, at least as newly inserted row above
	case err != nil:
		return []RunEntityRow{}, errors.New("insert microdata failed: " + entityName + ": " + err.Error())
	}

	// if microdata values already exist then update base run id
	// and remove duplicate values
	if runId != nBase {

		err = TrxUpdate(trx,
			"UPDATE run_entity SET base_run_id = "+strconv.Itoa(nBase)+
				" WHERE run_id = "+sRunId+
				" AND entity_gen_hid ="+sGenHid)
		if err != nil {
			return []RunEntityRow{}, errors.New("insert microdata failed: " + entityName + ": " + err.Error())
		}
		err = TrxUpdate(trx, "DELETE FROM "+entityGen.DbEntityTable+" WHERE run_id = "+sRunId)
		if err != nil {
			return []RunEntityRow{}, errors.New("insert microdata failed: " + entityName + ": " + err.Error())
		}
	}

	// select actual list of microdata run value digests ordered by entity generation Hid
	reRows := []RunEntityRow{}

	err = TrxSelectRows(trx,
		"SELECT run_id, entity_gen_hid, row_count, value_digest"+
			" FROM run_entity"+
			" WHERE run_id = "+strconv.Itoa(runId)+
			" ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var r RunEntityRow
			var nId int
			var svd sql.NullString
			if err := rows.Scan(&nId, &r.GenHid, &r.RowCount, &svd); err != nil {
				return err
			}
			if svd.Valid {
				r.ValueDigest = svd.String
			}
			reRows = append(reRows, r)
			return nil
		})
	if err != nil {
		return []RunEntityRow{}, errors.New("insert microdata failed: " + entityName + ": " + err.Error())
	}

	// completed OK, restore run_lst values
	err = TrxUpdate(trx,
		"UPDATE run_lst SET sub_restart = sub_restart + 1 WHERE run_id = "+sRunId)
	if err != nil {
		return []RunEntityRow{}, errors.New("insert microdata failed: " + entityName + ": " + err.Error())
	}

	return reRows, nil
}

// make sql to insert microdata values into model run or workset
func makeSqlInsertMicroValue(dbTable string, attrs []EntityAttrRow, toId int) string {

	// INSERT INTO Person_g87abcdef (run_id, entity_key, attr0, attr3) VALUES (?, ?, ?, ?)
	q := "INSERT INTO " + dbTable + "(run_id, entity_key"

	for k := range attrs {
		q += ", " + attrs[k].colName
	}

	q += ") VALUES (" + strconv.Itoa(toId) + ", ?"

	for k := 0; k < len(attrs); k++ {
		q += ", ?"
	}
	q += ")"

	return q
}

// prepare put() closure to convert each cell into parameters of insert sql statement until from() return not nil CellMicro value.
func putInsertMicroFrom(
	entityName string, attrs []EntityAttrRow, from func() (interface{}, error), digestFrom func(interface{}) error,
) func() (bool, []interface{}, error) {

	// converter from value into db value
	nAttr := len(attrs)
	fv := make([]func(bool, interface{}) (interface{}, error), nAttr)

	for k := 0; k < nAttr; k++ {
		fv[k] = cvtToSqlDbValue(attrs[k].typeOf.IsFloat(), attrs[k].typeOf, entityName+"."+attrs[k].Name)
	}

	// for each cell of microdata put into row of sql statement parameters
	row := make([]interface{}, 1+nAttr)

	put := func() (bool, []interface{}, error) {

		// get next input row
		c, err := from()
		if err != nil {
			return false, nil, err
		}
		if c == nil {
			return false, nil, nil // end of data
		}

		// convert and check input row
		cell, ok := c.(CellMicro)
		if !ok {
			return false, nil, errors.New("invalid type, expected: microdata cell (internal error)")
		}

		n := len(cell.Attr)
		if len(row) != 1+n {
			return false, nil, errors.New("invalid size of row buffer, expected: " + strconv.Itoa(1+n))
		}

		// set sql statement microdata values: entity key, attributes value
		row[0] = int64(cell.Key)

		for k, av := range cell.Attr {

			if v, err := fv[k](av.IsNull, av.Value); err == nil {
				row[k+1] = v
			} else {
				return false, nil, err
			}
		}

		// append row digest to microdata value digest
		err = digestFrom(cell)
		if err != nil {
			return false, nil, err
		}

		return true, row, nil // return current row to sql statement
	}

	return put
}

// digestMicrodataFrom start run microdata digest calculation and return closure to add microdata row to digest.
func digestMicrodataFrom(modelDef *ModelMeta, entityName string, entityGen *EntityGenMeta, rowCount *int, doubleFmt string) (hash.Hash, func(interface{}) error, error) {

	// start from entity name and generation digest
	hMd5 := md5.New()
	_, err := hMd5.Write([]byte("entity_name,gen_digest\n"))
	if err != nil {
		return nil, nil, err
	}

	_, err = hMd5.Write([]byte(entityName + "," + entityGen.GenDigest + "\n"))
	if err != nil {
		return nil, nil, err
	}

	// create microdata row digester append digest of microdata cells
	cvtMicro := &CellMicroConverter{CellEntityConverter: CellEntityConverter{
		ModelDef:  modelDef,
		Name:      entityName,
		EntityGen: entityGen,
		IsIdCsv:   true,
		DoubleFmt: doubleFmt,
	}}

	digestRow, err := digestMicrodataCellsFrom(hMd5, modelDef, rowCount, cvtMicro)
	if err != nil {
		return nil, nil, err
	}

	return hMd5, digestRow, nil
}
