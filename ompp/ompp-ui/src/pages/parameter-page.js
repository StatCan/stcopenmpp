import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'
import * as Mdf from 'src/model-common'
import * as Idb from 'src/idb/idb'
import RunBar from 'components/RunBar.vue'
import WorksetBar from 'components/WorksetBar.vue'
import RefreshRun from 'components/RefreshRun.vue'
import RefreshWorkset from 'components/RefreshWorkset.vue'
import RunInfoDialog from 'components/RunInfoDialog.vue'
import WorksetInfoDialog from 'components/WorksetInfoDialog.vue'
import ParameterInfoDialog from 'components/ParameterInfoDialog.vue'
import EditDiscardDialog from 'components/EditDiscardDialog.vue'
import { VueDraggableNext } from 'vue-draggable-next'
import * as Pcvt from 'components/pivot-cvt'
import * as Puih from './pivot-ui-helper'
import PvTable from 'components/PvTable.vue'
import MarkdownEditor from 'components/MarkdownEditor.vue'
import { openURL } from 'quasar'

/* eslint-disable no-multi-spaces */
const SMALL_PAGE_SIZE = 1000                    // small page size: do not show page controls
const LAST_PAGE_OFFSET = 2 * 1024 * 1024 * 1024 // large page offset to get the last page
/* eslint-enable no-multi-spaces */

export default {
  name: 'ParameterPage',
  components: {
    draggable: VueDraggableNext,
    PvTable,
    RunBar,
    WorksetBar,
    RefreshRun,
    RefreshWorkset,
    RunInfoDialog,
    WorksetInfoDialog,
    ParameterInfoDialog,
    EditDiscardDialog,
    MarkdownEditor
  },

  props: {
    digest: { type: String, default: '' },
    runDigest: { type: String, default: '' },
    worksetName: { type: String, default: '' },
    parameterName: { type: String, default: '' },
    refreshTickle: { type: Boolean, default: false }
  },

  /* eslint-disable no-multi-spaces */
  data () {
    return {
      loadDone: false,
      loadWait: false,
      saveStarted: false,
      saveWait: false,
      isNullable: false,    // if true then parameter value can be NULL
      isScalar: false,      // if true then it is scalar parameter with single sub-value
      rank: 0,              // parameter rank
      paramText: Mdf.emptyParamText(),
      paramType: Mdf.emptyTypeText(),
      paramRunSet: Mdf.emptyParamRunSet(),
      subCount: 0,
      dimProp: [],
      colFields: [],
      rowFields: [],
      otherFields: [],
      filterState: {},
      inpData: Object.freeze([]),
      ctrl: {
        isRowColControls: true,
        isRowColModeToggle: true,
        isPvTickle: false,      // used to update view of pivot table (on data selection change, on edit/save change)
        isPvDimsTickle: false,  // used to update dimensions in pivot table (on label change)
        isRawShow: false,       // for number editor this is raw value format status before editor started
        //
        isRawUseView: false, // vue3 copy of isRawUse
        isRawView: false,    // vue3 copy of isRawValue
        isFloatView: false,  // vue3 copy of isFloat
        isMoreView: false,   // vue3 copy of isDoMore
        isLessView: false    // vue3 copy of isDoLess
      },
      pvc: {
        rowColMode: Pcvt.SPANS_AND_DIMS_PVT,  // rows and columns mode: 2 = use spans and show dim names
        isShowNames: false,                   // if true then show dimension names and item names instead of labels
        reader: void 0,                       // return row reader: if defined then methods to read next row, read() dimension items and readValue()
        processValue: Pcvt.asIsPval,          // default value processing: return as is
        formatter: Pcvt.formatDefault,        // disable format(), parse() and validation by default
        cellClass: 'pv-cell-right',           // default cell value style: right justified number
        dimItemKeys: void 0                   // interface to find dimension item key (enum id) by row or column number
      },
      pvKeyPos: [],           // position of each dimension item in cell key
      edt: Pcvt.emptyEdit(),  // editor options and state shared with child
      isDragging: false,      // if true then user is dragging dimension select control
      selectDimName: '',      // selected dimension name
      isPages: false,
      pageStart: 0,
      pageSize: 0,
      isLastPage: false,
      isShowPageControls: false,
      loadRunWait: false,
      refreshRunTickle: false,
      loadWsWait: false,
      refreshWsTickle: false,
      showEditDiscardTickle: false,
      runInfoTickle: false,
      worksetInfoTickle: false,
      paramInfoTickle: false,
      noteEditorShow: false,
      noteEditorNotes: '',
      noteEditorLangCode: '',
      showDiscardParamNoteTickle: false,
      pathToRouteLeave: '',       // router component leave guard: path-to leave if user confirm changes discard
      isYesToLeave: false,        // if true then do router push to leave the page
      isUploadEnabled: false,
      uploadFileSelect: false,
      subCountUpload: 1,
      defaultSubUpload: 0,
      uploadFile: null,
      isModelTab: this.$route.path.startsWith('/model/') // if true then model page tab else parameter top page
    }
  },
  /* eslint-enable no-multi-spaces */

  computed: {
    isFromRun () { return (this.runDigest || '') !== '' },
    routeKey () {
      return (this.runDigest || '') !== ''
        ? Mdf.parameterRunPath(this.digest, this.runDigest, this.parameterName)
        : Mdf.parameterWorksetPath(this.digest, this.worksetName, this.parameterName)
    },
    paramDescr () { return Mdf.descrOfDescrNote(this.paramText) },
    fileSelected () { return !(this.uploadFile === null) },
    isEditUpdated () { return this.edt.isUpdated },
    isReadonlyServer () { return this.serverConfig?.IsReadonly ?? true },

    ...mapState(useModelStore, [
      'theModel',
      'theModelUpdated',
      'worksetTextListUpdated'
    ]),
    ...mapState(useServerStateStore, {
      omsUrl: 'omsUrl',
      serverConfig: 'config'
    }),
    ...mapState(useUiStateStore, [
      'idCsvDownload',
      'uiLang'
    ])
  },

  watch: {
    routeKey () { this.doRefresh() },
    refreshTickle () { this.doRefresh() },
    theModelUpdated () { this.doRefresh() },
    isEditUpdated () { this.$emit('edit-updated', this.edt.isUpdated, this.routeKey) },
    worksetTextListUpdated () { this.onWorksetUpdated() }
  },

  emits: [
    'edit-updated',
    'new-run-select',
    'set-update-readonly',
    'user-view-updated',
    'tab-mounted'
  ],

  methods: {
    ...mapActions(useModelStore, [
      'runTextByDigest',
      'worksetTextByName'
    ]),
    ...mapActions(useUiStateStore, [
      'paramViewWorksetUpdatedCount',
      'paramView',
      //
      'dispatchParamView',
      'dispatchParamViewDelete'
    ]),

    // show run parameter notes dialog
    doShowParamNote () {
      this.paramInfoTickle = !this.paramInfoTickle
    },
    // show run notes dialog
    doShowRunNote (modelDgst, runDgst) {
      this.runInfoTickle = !this.runInfoTickle
    },
    // show current workset notes dialog
    doShowWorksetNote (modelDgst, name) {
      this.worksetInfoTickle = !this.worksetInfoTickle
    },
    // on button click "new model run": pass event from child up to the next level
    onNewRunClick () {
      this.$emit('new-run-select', this.worksetName)
    },

    // on button click "toggle workset readonly status": pass event from child up to the next level
    onWorksetReadonlyToggle (dgst, name, isReadonly) {
      this.$emit('set-update-readonly', dgst, name, isReadonly)
    },
    // update workset read only status:
    //   set read-write if argument isReadonly is false
    //   set read-only if all parameters edit saved and parameter note editor closed
    updateWorksetReadonly (isReadonly) {
      if (this.isFromRun || (this.worksetName || '') === '') return // exit: this is not a workset parameter

      if (isReadonly) {
        if (this.paramViewWorksetUpdatedCount({ digest: this.digest, worksetName: this.worksetName }) > 0) {
          return // not all parameters saved for current workset
        }
        if (this.noteEditorShow) return // note editor active
      }
      this.onWorksetReadonlyToggle(this.digest, this.worksetName, isReadonly)
    },
    // workset updated: check read-only status and adjust controls
    // if sub-values count changed as result of parameter upload then reset view and reload data
    onWorksetUpdated () {
      const nSub = this.subCount
      const { isFound, src } = this.initParamRunSet()
      this.edt.isEnabled = this.isUploadEnabled = !this.isFromRun && isFound && Mdf.isNotEmptyWorksetText(src) // && !src.IsReadonly

      if (isFound && nSub !== (this.paramRunSet.SubCount || 0)) {
        this.dispatchParamViewDelete(this.routeKey)
        this.doRefresh()
      }
    },

    // show or hide row/column/other bars
    onToggleRowColControls () {
      this.ctrl.isRowColControls = !this.ctrl.isRowColControls
      this.dispatchParamView({ key: this.routeKey, isRowColControls: this.ctrl.isRowColControls })
    },
    onSetRowColMode (mode) {
      this.pvc.rowColMode = (4 + mode) % 4
      this.dispatchParamView({ key: this.routeKey, rowColMode: this.pvc.rowColMode })
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
      this.ctrl.isRawView = this.pvc.formatter.options().isRawValue // this.pvc.formatter.options().isRawValue
      this.ctrl.isPvTickle = !this.ctrl.isPvTickle
    },
    // copy tab separated values to clipboard: forward actions to pivot table component
    onCopyToClipboard () {
      this.$refs.omPivotTable.onCopyTsv()
    },

    // download parameter as csv file
    onDownload () {
      const udgst = encodeURIComponent(this.digest)
      const uname = encodeURIComponent(this.parameterName)
      let u = this.isFromRun
        ? this.omsUrl +
          '/api/model/' + udgst + '/run/' + encodeURIComponent(this.runDigest) + '/parameter/' + uname
        : this.omsUrl +
          '/api/model/' + udgst + '/workset/' + encodeURIComponent(this.worksetName) + '/parameter/' + uname
      u += (this.$q.platform.is.win) ? (this.idCsvDownload ? '/csv-id-bom' : '/csv-bom') : (this.idCsvDownload ? '/csv-id' : '/csv')

      openURL(u)
    },

    // view parameter rows by pages
    onPageSize (size) {
      this.pageSize = size
      if (this.isAllPageSize()) {
        this.pageStart = 0
        this.pageSize = 0
      }
      this.dispatchParamView({ key: this.routeKey, pageSize: size })
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
      this.isShowPageControls = this.pageSize > 0

      this.doRefreshDataPage(true)
    },
    isAllPageSize () {
      return !this.pageSize || typeof this.pageSize !== typeof 1 || this.pageSize <= 0
    },

    // show parameter csv upload dialog
    doShowFileSelect () {
      this.uploadFileSelect = true
      this.updateWorksetReadonly(false) // set workset read-write
    },
    // hides input parameter csv upload dialog
    doCancelFileSelect () {
      this.uploadFileSelect = false
      this.uploadFile = null
      this.updateWorksetReadonly(true) // set workset read only if all parameters saved and workset note editor closed
    },

    // reload parameter data and reset pivot view to default
    async onReloadDefaultView () {
      if (this.pvc.formatter) {
        this.pvc.formatter.resetOptions()
      }
      this.dispatchParamViewDelete(this.routeKey) // clean current view
      await this.restoreDefaultView()
      await this.setPageView()
      this.doRefreshDataPage()
    },
    // save current view as default parameter view
    async onSaveDefaultView () {
      const pv = this.paramView(this.routeKey)
      if (!pv) {
        console.warn('Parameter view not found:', this.routeKey)
        return
      }

      // convert selection values from enum Ids to enum codes for rows, columns, others dimensions
      const enumIdsToCodes = (pvSrc) => {
        const dst = []
        for (const p of pvSrc) {
          if (!p.values) continue // skip if no items selected in the dimension

          // if regular dimension then make array of enum codes
          let cArr = []
          if (p.name !== Puih.SUB_ID_DIM) {
            const dt = this.paramText.ParamDimsTxt.find((d) => d.Dim.Name === p.name)
            if (!dt) {
              console.warn('Error: dimension not found:', p.name)
              continue
            }
            const t = Mdf.typeTextById(this.theModel, (dt.Dim.TypeId || 0))
            cArr = Mdf.enumIdArrayToCodeArray(t, p.values)
          } else {
            // sub-value dimension: enum code same as enum id
            for (const eid of p.values) {
              cArr.push(eid.toString())
            }
          }
          dst.push({
            name: p.name,
            values: cArr
          })
        }
        return dst
      }

      // convert parameter view to "default" view: replace enum Ids with enum codes, ignore edit {} state
      const dv = {
        rows: enumIdsToCodes(pv.rows),
        cols: enumIdsToCodes(pv.cols),
        others: enumIdsToCodes(pv.others),
        isRowColControls: this.ctrl.isRowColControls,
        rowColMode: this.pvc.rowColMode,
        pageStart: this.isPages ? this.pageStart : 0,
        pageSize: this.isPages ? this.pageSize : 0
      }

      // save into indexed db
      try {
        const dbCon = await Idb.connection()
        const rw = await dbCon.openReadWrite(Mdf.modelName(this.theModel))
        await rw.put(this.parameterName, dv)
      } catch (e) {
        console.warn('Unable to save default parameter view', e)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to save default parameter view: ' + this.parameterName) })
        return
      }
      this.$q.notify({ type: 'info', message: this.$t('Default view of parameter saved: ') + this.parameterName })
            this.$emit('user-view-updated', this.digest, Mdf.modelName(this.theModel))

    },

    // restore default parameter view
    async restoreDefaultView () {
      // select default parameter view from inxeded db
      let dv
      try {
        const dbCon = await Idb.connection()
        const rd = await dbCon.openReadOnly(Mdf.modelName(this.theModel))
        dv = await rd.getByKey(this.parameterName)
      } catch (e) {
        console.warn('Unable to restore default parameter view', this.parameterName, e)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to restore default parameter view: ') + this.parameterName })
        return
      }
      // exit if not found or empty
      if (!dv || !dv?.rows || !dv?.cols || !dv?.others) {
        return
      }

      // convert parameter view from "default" view: replace enum codes with enum Ids
      const enumCodesToIds = (pvSrc) => {
        const dst = []
        for (const p of pvSrc) {
          if (!p.values) continue // empty selection

          // if regular dimension then make array of enum id's
          let eArr = []
          if (p.name !== Puih.SUB_ID_DIM) {
            const dt = this.paramText.ParamDimsTxt.find((d) => d.Dim.Name === p.name)
            if (!dt) {
              continue // unknown dimension: skip
            }
            const t = Mdf.typeTextById(this.theModel, (dt.Dim.TypeId || 0))
            eArr = Mdf.codeArrayToEnumIdArray(t, p.values)
          } else {
            // sub-value dimension: enum id the same as enum code
            for (const ec of p.values) {
              const n = parseInt(ec)
              if (!isNaN(n)) eArr.push(n)
            }
          }
          dst.push({
            name: p.name,
            values: eArr
          })
        }
        return dst
      }
      const rows = enumCodesToIds(dv.rows)
      const cols = enumCodesToIds(dv.cols)
      const others = enumCodesToIds(dv.others)

      // restore default page offset and size
      if (this.isPages) {
        this.pageStart = (typeof dv?.pageStart === typeof 1) ? (dv?.pageStart || 0) : 0
        this.pageSize = (typeof dv?.pageSize === typeof 1 && dv?.pageSize >= 0) ? dv.pageSize : SMALL_PAGE_SIZE
      } else {
        this.pageStart = 0
        this.pageSize = 0
      }
      this.isShowPageControls = this.pageSize > 0

      // if is not empty any of selection rows, columns, other dimensions
      // then store pivot view: do insert or replace of the view
      if (Mdf.isLength(rows) || Mdf.isLength(cols) || Mdf.isLength(others)) {
        const vs = Pcvt.pivotState(rows, cols, others, dv.isRowColControls, dv.rowColMode || Pcvt.SPANS_AND_DIMS_PVT)
        vs.edit = this.edt // edit state exist only for parameters
        vs.pageStart = this.isPages ? this.pageStart : 0
        vs.pageSize = this.isPages ? this.pageSize : 0

        this.dispatchParamView({
          key: this.routeKey,
          view: vs,
          digest: this.digest || '',
          modelName: Mdf.modelName(this.theModel),
          runDigest: this.runDigest || '',
          worksetName: this.worksetName || '',
          parameterName: this.parameterName || ''
        })
      }
    },

    // pivot table view updated: item keys layout updated
    onPvKeyPos (keyPos) { this.pvKeyPos = keyPos },

    // start of editor methods
    //
    // start or stop parameter editing
    doEditToogle () {
      if (this.edt.isEdit && this.edt.isUpdated) { // redirect to dialog to confirm "discard changes?"
        this.showEditDiscardTickle = !this.showEditDiscardTickle
        return
      }
      // else: start editing or stop editing (no changes in data)
      const isEditNow = this.edt.isEdit
      Pcvt.resetEdit(this.edt)
      this.edt.isEdit = !isEditNow

      // for numeric editor display raw value during editing and restore raw value format status after edit completed
      if (this.edt.kind === Pcvt.EDIT_NUMBER && this.pvc.formatter) {
        if (this.edt.isEdit) {
          this.ctrl.isRawShow = this.pvc.formatter.options().isRawValue
          if (!this.ctrl.isRawShow) this.pvc.formatter.doRawValue() // show raw value on edit start
        } else {
          if (this.pvc.formatter.options().isRawValue !== this.ctrl.isRawShow) this.pvc.formatter.doRawValue() // switch back to formatted value on edit complete
        }
      }

      this.dispatchParamView({ key: this.routeKey, edit: this.edt })
      this.updateWorksetReadonly(!this.edt.isEdit)
    },
    // parameter editor question: "Discard all changes?", user answer: "yes"
    onYesDiscardChanges () {
      Pcvt.resetEdit(this.edt)
      this.dispatchParamView({ key: this.routeKey, edit: this.edt })

      if (this.edt.kind === Pcvt.EDIT_NUMBER && this.pvc.formatter) {
        if (this.pvc.formatter.options().isRawValue !== this.ctrl.isRawShow) this.pvc.formatter.doRawValue() // switch back to formatted value on edit complete
      }
      this.updateWorksetReadonly(true) // set workset read only if all parameters saved and workset note editor closed
    },

    // save if data editied
    onEditSave () {
      this.doSaveDataPage()
      if (this.edt.kind === Pcvt.EDIT_NUMBER && this.pvc.formatter) {
        if (this.pvc.formatter.options().isRawValue !== this.ctrl.isRawShow) this.pvc.formatter.doRawValue() // switch back to formatted value on edit complete
      }
    },

    // undo and redo last edit changes: forward actions to pivot table component
    onUndo () { this.$refs.omPivotTable.doUndo() },
    onRedo () { this.$refs.omPivotTable.doRedo() },

    // on edit events: input confirm (data entered), undo, redo, paste
    onPvEdit () {
      this.dispatchParamView({ key: this.routeKey, edit: this.edt })
    },
    //
    // end of editor methods

    // dimensions drag, drop and selection filter
    //
    onDrag () {
      // drag started
      this.isDragging = true
    },
    onDrop (e) {
      // drag completed: drop
      this.isDragging = false

      // make sure at least one item selected in each dimension
      // other dimensions: use single-select dropdown
      for (const f of this.dimProp) {
        if (f.selection.length < 1) {
          f.selection.push(f.enums[0])
        }
      }
      for (const f of this.otherFields) {
        if (f.selection.length > 1) {
          f.selection.splice(1)
        }
      }
      for (const f of this.dimProp) {
        f.singleSelection = f.selection[0]
      }

      // update pivot view:
      //  if other dimesion(s) filters same as before
      //  then update pivot table view now
      //  else refresh data
      if (Puih.equalFilterState(this.filterState, this.otherFields)) {
        this.ctrl.isPvTickle = !this.ctrl.isPvTickle
      } else {
        this.doRefreshDataPage()
      }
      // update pivot view rows, columns, other dimensions
      this.dispatchParamView({
        key: this.routeKey,
        rows: Pcvt.pivotStateFields(this.rowFields),
        cols: Pcvt.pivotStateFields(this.colFields),
        others: Pcvt.pivotStateFields(this.otherFields)
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
        f.selection.splice(0, f.selection.length)
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
      if (panel !== 'other' || Puih.equalFilterState(this.filterState, this.otherFields)) {
        this.ctrl.isPvTickle = !this.ctrl.isPvTickle
      } else {
        this.doRefreshDataPage()
      }
      // update pivot view rows, columns, other dimensions
      this.dispatchParamView({
        key: this.routeKey,
        rows: Pcvt.pivotStateFields(this.rowFields),
        cols: Pcvt.pivotStateFields(this.colFields),
        others: Pcvt.pivotStateFields(this.otherFields)
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
      if (!isOther || Puih.equalFilterState(this.filterState, this.otherFields)) {
        this.ctrl.isPvTickle = !this.ctrl.isPvTickle
      } else {
        this.doRefreshDataPage()
      }
      // update pivot view rows, columns, other dimensions
      this.dispatchParamView({
        key: this.routeKey,
        rows: Pcvt.pivotStateFields(this.rowFields),
        cols: Pcvt.pivotStateFields(this.colFields),
        others: Pcvt.pivotStateFields(this.otherFields)
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
        f.selection.splice(0, f.selection.length)
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

      // update pivot view rows, columns, other dimensions
      this.dispatchParamView({
        key: this.routeKey,
        rows: Pcvt.pivotStateFields(this.rowFields),
        cols: Pcvt.pivotStateFields(this.colFields),
        others: Pcvt.pivotStateFields(this.otherFields)
      })
    },

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
    //
    // end of dimensions drag, drop and selection filter

    doRefresh () {
      this.initViewRefreshData()
      if (!this.isFromRun) {
        this.$emit('edit-updated', this.edt.isUpdated, this.routeKey)
      }
    },
    async initViewRefreshData () {
      await this.initView()
      this.doRefreshDataPage()
    },

    // initialize current page view on mounted or tab switch
    async initView () {
      // check if parameter exist in model run or in workset
      const { isFound, src } = this.initParamRunSet()
      if (!isFound) {
        return // exit on error
      }

      // find parameter, parameter type and size, including run sub-values count
      this.paramText = Mdf.paramTextByName(this.theModel, this.parameterName)
      this.paramType = Mdf.typeTextById(this.theModel, (this.paramText.Param.TypeId || 0))
      const paramSize = Mdf.paramSizeByName(this.theModel, this.parameterName)
      this.rank = paramSize?.rank || 0

      this.isPages = paramSize?.dimTotal > SMALL_PAGE_SIZE // disable pages for small table
      this.pageStart = 0
      this.pageSize = 0 // by default show all rows
      this.isShowPageControls = this.pageSize > 0

      this.isNullable = this.paramText.Param?.IsExtendable || false
      this.subCount = this.paramRunSet.SubCount || 0
      this.isScalar = this.rank <= 0 && this.subCount <= 1

      // adjust controls
      this.edt.isEnabled = this.isUploadEnabled = !this.isFromRun && Mdf.isNotEmptyWorksetText(src) // && !src.IsReadonly
      Pcvt.resetEdit(this.edt) // clear editor state

      const isRc = !this.isScalar
      this.pvc.rowColMode = isRc ? Pcvt.SPANS_AND_DIMS_PVT : Pcvt.NO_SPANS_NO_DIMS_PVT
      this.ctrl.isRowColModeToggle = isRc
      this.ctrl.isRowColControls = isRc
      this.pvKeyPos = []

      // make dimensions:
      //  [rank] of enum-based dimensions
      //  sub-value id dimension, if parameter has sub-values
      this.dimProp = []

      for (let n = 0; n < this.paramText.ParamDimsTxt.length; n++) {
        const dt = this.paramText.ParamDimsTxt[n]
        const t = Mdf.typeTextById(this.theModel, (dt.Dim.TypeId || 0))
        const f = {
          name: dt.Dim.Name || '',
          label: Mdf.descrOfDescrNote(dt) || dt.Dim.Name || '',
          enums: [],
          options: [],
          selection: [],
          singleSelection: {},
          filter: (val, update, abort) => {}
        }
        const eLst = Array(Mdf.typeEnumSize(t))
        for (let j = 0; j < eLst.length; j++) {
          eLst[j] = Mdf.enumItemByIdx(t, j)
        }
        f.enums = Object.freeze(eLst)
        f.options = f.enums
        f.filter = Puih.makeFilter(f)

        this.dimProp.push(f)
      }

      // if parameter has sub-values then add sub-value id dimension
      if (this.subCount > 1) {
        const f = {
          name: Puih.SUB_ID_DIM,
          label: this.$t('Sub #'),
          enums: [],
          options: [],
          selection: [],
          singleSelection: {},
          filter: (val, update, abort) => {}
        }

        const eLst = Array(this.subCount)
        for (let k = 0; k < this.subCount; k++) {
          eLst[k] = { value: k, name: k.toString(), label: k.toString() }
        }
        f.enums = Object.freeze(eLst)
        f.options = f.enums
        f.filter = Puih.makeFilter(f)

        this.dimProp.push(f)
      }

      // read parameter rows: one row for each parameter value
      this.pvc.reader = {
        isScale: false, // value scaling is not defined for parameters

        rowReader: (src) => {
          // no data to read: if source rows are empty or invalid return undefined reader
          if (!src || (src?.length || 0) <= 0) return void 0

          const srcLen = src.length
          let nSrc = 0

          const rd = { // reader to return
            readRow: () => {
              return (nSrc < srcLen) ? src[nSrc++] : void 0 // parameter row: one row for each parameter value
            },
            readDim: {},
            readValue: (r) => (!r.IsNull ? r.Value : void 0) // parameter value
          }

          // read dimension item value: enum id, sub-value id, parameter id
          for (let n = 0; n < this.rank; n++) {
            rd.readDim[this.dimProp[n].name] = (r) => (r.DimIds.length > n ? r.DimIds[n] : void 0)
          }
          rd.readDim[Puih.SUB_ID_DIM] = (r) => (r.SubId) // read parameter sub-value id

          return rd
        }
      }

      // setup process value and format value handlers:
      //  if parameter type is one of built-in then process and format value as float, int, boolen or string
      //  else parameter type is enum-based: process and format value as int enum id
      let lc = this.uiLang || this.$q.lang.getLocale() || ''
      if (lc) {
        try {
          const cla = Intl.getCanonicalLocales(lc)
          lc = cla?.[0] || ''
        } catch (e) {
          lc = ''
          console.warn('Error: undefined canonical locale:', e)
        }
      }
      this.pvc.processValue = Pcvt.asIsPval
      this.pvc.formatter = Pcvt.formatDefault({ isNullable: this.isNullable, locale: lc })
      this.pvc.cellClass = 'pv-cell-right' // numeric cell value style by default
      this.edt.kind = Pcvt.EDIT_NUMBER

      if (Mdf.isBuiltIn(this.paramType.Type)) {
        if (Mdf.isFloat(this.paramType.Type)) {
          this.pvc.processValue = Pcvt.asFloatPval
          this.pvc.formatter = Pcvt.formatFloat({ isNullable: this.isNullable, locale: lc, isAllDecimal: true }) // show all deciamls for the float value
        }
        if (Mdf.isInt(this.paramType.Type)) {
          this.pvc.processValue = Pcvt.asIntPval
          this.pvc.formatter = Pcvt.formatInt({ isNullable: this.isNullable, locale: lc })
        }
        if (Mdf.isBool(this.paramType.Type)) {
          this.pvc.processValue = Pcvt.asBoolPval
          this.pvc.cellClass = 'pv-cell-center'
          this.pvc.formatter = Pcvt.formatBool({})
          this.edt.kind = Pcvt.EDIT_BOOL
        }
        if (Mdf.isString(this.paramType.Type)) {
          this.pvc.cellClass = 'pv-cell-left' // no process or format value required for string type
          this.edt.kind = Pcvt.EDIT_STRING
        }
      } else {
        // if parameter is enum-based then value is integer enum id and format(value) should return enum description to display
        const t = this.paramType
        const valEnums = Array(Mdf.typeEnumSize(t))
        for (let j = 0; j < valEnums.length; j++) {
          valEnums[j] = Mdf.enumItemByIdx(t, j)
        }
        this.pvc.processValue = Pcvt.asIntPval
        this.pvc.formatter = Pcvt.formatEnum({ enums: valEnums })
        this.pvc.cellClass = 'pv-cell-left'
        this.edt.kind = Pcvt.EDIT_ENUM
      }

      // format view controls
      this.ctrl.isRawUseView = this.pvc.formatter.options().isRawUse
      this.ctrl.isRawView = this.pvc.formatter.options().isRawValue
      this.ctrl.isFloatView = this.pvc.formatter.options().isFloat
      this.ctrl.isMoreView = this.pvc.formatter.options().isDoMore
      this.ctrl.isLessView = this.pvc.formatter.options().isDoLess

      // set columns layout and refresh the data
      await this.setPageView()
    },

    // set page view: use previous page view from store or default
    async setPageView () {
      // if previous page view exist in session store
      let pv = this.paramView(this.routeKey)
      if (!pv) {
        await this.restoreDefaultView() // restore default parameter view, if exist
        pv = this.paramView(this.routeKey) // check if default view of parameter restored
        if (!pv) {
          this.setInitialPageView() // setup and use initial view of parameter
          return
        }
      }
      // else: restore previous view

      // restore rows, columns, others layout and items selection
      const restore = (pvSrc, dst) => {
        dst.splice(0, dst.length)
        for (const p of pvSrc) {
          const f = this.dimProp.find((d) => d.name === p.name)
          if (!f) continue

          f.selection.splice(0, f.selection.length)
          for (const v of p.values) {
            const e = f.enums.find((fe) => fe.value === v)
            if (e) {
              f.selection.push(e)
            }
          }
          f.singleSelection = (f.selection.length > 0) ? f.selection[0] : {}

          dst.push(f)
        }
      }
      restore(pv.rows, this.rowFields)
      restore(pv.cols, this.colFields)
      restore(pv.others, this.otherFields)

      // if there are any dimensions which are not in rows, columns or others then push it to others
      // it is possible if view restored and sub-value dimension is added by parameter upload
      for (const f of this.dimProp) {
        if (this.rowFields.findIndex((p) => f.name === p.name) >= 0) continue
        if (this.colFields.findIndex((p) => f.name === p.name) >= 0) continue
        if (this.otherFields.findIndex((p) => f.name === p.name) >= 0) continue

        // append to other fields
        f.selection.splice(0, f.selection.length)
        f.selection.push(f.enums[0])
        f.singleSelection = (f.selection.length > 0) ? f.selection[0] : {}
        this.otherFields.push(f)
      }

      // restore edit state and controls state
      if (this.edt.isEnabled) {
        this.edt = pv.edit
        this.edt.isEnabled = true
      }

      this.ctrl.isRowColControls = !!pv.isRowColControls
      this.pvc.rowColMode = typeof pv.rowColMode === typeof 1 ? pv.rowColMode : Pcvt.NO_SPANS_NO_DIMS_PVT

      // restore page offset and size
      if (this.isPages) {
        this.pageStart = (typeof pv?.pageStart === typeof 1) ? (pv?.pageStart || 0) : 0
        this.pageSize = (typeof pv?.pageSize === typeof 1 && pv?.pageSize >= 0) ? pv.pageSize : SMALL_PAGE_SIZE
      } else {
        this.pageStart = 0
        this.pageSize = 0
      }
      this.isShowPageControls = this.pageSize > 0

      // refresh pivot view: both dimensions labels and table body
      this.ctrl.isPvDimsTickle = !this.ctrl.isPvDimsTickle
      this.ctrl.isPvTickle = !this.ctrl.isPvTickle
    },

    // set initial page view for parameter
    setInitialPageView () {
      // set rows, columns and other:
      //   last-1 dimension on rows
      //   last dimension on columns
      //   the rest on other fields
      this.rowFields.splice(0, this.rowFields.length)
      this.colFields.splice(0, this.colFields.length)
      this.otherFields.splice(0, this.otherFields.length)

      if (this.dimProp.length === 1) this.rowFields.push(this.dimProp[0])
      if (this.dimProp.length > 1) {
        this.rowFields.push(this.dimProp[this.dimProp.length - 2])
        this.colFields.push(this.dimProp[this.dimProp.length - 1])
      }

      for (let k = 0; k < this.dimProp.length; k++) {
        const f = this.dimProp[k]
        f.selection.splice(0, f.selection.length)

        // if other then single selection else rows and columns: multiple selection
        if (k < this.dimProp.length - 2) {
          this.otherFields.push(f)
          f.selection.push(f.enums[0])
        } else {
          f.selection.push(...f.enums)
        }
        f.singleSelection = (f.selection.length > 0) ? f.selection[0] : {}
      }

      // default row-column mode: no row-column headers for scalar parameters without sub-values
      this.pvc.rowColMode = !this.isScalar ? Pcvt.SPANS_AND_DIMS_PVT : Pcvt.NO_SPANS_NO_DIMS_PVT

      Pcvt.resetEdit(this.edt) // clear editor state

      // store pivot view
      const vs = Pcvt.pivotStateFromFields(this.rowFields, this.colFields, this.otherFields, this.ctrl.isRowColControls, this.pvc.rowColMode)
      vs.edit = this.edt // edit state exist only for parameters
      vs.pageStart = 0
      vs.pageSize = this.isPages ? ((typeof this.pageSize === typeof 1 && this.pageSize >= 0) ? this.pageSize : SMALL_PAGE_SIZE) : 0

      this.dispatchParamView({
        key: this.routeKey,
        view: vs,
        digest: this.digest || '',
        modelName: Mdf.modelName(this.theModel),
        runDigest: this.runDigest || '',
        worksetName: this.worksetName || '',
        parameterName: this.parameterName || ''
      })

      // refresh pivot view: both dimensions labels and table body
      this.ctrl.isPvDimsTickle = !this.ctrl.isPvDimsTickle
      this.ctrl.isPvTickle = !this.ctrl.isPvTickle
    },

    // find model run or input workset and check if parameter exist in model or workset
    initParamRunSet () {
      if (!this.digest) {
        console.warn('Invalid (empty) model digest')
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) model digest') })
        return { isFound: false, src: Mdf.emptyWorksetText() }
      }
      if (!this.runDigest && !this.worksetName) {
        console.warn('Unable to show parameter: scenario name and run digest are empty')
        this.$q.notify({ type: 'negative', message: this.$t('Unable to show parameter: scenario name and run digest are empty') })
        return { isFound: false, src: Mdf.emptyWorksetText() }
      }

      if (this.isFromRun) {
        const runSrc = this.runTextByDigest({ ModelDigest: this.digest, RunDigest: this.runDigest })
        if (!Mdf.isNotEmptyRunText(runSrc)) {
          console.warn('Model run not found:', this.digest, this.runDigest)
          this.$q.notify({ type: 'negative', message: this.$t('Model run not found: ') + this.runDigest })
          return { isFound: false, src: runSrc }
        }

        this.paramRunSet = Mdf.paramRunSetByName(runSrc, this.parameterName)
        if (!Mdf.isNotEmptyParamRunSet(this.paramRunSet)) {
          console.warn('Parameter not found in model run:', this.parameterName, this.runDigest)
          this.$q.notify({ type: 'negative', message: this.$t('Parameter not found in model run: ') + this.parameterName + ' ' + this.runDigest })
          return { isFound: false, src: runSrc }
        }
        return { isFound: true, src: runSrc }
      }
      // else: find workset and parameter in workset
      const wsSrc = this.worksetTextByName({ ModelDigest: this.digest, Name: this.worksetName })
      if (!Mdf.isNotEmptyWorksetText(wsSrc)) {
        console.warn('Input scenario not found:', this.digest, this.worksetName)
        this.$q.notify({ type: 'negative', message: this.$t('Input scenario not found: ') + this.worksetName })
        return { isFound: false, src: wsSrc }
      }

      this.paramRunSet = Mdf.paramRunSetByName(wsSrc, this.parameterName)
      if (!Mdf.isNotEmptyParamRunSet(this.paramRunSet)) {
        console.warn('Parameter not found in scenario:', this.parameterName, this.worksetName)
        this.$q.notify({ type: 'negative', message: this.$t('Parameter not found in scenario: ') + this.parameterName + ' ' + this.worksetName })
        return { isFound: false, src: wsSrc }
      }
      return { isFound: true, src: wsSrc }
    },

    // edit parameter value notes
    onEditParamNote () {
      this.noteEditorNotes = Mdf.noteOfTxt(this.paramRunSet)
      this.noteEditorLangCode = this.uiLang || this.$q.lang.getLocale() || ''
      this.noteEditorShow = true
      this.updateWorksetReadonly(false) // set workset read-write
    },
    // ask user to confirm cancel if notes changed
    // return true if no changes in parameter notes
    onCancelParamNote () {
      if (!this.noteEditorShow) return true // note editor not active: allow to leave parameter tab

      const udn = this.$refs['param-note-editor'].getDescrNote()
      if (udn.isUpdated) { // redirect to dialog to confirm "discard changes?"
        this.showDiscardParamNoteTickle = !this.showDiscardParamNoteTickle
        return false // notes updated and not saved: disable route to other page or tab
      }
      // else:
      // close notes editor (no changes in data)
      this.noteEditorShow = false
      this.updateWorksetReadonly(true) // set workset read only if all parameters saved and workset note editor closed
      return true // no changes: allow to leave parameter tab
    },
    // on user selecting "Yes" from "discard changes?" pop-up alert
    onYesDiscardParamNote () {
      this.noteEditorShow = false
      this.updateWorksetReadonly(true) // set workset read only if all parameters saved and workset note editor closed
      this.isYesToLeave = (this.pathToRouteLeave || '') !== ''
      if (this.isYesToLeave) {
        this.$router.push(this.pathToRouteLeave)
      }
    },
    // save parameter value notes
    onSaveParamNote () {
      const udn = this.$refs['param-note-editor'].getDescrNote()
      if (this.isFromRun) {
        this.doSaveRunParameterNote(this.noteEditorLangCode, udn.note)
      } else {
        this.doSaveSetParameterNote(this.noteEditorLangCode, udn.note)
      }
      this.noteEditorShow = false
    },

    // get page of parameter data from current model run or workset
    async doRefreshDataPage (isFullPage = false) {
      const r = this.initParamRunSet()
      if (!r.isFound) {
        return // exit on error
      }

      this.loadDone = false
      this.loadWait = true

      // save filters: other dimensions selected items
      this.filterState = Puih.makeFilterState(this.otherFields)

      // make parameter read layout and url
      const layout = Puih.makeSelectLayout(this.parameterName, this.otherFields, Puih.SUB_ID_DIM)

      // if sub_id on other dimensions then add filter by sub-value id
      const fSub = this.filterState?.[Puih.SUB_ID_DIM]
      if (fSub) {
        layout.IsSubId = Array.isArray(fSub) && fSub.length > 0
        if (layout.IsSubId) layout.SubId = fSub[0]
      }

      layout.Offset = 0
      layout.Size = 0
      layout.IsFullPage = false
      if (this.isPages) {
        layout.Offset = this.pageStart || 0
        layout.Size = (!!this.pageSize && typeof this.pageSize === typeof 1) ? (this.pageSize || 0) : 0
        layout.IsFullPage = !!isFullPage
      }
      const udgst = encodeURIComponent(this.digest)

      const u = this.isFromRun
        ? this.omsUrl + '/api/model/' + udgst + '/run/' + encodeURIComponent(this.runDigest) + '/parameter/value-id'
        : this.omsUrl + '/api/model/' + udgst + '/workset/' + encodeURIComponent(this.worksetName) + '/parameter/value-id'

      // retrieve page from server, it must be: {Layout: {...}, Page: [...]}
      let isOk = false
      try {
        const response = await this.$axios.post(u, layout)
        const rsp = response.data

        if (!Puih.isPageLayoutRsp(rsp)) {
          console.warn('Invalid response to:', u)
          const em = Puih.errorFromPageLayoutRsp(rsp)
          this.$q.notify({ type: 'negative', message: (((em || '') !== '') ? em : this.$t('Server offline or parameter data not found: ') + this.parameterName) })
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

          // update pivot table view
          this.inpData = Object.freeze(d)
          this.loadDone = true
          this.ctrl.isPvTickle = !this.ctrl.isPvTickle
          isOk = true
        }
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or parameter data not found', em)
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or parameter data not found: ') + this.parameterName })
      }

      this.loadWait = false
      if (isOk) {
        this.dispatchParamView({
          key: this.routeKey,
          pageStart: this.isPages ? this.pageStart : 0,
          pageSize: this.isPages ? this.pageSize : 0
        })
      }
    },

    // save page of parameter data into current workset
    async doSaveDataPage () {
      const r = this.initParamRunSet()
      if (!r.isFound) {
        return // exit on error
      }
      if (this.isFromRun) {
        console.warn('Attempt to save data into run parameter', this.parameterName, this.runDigest, this.worksetName)
        return // exit on error
      }

      this.saveStarted = true
      this.saveWait = true

      // prepare parameter data for save, exit with error if no changes found
      const pv = Puih.makePageForSave(
        this.dimProp, this.pvKeyPos, this.rank, Puih.SUB_ID_DIM, (this.paramRunSet?.DefaultSubId || 0), this.isNullable, this.edt.updated
      )
      if (!Mdf.lengthOf(pv)) {
        console.warn('No parameter changes, nothing to save:', this.parameterName, this.worksetName)
        this.$q.notify({ type: 'warning', message: this.$t('No parameter changes, nothing to save.') })
        this.saveWait = false
        return
      }

      // url to update parameter data
      const u = this.omsUrl +
        '/api/model/' + encodeURIComponent(this.digest) +
        '/workset/' + encodeURIComponent(this.worksetName) +
        '/parameter/' + encodeURIComponent(this.parameterName) + '/new/value-id'

      // send data page to the server, response body expected to be empty
      try {
        const response = await this.$axios.patch(u, pv)
        const rsp = response.data
        if ((rsp || '') !== '') console.warn('Server reply on save:', rsp)

        // success: clear edit history and refersh data
        this.saveStarted = false
        Pcvt.resetEdit(this.edt)
        this.dispatchParamView({ key: this.routeKey, edit: this.edt })

        this.$nextTick(() => {
          this.doRefreshDataPage()
        })
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or parameter save failed:', em)
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or parameter save failed: ') + this.parameterName })
      }
      this.saveWait = false

      this.updateWorksetReadonly(true) // set workset read only if all parameters saved and workset note editor closed
    },

    // upload csv file to replace workset parameter value
    async onUploadParameter () {
      // validate sub-values count and default sub-value
      const nSub = Mdf.cleanIntNonNegativeInput(this.subCountUpload, 1)
      if (nSub < 1 || nSub > 8192 || !Number.isInteger(nSub)) {
        this.$q.notify({ type: 'negative', message: this.$t('Invalid number of sub-value to upload') + toString(nSub) })
        return
      }
      const nSubDefault = Mdf.cleanIntNonNegativeInput(this.defaultSubUpload, 0)

      // check file name: warning if it is not parameterName.csv
      const csvName = this.parameterName + '.csv'
      const fName = this.uploadFile?.name
      this.$q.notify({ type: (fName === csvName ? 'info' : 'warning'), message: this.$t('Uploading: ') + fName })

      // make upload multipart form
      const u = this.omsUrl + '/api/workset-merge'
      const wt = {
        ModelDigest: this.digest,
        Name: this.worksetName,
        Param: [{
          Name: this.parameterName,
          SubCount: nSub,
          DefaultSubId: nSubDefault
        }]
      }
      const fd = new FormData()
      fd.append('workset', JSON.stringify(wt))
      fd.append('parameter-csv', this.uploadFile, csvName) // file name must be the parameterName.csv

      try {
        // update parameter value, drop response on success
        await this.$axios.patch(u, fd)
      } catch (e) {
        let msg = ''
        try {
          if (e.response) msg = e.response.data || ''
        } finally {}
        console.warn('Unable to update input scenario', msg)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to update input scenario') })
        return
      }

      // notify user and close upload controls
      this.doCancelFileSelect()
      this.$q.notify({ type: 'info', message: this.$t('Uploaded: ') + fName })

      // refresh parameter view on success
      // if sub-values count the same then refresh only data
      // else refresh workset, reset view to default and and refersh the data
      if (nSub === this.subCount) {
        this.doRefreshDataPage()
      } else {
        this.refreshWsTickle = !this.refreshWsTickle // refersh workset
      }
    },

    // save run parameter value notes
    async doSaveRunParameterNote (langCode, note) {
      let isOk = false
      let msg = ''
      const dgst = this.runDigest

      // validate language code: it cannot be empty
      if (!langCode) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to save parameter value notes, language is unknown') })
        return
      }
      this.loadWait = true

      const u = this.omsUrl +
        '/api/model/' + encodeURIComponent(this.digest) +
        '/run/' + encodeURIComponent(dgst) + '/parameter-text'
      const pt = [{
        Name: this.parameterName,
        Txt: [{
          LangCode: langCode,
          Note: note || ''
        }]
      }]
      try {
        // send parameter value notes to the server, ignore response on success
        await this.$axios.patch(u, pt)
        isOk = true
      } catch (e) {
        try {
          if (e.response) msg = e.response.data || ''
        } finally {}
        console.warn('Unable to save run parameter value notes', msg)
      }
      this.loadWait = false
      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to save parameter value notes') + (msg ? ('. ' + msg) : '') })
        return
      }

      this.$q.notify({ type: 'info', message: this.$t('Parameter value notes saved: ') + this.parameterName })
      this.refreshRunTickle = !this.refreshRunTickle
    },

    // save workset parameter value notes
    async doSaveSetParameterNote (langCode, note) {
      let isOk = false
      let msg = ''
      const wsName = this.worksetName

      // validate language code: it cannot be empty
      if (!langCode) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to save parameter value notes, language is unknown') })
        return
      }
      this.loadWait = true

      const u = this.omsUrl +
        '/api/model/' + encodeURIComponent(this.digest) +
        '/workset/' + encodeURIComponent(wsName) + '/parameter-text'
      const pt = [{
        Name: this.parameterName,
        Txt: [{
          LangCode: langCode,
          Note: note || ''
        }]
      }]
      try {
        // send parameter value notes to the server, ignore response on success
        await this.$axios.patch(u, pt)
        isOk = true
      } catch (e) {
        try {
          if (e.response) msg = e.response.data || ''
        } finally {}
        console.warn('Unable to save workset parameter value notes', msg)
      }
      this.loadWait = false
      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to save parameter value notes') + (msg ? ('. ' + msg) : '') })
        return
      }

      this.updateWorksetReadonly(true) // set workset read only if all parameters saved and workset note editor closed

      this.$q.notify({ type: 'info', message: this.$t('Parameter value notes saved: ') + this.parameterName })
      this.refreshWsTickle = !this.refreshWsTickle
    }
  },

  // route leave guard: on leaving parameter page check
  // if parameter notes edited and not saved then ask user to confirm "discard all changes?"
  beforeRouteLeave (to, from) {
    // if this is parameter page (not model tab) editor then ask suer to save or cancle changes
    if (!this.isModelTab && this.edt.isEdit && this.edt.isUpdated) {
        this.showEditDiscardTickle = !this.showEditDiscardTickle
        return false // redirect to dialog to confirm "discard changes?"
    }
    // if user already confimed "yes" to leave the page
    if (this.isYesToLeave) {
      this.isYesToLeave = false
      return true // leave parameter page and route to next page
    }
    // else:
    //  if there edited and unsaved parameter notes
    this.isYesToLeave = this.onCancelParamNote()
    if (this.isYesToLeave) {
      this.isYesToLeave = false
      return true // no unsaved changes: leave parameter page and route to next page
    }
    // else:
    //  redirect to dialog to confirm "discard all changes?"
    this.pathToRouteLeave = to.path || '' // store path-to leave if user confirm "yes" to "discard changes?"
    return false
  },

  mounted () {
    this.initViewRefreshData()
    if (this.isFromRun) {
      this.$emit('tab-mounted', 'run-parameter', { digest: this.digest, runDigest: this.runDigest, parameterName: this.parameterName })
    } else {
      this.$emit('tab-mounted', 'set-parameter', { digest: this.digest, worksetName: this.worksetName, parameterName: this.parameterName })
      this.$nextTick(() => {
        this.$emit('edit-updated', this.edt.isUpdated, this.routeKey)
      })
    }
    if (!this.isModelTab) document.title = this.parameterName
  }
}
