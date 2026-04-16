<template>
<q-dialog v-model="showDlg" seamless position="bottom" full-width>
  <q-card>

    <q-bar class="bg-primary text-white">
      <q-btn
        @click="onApply()"
        :disable="(!filterList.length && !valueFilter.length) || isAnyEmpty()"
        icon="mdi-check-bold"
        :label="$t('Apply')"
        outline rounded
        />
      <q-space />
      <q-btn icon="mdi-close" :label="$t('Cancel')" outline rounded v-close-popup />
    </q-bar>

    <q-card-section class="text-body1">
      <table class="om-p-table full-width">
          <thead>
            <tr>
              <th class="om-p-head-center text-weight-medium"></th>
              <th class="om-p-head-center text-weight-medium">{{ $t('Measure') }}</th>
              <th class="om-p-head-center text-weight-medium flt-col-width">{{ $t('Filter') }}</th>
              <th class="om-p-head-center text-weight-medium">{{ $t('Values') }}</th>
            </tr>
          </thead>
        <tbody>
          <template v-for="(f, idx) in filterList" :key="'vft-' + f.fltId.toString()">
            <tr>
              <td class="om-p-cell-center">
                <q-btn
                  @click="onDelete(idx)"
                  unelevated
                  round
                  dense
                  class="text-primary"
                  icon="mdi-delete"
                  :title="$t('Delete')"
                  :aria-label="$t('Delete')" />
              </td>
              <td class="om-p-cell">
                <q-select
                  v-model="f.name"
                  :options="itemList"
                  option-value="name"
                  :option-label="isShowNames ? 'name' : 'label'"
                  emit-value
                  use-input
                  hide-selected
                  fill-input
                  clearable
                  @filter="onInputMeasureFilter"
                  outlined
                  dense
                  options-dense
                  bottom-slots
                  :title="$t('Select measure to filter')"
                  >
                  <template v-slot:no-option>
                    <q-item><q-item-section class="om-text-descr">{{ $t('Not found') }}</q-item-section></q-item>
                  </template>
                  <template v-slot:hint>
                    <div>{{ measureHint(f?.name) }}</div>
                  </template>
                </q-select>
              </td>
              <td class="om-p-cell">
                <q-select
                  v-model="f.op"
                  :options="opList"
                  option-value="code"
                  :option-label="isShowNames ? 'code' : 'label'"
                  emit-value
                  outlined
                  dense
                  options-dense
                  bottom-slots
                  :title="$t('Select filter condition')"
                  >
                  <template v-slot:hint>
                    <div>{{ opHint(f?.op) }}</div>
                  </template>
                </q-select>
              </td>
              <td class="om-p-cell mono">
                <q-input
                  v-model="f.inp"
                  maxlength="255"
                  size="80"
                  :rules="[ val => (val || '') !== '' ]"
                  dense
                  outlined
                  clearable
                  bottom-slots
                  :placeholder="(f.op !== 'IN' && f.op !== 'BETWEEN') ? $t('Enter filter value') : ((f.op === 'IN') ? $t('Enter list of comma separated values') : $t('Enter min, max values separated by comma'))"
                  :title="(f.op !== 'IN' && f.op !== 'BETWEEN') ? $t('Enter filter value') : ((f.op === 'IN') ? $t('Enter list of comma separated values') : $t('Enter min, max values separated by comma'))"
                  >
                  <template v-slot:hint>
                    <div>{{ (f.op !== 'IN' && f.op !== 'BETWEEN') ? $t('Enter filter value') : ((f.op === 'IN') ? $t('Enter list of comma separated values') : $t('Enter min, max values separated by comma')) }}</div>
                  </template>
                </q-input>
              </td>
            </tr>
          </template>
          <tr>
            <td class="om-p-cell-center">
              <q-btn
                @click="doAppend()"
                unelevated
                round
                dense
                class="text-primary"
                icon="mdi-plus-thick"
                :title="$t('Append')"
                :aria-label="$t('Append')" />
            </td>
            <td colspan="3" class="om-p-head"></td>
          </tr>
          <template v-for="(f, idx) in skipFilter" :key="'sft-' + idx.toString() + '-' + f.name">
            <tr>
              <td class="om-p-cell-center">
                <q-btn
                  @click="onSkipDelete(idx)"
                  unelevated
                  round
                  dense
                  class="text-primary"
                  icon="mdi-delete"
                  :title="$t('Delete')"
                  :aria-label="$t('Delete')" />
              </td>
              <td colspan="3" class="om-p-head-left mono">{{ (f?.label || '') + ' (' + f.name + ') ' + f.op  + ' ' + f.value.join(', ') }}</td>
            </tr>
          </template>
        </tbody>
      </table>
    </q-card-section>

  </q-card>
</q-dialog>
</template>

<script>
import * as Mdf from 'src/model-common'

export default {
  name: 'ValueFilterDialog',

  props: {
    showTickle: { type: Boolean, default: false, required: true },
    measureList: { type: Array, required: true, default: () => [] },
    valueFilter: { type: Array, required: true, default: () => [] },
    isShowNames: { type: Boolean, default: false }
  },

  data () {
    return {
      showDlg: false,
      nextId: 1,
      itemList: [],
      opList: [],
      filterList: [],
      skipFilter: []
    }
  },
  /* filter:
  {
    fltId: 1,
    name: 'Salary',
    label: 'Salary',
    op: '>',
    value: ['12', '34'],
    inp: '12, 34'
  }
  filters to skip:
  {
    fltId: 2,
    name: 'ex_12001',
    label: 'Average Income',
    op: '>',
    value: ['12', '34']
  }
  */

  watch: {
    showTickle () {
      this.initView()
      this.showDlg = true
    },
    updateTickle () { this.updateOpMeasure() },
    isShowNames () { this.updateOpMeasure() }
  },

  emits: ['value-filter-apply'],

  methods: {
    // initial view: open dialog
    initView () {
      this.updateOpMeasure()

      // use current value filters where measure name exists in current list measures
      // skip filters where measure name is not in current list of measures
      this.filterList = []
      this.skipFilter = []

      for (const f of this.valueFilter) {
        if ((f.name || '') === '' || (f.op || '') === '' || !Array.isArray(f.value)) {
          this.$q.notify({ type: 'info', message: this.$t('Remove filter: ') + (f.name || '') + ' ' + (f.op || '') + (((f.name || '') !== '') ? ' ' + '\u2026' : '') })
          continue
        }
        if (this.measureList.findIndex(m => m.name === f.name) >= 0) {
          this.filterList.push({
            fltId: this.nextId++,
            name: f.name,
            label: f?.label || '',
            op: f.op,
            value: Array.isArray(f.value) ? f.value : [],
            inp: Array.isArray(f.value) ? f.value.join(', ') : ''
          })
        } else {
          this.skipFilter.push({
            fltId: this.nextId++,
            name: f.name,
            label: f?.label || '',
            op: f.op,
            value: Array.isArray(f.value) ? f.value : []
          })
        }
      }
      // if filter is empty then add one initial filter with empty value
      if (this.filterList.length <= 0) this.doAppend()
    },

    // check if any value filter is empty
    isAnyEmpty () {
      return this.filterList.findIndex(f => (f.name || '') === '' || (f.op || '') === '' || (f.inp || '') === '') >= 0
    },

    // set select drop downs from measure list and filters condition list
    updateOpMeasure () {
      this.copyMeasureList()

      this.opList = []
      for (const op of Mdf.filterOpList) {
        this.opList.push({
          code: op.code,
          label: this.isShowNames ? op.code : this.$t(op.label)
        })
      }
    },
    // copy initial measure list
    copyMeasureList () {
      this.itemList = []
      for (const m of this.measureList) {
        this.itemList.push({
          name: m.name,
          label: m.label
        })
      }
    },
    // find measure label by name
    measureHint (name) {
      if ((name || '') === '' || typeof name !== typeof 'string') return this.$t('Select measure to filter')

      const n = this.measureList.findIndex(m => m.name === name)
      return n >= 0 ? this.measureList[n].label : this.$t('Select measure to filter')
    },
    opHint (code) {
      if ((code || '') === '' || typeof code !== typeof 'string') return this.$t('Select filter condition')

      const n = Mdf.filterOpList.findIndex(op => op.code === code)
      return n >= 0 ? this.$t(Mdf.filterOpList[n].label) : this.$t('Select filter condition')
    },

    // input measure filter: update list of measure by user input typing
    onInputMeasureFilter (val, doUpdate) {
      if ((val || '') === '' || typeof val !== typeof 'string' || val.trim() === '') {
        doUpdate(() => {
          this.copyMeasureList()
        })
        return // restore initial list of measures
      }
      // else filter by search measure name or label
      doUpdate(() => {
        const vlc = val.trim().toLocaleLowerCase()
        this.itemList = []
        for (const m of this.measureList) {
          if (this.isShowNames) {
            if (this.measureList.findIndex(m => m.name.toLowerCase().indexOf(vlc) >= 0) < 0) continue
          } else {
            if (this.measureList.findIndex(m => m.label.toLocaleLowerCase().indexOf(vlc) >= 0) < 0) continue
          }
          this.itemList.push({
            name: m.name,
            label: m.label
          })
        }
      })
    },

    // validate and send updated version of filters
    onApply () {
      let n = 0
      const fltLst = []

      for (const f of this.filterList) {
        // measure name, condition and value cannot be empty
        if ((f.name || '') === '' || this.measureList.findIndex(m => m.name === f.name) < 0) {
          this.$q.notify({ type: 'negative', message: this.$t('Invalid (or empty) filter measure') + ' [' + n.toString() + '] : ' + (f.name || '') })
          return
        }
        if ((f.op || '') === '' || Mdf.filterOpList.findIndex(op => op.code === f.op) < 0) {
          this.$q.notify({ type: 'negative', message: this.$t('Invalid (or empty) filter condition') + ' [' + n.toString() + '] : ' + (f.op || '') })
          return
        }
        if ((f.inp || '') === '' || (typeof f.inp === typeof 'string' && f.inp.trim() === '')) {
          this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) value entered') + ' [' + n.toString() + ']' })
          return
        }

        // single value expected if it is not in or between filter
        if (f.op !== 'IN' && f.op !== 'BETWEEN') {
          const s = (typeof f.inp !== typeof 'string') ? f.inp.toString() : f.inp
          f.value = [s]
        } else { // split values and check: for between it must be 2 values
          f.value = Mdf.splitCsv(f.inp)
          if (f.value.length <= 0) {
            this.$q.notify({ type: 'negative', message: this.$t('Invalid (or empty) value entered') + ' [' + n.toString() + '] : ' + f.inp })
            return
          }
          if (f.op === 'BETWEEN' && f.value.length !== 2) {
            this.$q.notify({ type: 'negative', message: this.$t('Invalid min, max values entered') + ' [' + n.toString() + '] : ' + f.inp })
            return
          }
        }

        fltLst.push({ // append to result
          name: f.name,
          label: f.label,
          op: f.op,
          value: f.value
        })
        n++
      }

      // append filters from skip list
      for (const f of this.skipFilter) {
        fltLst.push({
          name: f.name,
          label: f.label,
          op: f.op,
          value: f.value
        })
      }

      this.$emit('value-filter-apply', fltLst) // send filters to parent page
      this.showDlg = false
    },

    // delete row from filters list
    onDelete (idx) {
      if (idx < 0 || idx >= this.filterList.length) return
      this.filterList.splice(idx, 1)
    },
    onSkipDelete (idx) {
      if (idx < 0 || idx >= this.skipFilter.length) return
      this.skipFilter.splice(idx, 1)
    },

    // append extra filter row with empty values
    doAppend () {
      this.filterList.push({
        fltId: this.nextId++,
        name: this.measureList.length > 0 ? this.measureList[0].name : '',
        label: this.measureList.length > 0 ? this.measureList[0]?.label : '',
        op: Mdf.filterOpList.length > 0 ? Mdf.filterOpList[0].code : '',
        value: [],
        inp: ''
      })
    }
  }
}
</script>

<style lang="scss" scope="local">
  .flt-col-width {
    min-width: 8rem;
  }
</style>
