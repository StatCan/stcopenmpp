import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'
import * as Mdf from 'src/model-common'
import * as Idb from 'src/idb/idb'
import RunBar from 'components/RunBar.vue'
import RefreshRun from 'components/RefreshRun.vue'
import RunInfoDialog from 'components/RunInfoDialog.vue'
import EntityInfoDialog from 'components/EntityInfoDialog.vue'
import EntityCalcDialog from 'components/EntityCalcDialog.vue'
import ValueFilterDialog from 'components/ValueFilterDialog.vue'
import { VueDraggableNext } from 'vue-draggable-next'
import * as Pcvt from 'components/pivot-cvt'
import * as Puih from './pivot-ui-helper'
import PvTable from 'components/PvTable.vue'
import { openURL } from 'quasar'

/* eslint-disable no-multi-spaces */
const ATTR_DIM_NAME = 'ATTRIBUTES_DIM'          // attributes measure dimension name
const KEY_DIM_NAME = 'ENTITY_KEY_DIM'           // entity key dimension name
const CALC_DIM_NAME = 'CALCULATED_DIM'          // calculated attributes measure dimension name
const RUN_DIM_NAME = 'RUN_DIM'                  // model run compare dimension name
const SMALL_PAGE_SIZE = 10                      // small page size: do not show page controls
const LAST_PAGE_OFFSET = 2 * 1024 * 1024 * 1024 // large page offset to get the last page
/* eslint-enable no-multi-spaces */

export default {
  name: 'EntityPage',
  components: { draggable: VueDraggableNext, PvTable, RunBar, RefreshRun, RunInfoDialog, EntityInfoDialog, EntityCalcDialog, ValueFilterDialog },

  props: {
    digest: { type: String, default: '' },
    runDigest: { type: String, default: '' },
    entityName: { type: String, default: '' },
    refreshTickle: { type: Boolean, default: false }
  },

  /* eslint-disable no-multi-spaces */
  data () {
    return {
      loadDone: false,
      loadWait: false,
      isNullable: true,       // entity always nullabale, value can be NULL
      isScalar: false,        // microdata never scalar, it always has at least one attribute
      rank: 0,                // entity key dimension + number of enum-based attributes
      attrCount: 0,           // number of attributes of built-in types: int, float, bool, string
      entityText: Mdf.emptyEntityText(),
      runEntity: Mdf.emptyRunEntity(),
      runText: Mdf.emptyRunText(),
      dimProp: [],
      attrEnums: [],          // meausre dimension enums
      calcEnums: [],          // calculation dimension enums for aggregated measure calculation
      groupBy: [],            // names of group by dimensions
      groupDimCalc: [],       // selected names of group by dimensions
      attrCalc: [],           // selected names of calculated measure attributes
      aggrCalc: '',           // name of aggregation function, ex.: AVG
      cmpCalc: '',            // name of comparison function, ex.: DIFF
      valueFilter: [],        // measure value filters
      valueFilterMeasure: [], // list of value filter measures: name and label of attribute enums or calculted enums
      colFields: [],
      rowFields: [],
      otherFields: [],
      filterState: {},
      inpData: Object.freeze([]),
      ctrl: {
        isRowColControls: true,
        isRowColModeToggle: true,
        isPvTickle: false,      // used to update view of pivot table (on data selection change)
        isPvDimsTickle: false,  // used to update dimensions in pivot table (on label change)
        kind: Puih.ekind.MICRO,  // entity view content: microdata, aggregated (calculated), run compare
        //
        isRawUseView: false, // vue3 copy of isRawUse
        isRawView: false,    // vue3 copy of isRawValue
        isFloatView: false,  // vue3 copy of isFloat
        isMoreView: false,   // vue3 copy of isDoMore
        isLessView: false    // vue3 copy of isDoLess
      },
      pvc: {
        rowColMode: Pcvt.NO_SPANS_AND_DIMS_PVT, // rows and columns mode: 3 = no spans and show dim names
        isShowNames: false,                     // if true then show dimension names and item names instead of labels
        reader: void 0,                         // return row reader: if defined then methods to read next row, read() dimension items and readValue()
        processValue: Pcvt.asIsPval,            // default value processing: return as is
        formatter: Pcvt.formatDefault,          // disable format(), parse() and validation by default
        cellClass: 'pv-cell-right',             // default cell value style: right justified number
        dimItemKeys: void 0                     // interface to find dimension item key (enum id) by row or column number
      },
      attrFormatter: {},          // microdata attributes formatter: by key formatters for each attribute
      readerMicro: void 0,        // microdata row reader
      pvKeyPos: [],               // position of each dimension item in cell key
      locale: '',                 // current locale to format values
      isDragging: false,          // if true then user is dragging dimension select control
      selectDimName: '',      // selected dimension name
      isOtherDropDisabled: false, // if true then others drop area disabled
      isPages: false,
      pageStart: 0,
      pageSize: 0,
      isLastPage: false,
      isHidePageControls: false,
      loadRunWait: false,
      refreshRunTickle: false,
      runInfoTickle: false,
      entityInfoTickle: false,
      valueFilterTickle: false,
      calcEditTickle: false,
      calcUpdateTickle: false,
      aggrCalcList: [{  // aggeragation calculations: aggregate measure  attributes over dimensions (aggregate numric attributes over enum-based or boolean attributes)
        code: 'AVG',
        label: 'Average'
      }, {
        code: 'COUNT',
        label: 'Count'
      }, {
        code: 'SUM',
        label: 'Sum'
      }, {
        code: 'MAX',
        label: 'Maximum'
      }, {
        code: 'MIN',
        label: 'Minimum'
      }, {
        code: 'VAR',
        label: 'Variance'
      }, {
        code: 'SD',
        label: 'Standard deviation'
      }, {
        code: 'SE',
        label: 'Standard error'
      }, {
        code: 'CV',
        label: 'Coefficient of variation'
      }],
      compareCalcList: [{  // comparison calculations: additional measures as aggregation over dimensions
        code: 'DIFF',
        label: 'Variant - Base'
      }, {
        code: 'RATIO',
        label: 'Variant / Base'
      }, {
        code: 'PERCENT',
        label: '100 * (Variant - Base) / Base'
      }],
      isModelTab: this.$route.path.startsWith('/model/') // if true then model page tab else entity top page
    }
  },
  /* eslint-enable no-multi-spaces */

  computed: {
    routeKey () { return Mdf.microdataPath(this.digest, this.runDigest, this.entityName) },
    entityDescr () { return this?.entityText?.EntityDescr || '' },
    isRunCompare () {
      const mv = this?.modelViewSelected(this.digest)
      return !!mv && Array.isArray(mv?.digestCompareList) && mv.digestCompareList.length > 0
    },

    ...mapState(useModelStore, [
      'theModel',
      'theModelUpdated'
    ]),
    ...mapState(useServerStateStore, {
      omsUrl: 'omsUrl'
    }),
    ...mapState(useUiStateStore, [
      'uiLang',
      'idCsvDownload'
    ])
  },

  watch: {
    routeKey () { this.doRefresh() },
    refreshTickle () { this.doRefresh() },
    theModelUpdated () { this.doRefresh() }
  },

  emits: ['user-view-updated', 'tab-mounted'],

  methods: {
    ...mapActions(useModelStore, [
      'runTextByDigest'
    ]),
    ...mapActions(useUiStateStore, [
      'modelViewSelected',
      'microdataView',
      //
      'dispatchMicrodataView',
      'dispatchMicrodataViewDelete']
    ),

    async doRefresh () {
      this.initView()
      await this.setPageView()
      this.doRefreshDataPage()
    },

    // initialize current page view on mounted or tab switch
    initView () {
      // check if model run exists and run entity microdata is included in model run results
      if (!this.initRunEntity()) return // exit on error

      this.isPages = this.runEntity.RowCount > SMALL_PAGE_SIZE // disable pages for small microdata
      this.pageStart = 0
      this.pageSize = this.isPages ? SMALL_PAGE_SIZE : 0

      this.isNullable = true // entity always nullabale, value can be NULL
      this.isScalar = false // microdata never scalar, it always has at least one attribute

      // adjust controls
      this.pvc.rowColMode = !this.isScalar ? Pcvt.NO_SPANS_AND_DIMS_PVT : Pcvt.NO_SPANS_NO_DIMS_PVT
      this.ctrl.isRowColModeToggle = !this.isScalar
      this.ctrl.isRowColControls = !this.isScalar
      this.pvKeyPos = []
      this.aggrCalc = ''
      this.cmpCalc = ''

      // default pivot table options
      this.locale = this.uiLang || this.$q.lang.getLocale() || ''
      if (this.locale) {
        try {
          const cla = Intl.getCanonicalLocales(this.locale)
          this.locale = cla?.[0] || ''
        } catch (e) {
          this.locale = ''
          console.warn('Error: undefined canonical locale:', e)
        }
      }
      this.pvc.formatter = Pcvt.formatDefault({ isNullable: this.isNullable, locale: this.locale })
      this.attrFormatter = {}
      this.pvc.cellClass = 'pv-cell-right' // numeric cell value style by default

      this.pvc.processValue = Pcvt.asIsPval // no value conversion required, only formatting

      // initialize dimensions, set microdata view formatter and create microdata row reader
      this.setupDims()
      this.makeMicroReader()

      this.ctrl.kind = Puih.ekind.MICRO // default view
      this.pvc.reader = this.readerMicro
    },
    // setup all dimensions and microdata view formatter
    setupDims () {
      // make dimensions:
      //  [0]           entity key dimension
      //  [1, rank - 1] enum-based dimensions
      //  [rank]        measure attributes dimension: list of non-enum based attributes, e.g. int, double,... attributes
      //  [rank + 1]: calculated attributes dimension: calculated attributes as enums
      //  [rank + 2]:   run compare dimension: run names and run descriptions as labels
      this.attrCount = 0
      this.rank = 0
      this.dimProp = []

      // entity key dimension at [0] position: initially empty
      const fk = {
        name: KEY_DIM_NAME,
        label: (Mdf.descrOfDescrNote(this.entityText) || this.$t('Entity')) + ' ' + this.$t('keys'),
        enums: [],
        isBool: false,
        selection: [],
        singleSelection: {},
        filter: (val, update, abort) => {}
      }
      this.dimProp.push(fk)

      // measure attributes dimensions
      // [rank]:     "normal" attributes dimension: attributes as enums
      // [rank + 1]: calculated attributes dimension: calculated attributes as enums
      const fa = {
        name: ATTR_DIM_NAME,
        label: Mdf.descrOfDescrNote(this.entityText) || this.$t('Attribute'),
        enums: [],
        isBool: false,
        options: [],
        selection: [],
        singleSelection: {},
        filter: (val, update, abort) => {}
      }
      const fc = {
        name: CALC_DIM_NAME,
        label: Mdf.descrOfDescrNote(this.entityText) || this.$t('Attribute'),
        enums: [],
        isBool: false,
        options: [],
        selection: [],
        singleSelection: {},
        filter: (val, update, abort) => {}
      }
      this.attrEnums = []
      this.calcEnums = []
      const aFmt = {} // formatters for built-in attributes
      let nPos = 0 // attribute position in microdata row

      for (const ea of this.entityText.EntityAttrTxt) {
        if (!Mdf.isNotEmptyEntityAttr(ea)) continue
        if (this.runEntity.Attr.findIndex(name => ea.Attr.Name === name) < 0) continue // skip: this attribute is not included in run microdata

        // find attribute type text
        const tTxt = Mdf.typeTextById(this.theModel, ea.Attr.TypeId)

        if (!Mdf.isBuiltIn(tTxt.Type) || Mdf.isBool(tTxt.Type)) { // enum based attribute or bool: use it as dimension
          const f = {
            name: ea.Attr.Name || '',
            label: Mdf.descrOfDescrNote(ea) || ea.Attr.Name || '',
            enums: [],
            isBool: Mdf.isBool(tTxt.Type),
            options: [],
            selection: [],
            singleSelection: {},
            filter: (val, update, abort) => {},
            attrPos: nPos++
          }

          const eLst = Array(Mdf.typeEnumSize(tTxt))
          for (let j = 0; j < eLst.length; j++) {
            eLst[j] = Mdf.enumItemByIdx(tTxt, j)
          }
          f.enums = eLst
          f.options = f.enums
          f.filter = Puih.makeFilter(f)

          this.dimProp.push(f)
        } else { // built-in attribute type: add to attributes measure dimension
          const aId = ea.Attr.AttrId
          const a = {
            value: aId,
            name: ea.Attr.Name || aId.toString(),
            label: Mdf.descrOfDescrNote(ea) || ea.Attr.Name || '' || aId.toString(),
            attrPos: nPos++,
            isFloat: false,
            isInt: false
          }
          this.attrEnums.push(a)

          // setup process value and format value handlers:
          //  if parameter type is one of built-in then process and format value as float, int, boolen or string
          if (Mdf.isFloat(tTxt.Type)) {
            // this.pvc.processValue = Pcvt.asFloatPval
            a.isFloat = true
            aFmt[aId] = Pcvt.formatFloat({
              isNullable: this.isNullable,
              locale: this.locale,
              nDecimal: Pcvt.maxDecimalDefault,
              maxDecimal: Pcvt.maxDecimalDefault,
              isAllDecimal: true // show all deciamls for the float value
            })
            this.pvc.cellClass = 'pv-cell-right' // numeric cell value style by default
          }
          if (Mdf.isInt(tTxt.Type)) {
            // this.pvc.processValue = Pcvt.asIntPval
            a.isInt = true
            aFmt[aId] = Pcvt.formatInt({ isNullable: this.isNullable, locale: this.locale })
            this.pvc.cellClass = 'pv-cell-right' // numeric cell value style by default
          }
          if (Mdf.isBool(tTxt.Type)) {
            // this.pvc.processValue = Pcvt.asBoolPval
            aFmt[aId] = Pcvt.formatBool({})
            this.pvc.cellClass = 'pv-cell-center'
          }
          if (Mdf.isString(tTxt.Type)) {
            // this.pvc.processValue = Pcvt.asIsPval
            aFmt[aId] = Pcvt.formatDefault({ isNullable: this.isNullable, locale: this.locale })
            this.pvc.cellClass = 'pv-cell-left' // no process or format value required for string type
          }
        }
      }
      this.rank = this.dimProp.length
      this.attrCount = this.attrEnums.length

      // if there are any attributes of built-in type then append measure dimensions:
      //   [rank]:     "normal" attributes dimension: attributes as enums
      //   [rank + 1]: calculated attributes dimension: calculated attributes as enums
      fa.enums = this.attrEnums
      fa.options = fa.enums
      fa.filter = Puih.makeFilter(fa)
      this.dimProp.push(fa) // measure attributes dimension at [rank] position

      fc.enums = this.calcEnums
      fc.options = fc.enums
      fc.filter = Puih.makeFilter(fc)
      this.dimProp.push(fc) // calculated measure dimension at [rank + 1] position

      // if this a run compare the append run compare dimension:
      //   [rank + 2]:   run compare dimension: run names and run descriptions as labels
      // add current run as first dimension item
      const fr = {
        name: RUN_DIM_NAME,
        label: this.$t('Model run'),
        enums: [],
        isBool: false,
        options: [],
        selection: [],
        singleSelection: {},
        filter: (val, update, abort) => {}
      }

      fr.enums = [{
        value: this.runText.RunId,
        name: this.runText.Name,
        label: Mdf.descrOfTxt(this.runText) || this.runText.Name,
        digest: this.runDigest
      }]
      if (this.isRunCompare) {
        const mv = this.modelViewSelected(this.digest)
        for (const d of mv.digestCompareList) {
          const rt = this.runTextByDigest({ ModelDigest: this.digest, RunDigest: d })
          if (Mdf.isNotEmptyRunText(rt)) {
            fr.enums.push({
              value: rt.RunId,
              name: rt.Name,
              label: Mdf.descrOfTxt(rt) || rt.Name,
              digest: d
            })
          }
        }
      }
      fr.options = fr.enums
      fr.filter = Puih.makeFilter(fr)

      this.dimProp.push(fr) // run compare dimension at [rank + 2] position

      // setup formatter
      this.attrFormatter = aFmt
      this.pvc.formatter = Pcvt.formatByKey({
        isNullable: this.isNullable,
        isByKey: true,
        isRawUse: true,
        isRawValue: false,
        locale: this.locale,
        formatter: this.attrFormatter
      })
      this.pvc.dimItemKeys = Pcvt.dimItemKeys(ATTR_DIM_NAME)
      this.pvc.cellClass = 'pv-cell-right' // numeric cell value style by default
      this.pvc.processValue = Pcvt.asIsPval // no value conversion required, only formatting

      // format view controls
      this.ctrl.isRawUseView = this.pvc.formatter.options().isRawUse
      this.ctrl.isRawView = this.pvc.formatter.options().isRawValue
      this.ctrl.isFloatView = this.pvc.formatter.options().isFloat
      this.ctrl.isMoreView = this.pvc.formatter.options().isDoMore
      this.ctrl.isLessView = this.pvc.formatter.options().isDoLess
    },
    // create microdata row reader
    makeMicroReader () {
      // read microdata rows, each row is { Key: integer, Attr:[{IsNull: false, Value: 19},...] }
      // array of attributes:
      //   dimensions are enum-based attributes or boolean
      //   meausre dimension values are values of built-in types attributes
      this.readerMicro = {
        isScale: false, // value scaling is not defined for microdata

        rowReader: (src) => {
          // no data to read: if source rows are empty or invalid return undefined reader
          if (!src || (src?.length || 0) <= 0) return void 0

          // entity key dimension is at [0] position
          // attribute id's: [1, rank - 1] enum based dimensions
          let dimPos = []
          if (this.rank > 1) {
            dimPos = Array(this.rank - 1)

            for (let k = 1; k < this.rank; k++) {
              dimPos[k - 1] = this.dimProp[k].attrPos
            }
          }

          // measure dimension at [rank] position: attribute id's are enum values of measure dimension
          const mIds = Array(this.attrCount)
          const attrPos = Array(this.attrCount)

          if (this.attrCount > 0) {
            let n = 0
            for (const e of this.dimProp[this.rank].enums) {
              mIds[n] = e.value
              attrPos[n++] = e.attrPos
            }
          }

          const srcLen = src.length
          let nSrc = 0
          let nAttr = -1 // after first read row must be nAttr = 0

          const rd = { // reader to return

            readRow: () => {
              nAttr++
              if (nAttr >= this.attrCount) {
                nAttr = 0
                nSrc++
              }
              return (nSrc < srcLen) ? (src[nSrc] || void 0) : void 0 // microdata row: key and array of enum-based attributes as dimensions and buit-in types attributes as values
            },
            readDim: {},
            readValue: (r) => {
              const a = r?.Attr || void 0
              const v = (a && !a[attrPos[nAttr]].IsNull) ? a[attrPos[nAttr]].Value : void 0 // measure value: built-in type attribute value
              return v
            }
          }

          // read entity key dimension
          rd.readDim[KEY_DIM_NAME] = (r) => r?.Key

          // read dimension item value: enum id for enum-based attributes
          for (let n = 1; n < this.rank; n++) {
            if (!this.dimProp[n].isBool) { // enum-based dimension
              rd.readDim[this.dimProp[n].name] = (r) => {
                const a = r?.Attr || void 0
                const cv = (a && dimPos[n - 1] < a.length) ? a[dimPos[n - 1]] : void 0
                return (cv && !cv.IsNull) ? cv.Value : void 0
              }
            } else { // boolean dimension: enum id's: 0 = false, 1 = true
              rd.readDim[this.dimProp[n].name] = (r) => {
                const a = r?.Attr || void 0
                const cv = (a && dimPos[n - 1] < a.length) ? a[dimPos[n - 1]] : void 0
                return (cv && !cv.IsNull) ? (cv.Value ? 1 : 0) : void 0
              }
            }
          }

          // read measure dimension: attribute id
          rd.readDim[ATTR_DIM_NAME] = (r) => (nAttr < mIds.length ? mIds[nAttr] : void 0)

          return rd
        }
      }
    },

    // set page view: use previous page view from store or default
    async setPageView () {
      // if previous page view exist in session store
      let mv = this.microdataView(this.routeKey)
      if (!mv) {
        await this.restoreDefaultView() // restore default microdata view, if exist
        mv = this.microdataView(this.routeKey) // check if default view of microdata restored
        if (!mv) {
          this.setInitialPageView() // setup and use initial view of microdata
          return
        }
      }
      // else: restore previous view
      this.ctrl.kind = (typeof mv?.kind === typeof 1) ? (mv.kind % 3 || Puih.ekind.MICRO) : Puih.ekind.MICRO // there are only 3 kinds of view possible for microdata
      this.aggrCalc = (typeof mv?.aggrCalc === typeof 'string') ? mv?.aggrCalc : ''
      this.cmpCalc = (typeof mv?.cmpCalc === typeof 'string') ? mv?.cmpCalc : ''
      this.groupBy = Array.isArray(mv?.groupBy) ? mv.groupBy : []
      this.calcEnums = Array.isArray(mv?.calcEnums) ? mv.calcEnums : []
      this.valueFilter = Array.isArray(mv?.valueFilter) ? mv.valueFilter : []
      if (this.ctrl.kind === Puih.ekind.CMP && !this.isRunCompare) this.ctrl.kind = Puih.ekind.CALC // no runs to compare

      this.groupDimCalc = Array.from(this.groupBy)
      this.attrCalc = []

      // [rank + 1]: calculated measure dimension
      this.dimProp[this.rank + 1].enums = this.calcEnums
      this.dimProp[this.rank + 1].options = this.dimProp[this.rank + 1].enums
      this.dimProp[this.rank + 1].filter = Puih.makeFilter(this.dimProp[this.rank + 1])

      // restore rows, columns, others layout and items selection
      const restore = (edSrc) => {
        const dst = []
        for (const ed of edSrc) {
          const f = this.dimProp.find((d) => d.name === ed.name)
          if (!f) continue
          if (!this.isDimKindValid(this.ctrl.kind, f.name)) continue // skip measure dimensions not applicable for that view kind

          f.selection = []
          for (const v of ed.values) {
            const e = f.enums.find((fe) => fe.value === v)
            if (e) {
              f.selection.push(e)
            }
          }
          f.singleSelection = (f.selection.length > 0) ? f.selection[0] : {}

          dst.push(f)
        }
        return dst
      }
      this.rowFields = restore(mv.rows)
      this.colFields = restore(mv.cols)
      this.otherFields = restore(mv.others)

      // if there are any dimensions which are not in rows, columns or others then push it to others
      for (const f of this.dimProp) {
        if (this.rowFields.findIndex((d) => f.name === d.name) >= 0) continue
        if (this.colFields.findIndex((d) => f.name === d.name) >= 0) continue
        if (this.otherFields.findIndex((d) => f.name === d.name) >= 0) continue
        if (!this.isDimKindValid(this.ctrl.kind, f.name)) continue // skip measure dimensions not applicable for that view kind

        // append to other fields, select first enum
        f.selection = []
        f.selection.push(f.enums[0])
        f.singleSelection = (f.selection.length > 0) ? f.selection[0] : {}
        this.otherFields.push(f)
      }

      // if entity key dimensions on others then move it to rows last position
      const n = this.otherFields.findIndex(f => f.name === KEY_DIM_NAME)
      if (n >= 0) {
        this.otherFields.splice(n, 1)
        this.rowFields.push(this.dimProp[0])
      }
      // entity dimension page: all keys always selected
      if (this.dimProp.length > 0) {
        this.dimProp[0].selection = Array.from(this.dimProp[0].enums)
        this.dimProp[0].singleSelection = (this.dimProp[0].selection.length > 0) ? this.dimProp[0].selection[0] : {}
      }

      // restore controls view state
      this.ctrl.isRowColControls = !!mv.isRowColControls
      this.pvc.rowColMode = typeof mv.rowColMode === typeof 1 ? mv.rowColMode : Pcvt.NO_SPANS_AND_DIMS_PVT

      // restore page offset and size
      if (this.isPages) {
        this.pageStart = (typeof mv?.pageStart === typeof 1) ? (mv?.pageStart || 0) : 0
        this.pageSize = (typeof mv?.pageSize === typeof 1 && mv.pageSize >= 0) ? mv.pageSize : SMALL_PAGE_SIZE
      } else {
        this.pageStart = 0
        this.pageSize = 0
      }

      // store view and refresh pivot view: both dimensions labels and table body
      if (this.ctrl.kind === Puih.ekind.CALC || this.ctrl.kind === Puih.ekind.CMP) this.doCalcPage()

      this.storeView()
      this.ctrl.isPvDimsTickle = !this.ctrl.isPvDimsTickle
      this.ctrl.isPvTickle = !this.ctrl.isPvTickle
    },
    // return true if dimension name valid for the view kind
    isDimKindValid (kind, name) {
      switch (kind) {
        case Puih.ekind.MICRO:
          return name !== CALC_DIM_NAME && name !== RUN_DIM_NAME
        case Puih.ekind.CALC:
          return name === CALC_DIM_NAME || (Array.isArray(this.groupBy) && this.groupBy.indexOf(name) >= 0)
        case Puih.ekind.CMP:
          return name === CALC_DIM_NAME || name === RUN_DIM_NAME || (Array.isArray(this.groupBy) && this.groupBy.indexOf(name) >= 0)
      }
      return false // invalid view kind
    },

    // set initial page view for microdata
    setInitialPageView () {
      // all dimensions on rows, entity key dimension is at last row, measure dimension on columns
      const rf = []
      const cf = []
      const tf = []
      this.ctrl.kind = Puih.ekind.MICRO
      this.aggrCalc = ''
      this.cmpCalc = ''
      this.groupBy = []
      this.calcEnums = []
      this.groupDimCalc = []
      this.attrCalc = []

      // rows: rank dimensions
      for (let nDim = 1; nDim < this.rank; nDim++) {
        rf.push(this.dimProp[nDim])
      }
      if (this.dimProp.length > 0) rf.push(this.dimProp[0]) // entity key dimension at rows on last position

      // columns: measure attribute dimension, it is at [rank] position in dimensions array
      if (this.dimProp.length > this.rank) cf.push(this.dimProp[this.rank])

      // for rows and columns select all items
      for (const f of cf) {
        f.selection = Array.from(f.enums)
        f.singleSelection = (f.selection.length > 0) ? f.selection[0] : {}
      }
      for (const f of rf) {
        f.selection = Array.from(f.enums)
        f.singleSelection = (f.selection.length > 0) ? f.selection[0] : {}
      }
      // other dimensions are empty by default
      this.rowFields = rf
      this.colFields = cf
      this.otherFields = tf

      // default row-column mode: row-column headers without spans
      // as it is today microdata cannot be scalar, always has at least one attribute
      this.pvc.rowColMode = !this.isScalar ? Pcvt.NO_SPANS_AND_DIMS_PVT : Pcvt.NO_SPANS_NO_DIMS_PVT
      this.storeView()

      // refresh pivot view: both dimensions labels and table body
      this.ctrl.isPvDimsTickle = !this.ctrl.isPvDimsTickle
      this.ctrl.isPvTickle = !this.ctrl.isPvTickle
    },
    // store pivot view
    storeView () {
      const vs = Pcvt.pivotStateFromFields(this.rowFields, this.colFields, this.otherFields, this.ctrl.isRowColControls, this.pvc.rowColMode, KEY_DIM_NAME)
      vs.kind = this.ctrl.kind || Puih.ekind.MICRO // view kind is specific to microdata
      vs.pageStart = 0
      vs.pageSize = this.isPages ? ((typeof this.pageSize === typeof 1 && this.pageSize >= 0) ? this.pageSize : SMALL_PAGE_SIZE) : 0
      vs.aggrCalc = this.aggrCalc || ''
      vs.cmpCalc = this.cmpCalc || ''
      vs.groupBy = Array.isArray(this.groupBy) ? this.groupBy : []
      vs.calcEnums = Array.isArray(this.calcEnums) ? this.calcEnums : []
      vs.valueFilter = Array.isArray(this.valueFilter) ? this.valueFilter : []

      this.dispatchMicrodataView({
        key: this.routeKey,
        view: vs,
        digest: this.digest || '',
        modelName: Mdf.modelName(this.theModel),
        runDigest: this.runDigest || '',
        entityName: this.entityName || ''
      })
    },

    // find model run and check if run entity exist in the model run
    initRunEntity () {
      if (!this.checkRunEntity()) return false // return error if model, run or entity name is empty

      this.runText = this.runTextByDigest({ ModelDigest: this.digest, RunDigest: this.runDigest })
      if (!Mdf.isNotEmptyRunText(this.runText)) {
        console.warn('Model run not found:', this.digest, this.runDigest)
        this.$q.notify({ type: 'negative', message: this.$t('Model run not found: ') + this.runDigest })
        return false
      }

      this.entityText = Mdf.entityTextByName(this.theModel, this.entityName)
      if (!Mdf.isNotEmptyEntityText(this.entityText)) {
        console.warn('Model entity not found:', this.entityName)
        this.$q.notify({ type: 'negative', message: this.$t('Model entity not found: ') + this.entityName })
        return false
      }
      this.runEntity = Mdf.runEntityByName(this.runText, this.entityName)
      if (!Mdf.isNotEmptyRunEntity(this.runEntity)) {
        console.warn('Entity microdata not found in model run:', this.entityName, this.runDigest)
        this.$q.notify({ type: 'negative', message: this.$t('Entity microdata not found in model run: ') + this.entityName + ' ' + this.runDigest })
        return false
      }
      return true
    },
    // check if model digeset, run digest and entity name is not empty
    checkRunEntity () {
      if (!this.digest) {
        console.warn('Invalid (empty) model digest')
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) model digest') })
        return false
      }
      if (!this.runDigest) {
        console.warn('Unable to show microdata: run digest is empty')
        this.$q.notify({ type: 'negative', message: this.$t('Unable to show microdata: run digest is empty') })
        return false
      }
      if (!this.entityName) {
        console.warn('Unable to show microdata: entity name is empty')
        this.$q.notify({ type: 'negative', message: this.$t('Unable to show microdata: entity name is empty') })
        return false
      }
      return true
    },

    // show microdata view
    doMicroPage () {
      // replace measure dimension
      const mNewIdx = this.rank // [rank]: measure attributes dimension index in dimesions list

      let n = this.replaceMeasureDim(CALC_DIM_NAME, mNewIdx, this.rowFields, false)
      if (n < 0) n = this.replaceMeasureDim(CALC_DIM_NAME, mNewIdx, this.colFields, false)
      if (n < 0) n = this.replaceMeasureDim(CALC_DIM_NAME, mNewIdx, this.otherFields, true)
      if (n < 0) {
        console.warn('Measure dimension not found:', CALC_DIM_NAME)
        this.$q.notify({ type: 'negative', message: this.$t('Measure dimension not found') })
        return
      }

      // remove run dimension, if exists (if it was run comparison)
      let isFound = this.removeDimField(this.rowFields, RUN_DIM_NAME)
      if (!isFound) isFound = this.removeDimField(this.colFields, RUN_DIM_NAME)
      if (!isFound) isFound = this.removeDimField(this.otherFields, RUN_DIM_NAME)

      // add missing dimensions on rows: rank-1 dimensions
      for (let nDim = 1; nDim < this.rank; nDim++) {
        if (this.rowFields.findIndex(d => d.name === this.dimProp[nDim].name) >= 0) continue
        if (this.colFields.findIndex(d => d.name === this.dimProp[nDim].name) >= 0) continue
        if (this.otherFields.findIndex(d => d.name === this.dimProp[nDim].name) >= 0) continue

        this.rowFields.push(this.dimProp[nDim])

        this.dimProp[nDim].selection = Array.from(this.dimProp[nDim].enums)
        this.dimProp[nDim].singleSelection = (this.dimProp[nDim].selection.length > 0) ? this.dimProp[nDim].selection[0] : {}
      }

      // add key dimension if not exists
      isFound = this.rowFields.findIndex(d => d.name === KEY_DIM_NAME) >= 0
      if (!isFound) isFound = this.colFields.findIndex(d => d.name === KEY_DIM_NAME) >= 0
      if (!isFound) this.removeDimField(this.otherFields, KEY_DIM_NAME)
      if (!isFound) {
        const n = this.rowFields.length
        this.rowFields.push(this.dimProp[0]) // key dimension default position: last on rows

        this.dimProp[n].selection = Array.from(this.dimProp[n].enums)
        this.dimProp[n].singleSelection = (this.dimProp[n].selection.length > 0) ? this.dimProp[n].selection[0] : {}
      }

      // use microdata row reader and format by key on measure attributes dimension
      this.pvc.reader = this.readerMicro
      this.pvc.dimItemKeys = Pcvt.dimItemKeys(ATTR_DIM_NAME)

      // setup formatter
      const fOpts = this.pvc.formatter.floatOptions() // shared options for float attributes
      this.pvc.formatter = Pcvt.formatByKey({
        isNullable: this.isNullable,
        isRawUse: true,
        isRawValue: this.ctrl.isRawView,
        locale: this.locale,
        formatter: this.attrFormatter
      })
      this.pvc.formatter.setDecimals(fOpts.nDecimal, fOpts.isAllDecimal) // set number of decimals for float attributes

      this.pvc.dimItemKeys = Pcvt.dimItemKeys(ATTR_DIM_NAME)
      this.pvc.cellClass = 'pv-cell-right' // numeric cell value style by default
      this.pvc.processValue = Pcvt.asIsPval // no value conversion required, only formatting

      // format view controls
      this.ctrl.isRawUseView = this.pvc.formatter.options().isRawUse
      this.ctrl.isRawView = this.pvc.formatter.options().isRawValue
      this.ctrl.isFloatView = this.pvc.formatter.options().isFloat
      this.ctrl.isMoreView = this.pvc.formatter.options().isDoMore
      this.ctrl.isLessView = this.pvc.formatter.options().isDoLess

      // set new view kind and  store pivot view
      this.ctrl.kind = Puih.ekind.MICRO
      this.storeView()
      this.doRefreshDataPage()
    },

    // open value filter dialog
    onValueFilter () {
      // set filter measures from attribute enums or calculated enums
      this.valueFilterMeasure = []

      if (this.ctrl.kind === Puih.ekind.MICRO) {
        for (const e of this.attrEnums) {
          this.valueFilterMeasure.push({
            name: e.name,
            label: e.label
          })
        }
      } else {
        for (const e of this.calcEnums) {
          this.valueFilterMeasure.push({
            name: e.name,
            label: e.label
          })
        }
      }

      this.valueFilterTickle = !this.valueFilterTickle
    },
    // update value filters
    onValueFilterApply (fltLst) {
      if (!Array.isArray(fltLst)) {
        this.valueFilter = []
        console.warn('Invalid (or empty) value filters', fltLst)
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (or empty) value filters') })
        return
      }

      // validate filters and save filter state
      const fLst = []
      for (const f of fltLst) {
        // measure must be an attribute name or calculation name
        if ((f?.name || '') === '' || typeof f?.name !== typeof 'string' ||
          (this.attrEnums.findIndex(e => e.name === f?.name) < 0 && this.calcEnums.findIndex(e => e.name === f?.name) < 0)) {
          this.$q.notify({ type: 'warning', message: this.$t('Invalid (or empty) filter measure') + ' [' + fLst.length.toString() + '] : ' + (f?.name || '') + ' ' + (f?.label || '') })
          // return
        }
        if ((f?.op || '') === '' || typeof f?.op !== typeof 'string' || Mdf.filterOpList.findIndex(op => op.code === f?.op) < 0) {
          this.$q.notify({ type: 'negative', message: this.$t('Invalid (or empty) filter condition') + ' [' + fLst.length.toString() + '] : ' + (f?.op || '') })
          return
        }
        if (!Array.isArray(f?.value) || f.value.length < 0) {
          this.$q.notify({ type: 'negative', message: this.$t('Invalid (or empty) value entered') + ' [' + fLst.length.toString() + ']' })
          return
        }
        fLst.push({
          name: f.name,
          label: f?.label || '',
          op: f.op,
          value: f.value
        })
      }
      this.valueFilter = fLst
      this.dispatchMicrodataView({ key: this.routeKey, valueFilter: this.valueFilter })

      this.doRefreshDataPage() // apply new value filters
    },

    // user click to show aggregated microdata view
    onCalcPage () {
      if (!Mdf.isLength(this.groupDimCalc)) {
        console.warn('Invalid (empty) list of aggregation dimensions', this.groupBy)
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) list of aggregation dimensions') })
        return
      }

      // calculated measures dimension: enums are aggregated attributes or aggreagted comparison of attributes
      if (Mdf.isLength(this.attrCalc)) {
        if (!this.aggrCalc) {
          console.warn('Invalid (empty) aggregation selected:', this.aggrCalc)
          this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) aggregation selected') })
          return
        }
        this.updateCalcDim()
      }
      if (!Mdf.isLength(this.calcEnums)) {
        console.warn('Invalid (empty) list of calculated measure attributes', this.attrCalc)
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) list of calculated measure attributes') })
        return
      }

      // show aggregated microdata view
      this.groupBy = Array.from(this.groupDimCalc)
      this.doCalcPage()
      this.storeView()
      this.doRefreshDataPage()
    },
    // show calculation edit dialog
    onCalcEdit () {
      if (this.calcEnums.length <= 0) {
        if (!this.aggrCalc) {
          console.warn('Invalid (empty) aggregation selected:', this.aggrCalc)
          this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) aggregation selected') })
          return
        }
        if (!Mdf.isLength(this.attrCalc)) {
          console.warn('Invalid (empty) list of calculated measure attributes', this.attrCalc)
          this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) list of calculated measure attributes') })
          return
        }
        this.updateCalcDim()
      }
      this.calcEditTickle = !this.calcEditTickle
    },
    // copy edited calcultion list into calcEnums
    onCalcEditApply (cLst) {
      if (!Mdf.isLength(cLst) || cLst.findIndex(c => !!c.calc && Mdf.isLength(c.calc.trim())) < 0) {
        console.warn('Invalid (empty) list of calculated measure attributes', cLst)
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) list of calculated measure attributes') })
        return
      }
      this.calcEnums = []

      let nId = Mdf.CALCULATED_ID_OFFSET
      for (let k = 0; k < cLst.length; k++) {
        const c = cLst[k]
        c.calc = (c.calc || '').trim()
        if (!c.calc) continue // skip empty calculations

        // calcId must be unique
        if (c.calcId < 0 || cLst.findIndex((e, idx) => { return idx !== k && e.calcId === c.calcId }) >= 0) {
          while (cLst.findIndex(e => e.calcId === nId) >= 0) {
            nId++
          }
          c.calcId = nId
        }

        // cleanup calculation name and make it unique
        c.name = Mdf.cleanColumnValueInput(c.name)

        if ((c.name || '') === '' || cLst.findIndex((e, idx) => { return idx !== k && e.name === c.name }) >= 0) {
          c.name = ((typeof c.name === typeof 'string' && (c.name || '') !== '') ? (c.name + '_') : '') + 'ex_' + c.calcId.toString()
          if (c.name.length > 32) {
            c.name = c.name.substring(c.name.length - 32)
          }
        }

        this.calcEnums.push({
          value: c.calcId,
          name: c.name,
          label: (c.label || '').trim() || c.name,
          isInt: false,
          isFloat: true,
          calc: c.calc
        })
      }

      this.aggrCalc = '' // disable update from menu
      this.cmpCalc = ''
      this.attrCalc = []

      // if dimension list defined then show aggregated microdata view
      if (!Mdf.isLength(this.groupDimCalc)) return

      this.groupBy = Array.from(this.groupDimCalc)
      this.doCalcPage()
      this.storeView()
      this.doRefreshDataPage()
    },
    // set aggregated calculation name
    onAggregateSet (src) {
      if (this.aggrCalcList.findIndex(c => src === c.code) < 0) {
        console.warn('Invalid aggregation selected:', src)
        this.$q.notify({ type: 'negative', message: this.$t('Invalid aggregation selected') + ' ' + (src || '') })
        return
      }
      this.aggrCalc = src
      this.calcEnums = [] // calculation updated
      if (this.aggrCalc && Mdf.isLength(this.aggrCalc)) this.updateCalcDim()
    },
    // set run comparison calculation name
    onCompareToogle (src) {
      if (!src) return
      if (src === this.cmpCalc) { // clear comparison
        this.cmpCalc = ''
      } else {
        if (this.compareCalcList.findIndex(c => src === c.code) < 0) {
          console.warn('Invalid aggregation selected:', src)
          this.$q.notify({ type: 'negative', message: this.$t('Invalid aggregation selected') + ' ' + (src || '') })
          return
        }
        this.cmpCalc = src // set new comparison
      }
      this.calcEnums = [] // calculation updated

      if (this.aggrCalc && Mdf.isLength(this.aggrCalc)) this.updateCalcDim()
    },
    // add or remove group by dimension name on click
    onGroupByToogle (name) {
      if (!name) return
      const n = this.groupDimCalc.indexOf(name)
      if (n >= 0) {
        this.groupDimCalc.splice(n, 1)
      } else {
        this.groupDimCalc.push(name)
      }
    },
    // return true if dimesion name is in group by dimensions list
    isGroupBy (name) {
      return !!name && this.groupDimCalc.indexOf(name) >= 0
    },
    // add or remove group by dimension name on click
    onCalcAttrToogle (name) {
      if (!name) return
      const n = this.attrCalc.indexOf(name)
      if (n >= 0) {
        this.attrCalc.splice(n, 1)
      } else {
        this.attrCalc.push(name)
      }
      this.calcEnums = [] // measure attributes updated
      if (this.aggrCalc && Mdf.isLength(this.aggrCalc)) this.updateCalcDim()
    },
    // return true if measure attribute name is in calculated attributes list
    isAttrCalc (name) {
      return !!name && this.attrCalc.indexOf(name) >= 0
    },
    // show aggregated microdata view
    doCalcPage () {
      if (!Mdf.isLength(this.groupBy)) {
        console.warn('Invalid (empty) list of aggregation dimensions', this.groupBy)
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) list of aggregation dimensions') })
        return
      }
      if (!Mdf.isLength(this.calcEnums)) {
        console.warn('Invalid (empty) list of calculated measure attributes', this.attrCalc)
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) list of calculated measure attributes') })
        return
      }

      // if there is no calculated measure dimension then push it on columns
      // [rank + 1]: calculated attributes dimension: calculated attributes as enums
      const mNewIdx = this.rank + 1
      this.dimProp[mNewIdx].enums = this.calcEnums

      let n = this.replaceMeasureDim(ATTR_DIM_NAME, mNewIdx, this.rowFields, false)
      if (n < 0) n = this.replaceMeasureDim(ATTR_DIM_NAME, mNewIdx, this.colFields, false)
      if (n < 0) n = this.replaceMeasureDim(ATTR_DIM_NAME, mNewIdx, this.otherFields, true)
      if (n < 0) {
        if (this.rowFields.findIndex(d => d.name === CALC_DIM_NAME) < 0 &&
          this.colFields.findIndex(d => d.name === CALC_DIM_NAME) < 0 &&
          this.otherFields.findIndex(d => d.name === CALC_DIM_NAME) < 0) { // if measure dimension not exist the push to columns
          this.colFields.push(this.dimProp[mNewIdx])
        }
      }

      // if calculated dimension on rows or columns then select all items
      // if calculated dimension on others then select first item
      if (this.otherFields.findIndex(f => f.name === CALC_DIM_NAME) >= 0) {
        this.dimProp[mNewIdx].selection = [this.dimProp[mNewIdx].enums[0]]
      } else {
        this.dimProp[mNewIdx].selection = Array.from(this.dimProp[mNewIdx].enums)
      }
      this.dimProp[mNewIdx].singleSelection = (this.dimProp[mNewIdx].selection.length > 0) ? this.dimProp[mNewIdx].selection[0] : {}

      // remove dimensions which are not group by dimensions
      for (let k = 0; k < this.rank; k++) {
        if (this.groupBy.indexOf(this.dimProp[k].name) < 0) {
          let isFound = this.removeDimField(this.rowFields, this.dimProp[k].name)
          if (!isFound) isFound = this.removeDimField(this.colFields, this.dimProp[k].name)
          if (!isFound) isFound = this.removeDimField(this.otherFields, this.dimProp[k].name)
        }
      }
      let isFound = this.removeDimField(this.rowFields, KEY_DIM_NAME)
      if (!isFound) isFound = this.removeDimField(this.colFields, KEY_DIM_NAME)
      if (!isFound) isFound = this.removeDimField(this.otherFields, KEY_DIM_NAME)

      // if any group by dimension not exist push it on rows
      for (const name of this.groupBy) {
        if (this.rowFields.findIndex(d => d.name === name) < 0 &&
          this.colFields.findIndex(d => d.name === name) < 0 &&
          this.otherFields.findIndex(d => d.name === name) < 0) {
          const n = this.dimProp.findIndex(d => d.name === name)
          if (n >= 0) {
            this.rowFields.push(this.dimProp[n])
            this.dimProp[n].selection = Array.from(this.dimProp[n].enums)
            this.dimProp[n].singleSelection = (this.dimProp[n].selection.length > 0) ? this.dimProp[n].selection[0] : {}
          }
        }
      }

      // if this is not run comparison then remove runs dimension
      // else insert runs dimension before measure dimension: before calculated dimension
      if (!this.isRunCompare) {
        isFound = this.removeDimField(this.rowFields, RUN_DIM_NAME)
        if (!isFound) isFound = this.removeDimField(this.colFields, RUN_DIM_NAME)
        if (!isFound) isFound = this.removeDimField(this.otherFields, RUN_DIM_NAME)
      } else {
        // if runs dimension not exist in the view then insert it before measure dimension: before calculated dimension
        if (this.rowFields.findIndex(d => d.name === RUN_DIM_NAME) < 0 &&
          this.colFields.findIndex(d => d.name === RUN_DIM_NAME) < 0 &&
          this.otherFields.findIndex(d => d.name === RUN_DIM_NAME) < 0) {
          //
          const fr = this.dimProp[this.rank + 2] // [rank + 2]: model runs dimension

          // insert or replace runs dimension before measure dimension: before calculated dimension
          const insertRunDim = (dims, isOther) => {
            // find measure position in dimension list
            const n = dims.findIndex(d => d.name === CALC_DIM_NAME)
            if (n < 0) return false

            dims.splice(n, 0, fr) // insert runs dimension before measure dimension

            // select all items if it rows or columns, select first item if it is others slicer
            if (isOther) {
              fr.selection = [fr.enums[0]]
            } else {
              fr.selection = Array.from(fr.enums)
            }
            fr.singleSelection = (fr.selection.length > 0) ? fr.selection[0] : {}

            return true
          }
          isFound = insertRunDim(this.rowFields, false)
          if (!isFound) isFound = insertRunDim(this.colFields, false)
          if (!isFound) isFound = insertRunDim(this.otherFields, true)
        }
      }

      // set new view kind, store pivot view and refersh the data
      this.ctrl.kind = this.isRunCompare ? Puih.ekind.CMP : Puih.ekind.CALC
      this.updateCalcFormat()
      this.pvc.reader = this.makeCalcReader()
    },
    // setup calculation measure dimension and format options
    updateCalcDim () {
      //
      const ceLst = []

      for (const a of this.attrEnums) {
        if (this.attrCalc.indexOf(a.name) < 0) continue // this attribute is not selected as calculated measure
        if (!a.isInt && !a.isFloat) {
          console.warn('Invalid type of calculated measure attribute: it must be numeric:', a.name)
          continue
        }

        // add to calculated measure both attributes: source attribute and calculated attribute
        let alb = this.aggrCalc
        const n = this.aggrCalcList.findIndex(c => this.aggrCalc === c.code)
        if (n >= 0) alb = this.$t(this.aggrCalcList[n].label)

        const ia = this.aggrCalc === 'COUNT' || (a.isInt && (this.aggrCalc === 'MIN' || this.aggrCalc === 'MAX'))
        const ae = {
          // value: a.value + Mdf.CALCULATED_ID_OFFSET,
          // name: this.aggrCalc + '_' + a.name,
          value: ceLst.length + Mdf.CALCULATED_ID_OFFSET,
          name: 'ex_' + (ceLst.length + Mdf.CALCULATED_ID_OFFSET).toString(),
          label: alb + ' ' + a.label,
          isInt: ia,
          isFloat: !ia,
          calc: Mdf.toAggregateCompareFnc(this.aggrCalc, '', a.name) // calculation: aggregate source attribute
        }
        ceLst.push(ae)

        if (this.isRunCompare && this.cmpCalc) {
          let clb = this.cmpCalc
          const n = this.compareCalcList.findIndex(c => this.cmpCalc === c.code)
          if (n >= 0) clb = this.$t(this.compareCalcList[n].label)

          const ic = this.aggrCalc === 'COUNT' || (a.isInt && (this.aggrCalc === 'MIN' || this.aggrCalc === 'MAX') && this.cmpCalc === 'DIFF')
          const ce = {
            // value: a.value + 2 * Mdf.CALCULATED_ID_OFFSET,
            // name: this.aggrCalc + '_' + this.cmpCalc + '_' + a.name,
            value: ceLst.length + Mdf.CALCULATED_ID_OFFSET,
            name: 'ex_' + (ceLst.length + Mdf.CALCULATED_ID_OFFSET).toString(),
            label: alb + ' ' + clb + ' ' + a.label,
            isInt: ic,
            isFloat: !ic,
            calc: Mdf.toAggregateCompareFnc(this.aggrCalc, this.cmpCalc, a.name) // calculation: aggregate source attribute
          }
          ceLst.push(ce)
        }
      }

      this.calcEnums = ceLst
      this.calcUpdateTickle = !this.calcUpdateTickle
    },
    // setup attributes format based on claculated measure items: float or integer
    updateCalcFormat () {
      const cFmt = {} // formatters for calculated attributes

      for (const ce of this.calcEnums) {
        if (ce.isInt) {
          cFmt[ce.value] = Pcvt.formatInt({ isNullable: this.isNullable, locale: this.locale })
        } else {
          cFmt[ce.value] = Pcvt.formatFloat({
            isNullable: this.isNullable,
            locale: this.locale,
            nDecimal: Pcvt.maxDecimalDefault,
            maxDecimal: Pcvt.maxDecimalDefault,
            isAllDecimal: true // show all deciamls for the float value
          })
        }
      }

      // [rank + 1]: calculated attributes dimension: calculated attributes as enums
      this.dimProp[this.rank + 1].enums = this.calcEnums
      this.dimProp[this.rank + 1].options = this.calcEnums
      this.dimProp[this.rank + 1].filter = Puih.makeFilter(this.dimProp[this.rank + 1])

      // setup formatter
      const fOpts = this.pvc.formatter.floatOptions() // shared options for float attributes
      this.pvc.formatter = Pcvt.formatByKey({
        isNullable: this.isNullable,
        isRawUse: true,
        isRawValue: this.ctrl.isRawView,
        locale: this.locale,
        formatter: cFmt
      })
      this.pvc.formatter.setDecimals(fOpts.nDecimal, fOpts.isAllDecimal) // set number of decimals for float attributes

      this.pvc.dimItemKeys = Pcvt.dimItemKeys(CALC_DIM_NAME)
      this.pvc.cellClass = 'pv-cell-right' // numeric cell value style by default
      this.pvc.processValue = Pcvt.asIsPval // no value conversion required, only formatting

      // format view controls
      this.ctrl.isRawUseView = this.pvc.formatter.options().isRawUse
      this.ctrl.isRawView = this.pvc.formatter.options().isRawValue
      this.ctrl.isFloatView = this.pvc.formatter.options().isFloat
      this.ctrl.isMoreView = this.pvc.formatter.options().isDoMore
      this.ctrl.isLessView = this.pvc.formatter.options().isDoLess
    },
    // create row reader for aggregated microdata or for microdata run comparison based on current group by dimensions
    makeCalcReader () {
      // read calculated rows: one row for each calculation
      // each row is { CalcId: 1208, RunId: 219, Attr:[{IsNull: false, Value: 19},...] }
      // array of attributes:
      //   group by dimensions are enum-based attributes or boolean
      //   last position is meausre dimension values of float or integer type
      return {
        isScale: false, // value scaling is not defined for microdata

        rowReader: (src) => {
          // no data to read: if source rows are empty or invalid return undefined reader
          if (!src || (src?.length || 0) <= 0) return void 0

          const srcLen = src.length
          let nSrc = 0

          const dimPos = []
          let rPos = 0
          for (const d of this.dimProp) {
            if (this.groupBy.indexOf(d.name) >= 0) dimPos.push(rPos++)
          }
          const nVal = this.groupBy.length // calculated attribute value at last position

          const rd = { // reader to return
            readRow: () => {
              return (nSrc < srcLen) ? src[nSrc++] : void 0 // calculated row: one row for each calculation
            },
            readDim: {},
            readValue: (r) => {
              const a = r?.Attr || void 0
              const cv = (a && nVal < a.length) ? a[nVal] : void 0 // calculated attribute value at last position
              return (cv && !cv.IsNull) ? cv.Value : void 0
            }
          }

          // read dimension item value: enum id for enum-based attributes
          let nPos = 0
          for (const d of this.dimProp) {
            if (this.groupBy.indexOf(d.name) < 0) continue // skip: this is not a group by diemnsion

            const n = dimPos[nPos++]
            if (!d.isBool) { // enum-based dimension
              rd.readDim[d.name] = (r) => {
                const a = r?.Attr || void 0
                const cv = (a && n < a.length) ? a[n] : void 0
                return (cv && !cv.IsNull) ? cv.Value : void 0
              }
            } else { // boolean dimension: enum id's: 0 = false, 1 = true
              rd.readDim[d.name] = (r) => {
                const a = r?.Attr || void 0
                const cv = (a && n < a.length) ? a[n] : void 0
                return (cv && !cv.IsNull) ? (cv.Value ? 1 : 0) : void 0
              }
            }
          }

          // read calculation dimension: calculation id
          // read run dimension: run id
          rd.readDim[CALC_DIM_NAME] = (r) => (r.CalcId)
          rd.readDim[RUN_DIM_NAME] = (r) => (r.RunId)

          return rd
        }
      }
    },
    // replace measure dimension in rows, columns or othres list by new dimension
    // for example, replace measure attributes dimension with calculated attributes measure
    replaceMeasureDim (mName, newDimIdx, dims, isOther) {
      // find measure position in dimension list
      const n = dims.findIndex(d => d.name === mName)
      if (n < 0) return n

      dims[n] = this.dimProp[newDimIdx] // replace existing measure with new

      // select all items if it rows or columns, select first item if it is others slicer
      if (isOther) {
        this.dimProp[newDimIdx].selection = [this.dimProp[newDimIdx].enums[0]]
      } else {
        this.dimProp[newDimIdx].selection = Array.from(this.dimProp[newDimIdx].enums)
      }
      this.dimProp[newDimIdx].singleSelection = (this.dimProp[newDimIdx].selection.length > 0) ? this.dimProp[newDimIdx].selection[0] : {}

      return n
    },
    // remove dimension from view dimensions: from rows, columns or others
    removeDimField (dims, name) {
      const n = dims.findIndex(d => d.name === name)
      if (n >= 0) dims.splice(n, 1)
      return n >= 0
    },

    // show entity notes dialog
    doShowEntityNote () {
      this.entityInfoTickle = !this.entityInfoTickle
    },
    // show run notes dialog
    doShowRunNote (modelDgst, runDgst) {
      this.runInfoTickle = !this.runInfoTickle
    },

    // show or hide row/column/other bars
    onToggleRowColControls () {
      this.ctrl.isRowColControls = !this.ctrl.isRowColControls
      this.dispatchMicrodataView({ key: this.routeKey, isRowColControls: this.ctrl.isRowColControls })
    },
    onSetRowColMode (mode) {
      this.pvc.rowColMode = (4 + mode) % 4
      this.dispatchMicrodataView({ key: this.routeKey, rowColMode: this.pvc.rowColMode })
    },
    // switch between show dimension names and item names or labels
    onShowItemNames () {
      this.pvc.isShowNames = !this.pvc.isShowNames
      this.ctrl.isPvDimsTickle = !this.ctrl.isPvDimsTickle
    },
    // show more decimals (or more details) in table body
    onShowMoreFormat () {
      if (!this.pvc.formatter) return
      this.pvc.formatter.doMore()
      this.ctrl.isMoreView = this.pvc.formatter.options().isDoMore
      this.ctrl.isLessView = this.pvc.formatter.options().isDoLess
      this.ctrl.isRawView = this.pvc.formatter.options().isRawValue
      this.ctrl.isPvTickle = !this.ctrl.isPvTickle
    },
    // show less decimals (or less details) in table body
    onShowLessFormat () {
      if (!this.pvc.formatter) return
      this.pvc.formatter.doLess()
      this.ctrl.isMoreView = this.pvc.formatter.options().isDoMore
      this.ctrl.isLessView = this.pvc.formatter.options().isDoLess
      this.ctrl.isRawView = this.pvc.formatter.options().isRawValue
      this.ctrl.isPvTickle = !this.ctrl.isPvTickle
    },
    // toogle to formatted value or to raw value in table body
    onToggleRawValue () {
      if (!this.pvc.formatter) return
      this.pvc.formatter.doRawValue()
      this.ctrl.isMoreView = this.pvc.formatter.options().isDoMore
      this.ctrl.isLessView = this.pvc.formatter.options().isDoLess
      this.ctrl.isRawView = this.pvc.formatter.options().isRawValue
      this.ctrl.isPvTickle = !this.ctrl.isPvTickle
    },
    // copy tab separated values to clipboard: forward actions to pivot table component
    onCopyToClipboard () {
      this.$refs.omPivotTable.onCopyTsv()
    },

    // view table rows by pages
    onPageSize (size) {
      this.pageSize = size
      if (this.isAllPageSize()) {
        this.pageStart = 0
        this.pageSize = 0
      }
      this.dispatchMicrodataView({ key: this.routeKey, pageSize: size })
      this.doRefreshDataPage()
    },
    onFirstPage () {
      this.pageStart = 0
      this.doRefreshDataPage()
    },
    onPrevPage () {
      this.pageStart = this.pageStart - this.pageSize
      if (this.pageStart < 0) this.pageStart = 0

      this.doRefreshDataPage()
    },
    onNextPage () {
      this.pageStart = this.pageStart + this.pageSize
      this.doRefreshDataPage()
    },
    onLastPage () {
      if (this.isAllPageSize() || this.pageSize > SMALL_PAGE_SIZE) { // limit last page size
        this.pageSize = SMALL_PAGE_SIZE
        this.$q.notify({ type: 'info', message: this.$t('Size reduced to: ') + this.pageSize })
      }
      this.pageStart = LAST_PAGE_OFFSET

      this.doRefreshDataPage(true)
    },
    isAllPageSize () {
      return !this.pageSize || typeof this.pageSize !== typeof 1 || this.pageSize <= 0
    },

    // download microdata as csv file
    onDownload () {
      // start with:
      //   api/model/:model/run/:run/microdata/:name
      let u = this.omsUrl +
        '/api/model/' + encodeURIComponent(this.digest) +
        '/run/' + encodeURIComponent(this.runDigest) +
        '/microdata/' + encodeURIComponent(this.entityName)

      // api/model/:model/run/:run/microdata/:name/csv
      // api/model/:model/run/:run/microdata/:name/group-by/:group-by/calc/:calc/csv
      // api/model/:model/run/:run/microdata/:name/group-by/:group-by/compare/:compare/variant/:variant/csv
      let cp = ''
      let pn = ''

      if (this.ctrl.kind !== Puih.ekind.MICRO) {
        // calculation can contain divide by / slash, use query paramters instead of url: calc/NONE/csv?calc=OM_AVG(x/y)
        for (const ce of this.calcEnums) {
          cp += (cp ? ',' : '') + ce.calc
        }

        u += '/group-by/' + encodeURIComponent(this.groupBy.join(','))

        if (this.ctrl.kind === Puih.ekind.CALC) {
          u += '/calc/NONE'
          pn = 'calc'
        } else {
          let dl = ''
          for (const e of this.dimProp[this.rank + 2].enums) { // [rank + 2]: run compare dimension
            if (e.digest !== this.digest) dl += (dl ? ',' : '') + e.digest
          }

          u += '/compare/NONE/variant/' + encodeURIComponent(dl)
          pn = 'compare'
        }
      }
      u += ((this.$q.platform.is.win) ? (this.idCsvDownload ? '/csv-id-bom' : '/csv-bom') : (this.idCsvDownload ? '/csv-id' : '/csv')) + '?' + pn + '=' + encodeURIComponent(cp)
      openURL(u)
    },

    // reload microdata data and reset pivot view to default
    async onReloadDefaultView () {
      if (this.pvc.formatter) {
        this.pvc.formatter.resetOptions()
      }
      this.dispatchMicrodataViewDelete(this.routeKey) // clean current view
      await this.restoreDefaultView()
      await this.setPageView()
      this.doRefreshDataPage()
    },
    // save current view as default microdata view
    async onSaveDefaultView () {
      const mv = this.microdataView(this.routeKey)
      if (!mv) {
        console.warn('Microdata view not found:', this.routeKey)
        return
      }

      // convert selection values from enum Ids to enum codes for rows, columns, others dimensions
      const enumIdsToCodes = (edSrc) => {
        const dst = []
        for (const ed of edSrc) {
          if (!ed?.values) continue // skip if no items selected in the dimension

          const f = this.dimProp.find((d) => d.name === ed.name)
          if (!f) {
            console.warn('Error: dimension not found:', ed.name)
            continue
          }
          const isRd = f.name === RUN_DIM_NAME
          let cArr = []

          if (ed.name !== KEY_DIM_NAME) { // skip items if it is entity key dimension
            const eLen = Mdf.lengthOf(ed.values)
            if (eLen > 0) {
              cArr = Array(eLen)
              let n = 0
              for (let k = 0; k < eLen; k++) {
                const i = f.enums.findIndex(e => e.value === ed.values[k])
                if (i >= 0) {
                  if (!isRd) {
                    cArr[n++] = f.enums[i].name
                  } else {
                    cArr[n++] = f.enums[i].digest // model runs dimension
                  }
                }
              }
              cArr.length = n // remove size of not found
            }
          }

          dst.push({
            name: ed.name,
            values: cArr
          })
        }
        return dst
      }

      // convert microdata view to "default" view: replace enum Ids with enum codes
      const dv = {
        rows: enumIdsToCodes(mv.rows),
        cols: enumIdsToCodes(mv.cols),
        others: enumIdsToCodes(mv.others),
        isRowColControls: this.ctrl.isRowColControls,
        rowColMode: this.pvc.rowColMode,
        kind: this.ctrl.kind || Puih.ekind.MICRO,
        pageStart: this.isPages ? this.pageStart : 0,
        pageSize: this.isPages ? this.pageSize : 0,
        aggrCalc: this.aggrCalc || '',
        cmpCalc: this.cmpCalc || '',
        groupBy: Array.isArray(this.groupBy) ? Array.from(this.groupBy) : [],
        calcEnums: Array.isArray(this.calcEnums) ? Array.from(this.calcEnums) : []
      }

      // save into indexed db
      try {
        const dbCon = await Idb.connection()
        const rw = await dbCon.openReadWrite(Mdf.modelName(this.theModel))
        await rw.put(this.entityName, dv)
      } catch (e) {
        console.warn('Unable to save default microdata view', e)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to save default microdata view: ') + this.entityName })
        return
      }

      this.$q.notify({ type: 'info', message: this.$t('Default view of microdata saved: ') + this.entityName })
      this.$emit('user-view-updated', this.digest, Mdf.modelName(this.theModel))
    },

    // restore default microdata view
    async restoreDefaultView () {
      // select default microdata view from inxeded db
      let dv
      try {
        const dbCon = await Idb.connection()
        const rd = await dbCon.openReadOnly(Mdf.modelName(this.theModel))
        dv = await rd.getByKey(this.entityName)
      } catch (e) {
        console.warn('Unable to restore default microdata view', this.entityName, e)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to restore default microdata view: ') + this.entityName })
        return
      }
      // exit if not found or empty
      if (!dv || !dv?.rows || !dv?.cols || !dv?.others) {
        return
      }

      // restore view kind, aggregation name, comparison name, group by and calculated measure
      this.ctrl.kind = (typeof dv?.kind === typeof 1) ? ((dv.kind % 3) || Puih.ekind.MICRO) : Puih.ekind.MICRO // there are only 3 kinds of view possible for output table
      this.aggrCalc = (typeof dv?.aggrCalc === typeof 'string') ? (dv?.aggrCalc || '') : ''
      this.cmpCalc = (typeof dv?.cmpCalc === typeof 'string') ? (dv?.cmpCalc || '') : ''
      this.groupBy = Array.isArray(dv?.groupBy) ? dv.groupBy : []
      this.calcEnums = Array.isArray(dv?.calcEnums) ? dv.calcEnums : []

      this.groupDimCalc = Array.from(this.groupBy)
      this.attrCalc = []

      // [rank + 1]: calculated measure dimension
      this.dimProp[this.rank + 1].enums = this.calcEnums
      this.dimProp[this.rank + 1].options = this.dimProp[this.rank + 1].enums
      this.dimProp[this.rank + 1].filter = Puih.makeFilter(this.dimProp[this.rank + 1])

      // convert output table view from "default" view: replace enum codes with enum Ids
      let isKey = false
      const enumCodesToIds = (edSrc) => {
        const dst = []
        for (const ed of edSrc) {
          if (!ed.values) continue // empty selection

          const f = this.dimProp.find((d) => d.name === ed.name)
          if (!f) {
            console.warn('Error: dimension not found:', ed.name)
            continue
          }

          const isRd = f.name === RUN_DIM_NAME
          let dl = []
          if (isRd) {
            dl = this?.modelViewSelected(this.digest)?.digestCompareList
            if (!Array.isArray(dl)) dl = []
          }
          let eArr = []

          if (ed.name === KEY_DIM_NAME) { // skip items if it is entity key dimension
            isKey = true
          } else {
            const eLen = Mdf.lengthOf(ed.values)
            if (eLen > 0) {
              eArr = Array(eLen)
              let n = 0
              for (let k = 0; k < eLen; k++) {
                if (!isRd) {
                  const i = f.enums.findIndex(e => e.name === ed.values[k])
                  if (i >= 0) {
                    eArr[n++] = f.enums[i].value
                  }
                } else {
                  if (ed.values[k] !== this.runDigest && dl.findIndex(d => d === ed.values[k]) < 0) continue

                  const rt = this.runTextByDigest({ ModelDigest: this.digest, RunDigest: ed.values[k] })
                  if (Mdf.isNotEmptyRunText(rt)) {
                    eArr[n++] = rt.RunId
                  }
                }
              }
              eArr.length = n // remove size of not found
            }
          }

          dst.push({
            name: ed.name,
            values: eArr
          })
        }
        return dst
      }
      const rows = enumCodesToIds(dv.rows)
      const cols = enumCodesToIds(dv.cols)
      const others = enumCodesToIds(dv.others)
      if (!isKey && this.ctrl.kind === Puih.ekind.MICRO) this.rowFields.push(this.dimProp[0]) // by default put key dimension to the last position on rows

      // restore default page offset and size
      if (this.isPages) {
        this.pageStart = (typeof dv?.pageStart === typeof 1) ? (dv?.pageStart || 0) : 0
        this.pageSize = (typeof dv?.pageSize === typeof 1 && dv?.pageSize >= 0) ? dv.pageSize : SMALL_PAGE_SIZE
      } else {
        this.pageStart = 0
        this.pageSize = 0
      }

      // if is not empty any of selection rows, columns, other dimensions
      // then store pivot view: do insert or replace of the view
      if (Mdf.isLength(rows) || Mdf.isLength(cols) || Mdf.isLength(others)) {
        const vs = Pcvt.pivotState(rows, cols, others, dv.isRowColControls, dv.rowColMode || Pcvt.NO_SPANS_AND_DIMS_PVT)
        vs.kind = this.ctrl.kind || Puih.ekind.MICRO // view kind is specific to microdata
        vs.pageStart = this.isPages ? this.pageStart : 0
        vs.pageSize = this.isPages ? this.pageSize : 0
        vs.aggrCalc = this.aggrCalc || ''
        vs.cmpCalc = this.cmpCalc || ''
        vs.groupBy = this.groupBy
        vs.calcEnums = this.calcEnums
        vs.valueFilter = Array.isArray(this.valueFilter) ? this.valueFilter : []

        this.dispatchMicrodataView({
          key: this.routeKey,
          view: vs,
          digest: this.digest || '',
          modelName: Mdf.modelName(this.theModel),
          runDigest: this.runDigest || '',
          entityName: this.entityName || ''
        })
      }
    },

    // pivot table view updated: item keys layout updated
    onPvKeyPos (keyPos) { this.pvKeyPos = keyPos },

    // dimensions drag, drop and selection filter
    //
    onChoose (e) {
      if (e?.item?.id === 'item-draggable-' + KEY_DIM_NAME) {
        this.isOtherDropDisabled = true // disable drop of entity key dimension on others
      }
    },
    onUnchoose (e) {
      this.isOtherDropDisabled = false
    },
    onDrag (e) {
      // drag started
      this.isDragging = true
    },
    onDrop (e) {
      // drag completed: drop
      this.isDragging = false

      // make sure at least one item selected in each dimension
      // other dimensions: use single-select dropdown
      let isMeasure = false
      for (const f of this.dimProp) {
        if (f.selection.length < 1 && this.isDimKindValid(this.ctrl.kind, f.name)) {
          f.selection.push(f.enums[0])
          if (f.name === ATTR_DIM_NAME || f.name === CALC_DIM_NAME) isMeasure = true
        }
      }
      for (const f of this.otherFields) {
        if (f.selection.length > 1) {
          f.selection = [f.selection[0]]
          if (f.name === ATTR_DIM_NAME || f.name === CALC_DIM_NAME) isMeasure = true
        }
      }
      for (const f of this.dimProp) {
        if (this.isDimKindValid(this.ctrl.kind, f.name)) f.singleSelection = f.selection[0]
      }

      // update pivot view:
      //  if other dimesion(s) filters same as before
      //  then update pivot table view now
      //  else refresh data
      if (Puih.equalFilterState(this.filterState, this.otherFields, [KEY_DIM_NAME, ATTR_DIM_NAME, CALC_DIM_NAME])) {
        this.ctrl.isPvTickle = !this.ctrl.isPvTickle
        if (isMeasure) {
          this.filterState = Puih.makeFilterState(this.otherFields)
        }
      } else {
        this.doRefreshDataPage()
      }
      // update pivot view rows, columns, other dimensions
      this.dispatchMicrodataView({
        key: this.routeKey,
        rows: Pcvt.pivotStateFields(this.rowFields),
        cols: Pcvt.pivotStateFields(this.colFields),
        others: Pcvt.pivotStateFields(this.otherFields),
        kind: this.ctrl.kind
      })
    },

    // dimension select input: selection changed
    onSelectInput (panel, name, vals) {
      if (this.isDragging) return // exit: this is drag-and-drop, no changes in selection yet

      const f = this.dimProp.find((d) => d.name === name)
      if (!f) return

      // sync other dimension(s) single selection value with selection array(s) filter
      if (panel === 'other') {
        f.singleSelection = {}
        f.selection = []
        if (vals) {
          f.singleSelection = vals
          f.selection.push(vals)
        }
      }
      f.selection.sort(
        (left, right) => (left.value === right.value) ? 0 : ((left.value < right.value) ? -1 : 1)
      )

      // update pivot view:
      //   if other dimesions filters same as before then update pivot table view now
      //   else refresh data
      if (panel !== 'other' || Puih.equalFilterState(this.filterState, this.otherFields, [KEY_DIM_NAME, ATTR_DIM_NAME])) {
        this.ctrl.isPvTickle = !this.ctrl.isPvTickle
        if (name === ATTR_DIM_NAME) {
          this.filterState = Puih.makeFilterState(this.otherFields)
        }
      } else {
        this.doRefreshDataPage()
      }
      // update pivot view rows, columns, other dimensions
      this.dispatchMicrodataView({
        key: this.routeKey,
        rows: Pcvt.pivotStateFields(this.rowFields),
        cols: Pcvt.pivotStateFields(this.colFields),
        others: Pcvt.pivotStateFields(this.otherFields),
        kind: this.ctrl.kind
      })
    },
    onUpdateSelect (vals) {
      if (this.isDragging) return // exit: this is drag-and-drop, no changes in selection yet

      // find dimension and check if it other panel
      const f = this.dimProp.find((d) => d.name === this.selectDimName)
      if (!f) {
        console.warn('Invalid (empty) selected dimension:', this.selectDimName)
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) selected dimension') })
        return
      }
      const isOther = this.otherFields.findIndex((p) => f.name === p.name) >= 0

      // sync dimension selection values
      f.selection.splice(0, f.selection.length)
      if (!isOther) {
        f.selection.push(...vals)
      } else {
        f.singleSelection = vals
        f.selection.push(vals)
      }
      f.selection.sort(
        (left, right) => (left.value === right.value) ? 0 : ((left.value < right.value) ? -1 : 1)
      )

      // update pivot view:
      //   if other dimesions filters same as before then update pivot table view now
      //   else refresh data
      if (!isOther || Puih.equalFilterState(this.filterState, this.otherFields, [KEY_DIM_NAME, ATTR_DIM_NAME])) {
        this.ctrl.isPvTickle = !this.ctrl.isPvTickle
        if (f.name === ATTR_DIM_NAME) {
          this.filterState = Puih.makeFilterState(this.otherFields)
        }
      } else {
        this.doRefreshDataPage()
      }
      // update pivot view rows, columns, other dimensions
      this.dispatchMicrodataView({
        key: this.routeKey,
        rows: Pcvt.pivotStateFields(this.rowFields),
        cols: Pcvt.pivotStateFields(this.colFields),
        others: Pcvt.pivotStateFields(this.otherFields),
        kind: this.ctrl.kind
      })
    },
    onFocusSelect (e) {
      this.selectDimName = ''
      const p = e?.target?.closest('div[id^=item-draggable-]')
      if (p) {
        this.selectDimName = (p?.getAttribute('id') || '').slice('item-draggable-'.length)
      }
      if (!this.selectDimName) {
        console.warn('Invalid (empty) selected dimension:', this.selectDimName)
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) selected dimension') })
      }
    },
    // do "select all" items: all which are visible through filter options
    onSelectAll (name) {
      const f = this.dimProp.find((d) => d.name === name)
      if (!f) return

      // if options not filtered then all select items in dimension
      // else append to current selection items from the filter
      if (f.options.length === f.enums.length) {
        f.selection = Array.from(f.options)
      } else {
        const a = f.options.filter(ov => f.selection.findIndex(sv => sv.value === ov.value) < 0)
        f.selection = f.selection.concat(a)
        f.selection.sort(
          (left, right) => (left.value === right.value) ? 0 : ((left.value < right.value) ? -1 : 1)
        )
      }

      f.singleSelection = f.selection.length > 0 ? f.selection[0] : {}
      f.options = f.enums

      this.updateSelectOrClearView(name)
    },
    // do "clear all" items: all which are visible through filter options
    onClearAll (name) {
      const f = this.dimProp.find((d) => d.name === name)
      if (!f) return

      // if options not filtered then clear all selection (select nothing)
      // else remove from selection all filtered options
      if (f.options.length === f.enums.length) {
        f.selection = []
      } else {
        f.selection = f.selection.filter(sv => f.options.findIndex(ov => ov.value === sv.value) < 0)
      }

      f.singleSelection = f.selection.length > 0 ? f.selection[0] : {}
      f.options = f.enums

      this.updateSelectOrClearView(name)
    },
    // update pivot view after "select all" or "clear all"
    updateSelectOrClearView (name) {
      this.ctrl.isPvTickle = !this.ctrl.isPvTickle
      if (name === ATTR_DIM_NAME) {
        this.filterState = Puih.makeFilterState(this.otherFields)
      }
      // update pivot view rows, columns, other dimensions
      this.dispatchMicrodataView({
        key: this.routeKey,
        rows: Pcvt.pivotStateFields(this.rowFields),
        cols: Pcvt.pivotStateFields(this.colFields),
        others: Pcvt.pivotStateFields(this.otherFields),
        kind: this.ctrl.kind
      })
    },
    //
    // end of dimensions drag, drop and selection filter

    // make a label for dimension item(s) select
    selectLabel (isNames, f) {
      const dsl = this.$t('Select')

      if (!f) return dsl + '\u2026'
      //
      switch (f.selection.length) {
        case 0: return dsl + ' ' + (isNames ? f.name : f.label) + '\u2026'
        case 1: return (isNames ? f.selection[0].name : f.selection[0].label)
      }
      return (isNames ? f.selection[0].name : f.selection[0].label) + ', ' + '\u2026'
    },

    // get page of microdata or aggregated microdata from current model run
    async doRefreshDataPage (isFullPage = false) {
      if (!this.checkRunEntity()) return // exit on error

      // value filters: filter only by measures from current view: only attribute enums or calculated enums
      const vfLst = []
      for (const f of this.valueFilter) {
        const isOk = (this.ctrl.kind === Puih.ekind.MICRO) ? (this.attrEnums.findIndex(e => e.name === f.name) >= 0) : (this.calcEnums.findIndex(e => e.name === f.name) >= 0)
        if (isOk) {
          vfLst.push(f)
        } else {
          const s = Array.isArray(f.value) ? (' ' + f.value.join(', ')) : ''
          this.$q.notify({
            type: 'info',
            message: this.$t('Skip filter: ') + (f?.label || '') + ' (' + (f.name || '') + ') ' + (f.op || '') + ' ' + (s.length > 40 ? (s.substring(0, 40) + '\u2026') : s)
          })
        }
      }

      this.filterState = Puih.makeFilterState(this.otherFields) // save filters: other dimensions selected items

      // make microdata read layout
      const layout = Puih.makeSelectLayout(this.entityName, this.otherFields, [KEY_DIM_NAME, ATTR_DIM_NAME, CALC_DIM_NAME, RUN_DIM_NAME], vfLst)
      layout.Offset = 0
      layout.Size = 0
      layout.IsFullPage = false
      if (this.isPages) {
        layout.Offset = this.pageStart || 0
        layout.Size = (!!this.pageSize && typeof this.pageSize === typeof 1) ? (this.pageSize || 0) : 0
        layout.IsFullPage = !!isFullPage
      }

      if (this.ctrl.kind === Puih.ekind.MICRO) {
        this.doRefreshMicroPage(layout)
        return
      }
      this.doRefreshCalcPage(layout)
    },

    // get page of microdata from current model run
    async doRefreshMicroPage (layout) {
      this.loadDone = false
      this.loadWait = true

      const u = this.omsUrl +
        '/api/model/' + encodeURIComponent(this.digest) +
        '/run/' + encodeURIComponent(this.runDigest) + '/microdata/value-id'

      // retrieve page from server, it must be: {Layout: {...}, Page: [...]}
      let isOk = false
      try {
        const response = await this.$axios.post(u, layout)
        const rsp = response.data

        if (!Puih.isPageLayoutRsp(rsp)) {
          const em = Puih.errorFromPageLayoutRsp(rsp)
          this.$q.notify({ type: 'negative', message: (((em || '') !== '') ? em : this.$t('Server offline or microdata not found: ') + this.entityName) })
        } else {
          let d = []
          if (!rsp) {
            this.pageStart = 0
            this.isLastPage = true
          } else {
            if ((rsp?.Page?.length || 0) > 0) {
              d = rsp.Page
            }
            this.pageStart = rsp?.Layout?.Offset || 0
            this.isLastPage = rsp?.Layout?.IsLastPage || false
          }

          // update pivot table view and set entity key dimension enums
          this.inpData = Object.freeze(d)

          const eLst = Array(this.inpData.length)
          for (let j = 0; j < this.inpData.length; j++) {
            eLst[j] = {
              value: this.inpData[j]?.Key || 0,
              name: this.inpData[j]?.Key.toString() || 'Invalid',
              label: this.inpData[j]?.Key.toString() || this.$t('Invalid key')
            }
          }
          this.dimProp[0].enums = eLst
          this.dimProp[0].selection = Array.from(eLst)
          this.dimProp[0].singleSelection = (this.dimProp[0].selection.length > 0) ? this.dimProp[0].selection[0] : {}

          this.loadDone = true
          this.ctrl.isPvDimsTickle = !this.ctrl.isPvDimsTickle
          this.ctrl.isPvTickle = !this.ctrl.isPvTickle
          isOk = true
        }
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or microdata not found', em)
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or microdata not found: ') + this.entityName })
      }

      this.loadWait = false
      if (isOk) {
        this.dispatchMicrodataView({
          key: this.routeKey,
          pageStart: this.isPages ? this.pageStart : 0,
          pageSize: this.isPages ? this.pageSize : 0
        })
      }
    },

    // get page of aggregated microdata from current model run
    async doRefreshCalcPage (layout) {
      this.loadDone = false
      this.loadWait = true

      // make calculation layout: add Calculation, GroupBy and Runs if it is run comparison
      /*
      {
          "Name": "Person",
          "Calculation": [{
                  "Calculate": "OM_AVG(Income) * (param.StartingSeed / 100)",
                  "CalcId": 2401,
                  "Name": "Avg_Income_adjusted"
              }, {
                  "Calculate": "OM_AVG(Salary + Pension + param.StartingSeed)",
                  "CalcId": 2404,
                  "Name": "Avg_Salary_Pension_adjusted"
              }
          ],
          "GroupBy": [
              "AgeGroup",
              "Sex"
          ],
          "Runs": [
              "abcd1234"
          ],
          "Offset": 0,
          "Size": 100,
          "IsFullPage": true,
          "Filter": [....]
      }
      */
      const ltc = Object.assign({}, layout,
        {
          Calculation: [],
          GroupBy: this.groupBy,
          Runs: []
        })
      for (const ce of this.calcEnums) {
        ltc.Calculation.push({
          Calculate: ce.calc,
          CalcId: ce.value,
          Name: ce.name
        })
      }
      // if it is run comparison and runs are not on filters then add run digests
      if (this.isRunCompare && this.otherFields.findIndex(d => d.name === RUN_DIM_NAME) < 0) {
        for (const e of this.dimProp[this.rank + 2].enums) { // [rank + 2]: run compare dimension
          if (e.digest !== this.digest) ltc.Runs.push(e.digest)
        }
      }

      const u = this.omsUrl +
        '/api/model/' + encodeURIComponent(this.digest) +
        '/run/' + encodeURIComponent(this.runDigest) + '/microdata/compare-id'

      // retrieve page from server, it must be: {Layout: {...}, Page: [...]}
      let isOk = false
      try {
        const response = await this.$axios.post(u, ltc)
        const rsp = response.data

        if (!Puih.isPageLayoutRsp(rsp)) {
          console.warn('Invalid response to:', u)
          const em = Puih.errorFromPageLayoutRsp(rsp)
          this.$q.notify({ type: 'negative', message: (((em || '') !== '') ? em : this.$t('Server offline or microdata not found: ') + this.entityName) })
        } else {
          let d = []
          if (!rsp) {
            this.pageStart = 0
            this.isLastPage = true
          } else {
            if ((rsp?.Page?.length || 0) > 0) {
              d = rsp.Page
            }
            this.pageStart = rsp?.Layout?.Offset || 0
            this.isLastPage = rsp?.Layout?.IsLastPage || false
          }
          this.inpData = Object.freeze(d)

          this.loadDone = true
          this.ctrl.isPvDimsTickle = !this.ctrl.isPvDimsTickle
          this.ctrl.isPvTickle = !this.ctrl.isPvTickle
          isOk = true
        }
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or microdata not found', em)
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or microdata not found: ') + this.entityName })
      }

      this.loadWait = false
      if (isOk) {
        this.dispatchMicrodataView({
          key: this.routeKey,
          pageStart: this.isPages ? this.pageStart : 0,
          pageSize: this.isPages ? this.pageSize : 0
        })
      }
    }
  },

  mounted () {
    this.doRefresh()
    this.$emit('tab-mounted', 'entity', { digest: this.digest, runDigest: this.runDigest, entityName: this.entityName })
    if (!this.isModelTab) document.title = this.entityName
  }
}
