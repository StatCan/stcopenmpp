/*
Properties:

rowFields[], colFields[], otherFields[]: array of dimensions, example of each element:
  {
    name: 'dim0',
    label: 'Salary',
    selection: [],
    enums: [
      { value: 100, name: 'low',    label: 'Low Value' },
      { value: 200, name: 'medium', label: 'Medium Value' },
      { value: 300, name: 'high',   label: 'High Value' }
    ]
  }

pvData: // array of table rows, for example:
    { DimIds: [100, 0], IsNull: false, Value: 895.5, ExprId: 0 }
  or:
    { DimIds: [10, 0], IsNull: false, Value: 0.1, SubId: 0 }
    ....
  dimension read() function expected to return enum value from row
  dimension selection[] is an array of selected enums for each dimension

pvControl: {
  rowColMode:   rows and columns mode:
                  3 = no spans and show dim names
                  2 = use spans, show dim names
                  1 = use spans, hide dim names
                  0 = no spans, hide dim names

  isShowNames:  if true then show item names (enums[k].name) instead of labels (enums[k].label) by default

  reader: {
    isScale: false by default, if true then reader can do value scaling: find value range

    rowReader (src) => { // row reader: methods to read next row, read() dimension items and readValue()
      readRow: () => {
        return next row or undefined if end of rorws
      },
      readDim[dimension name]: (r) => {
        return dimension item id by diemnsion name from row (r), for example: (r) => (r.DimIds.length > 0 ? r.DimIds[0] : void 0),
      },
      readValue: (r) => {
        read table cell value from row (r), for example: (r) => (!r.IsNull ? r.Value : void 0)
      }
    },

    scaleReader: {
      isRange: () => { retrun true if range is defined }
      rangeDef: () => { retrun range scale id, calculation and interval count }
      setRange: (eId, calc, n) => { calculate values range bounds for measure scale id }
      findRange: (val) => { find range index of the measuare value }
      getScale: () => { return scale for all measure expresions }
    }
  },

  processValue: functions are used to aggregate cell value(s), object with two methods:
  processValue: {
    init()      // call once for each table cell, it return initial cell state
    doNext()    // call for each input row after readValue()
    // example of processValue "sum":
    {
      init: () => ({ result: void 0 }), // returm initial state for each cell
      doNext: (val, state) => {
        let v = parseFloat(val)
        if (!isNaN(v)) state.result = (state.result || 0) + v
        return state.result
      }
    }
    default processValue{} return value as is, (no conversion or aggregation)
  }

  formatter: {
    options():  function to return formatter options
    format():   function (if defined) to convert cell value to string
    parse():    function (if defined) to convert cell string to value, ex: parseFloat(), used by editor only
    isValid():  function to validate cell value, used by editor only
    enumIdByLabel(): function to return enum id by enum label, used by editor only
  }

  dimItemKeys: {  // optional, undefined by default
                  // interface to find dimension item key (enum id) by row or column number
                  // defined for measure dimension and used to find value format by measure dimension enum id
  }

  cellClass: 'pv-cell-right'  // cell value style: by default right justified number
}

refreshViewTickle: watch true/false, on change pivot table view updated
  it is recommended to set
    refreshViewTickle = !refreshViewTickle
  after pvData[] or any selection[] changed

refreshDimsTickle: watch true/false, on change dimension properties updated
  it is recommended to set
    refreshDimsTickle = !refreshDimsTickle
  after dimension arrays initialized (after init of: rowFields[], colFields[], otherFields[])

refreshRangeTickle: watch true/false to update cell class on change of measure value scale range

if you are using pivot table editor then pvEdit is editor options and state shared with parent
pvEdit: {
  isEnabled: false,       // if true then edit value
  kind: Pcvt.EDIT_NUMBER, // numeric editor by default, other kinds: text, bool checkbox or enums dropdown
        // current editor state
  isEdit: false,    // if true then edit in progress
  isUpdated: false, // if true then cell value(s) updated
  cellKey: '',      // current eidtor focus cell
  cellValue: '',    // current eidtor input value
  updated: {},      // updated cells
  history: [],      // update history
  lastHistory: 0    // length of update history, changed by undo-redo
}
*/

import * as Pcvt from './pivot-cvt'

export default {
  /* eslint-disable no-multi-spaces */
  props: {
    rowFields: { type: Array, default: () => [] },
    colFields: { type: Array, default: () => [] },
    otherFields: { type: Array, default: () => [] },
    pvData: {
      type: Array, default: () => [] // input data as array of objects
    },
    pvControl: {
      type: Object,
      default: () => ({
        rowColMode: Pcvt.NO_SPANS_NO_DIMS_PVT,  // rows and columns mode: 3 = no spans and show dim names, 2 = use spans and show dim names, 1 = use spans and hide dim names, 0 = no spans and hide dim names
        isShowNames: false,                     // if true then show dimension names and item names instead of labels
        reader: void 0,                   // return row reader: if defined then methods to read next row, read() dimension items and readValue()
        processValue: Pcvt.asIsPval,      // default value processing: return as is
        formatter: Pcvt.formatDefault,    // disable format(), parse() and validation by default
        cellClass: 'pv-cell-right',       // default cell value style: right justified number
        dimItemKeys: void 0               // interface to find dimension item key (enum id) by row or column number
      })
    },
    refreshViewTickle: {
      type: Boolean, required: true // on refreshViewTickle change pivot table view updated
    },
    refreshDimsTickle: {
      type: Boolean, required: true // on refreshDimsTickle change items labels updated
    },
    refreshRangeTickle: {
      type: Boolean, default: false // on refreshDimsTickle change items labels updated
    },
    pvEdit: { // editor options and state shared with parent
      type: Object,
      default: Pcvt.emptyEdit
    }
  },

  data () {
    return {
      pvt: {
        rowCount: 0,      // table size: number of rows
        colCount: 0,      // table size: number of columns
        labels: {},       // row, column, other dimensions item labels, key: {dimensionName, itemCode}
        rows: Object.freeze([]),      // for each row:    array of item keys
        cols: Object.freeze([]),      // for each column: array of item keys
        rowKeys: Object.freeze([]),   // for each table row:    itemsToKey(rows[i])
        colKeys: Object.freeze([]),   // for each table column: itemsToKey(cols[j])
        cells: Object.freeze({}),     // body cells value, object key: cellKey
        cellKeys: Object.freeze([]),  // keys of table body values ordered by [row index, column index]
        rowSpans: Object.freeze({}),  // row span for each row label
        colSpans: Object.freeze({})   // column span for each column label
      },
      keyRenderCount: 0,  // table body cell key suffix to force update
      renderKeys: {},     // for each cellKey string of: 'cellKey-keyRenderCount'
      keyPos: [],         // position of each dimension item in cell key
      valueLen: 0,        // value input text size
      isRange: false,                 // if true then measure diemnsion has scale ranges defined
      rangeDef: Pcvt.emptyRangeDef(), // current range definition
      heatStyles: Pcvt.HeatMixStyle,  // heat map color styles
      itemKeyByRowCol: (nRow, nCol) => (void 0) // return dimension item key (enum id) by row or column number
    }
  },
  /* eslint-enable no-multi-spaces */

  watch: {
    refreshViewTickle () { this.setData(this.pvData) },
    refreshDimsTickle () { this.setDimItemLabels() },
    refreshRangeTickle () {
      this.isRange = !!this.pvControl.reader && this.pvControl.reader?.isScale && this.pvControl.reader?.scaleReader?.isRange()
      if (this.isRange) this.setRangeInfo()
    },

    // on edit started event: set focus on top left cell
    isPvEditNow () {
      if (this.pvEdit.isEdit) {
        this.$nextTick(() => {
          this.focusToRowCol(0, 0)
          // this.updateValueLenByRowCol(0, 0)
        })
      }
    }
  },
  computed: {
    // return true if pivot table in edit mode
    isPvEditNow () {
      return this.pvEdit.isEdit
    },
    // return true if pivot table in edit mode and cell input mode
    isEditInput () {
      return this.pvEdit.isEdit && typeof this.pvEdit.cellKey === typeof 'string' && this.pvEdit.cellKey !== ''
    }
  },

  emits: ['pv-edit', 'pv-key-pos'],

  methods: {
    // table body cell render keys to force update
    getRenderKey (key) {
      return this.renderKeys.hasOwnProperty(key) ? this.renderKeys[key] : void 0
    },
    makeRenderKey (key) { return [key, this.keyRenderCount].join('-') },
    changeRenderKey (key) {
      this.renderKeys[key] = [key, ++this.keyRenderCount].join('-')
    },
    // return formatted cell value
    getCellValueFmt (nRow, nCol) {
      const v = this.pvt.cells[this.pvt.cellKeys[nRow * this.pvt.colCount + nCol]]

      if (this.pvControl.formatter.options()?.isByKey) {
        return this.pvControl.formatter.format(v, this.itemKeyByRowCol(nRow, nCol))
      }
      return this.pvControl.formatter.format(v)
    },
    // set or clear range definition and heat map stlyes
    setRangeInfo () {
      this.rangeDef = Pcvt.emptyRangeDef()
      this.heatStyles = Pcvt.HeatMixStyle

      if (this.isRange) {
        this.rangeDef = this.pvControl.reader.scaleReader.rangeDef()
        if (this.rangeDef.color === Pcvt.HOT_RANGE) this.heatStyles = Pcvt.HeatHotStyle
        if (this.rangeDef.color === Pcvt.COLD_RANGE) this.heatStyles = Pcvt.HeatColdStyle
      }
    },
    // return additional cell class style
    getExtraCellClass (nRow, nCol) {
      if (!this.isRange) return ''

      const eId = this.itemKeyByRowCol(nRow, nCol)
      if (eId !== this.rangeDef.scaleId) return '' // this is not scale expression: use default style

      const v = this.pvt.cells[this.pvt.cellKeys[nRow * this.pvt.colCount + nCol]]
      const rIdx = this.pvControl.reader.scaleReader.findRange(v)

      return (rIdx >= 0 && rIdx < this.heatStyles.length) ? this.heatStyles[rIdx] : ''
    },

    // start of editor methods
    //
    // return updated cell value or default if value not updated
    getUpdatedToDisplay (nRow, nCol) {
      const v = this.getUpdatedFmt(nRow, nCol)
      return (v !== void 0 && v !== '') ? v : '\u00a0' // value or &nbsp;
    },
    getUpdatedFmt (nRow, nCol) {
      const v = this.getUpdatedSrc(this.pvt.cellKeys[nRow * this.pvt.colCount + nCol])

      if (this.pvControl.formatter.options()?.isByKey) {
        return this.pvControl.formatter.format(v, this.itemKeyByRowCol(nRow, nCol))
      }
      return this.pvControl.formatter.format(v)
    },
    getUpdatedSrc (key) {
      return this.pvEdit.isUpdated && this.pvEdit.updated.hasOwnProperty(key) ? this.pvEdit.updated[key] : this.pvt.cells[key]
    },

    // start cell edit: enter into input control
    onKeyEnter (e) {
      const rc = this.rowColAttrs(e)
      if (rc) this.cellInputStart(rc.cRow, rc.cCol) // if this is table body cell then start input
    },
    onDblClick (e) {
      const rc = this.rowColAttrs(e)
      if (rc) this.cellInputStart(rc.cRow, rc.cCol) // if this is table body cell then start input
    },
    cellInputStart (nRow, nCol) {
      if (typeof nRow !== typeof 1 || typeof nCol !== typeof 1) {
        console.warn('Fail to start cell editing: invalid or undefined row or column')
        return
      }
      const key = this.pvt.cellKeys[nRow * this.pvt.colCount + nCol]
      this.updateValueLenByKey(key)
      this.pvEdit.cellKey = key
      this.pvEdit.cellValue = this.getUpdatedSrc(key)
      this.focusNextTick('input-cell')
    },

    // cancel input edit by escape
    onCellInputEscape () {
      const cKey = this.pvEdit.cellKey
      this.$nextTick(() => {
        this.focusNextTick(cKey)
      })
      this.pvEdit.cellKey = ''
      this.pvEdit.cellValue = ''
    },

    // confirm input edit by Enter: finish cell edit, move to next cell and start next cell edit
    onCellInputEnter (nRow, nCol) {
      const cKey = this.pvEdit.cellKey
      const isOk = this.cellInputConfirm(this.pvEdit.cellValue, cKey)
      this.pvEdit.cellKey = ''
      this.pvEdit.cellValue = ''
      this.$emit('pv-edit')
      // if invalid input or end of the table then stop edit and keep focus at the same cell
      if (!isOk || ((nRow || 0) >= this.pvt.rowCount - 1 && (nCol || 0) >= this.pvt.colCount - 1)) {
        this.$nextTick(() => { this.focusNextTick(cKey) })
        return
      }
      // else: move to next cell right and down
      let nr = nRow || 0
      let nc = nCol || 0
      if (nc < this.pvt.colCount - 1) {
        nc++
      } else {
        nr++
        nc = 0
      }
      // move focus to next cell (next right, next down) and start edit
      this.$nextTick(() => {
        this.focusToRowCol(nr, nc)
        this.$nextTick(() => { this.cellInputStart(nr, nc) })
      })
    },
    // confirm input edit: finish cell edit and keep focus at the same cell
    onCellInputConfirm () {
      const cKey = this.pvEdit.cellKey
      this.cellInputConfirm(this.pvEdit.cellValue, cKey)
      this.$nextTick(() => {
        this.focusNextTick(cKey)
      })
      this.pvEdit.cellKey = ''
      this.pvEdit.cellValue = ''
      this.$emit('pv-edit')
    },
    // confirm input edit by lost focus
    onCellInputBlur () {
      const cKey = this.pvEdit.cellKey
      if ((cKey || '') === '') return // exit on blur before focus: Chrome vs Firefox
      this.cellInputConfirm(this.pvEdit.cellValue, cKey)
      this.pvEdit.cellKey = ''
      this.pvEdit.cellValue = ''
      this.$emit('pv-edit')
    },

    // confirm input edit: validate and save changes in edit history
    cellInputConfirm (val, iKey) {
      // validate input
      if (!this.pvControl.formatter.isValid(val)) {
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (or empty) value entered') })
        return false
      }

      // compare input value with previous
      const iNow = this.pvControl.formatter.parse(val)
      const iPrev = this.pvEdit.updated.hasOwnProperty(iKey) ? this.pvEdit.updated[iKey] : this.pvt.cells[iKey]

      if (iNow === iPrev || (iNow === '' && iPrev === void 0)) return true // exit if value not changed

      // store updated value and append it change history
      this.pvEdit.updated[iKey] = iNow
      this.pvEdit.isUpdated = true

      if (this.pvEdit.lastHistory < this.pvEdit.history.length) {
        this.pvEdit.history.splice(this.pvEdit.lastHistory)
      }
      this.pvEdit.history.push({
        key: iKey,
        now: iNow,
        prev: iPrev
      })
      this.pvEdit.lastHistory = this.pvEdit.history.length

      this.changeRenderKey(iKey)
      this.updateValueLenByValue(iNow)
      return true
    },

    // undo last edit changes
    doUndo () {
      if (!this.pvEdit.isEdit || this.pvEdit.lastHistory <= 0) return // exit: entire history already undone

      const n = --this.pvEdit.lastHistory
      const cKey = this.pvEdit.history[n].key

      let isPrev = false
      for (let k = 0; !isPrev && k < n; k++) {
        isPrev = this.pvEdit.history[k].key === cKey
      }
      if (isPrev) {
        this.pvEdit.updated[cKey] = this.pvEdit.history[n].prev
      } else {
        delete this.pvEdit.updated[cKey]
        this.pvEdit.isUpdated = !!this.pvEdit.updated && this.pvEdit.lastHistory > 0
      }

      // update display value
      this.changeRenderKey(cKey)
      this.focusNextTick(cKey)
      this.$emit('pv-edit')
    },
    // redo most recent undo
    doRedo () {
      if (!this.pvEdit.isEdit || this.pvEdit.lastHistory >= this.pvEdit.history.length) return // exit: already at the end of history

      const n = this.pvEdit.lastHistory++
      const cKey = this.pvEdit.history[n].key
      this.pvEdit.updated[cKey] = this.pvEdit.history[n].now
      this.pvEdit.isUpdated = true

      // update display value
      this.changeRenderKey(cKey)
      this.focusNextTick(cKey)
      this.$emit('pv-edit')
    },

    // arrows navigation
    onLeftArrow (e) {
      const rc = this.rowColAttrs(e)
      if (rc && rc.cCol > 0) this.focusToRowCol(rc.cRow, rc.cCol - 1) // move focus left if this is table body cell
    },
    onRightArrow (e) {
      const rc = this.rowColAttrs(e)
      if (rc && rc.cCol < this.pvt.colCount - 1) this.focusToRowCol(rc.cRow, rc.cCol + 1) // move focus right if this is table body cell
    },
    onDownArrow (e) {
      const rc = this.rowColAttrs(e)
      if (rc && rc.cRow < this.pvt.rowCount - 1) this.focusToRowCol(rc.cRow + 1, rc.cCol) // move focus down if this is table body cell
    },
    onUpArrow (e) {
      const rc = this.rowColAttrs(e)
      if (rc && rc.cRow > 0) this.focusToRowCol(rc.cRow - 1, rc.cCol) // move focus up if this is table body cell
    },

    // set cell focus to (rowNumber,columnNumber) coordinates
    focusToRowCol (nRow, nCol) {
      const cKey = this.cellKeyByRowCol(nRow, nCol)
      if (cKey && this.$refs[cKey] && (this.$refs[cKey].length || 0) === 1) this.$refs[cKey][0].focus()
    },
    // set cell focus on next tick
    focusNextTick (key) {
      this.$nextTick(() => {
        if (this.$refs[key] && (this.$refs[key].length || 0) === 1) this.$refs[key][0].focus()
      })
    },
    // get cell key by (row, column)
    cellKeyByRowCol (nRow, nCol) {
      if (typeof nRow !== typeof 1 || typeof nCol !== typeof 1) return void 0
      if (nRow < 0 || nRow > this.pvt.rowCount - 1 || nCol < 0 || nCol > this.pvt.colCount - 1) return void 0 // outside of table body

      return this.pvt.cellKeys[nRow * this.pvt.colCount + nCol]
    },

    // get body cell {row,column} from event target attributes
    rowColAttrs (e) {
      if (!e || !e.target) {
        console.warn('Failed to get row and column: event target unknown')
        return void 0
      }
      const nRow = parseInt((e.target.getAttribute('data-om-nrow') || ''), 10)
      const nCol = parseInt((e.target.getAttribute('data-om-ncol') || ''), 10)

      return (!isNaN(nRow) && !isNaN(nCol)) ? { cRow: nRow, cCol: nCol } : void 0 // return undefined if taregt has no row or column attribute
    },

    // on recalculate size of input text value based on client width
    updateValueLenByRowCol (nRow, nCol) {
      this.updateValueLenByKey(this.cellKeyByRowCol(nRow, nCol))
    },
    updateValueLenByKey (key) {
      if (key && this.$refs[key] && (this.$refs[key].length || 0) === 1) {
        const n = this.$refs[key][0].clientWidth || 0
        if (n) {
          const nc = Math.floor((n - 9) / 9) - 1
          if (this.valueLen < nc) this.valueLen = nc
        }
      }
    },
    // on recalculate size of input text value
    updateValueLenByValue (val) {
      if (val !== void 0 && val !== '') {
        const n = (val.toString() || '').length
        if (this.valueLen < n) this.valueLen = n
      }
    },

    // keyup event handler in editor mode: capture hot keys if it is not input text
    onKeyUpEdit (e) {
      if (e.defaultPrevented) return

      // capture hot keys: ctrl-z and ctrl-y
      if (!e.ctrlKey || e.altKey || e.shiftKey || e.metaKey) return
      switch (e.key) {
        case 'z':
        case 'Z':
          this.doUndo()
          break
        case 'y':
        case 'Y':
          this.doRedo()
          break
        default:
          return // not a hot key: pass to default
      }
      e.preventDefault() // done with hot keys
    },

    // keydown event handler in editor mode: capture hot keys if it is not input text
    onKeyDownEdit (e) {
      if (e.defaultPrevented) return

      // capture hot keys: enter and arrows
      if (e.ctrlKey || e.altKey || e.shiftKey || e.metaKey) return
      switch (e.key) {
        case 'Enter':
          this.onKeyEnter(e)
          break
        case 'Left':
        case 'ArrowLeft':
          this.onLeftArrow(e)
          break
        case 'Right':
        case 'ArrowRight':
          this.onRightArrow(e)
          break
        case 'Up':
        case 'ArrowUp':
          this.onUpArrow(e)
          break
        case 'Down':
        case 'ArrowDown':
          this.onDownArrow(e)
          break
        default:
          return // not a hot key: pass to default
      }
      e.preventDefault() // done with hot keys
    },

    // paste tab separated values from clipboard, event.preventDefault done by event modifier
    onPasteTsv (e) {
      const rc = this.rowColAttrs(e)
      if (!rc) {
        return false // current cell is unknown: paste must be done into table cell
      }

      // get pasted clipboard values
      let pasted = ''
      if (e.clipboardData && e.clipboardData.getData) {
        pasted = e.clipboardData.getData('text') || ''
      } else {
        if (window.clipboardData && window.clipboardData.getData) pasted = window.clipboardData.getData('Text') || ''
      }

      // parse tab separated values
      const pv = Pcvt.parseTsv(pasted, this.pvt.rowCount - rc.cRow, this.pvt.colCount - rc.cCol)

      if (!pv || pv.rowSize <= 0 || (pv.colSize <= 0 && rc.cRow + pv.rowSize <= this.pvt.rowCount)) {
        this.$q.notify({ type: 'negative', message: this.$t('Empty (or invalid) paste: tab separated values expected') })
        return false
      }
      if (rc.cRow + pv.rowSize > this.pvt.rowCount) {
        this.$q.notify({ type: 'negative', message: this.$t('Too many rows pasted: ') + pv.rowSize.toString() })
        return false
      }
      if (rc.cCol + pv.colSize > this.pvt.colCount) {
        this.$q.notify({ type: 'negative', message: this.$t('Too many columns pasted: ') + pv.colSize.toString() })
        return false
      }

      // for each value do input into the cell
      // if this is enum based parameter and cell values are labels then convert enum labels to enum id's
      const isToEnumId = this.pvEdit.kind === Pcvt.EDIT_ENUM && !this.pvControl.formatter.options().isRawValue

      for (let k = 0; k < pv.arr.length; k++) {
        for (let j = 0; j < pv.arr[k].length; j++) {
          const cKey = this.pvt.cellKeys[(rc.cRow + k) * this.pvt.colCount + (rc.cCol + j)]

          let val = pv.arr[k][j]
          if (isToEnumId) {
            val = this.pvControl.formatter.enumIdByLabel(val)
            if (val === void 0 || val === '') {
              this.$q.notify({
                type: 'negative',
                message: this.$t('Invalid enum label at row: ') + k.toString() + ' ' + this.$t('column: ') + j.toString()
              })
              return false
            }
          }

          // do input into the cell
          if (!this.cellInputConfirm(val, cKey)) return false // input validation failed
          this.$emit('pv-edit')
          this.focusNextTick(cKey)
        }
      }
      return true // success
    },
    //
    // end of editor methods

    // copy tab separated values to clipboard
    onCopyTsv () {
      let tsv = ''

      // prefix for each column header: empty '' value * by count of row dimensions
      let cp = ''
      for (let nFld = 0; nFld < this.rowFields.length; nFld++) {
        cp += '\t'
      }

      // table header: items of column dimensions
      for (let nFld = 0; nFld < this.colFields.length; nFld++) {
        tsv += cp
        for (let nCol = 0; nCol < this.pvt.colCount; nCol++) {
          tsv += (this.pvt.labels[this.colFields[nFld].name][this.pvt.cols[nCol][nFld]] || '') + (nCol < this.pvt.colCount - 1 ? '\t' : '\r\n')
        }
      }

      // table body: headers of each row and body cell values
      for (let nRow = 0; nRow < this.pvt.rowCount; nRow++) {
        // items of row dimemensions
        for (let nFld = 0; nFld < this.rowFields.length; nFld++) {
          tsv += (this.pvt.labels[this.rowFields[nFld].name][this.pvt.rows[nRow][nFld]] || '') + '\t'
        }

        // body cell values
        if (!this.pvEdit.isEdit) {
          for (let nCol = 0; nCol < this.pvt.colCount; nCol++) {
            const cKey = this.pvt.cellKeys[nRow * this.pvt.colCount + nCol]
            tsv += this.pvControl.formatter.format(this.pvt.cells[cKey]) + (nCol < this.pvt.colCount - 1 ? '\t' : '\r\n')
          }
        } else {
          for (let nCol = 0; nCol < this.pvt.colCount; nCol++) {
            tsv += this.getUpdatedFmt(nRow, nCol) + (nCol < this.pvt.colCount - 1 ? '\t' : '\r\n')
          }
        }
      }

      // copy to clipboard
      const isOk = Pcvt.toClipboard(tsv)

      if (isOk) {
        this.$q.notify({
          type: 'info',
          message: this.$t('Copy tab separated values to clipboard: ') + tsv.length + ' ' + this.$t('characters')
        })
      } else {
        this.$q.notify({
          type: 'negative',
          message: this.$t('Failed to copy tab separated values to clipboard')
        })
      }
      return isOk
    },

    // set item labels for for all dimensions: rows, columns, other
    setDimItemLabels () {
      const makeLabels = (dims) => {
        for (const f of dims) {
          const ls = {}

          // make enum labels
          for (const e of f.enums) {
            ls[e.value] = !this.pvControl.isShowNames ? e.label : e.name
          }
          this.pvt.labels[f.name] = ls
        }
      }

      this.pvt.labels = {}
      makeLabels(this.rowFields)
      makeLabels(this.colFields)
      makeLabels(this.otherFields)
    },

    // get enum label by dimension name and enum item value
    getDimItemLabel (dimName, enumValue) {
      return this.pvt.labels.hasOwnProperty(dimName) ? (this.pvt.labels[dimName][enumValue] || '') : ''
    },

    // update pivot table from input data page
    setData (data) {
      // clean existing view
      this.pvt.rowCount = 0
      this.pvt.colCount = 0
      this.pvt.rows = Object.freeze([])
      this.pvt.cols = Object.freeze([])
      this.pvt.rowKeys = Object.freeze([])
      this.pvt.colKeys = Object.freeze([])
      this.pvt.cells = Object.freeze({})
      this.pvt.cellKeys = Object.freeze([])
      this.pvt.rowSpans = Object.freeze({})
      this.pvt.colSpans = Object.freeze({})
      this.renderKeys = {}
      this.keyPos = []
      this.itemKeyByRowCol = (nRow, nCol) => (void 0)
      this.isRange = false
      this.rangeDef = Pcvt.emptyRangeDef()
      this.heatStyles = Pcvt.HeatMixStyle

      // if response is empty or invalid: clean table and exit
      if (!data || (data?.length || 0) <= 0) return

      if (!this.pvControl.reader) {
        console.warn('Reader not ready, data length:', data?.length)
        return
      }

      const rdr = this.pvControl.reader.rowReader(data)
      if (!rdr) {
        console.warn('Unexpected empty row reader, data length:', data?.length)
        return // data is empty: unexpected
      }
      const emptyRead = (r) => (void 0)

      // make dimensions field selectors to read, filter and sort dimension fields
      const dimProc = []
      const rCmp = []
      const cCmp = []
      for (const f of this.rowFields) {
        const p = Pcvt.dimField(f, ((rdr?.readDim?.[f.name]) || emptyRead), true, false)
        dimProc.push(p)
        rCmp.push(p.compareEnumIdByIndex)
      }
      for (const f of this.colFields) {
        const p = Pcvt.dimField(f, ((rdr?.readDim?.[f.name]) || emptyRead), false, true)
        dimProc.push(p)
        cCmp.push(p.compareEnumIdByIndex)
      }
      for (const f of this.otherFields) {
        dimProc.push(Pcvt.dimField(f, ((rdr?.readDim?.[f.name]) || emptyRead), false, false))
      }

      const isScalar = dimProc.length === 0 // scalar parameter with only one sub-value

      let cellKeyLen = 0
      for (const p of dimProc) {
        if (!p.isCellKey) continue
        cellKeyLen++
        for (const rp of dimProc) {
          if (!rp.isCellKey) continue
          if (p.name > rp.name) p.keyPos++
        }
      }

      // process all input records: check if it match selection filters and aggregate values
      const rowKeyLen = this.rowFields.length
      const colKeyLen = this.colFields.length
      const vrows = []
      const vcols = []
      const rKeys = {}
      const cKeys = {}
      const vcells = {}
      const vstate = {}

      while (true) {
        const rRow = rdr.readRow()
        if (!rRow) {
          break // end of data
        }

        // check if record match selection filters
        let isSel = true
        const r = Array(rowKeyLen)
        const c = Array(colKeyLen)
        const b = Array(cellKeyLen)
        let i = 0
        let j = 0
        for (const p of dimProc) {
          const v = p.read(rRow)
          isSel = p.filter(v)
          if (!isSel) break
          if (p.isRow) r[i++] = v
          if (p.isCol) c[j++] = v
          if (p.isCellKey) b[p.keyPos] = v
        }
        if (!isSel) continue // skip row: dimension item is not in filter values

        // build list of rows and columns keys
        const rk = Pcvt.itemsToKey(r)
        if (!rKeys[rk]) {
          rKeys[rk] = true
          vrows.push(r)
        }
        const ck = Pcvt.itemsToKey(c)
        if (!cKeys[ck]) {
          cKeys[ck] = true
          vcols.push(c)
        }

        // extract value(s) from record and aggregate
        const v = rdr.readValue(rRow)

        const bkey = isScalar ? Pcvt.PV_KEY_SCALAR : Pcvt.itemsToKey(b)
        if (!vstate.hasOwnProperty(bkey)) {
          vstate[bkey] = this.pvControl.processValue.init()
          vcells[bkey] = void 0
        }
        if (v !== void 0 || v !== null) {
          vcells[bkey] = this.pvControl.processValue.doNext(v, vstate[bkey])
        }
      }

      // if value scale enabled and measure value range defined
      this.isRange = this.pvControl.reader?.isScale && this.pvControl.reader?.scaleReader?.isRange()
      if (this.isRange) this.setRangeInfo()

      // sort row keys and column keys in the order of dimension items
      vrows.sort(Pcvt.compareKeys(rowKeyLen, rCmp))
      vcols.sort(Pcvt.compareKeys(colKeyLen, cCmp))

      // calculate span for rows and columns
      const rsp = Pcvt.itemSpans(rowKeyLen, vrows)
      const csp = Pcvt.itemSpans(colKeyLen, vcols)

      // sorted keys: row keys, column keys, body value keys
      const rowCount = vrows.length
      const colCount = vcols.length

      const vrk = Array(rowCount)
      let n = 0
      for (const r of vrows) {
        vrk[n++] = Pcvt.itemsToKey(r)
      }

      const vck = Array(colCount)
      n = 0
      for (const c of vcols) {
        vck[n++] = Pcvt.itemsToKey(c)
      }

      // body cell key is row dimension(s) items, column dimension(s) items
      // and filter items: items other dimension(s) where only one item selected
      const b = Array(cellKeyLen)
      const nkp = Array(cellKeyLen)
      const kp = {}
      for (let k = 0; k < dimProc.length; k++) {
        if (!dimProc[k].isCellKey) continue
        nkp[dimProc[k].keyPos] = {
          name: dimProc[k].name,
          pos: dimProc[k].keyPos
        }
        kp[dimProc[k].name] = dimProc[k].keyPos
        if (dimProc[k].isSingleKey) {
          b[dimProc[k].keyPos] = dimProc[k].singleKey // add selected item to cell key
        }
      }

      const vkeys = Array(rowCount * colCount)
      n = 0
      for (const r of vrows) {
        for (let k = 0; k < rowKeyLen; k++) { // add row items to key
          b[kp[this.rowFields[k].name]] = r[k]
        }
        for (const c of vcols) {
          for (let k = 0; k < colKeyLen; k++) { // add column items to key
            b[kp[this.colFields[k].name]] = c[k]
          }
          vkeys[n++] = Pcvt.itemsToKey(b)
        }
      }
      // scalar parameter with only one sub-value
      if (isScalar && vkeys.length === 1) {
        if (vkeys[0] === '') vkeys[0] = Pcvt.PV_KEY_SCALAR
      }

      this.keyRenderCount++
      const vupd = {}
      for (const bkey of vkeys) {
        vupd[bkey] = this.makeRenderKey(bkey)
      }

      // max length of cell value as string
      this.valueLen = 0
      let ml = 0
      for (const bkey in vcells) {
        if (vcells[bkey] !== void 0) {
          const n = (vcells[bkey].toString() || '').length
          if (ml < n) ml = n
        }
      }
      if (this.valueLen < ml) this.valueLen = ml

      // done
      this.pvt.rowCount = rowCount
      this.pvt.colCount = colCount
      this.pvt.rows = Object.freeze(vrows)
      this.pvt.cols = Object.freeze(vcols)
      this.pvt.rowKeys = Object.freeze(vrk)
      this.pvt.colKeys = Object.freeze(vck)
      this.pvt.cells = Object.freeze(vcells)
      this.pvt.cellKeys = Object.freeze(vkeys)
      this.pvt.rowSpans = Object.freeze(rsp)
      this.pvt.colSpans = Object.freeze(csp)
      this.renderKeys = vupd
      this.keyPos = nkp
      if (this.pvControl.dimItemKeys) {
        this.itemKeyByRowCol = this.pvControl.dimItemKeys.itemKeys(this.rowFields, this.colFields, this.otherFields, this.pvt.rows, this.pvt.cols).byRowCol
      }

      this.$emit('pv-key-pos', this.keyPos)
    }
  },

  mounted () {
    this.keyRenderCount = 0
    this.setDimItemLabels()
    this.setData(this.pvData)
  }
}
