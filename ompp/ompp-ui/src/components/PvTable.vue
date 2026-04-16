<!-- pivot table view -->
<template>

  <table
    class="pv-main-table"
    v-on="!isEditInput ? {'copy': onCopyTsv} : {}">

    <thead v-if="pvControl.rowColMode === 2"><!-- use spans and show dim names -->

      <tr v-for="(cf, nFld) in colFields" :key="cf.name">
        <th
          v-if="nFld === 0 && !!rowFields.length"
          :colspan="rowFields.length"
          :rowspan="colFields.length"
          class="pv-rc-pad"></th>
        <th
          class="pv-rc-cell">{{!pvControl.isShowNames ? cf.label : cf.name}}</th>
        <template v-for="(col, nCol) in pvt.cols">
          <th
            :key="pvt.colKeys[nCol]"
            v-if="!!pvt.colSpans[nCol * colFields.length + nFld]"
            :colspan="pvt.colSpans[nCol * colFields.length + nFld]"
            :rowspan="(!!rowFields.length && nFld === colFields.length - 1) ? 2 : 1"
            class="pv-col-head">
              {{getDimItemLabel(cf.name, col[nFld])}}
          </th>
        </template>
      </tr>
      <tr v-if="!!rowFields.length">
        <th v-for="rf in rowFields"
          :key="rf.name"
          class="pv-rc-cell">
            {{!pvControl.isShowNames ? rf.label : rf.name}}
        </th>
        <th class="pv-rc-pad"></th>
      </tr>

    </thead>
    <thead v-else-if="pvControl.rowColMode === 3"><!-- no spans and show dim names -->

      <tr v-for="(cf, nFld) in colFields" :key="cf.name">
        <th
          v-if="nFld === 0 && !!rowFields.length"
          :colspan="rowFields.length"
          :rowspan="colFields.length"
          class="pv-rc-pad"></th>
        <th
          class="pv-rc-cell">{{!pvControl.isShowNames ? cf.label : cf.name}}</th>
        <template v-for="(col, nCol) in pvt.cols" :key="pvt.colKeys[nCol]">
          <th
            :rowspan="(!!rowFields.length && nFld === colFields.length - 1) ? 2 : 1"
            class="pv-col-head">
              {{getDimItemLabel(cf.name, col[nFld])}}
          </th>
        </template>
      </tr>
      <tr v-if="!!rowFields.length">
        <th v-for="rf in rowFields"
          :key="rf.name"
          class="pv-rc-cell">
            {{!pvControl.isShowNames ? rf.label : rf.name}}
        </th>
        <th class="pv-rc-pad"></th>
      </tr>

    </thead>
    <thead v-else-if="pvControl.rowColMode === 1"><!-- use spans and hide dim names -->

      <tr v-for="(cf, nFld) in colFields" :key="cf.name">
        <th
          v-if="nFld === 0 && !!rowFields.length"
          :colspan="rowFields.length"
          :rowspan="colFields.length"
          class="pv-rc-pad"></th>
        <template v-for="(col, nCol) in pvt.cols">
          <th
            :key="pvt.colKeys[nCol]"
            v-if="!!pvt.colSpans[nCol * colFields.length + nFld]"
            :colspan="pvt.colSpans[nCol * colFields.length + nFld]"
            class="pv-col-head">
              {{getDimItemLabel(cf.name, col[nFld])}}
          </th>
        </template>
      </tr>

    </thead>
    <thead v-else><!-- no spans and hide dim names -->

      <tr v-for="(cf, nFld) in colFields" :key="cf.name">
        <th
          v-if="nFld === 0 && !!rowFields.length"
          :colspan="rowFields.length"
          :rowspan="colFields.length"
          class="pv-rc-pad"></th>
        <template v-for="(col, nCol) in pvt.cols" :key="pvt.colKeys[nCol]">
          <th
            class="pv-col-head">
              {{getDimItemLabel(cf.name, col[nFld])}}
          </th>
        </template>
      </tr>

    </thead>

    <!-- table body: pivot view -->
    <tbody v-if="!pvEdit.isEdit">

      <tr v-for="(row, nRow) in pvt.rows" :key="pvt.rowKeys[nRow]">

        <!-- row headers: row dimension(s) item label -->
        <template v-for="(rf, nFld) in rowFields">
          <th
            :key="pvt.rowKeys[nRow] + ':' + nFld"
            v-if="pvControl.rowColMode === 0 || pvControl.rowColMode === 3 || !!pvt.rowSpans[nRow * rowFields.length + nFld]"
            :rowspan="(pvControl.rowColMode === 2 || pvControl.rowColMode === 1) && pvt.rowSpans[nRow * rowFields.length + nFld]"
            :colspan="((pvControl.rowColMode === 2 || pvControl.rowColMode === 3) && !!colFields.length && nFld === rowFields.length - 1) ? 2 : 1"
            class="pv-row-head">
              {{getDimItemLabel(rf.name, row[nFld])}}
          </th>
        </template>
        <!-- empty row headers if all dimensions on the columns -->
        <th v-if="(pvControl.rowColMode === 2 || pvControl.rowColMode === 3) && !rowFields.length && !!colFields.length" class="pv-rc-pad"></th>

        <!-- table body value cells -->
         <template v-if="isRange">
          <td v-for="(col, nCol) in pvt.cols" :key="pvt.cellKeys[nRow * pvt.colCount + nCol]"
          :class="this.pvControl.cellClass + ' ' + getExtraCellClass(nRow, nCol)">{{getCellValueFmt(nRow, nCol)}}</td>
        </template>
        <template v-else>
          <td v-for="(col, nCol) in pvt.cols" :key="pvt.cellKeys[nRow * pvt.colCount + nCol]"
          :class="pvControl.cellClass">{{getCellValueFmt(nRow, nCol)}}</td>
        </template>

      </tr>

    </tbody>
    <!-- table body: pivot editor -->
    <tbody v-else
      @dblclick="onDblClick($event)"
      v-on="!isEditInput ? {'paste': onPasteTsv, 'keyup': onKeyUpEdit, 'keydown': onKeyDownEdit} : {}"
      >

      <tr v-for="(row, nRow) in pvt.rows" :key="pvt.rowKeys[nRow]">

        <!-- row headers: row dimension(s) item label -->
        <template v-for="(rf, nFld) in rowFields">
          <th
            :key="pvt.rowKeys[nRow] + ':' + nFld"
            v-if="pvControl.rowColMode === 0 || pvControl.rowColMode === 3 || !!pvt.rowSpans[nRow * rowFields.length + nFld]"
            :rowspan="(pvControl.rowColMode === 2 || pvControl.rowColMode === 1) && pvt.rowSpans[nRow * rowFields.length + nFld]"
            :colspan="((pvControl.rowColMode === 2 || pvControl.rowColMode === 3) && !!colFields.length && nFld === rowFields.length - 1) ? 2 : 1"
            class="pv-row-head">
              {{getDimItemLabel(rf.name, row[nFld])}}
          </th>
        </template>
        <!-- empty row headers if all dimensions on the columns -->
        <th v-if="(pvControl.rowColMode === 2 || pvControl.rowColMode === 3) && !rowFields.length && !!colFields.length" class="pv-rc-pad"></th>

        <!-- table body value cells: readonly cells and edit input cell (if edit in progress) -->
        <td v-for="(col, nCol) in pvt.cols"
          :key="getRenderKey(pvt.cellKeys[nRow * pvt.colCount + nCol])"
          :ref="pvt.cellKeys[nRow * pvt.colCount + nCol]"
          :data-om-nrow="nRow"
          :data-om-ncol="nCol"
          :tabindex="(pvt.cellKeys[nRow * pvt.colCount + nCol] !== pvEdit.cellKey) ? 0 : void 0"
          role="button"
          :class="pvControl.cellClass"
          class="pv-cell-view"
          >

          <!-- eidtor: readonly cells -->
          <template v-if="pvt.cellKeys[nRow * pvt.colCount + nCol] !== pvEdit.cellKey">
            {{getUpdatedToDisplay(nRow, nCol)}}
          </template>
          <template v-else>

            <!-- checkbox editor for boolean value -->
            <template v-if="pvEdit.kind === 2">
              <input
                type="checkbox"
                ref="input-cell"
                v-model.lazy="pvEdit.cellValue"
                @keydown.enter.stop="onCellInputEnter(nRow, nCol)"
                @blur="onCellInputBlur"
                @keydown.esc="onCellInputEscape"
                tabindex="0"
                class="mdc-typography--body1" />
            </template>

            <!-- select dropdown editor for enum value -->
            <template v-else-if="pvEdit.kind === 3">
              <select
                ref="input-cell"
                v-model.lazy="pvEdit.cellValue"
                @keydown.enter.stop="onCellInputEnter(nRow, nCol)"
                @blur="onCellInputBlur"
                @keydown.esc="onCellInputEscape"
                tabindex="0"
                class="pv-cell-font mdc-typography--body1">
                  <option v-for="opt in pvControl.formatter.getEnums()" :key="opt.value" :value="opt.value">{{opt.label}}</option>
              </select>
            </template>

            <!-- default editor for float, integer or string value -->
            <template v-else>
              <input
                type="text"
                ref="input-cell"
                v-model="pvEdit.cellValue"
                @keydown.enter.stop="onCellInputEnter(nRow, nCol)"
                @dblclick.stop="onCellInputConfirm"
                @blur="onCellInputBlur"
                @keydown.esc="onCellInputEscape"
                tabindex="0"
                class="pv-cell-input mdc-typography--body1"
                :size="valueLen" />
            </template>
          </template>

        </td>
      </tr>

    </tbody>
  </table>

</template>

<script src="./pv-table.js"></script>

<style lang="scss" scoped>
.pv-main-table {
  text-align: left;
  border-collapse: collapse;
}
.pv-cell-font {
  font-size: 0.875rem;
}
.pv-cell {
  padding: 0.25rem;
  border: 1px solid lightgrey;
  @extend .pv-cell-font;
}

// table header
.pv-hdr {
  font-weight: map-get($text-weights, "medium"); // 500;
  background-color: whitesmoke;
  @extend .pv-cell;
}
.pv-col-head {
  text-align: center;
  @extend .pv-hdr;
}
.pv-row-head {
  text-align: left;
  @extend .pv-hdr;
}
.pv-rc {
  @extend .pv-hdr;
  background-color: $grey-4; // #e0e0e0
  // background-color: #e6e6e6;
}
.pv-rc-pad {
  @extend .pv-rc;
}
.pv-rc-cell {
  text-align: left;
  @extend .pv-rc;
}

// table body cells: readonly view or editor input
.pv-cell-right {
  text-align: right;
  @extend .pv-cell;
}
.pv-cell-left {
  text-align: left;
  @extend .pv-cell;
}
.pv-cell-center {
  text-align: center;
  @extend .pv-cell;
}
.pv-cell-view {
  &:focus {
    background-color: gainsboro;
  }
  @extend .pv-cell-font;
}
.pv-cell-input { // input text
  border: 0;
  padding: 0px 1px;
  background-color: #e6e6e6;
  @extend .pv-cell-font;
}
</style>
