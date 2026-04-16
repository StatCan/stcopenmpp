<template>
<q-dialog v-model="showDlg" seamless position="bottom" full-width>
  <q-card>

    <q-bar class="bg-primary text-white">
      <q-btn
        @click="onApply()"
        :disable="!isEdited || !calcList.length || isAnyEmpty()"
        icon="mdi-check-bold"
        :label="$t('Apply')"
        outline rounded v-close-popup
        />
      <q-space />
      <q-btn icon="mdi-close" :label="$t('Cancel')" outline rounded v-close-popup />
    </q-bar>

    <q-card-section class="text-body1">
      <table class="om-p-table full-width">
          <thead>
            <tr>
              <th class="om-p-head-center text-weight-medium"></th>
              <th class="om-p-head-center text-weight-medium">{{ $t('Aggregation') }}</th>
              <th class="om-p-head-center text-weight-medium">{{ $t('Name') }}</th>
              <th class="om-p-head-center text-weight-medium">{{ $t('Id') }}</th>
            </tr>
          </thead>
        <tbody>
          <template v-for="(c, idx) in calcList"  :key="'cln-' + (c.calcId.toString() || idx.toString())">
            <tr>
              <td rowspan="2" class="om-p-cell-center">
                <q-btn
                  @click="onDelete(idx)"
                  unelevated
                  round
                  dense
                  class="col-auto text-primary"
                  icon="mdi-delete"
                  :title="$t('Delete')"
                  :aria-label="$t('Delete')" />
              </td>
              <td class="om-p-cell mono">
                <q-input
                  v-model="c.calc"
                  maxlength="255"
                  size="80"
                  required
                  @blur="onCalcBlur(idx)"
                  :rules="[ val => (val || '') !== '' ]"
                  dense
                  outlined
                  hide-bottom-space
                  :placeholder="$t('Aggregation expression') + ' (' + $t('Required') + ')'"
                  :title="$t('Aggregation expression')"
                  >
                </q-input>
              </td>
              <td class="om-p-cell mono">
                <q-input
                  v-model="c.name"
                  maxlength="32"
                  size="32"
                  required
                  @blur="onNameBlur(idx)"
                  :rules="[ val => (val || '') !== '' ]"
                  dense
                  outlined
                  hide-bottom-space
                  :placeholder="$t('Unique name') + ' (' + $t('Required') + ')'"
                  :title="$t('Unique name')"
                  >
                </q-input>
              </td>
              <td class="om-p-cell mono">{{ !!c.calcId ? c.calcId.toString() : '' }}</td>
            </tr>
            <tr>
              <td colspan="3" class="om-p-cell">
                <q-input
                  v-model="c.label"
                  maxlength="255"
                  size="80"
                  @blur="onDescrBlur"
                  dense
                  outlined
                  clearable
                  hide-bottom-space
                  :placeholder="$t('Description') + ' (' + $t('label') + ')'"
                  :title="$t('Description') + ' (' + $t('label') + ')'"
                  >
                </q-input>
              </td>
            </tr>
          </template>
          <tr>
            <td class="om-p-cell-center">
              <q-btn
                @click="onAppend()"
                :disable="isAnyEmpty()"
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
        </tbody>
      </table>
    </q-card-section>

  </q-card>
</q-dialog>
</template>

<script>
import * as Mdf from 'src/model-common'

export default {
  name: 'EntityCalcDialog',

  props: {
    showTickle: { type: Boolean, default: false, required: true },
    updateTickle: { type: Boolean, default: false },
    calcEnums: { type: Array, default: () => [] }
  },

  data () {
    return {
      showDlg: false,
      isEdited: false,
      nextId: 1,
      calcList: []
    }
  },

  watch: {
    updateTickle () {
      this.updateCalc()
    },
    showTickle () {
      this.updateCalc()
      this.showDlg = true
    }
  },

  emits: ['calc-list-apply'],

  methods: {
    // check if any calculation is empty
    isAnyEmpty () { return this.calcList.findIndex(c => (c?.calc || '').trim() === '') >= 0 },

    // check if there are any different values as resut of editing
    isAnyDiff () {
      if (this.calcList.length !== this.calcEnums.length) return true

      for (let k = 0; k < this.calcEnums.length; k++) {
        const ce = this.calcEnums[k]
        if (ce?.value !== this.calcList[k]?.calcId) return true
        if (ce?.name !== this.calcList[k]?.name) return true
        if (ce?.calc !== this.calcList[k]?.calc) return true
        if (ce?.label !== this.calcList[k]?.label) return true
      }
    },

    // copy calcEnums into calculation list for editing
    updateCalc () {
      this.isEdited = false
      this.calcList = []

      for (let k = 0; k < this.calcEnums.length; k++) {
        const ce = this.calcEnums[k]
        if (!(ce?.calc || '').trim()) continue

        this.calcList.push({
          calcId: this.calcEnums[k].value,
          name: this.calcEnums[k].name,
          calc: ce.calc.trim(),
          label: ce.label || this.calcEnums[k].name
        })
      }

      for (let k = 0; k < this.calcList.length; k++) {
        this.adjustLineNameCalcId(k)
      }
      this.isEdited = this.isAnyDiff()
    },

    // send updated version of calculted enums
    onApply () {
      this.$emit('calc-list-apply', this.calcList)
    },

    // delete row from calculations list
    onDelete (lineIdx) {
      if (lineIdx < 0 || lineIdx >= this.calcList.length) return
      this.calcList.splice(lineIdx, 1)

      this.adjustLineNameCalcId(lineIdx)
      this.isEdited = this.isAnyDiff()
    },

    // append extra calculation row
    onAppend () {
      if (this.isAnyEmpty()) return

      let nId = this.calcList.length + Mdf.CALCULATED_ID_OFFSET
      while (this.calcList.findIndex(e => e.calcId === nId) >= 0) {
        nId++
      }
      const nm = 'ex_' + nId.toString()

      this.calcList.push({
        calcId: nId,
        name: nm,
        calc: '',
        label: nm
      })

      for (let k = 0; k < this.calcList.length; k++) {
        this.adjustLineNameCalcId(k)
      }
      this.isEdited = this.isAnyDiff()
    },

    // calcultion blur: trim and check if any calculation is empty
    onCalcBlur (lineIdx) {
      for (let k = 0; k < this.calcList.length; k++) {
        if ((this.calcList[k]?.calc || '') === '' || typeof (this.calcList[k]?.calc || '') !== typeof 'string') {
          this.calcList[k].calc = ''
        } else {
          this.calcList[k].calc = this.calcList[k].calc.trim()
        }
      }
      this.adjustLineNameCalcId(lineIdx)
      this.isEdited = this.isAnyDiff()
    },

    // expression name blur: cleanup name, truncate and make it unique
    onNameBlur (lineIdx) {
      this.adjustLineNameCalcId(lineIdx)
      this.isEdited = this.isAnyDiff()
    },

    // make unique name and calcId
    adjustLineNameCalcId (lineIdx) {
      if (typeof lineIdx !== typeof 1 || lineIdx < 0 || lineIdx >= this.calcList.length) return // index out of range

      const c = this.calcList[lineIdx]
      if ((c.calc || '') === '') return // skip empty calcutions

      // calcId must be unique
      let nId = Mdf.CALCULATED_ID_OFFSET

      if (c.calcId < 0 || this.calcList.findIndex((e, idx) => { return idx !== lineIdx && e.calcId === c.calcId }) >= 0) {
        while (this.calcList.findIndex(e => e.calcId === nId) >= 0) {
          nId++
        }
        c.calcId = nId
      }

      // cleanup name and make it unique
      c.name = Mdf.cleanColumnValueInput(c.name)

      if ((c.name || '') === '' || this.calcList.findIndex((e, idx) => { return idx !== lineIdx && e.name === c.name }) >= 0) {
        c.name = ((typeof c.name === typeof 'string' && (c.name || '') !== '') ? (c.name + '_') : '') + 'ex_' + c.calcId.toString()
        if (c.name.length > 32) {
          c.name = c.name.substring(c.name.length - 32)
        }
      }
    },

    // description label blur: trim labels
    onDescrBlur (e) {
      for (let k = 0; k < this.calcList.length; k++) {
        if ((this.calcList[k]?.label || '') === '' || typeof (this.calcList[k]?.label || '') !== typeof 'string') {
          this.calcList[k].label = ''
        } else {
          this.calcList[k].label = this.calcList[k].label.trim()
        }
      }
      this.isEdited = this.isAnyDiff()
    }
  }
}
</script>
