<template>
<div class="text-body1">

  <div class="q-pa-sm">
    <q-toolbar class="shadow-1 rounded-borders">

      <run-bar
        :model-digest="digest"
        :run-digest="runDigest"
        @run-info-click="doShowRunNote"
        >
      </run-bar>

    </q-toolbar>
  </div>

  <!-- output table header -->
  <div class="q-mx-sm q-mb-sm">
    <q-toolbar class="row reverse-wrap items-center shadow-1 rounded-borders">

    <!-- menu -->
    <q-btn
      outline
      dense
      class="col-auto text-primary rounded-borders q-mr-xs"
      icon="menu"
      :title="$t('Menu')"
      :aria-label="$t('Menu')"
      >
      <q-menu>
        <q-list dense>

          <q-item
            @click="doShowTableNote"
            clickable
            v-close-popup
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-information-outline" />
            </q-item-section>
            <q-item-section>{{ $t('About') + ' ' + tableName }}</q-item-section>
          </q-item>
          <q-item
            clickable
            v-close-popup
            tag="a"
            target="_blank"
            :href="'/?model=' + encodeURIComponent(digest) + '&run=' + encodeURIComponent(runDigest) + '&table=' + encodeURIComponent(tableName)"
            >
            <q-item-section avatar>
              <q-icon color="primary" name="open_in_new" />
            </q-item-section>
            <q-item-section>{{ $t('Open in new window') }}</q-item-section>
          </q-item>

          <q-separator />

          <q-item
            @click="doExpressionPage"
            :disable="ctrl.kind === 0"
            clickable
            v-close-popup
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-sigma" />
            </q-item-section>
            <q-item-section>{{ $t('View table expressions') }}</q-item-section>
          </q-item>
          <q-item
            @click="doAccumulatorPage"
            :disable="ctrl.kind === 1"
            clickable
            v-close-popup
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-variable" />
            </q-item-section>
            <q-item-section>{{ $t('View accumulators and sub-values (sub-samples)') }}</q-item-section>
          </q-item>
          <q-item
            @click="doAllAccumulatorPage"
            :disable="ctrl.kind === 2"
            clickable
            v-close-popup
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-application-variable-outline" />
            </q-item-section>
            <q-item-section>{{ $t('View all accumulators and sub-values (sub-samples)') }}</q-item-section>
          </q-item>

          <!-- calculated measures menu -->
          <q-item
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-function-variant" />
            </q-item-section>
            <q-item-section>{{ $t('Calculate expression') }}</q-item-section>
            <q-item-section side>
              <q-icon name="mdi-menu-right" />
            </q-item-section>
            <q-menu auto-close anchor="top end" self="top start">
              <q-list dense>
                <template v-if="isCompare">
                  <q-item
                    v-for="c in compareCalcList"
                    :key="c.code"
                    @click="doCalcPage(c.code, true)"
                    :class="{ 'text-primary' : srcCalc === c.code }"
                    clickable
                    >
                    <q-item-section>{{ $t(c.label) }}</q-item-section>
                    <q-item-section class="mono" :class="{ 'text-primary' : srcCalc === c.code }" side>{{ c.code }}</q-item-section>
                  </q-item>
                  <q-separator />
                </template>
                <q-item
                  v-for="c in aggrCalcList"
                  :key="c.code"
                  @click="doCalcPage(c.code, false)"
                  :class="{ 'text-primary' : srcCalc === c.code }"
                  clickable
                  >
                  <q-item-section>{{ $t(c.label) }}</q-item-section>
                  <q-item-section class="mono" :class="{ 'text-primary' : srcCalc === c.code }" side>{{ c.code }}</q-item-section>
                </q-item>
              </q-list>
            </q-menu>
          </q-item>
          <!-- end of calculated measures menu -->

          <q-item
            @click="onValueFilter"
            v-close-popup
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" :name="!!valueFilter.length ? 'mdi-filter-outline' : 'mdi-filter'" />
            </q-item-section>
            <q-item-section>{{ $t('Filter by values') + (!!valueFilter.length ? ' [ ' + valueFilter.length.toLocaleString() + ' ]' : '\u2026') }}</q-item-section>
          </q-item>

          <!-- calculated scale menu -->
          <q-item
            :disable="!isScaleEnabled() || isScalar"
            :clickable="isScaleEnabled() && !isScalar"
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-sun-thermometer" />
            </q-item-section>
            <q-item-section>{{ $t('Heat map') }}</q-item-section>
            <q-item-section side>
              <q-icon name="mdi-menu-right" />
            </q-item-section>
            <q-menu
              v-if="isScaleEnabled() && !isScalar"
              anchor="top end"
              self="top start"
              >
              <q-list dense>
                <q-item
                  @click="doClearScale()"
                  :disable="scaleId < 0 && !scaleCalc"
                  v-close-popup
                  clickable
                  >
                  <q-item-section><span><span class="mono check-menu">&nbsp;</span>{{ $t('Clear') }}</span></q-item-section>
                </q-item>
                <q-separator />
                <q-item
                  v-for="c in scaleItems"
                  :key="c.value"
                  @click="onScaleItem(c.value, c.name)"
                  clickable
                  >
                  <template v-if="scaleId === c.value">
                    <q-item-section class="text-primary"><span><span class="mono check-menu">&check;</span>{{ $t(c.label) }}</span></q-item-section>
                  </template>
                  <template v-else>
                    <q-item-section><span><span class="mono check-menu">&nbsp;</span>{{ $t(c.label) }}</span></q-item-section>
                  </template>
                </q-item>
                <q-separator />
                <q-item
                  v-for="c in scaleCalcList"
                  :key="c.code"
                  @click="onScaleCalc(c.value, c.code)"
                  clickable
                  >
                  <template v-if="scaleCalc === c.value">
                    <q-item-section class="text-primary"><span class="mono"><span class="mono check-menu">&check;</span>{{ $t(c.label) }}</span></q-item-section>
                  </template>
                  <template v-else>
                    <q-item-section><span class="mono"><span class="mono check-menu">&nbsp;</span>{{ $t(c.label) }}</span></q-item-section>
                  </template>
                </q-item>
              </q-list>
            </q-menu>
          </q-item>
          <!-- end of calculated scale menu -->

          <q-separator />

          <!-- chart menu -->
          <q-item
            @click="onChartToggle()"
            :disable="isScalar"
            clickable
            v-close-popup
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-chart-bar" />
            </q-item-section>
            <q-item-section>{{ isShowChart ? $t('Hide chart') : $t('Show chart') }}</q-item-section>
          </q-item>
          <q-item
            :disable="isScalar"
            @click="showChart('col')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'col') }"
            :clickable="!isShowChart || chartType !== 'col'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="bar_chart" /></q-item-section>
            <q-item-section>{{ $t('Column chart') }}</q-item-section>
          </q-item>
          <q-item
            :disable="isScalar"
            @click="showChart('col-stack')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'col-stack') }"
            :clickable="!isShowChart || chartType !== 'col-stack'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="stacked_bar_chart" /></q-item-section>
            <q-item-section>{{ $t('Column stacked') }}</q-item-section>
          </q-item>
          <q-item
            :disable="isScalar"
            @click="showChart('col-100')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'col-100') }"
            :clickable="!isShowChart || chartType !== 'col-100'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="stacked_bar_chart" /></q-item-section>
            <q-item-section>{{ $t('Column stacked 100%') }}</q-item-section>
          </q-item>
          <q-item
          :disable="isScalar"
          @click="showChart('row')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'row') }"
            :clickable="!isShowChart || chartType !== 'row'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="bar_chart" style="rotate: 90deg;" /></q-item-section>
            <q-item-section>{{ $t('Row chart') }}</q-item-section>
          </q-item>
          <q-item
          :disable="isScalar"
          @click="showChart('row-stack')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'row-stack') }"
            :clickable="!isShowChart || chartType !== 'row-stack'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="stacked_bar_chart" style="rotate: 90deg;" /></q-item-section>
            <q-item-section>{{ $t('Row stacked') }}</q-item-section>
          </q-item>
          <q-item
          :disable="isScalar"
          @click="showChart('row-100')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'row-100') }"
            :clickable="!isShowChart || chartType !== 'row-100'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="stacked_bar_chart" style="rotate: 90deg;" /></q-item-section>
            <q-item-section>{{ $t('Row stacked 100%') }}</q-item-section>
          </q-item>
          <q-item
          :disable="isScalar"
            @click="showChart('line')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'line') }"
            :clickable="!isShowChart || chartType !== 'line'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="mdi-chart-line" /></q-item-section>
            <q-item-section>{{ $t('Line chart') }}</q-item-section>
          </q-item>
          <q-item
          :disable="isScalar"
          @click="showChart('spline')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'spline') }"
            :clickable="!isShowChart || chartType !== 'spline'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="mdi-chart-bell-curve" /></q-item-section>
            <q-item-section>{{ $t('Spline chart') }}</q-item-section>
          </q-item>
          <q-item
          :disable="isScalar"
          @click="showChart('cubic')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'cubic') }"
            :clickable="!isShowChart || chartType !== 'cubic'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="wifi_channel" /></q-item-section>
            <q-item-section>{{ $t('Cubic spline') }}</q-item-section>
          </q-item>
          <q-item
          :disable="isScalar"
          @click="showChart('pie')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'pie') }"
            :clickable="!isShowChart || chartType !== 'pie'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="pie_chart" /></q-item-section>
            <q-item-section>{{ $t('Pie chart') }}</q-item-section>
          </q-item>
          <q-item
          :disable="isScalar"
          @click="showChart('donut')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'donut') }"
            :clickable="!isShowChart || chartType !== 'donut'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="mdi-chart-donut" /></q-item-section>
            <q-item-section>{{ $t('Donut chart') }}</q-item-section>
          </q-item>
          <q-item
          :disable="isScalar || !isShowChart"
            @click="onChartLabelToggle()"
            clickable
            >
            <template v-if="isChartLabels">
              <q-item-section avatar><q-icon name="mdi-numeric-off" /></q-item-section>
              <q-item-section class="text-primary">{{ $t('Hide chart values') }}</q-item-section>
            </template>
            <template v-else>
              <q-item-section avatar><q-icon name="mdi-numeric" /></q-item-section>
              <q-item-section>{{ $t('Show chart values') }}</q-item-section>
            </template>
          </q-item>
          <q-item
            :disable="isScalar || !isShowChart"
            >
            <q-item-section avatar><q-icon name="mdi-resize" /></q-item-section>
            <q-item-section class="q-pb-lg q-pr-xs q-ma-xs">
              <q-slider
                v-model="prctChartSize"
                :step="10"
                snap
                :min="20"
                :max="100"
                :label-value="prctChartSize + '%'"
                label-always
                switch-label-side
              />
            </q-item-section>
          </q-item>
          <!-- end of chart menu -->

          <q-separator />

          <q-item
            @click="onCopyToClipboard"
            clickable
            v-close-popup
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-content-copy" />
            </q-item-section>
            <q-item-section>{{ $t('Copy tab separated values to clipboard: ') + 'Ctrl+C' }}</q-item-section>
          </q-item>
          <q-item
            @click="onDownload"
            clickable
            v-close-popup
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-download" />
            </q-item-section>
            <q-item-section>{{ $t('Download') + ' '  + tableName + ' ' + $t('as CSV') }}</q-item-section>
          </q-item>

          <q-separator />

          <q-item
            @click="onToggleRowColControls"
            :disable="!ctrl.isRowColModeToggle"
            clickable
            v-close-popup
            >
            <q-item-section avatar>
              <q-icon color="primary" :name="ctrl.isRowColControls ? 'mdi-table-headers-eye-off' : 'mdi-table-headers-eye'" />
            </q-item-section>
            <q-item-section>{{ ctrl.isRowColControls ? $t('Hide rows and columns bars') : $t('Show rows and columns bars') }}</q-item-section>
          </q-item>

          <template v-if="ctrl.isRowColModeToggle">
            <q-item
              @click="onSetRowColMode(2)"
              :disable="pvc.rowColMode === 2"
              clickable
              v-close-popup
              >
              <q-item-section avatar>
                <q-icon color="primary" name="mdi-view-quilt-outline" />
              </q-item-section>
              <q-item-section>{{ $t('Switch to default pivot view') }}</q-item-section>
            </q-item>
            <q-item
              @click="onSetRowColMode(1)"
              :disable="pvc.rowColMode === 1"
              clickable
              v-close-popup
              >
              <q-item-section avatar>
                <q-icon color="primary" name="mdi-view-compact-outline" />
              </q-item-section>
              <q-item-section>{{ $t('Table view: hide rows and columns name') }}</q-item-section>
            </q-item>
            <q-item
              @click="onSetRowColMode(3)"
              :disable="pvc.rowColMode === 3"
              clickable
              v-close-popup
              >
              <q-item-section avatar>
                <q-icon color="primary" name="mdi-view-list-outline" />
              </q-item-section>
              <q-item-section>{{ $t('Table view: always show rows and columns item and name') }}</q-item-section>
            </q-item>
            <q-item
              @click="onSetRowColMode(0)"
              :disable="pvc.rowColMode === 0"
              clickable
              v-close-popup
              >
              <q-item-section avatar>
                <q-icon color="primary" name="mdi-view-module-outline" />
              </q-item-section>
              <q-item-section>{{ $t('Table view: always show rows and columns item') }}</q-item-section>
            </q-item>
          </template>

          <q-item
            @click="onShowMoreFormat"
            :disable="!ctrl.isMoreView"
            clickable
            v-close-popup
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-decimal-increase" />
            </q-item-section>
            <q-item-section>{{ $t('Increase precision') }}</q-item-section>
          </q-item>
          <q-item
            @click="onShowLessFormat"
            :disable="!ctrl.isLessView"
            clickable
            v-close-popup
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-decimal-decrease" />
            </q-item-section>
            <q-item-section>{{ $t('Decrease precision') }}</q-item-section>
          </q-item>
          <q-item
            v-if="ctrl.isRawUseView"
            @click="onToggleRawValue"
            clickable
            v-close-popup
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-loupe" />
            </q-item-section>
            <q-item-section>{{ !ctrl.isRawView ? $t('Show raw source value') : $t('Show formatted value') }}</q-item-section>
          </q-item>
          <q-item
            @click="onShowItemNames"
            :disable="isScalar"
            clickable
            v-close-popup
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-label-outline" />
            </q-item-section>
            <q-item-section>{{ pvc.isShowNames ? $t('Show labels') : $t('Show names') }}</q-item-section>
          </q-item>

          <q-separator />

          <q-item
            @click="onSaveDefaultView"
            clickable
            v-close-popup
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-content-save-cog" />
            </q-item-section>
            <q-item-section>{{ $t('Save table view as default view of') + ' ' + tableName }}</q-item-section>
          </q-item>
          <q-item
            @click="onReloadDefaultView"
            clickable
            v-close-popup
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-cog-refresh-outline" />
            </q-item-section>
            <q-item-section>{{ $t('Reset table view to default and reload') + ' ' + tableName }}</q-item-section>
          </q-item>

        </q-list>
      </q-menu>
    </q-btn>
    <!-- end of menu -->

    <q-btn
      @click="doShowTableNote"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-mr-xs"
      icon="mdi-information"
      :title="$t('About') + ' ' + tableName"
      />
    <q-btn
      tag="a"
      target="_blank"
      :href="'/?model=' + encodeURIComponent(digest) + '&run=' + encodeURIComponent(runDigest) + '&table=' + encodeURIComponent(tableName)"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders"
      icon="open_in_new"
      :title="$t('Open in new window')"
      />
    <q-separator vertical inset spaced="sm" color="secondary" />

    <template v-if="isPages">
      <q-btn
        @click="isShowPageControls = !isShowPageControls"
        :flat="!isShowPageControls"
        :outline="isShowPageControls"
        dense
        :class="!isShowPageControls ? 'bar-button-on' : 'bar-button-off'"
        class="col-auto rounded-borders q-mr-xs"
        icon="mdi-unfold-more-vertical"
        :title="isShowPageControls ? $t('Hide pagination controls') : $t('Show pagination controls')"
        />
      <template v-if="isShowPageControls">
        <q-btn
          @click="onFirstPage"
          :disable="pageStart === 0"
          flat
          dense
          class="col-auto bg-primary text-white rounded-borders q-mr-xs"
          icon="mdi-page-first"
          :title="$t('First page')"
          />
        <q-btn
          @click="onPrevPage"
          :disable="pageStart === 0"
          flat
          dense
          class="col-auto bg-primary text-white rounded-borders q-mr-xs"
          icon="mdi-chevron-left"
          :title="$t('Previous page')"
          />
        <div
          class="page-start-item rounded-borders om-text-secondary q-px-xs q-py-xs q-mr-xs"
          :title="$t('Position')"
          >{{ (!!pageStart && typeof pageStart === typeof 1) ? pageStart.toLocaleString() : pageStart }}</div>
        <q-btn
          @click="onNextPage"
          :disable="isLastPage"
          flat
          dense
          class="col-auto bg-primary text-white rounded-borders q-mr-xs"
          icon="mdi-chevron-right"
          :title="$t('Next page')"
          />
        <q-btn
          @click="onLastPage"
          :disable="isLastPage"
          flat
          dense
          class="col-auto bg-primary text-white rounded-borders q-mr-xs"
          icon="mdi-page-last"
          :title="$t('Last page')"
          />
        <q-select
          :model-value="pageSize"
           @update:model-value="onPageSize"
          :options="[10, 40, 100, 200, 400, 1000, 2000, 4000, 10000, 20000, 0]"
          :option-label="(val) => (!val || typeof val !== typeof 1 || val <= 0) ? $t('All') : val.toLocaleString()"
          outlined
          options-dense
          dense
          :label="$t('Size')"
          class="col-auto"
          style="min-width: 6rem"
          >
        </q-select>
      </template>
      <q-separator vertical inset spaced="sm" color="secondary" />
    </template>

    <q-btn
      @click="doExpressionPage"
      :disable="ctrl.kind === 0"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-mr-xs"
      icon="mdi-sigma"
      :title="$t('View table expressions')"
      />
    <q-btn
      @click="doAccumulatorPage"
      :disable="ctrl.kind === 1"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-mr-xs"
      icon="mdi-variable"
      :title="$t('View accumulators and sub-values (sub-samples)')"
      />
    <q-btn
      @click="doAllAccumulatorPage"
      :disable="ctrl.kind === 2"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-mr-xs"
      icon="mdi-application-variable-outline"
      :title="$t('View all accumulators and sub-values (sub-samples)')"
      />
    <!-- calculated measures menu -->
    <q-btn
      :flat="ctrl.kind !== 3 && ctrl.kind !== 4"
      :outline="ctrl.kind === 3 || ctrl.kind === 4"
      dense
      :class="(ctrl.kind !== 3 && ctrl.kind !== 4) ? 'bar-button-on' : 'bar-button-off'"
      class="col-auto rounded-borders q-mr-xs"
      icon="mdi-function-variant"
      :title="$t('Calculate expression')"
      :aria-label="$t('Calculate expression')"
      >
      <q-menu auto-close>
        <q-list dense>
          <template v-if="isCompare">
            <q-item
              v-for="c in compareCalcList"
              :key="c.code"
              @click="doCalcPage(c.code, true)"
              :class="{ 'text-primary' : srcCalc === c.code }"
              clickable
              >
              <q-item-section>{{ $t(c.label) }}</q-item-section>
              <q-item-section class="mono" :class="{ 'text-primary' : srcCalc === c.code }" side>{{ c.code }}</q-item-section>
            </q-item>
            <q-separator />
          </template>
          <q-item
            v-for="c in aggrCalcList"
            :key="c.code"
            @click="doCalcPage(c.code, false)"
            :class="{ 'text-primary' : srcCalc === c.code }"
            clickable
            >
            <q-item-section>{{ $t(c.label) }}</q-item-section>
            <q-item-section class="mono" :class="{ 'text-primary' : srcCalc === c.code }" side>{{ c.code }}</q-item-section>
          </q-item>
        </q-list>
      </q-menu>
    </q-btn>
    <!-- end of calculated measures menu -->
    <q-btn
      @click="onValueFilter"
      unelevated
      dense
      color="primary"
      class="col-auto rounded-borders q-mr-xs"
      :icon="!!valueFilter.length ? 'mdi-filter-outline' : 'mdi-filter'"
      :label="!!valueFilter.length ? '[ ' + valueFilter.length.toLocaleString() + ' ]' : ''"
      :title="$t('Filter by values') + (!!valueFilter.length ? ' [ ' + valueFilter.length.toLocaleString() + ' ]' : '\u2026')"
      :aria-label="$t('Filter by values')"
      />
    <!-- calculated scale menu -->
    <q-btn
      :disable="!isScaleEnabled() || isScalar"
      unelevated
      dense
      color="primary"
      class="col-auto rounded-borders q-mr-xs"
      icon="mdi-sun-thermometer"
      :title="$t('Heat map')"
      :aria-label="$t('Heat map')"
      >
      <q-menu>
        <div class=" q-px-sm q-mt-sm q-mb-none">
          <q-btn
          @click="doClearScale()"
          :disable="scaleId < 0 && !scaleCalc"
          v-close-popup
          unelevated
          :label="$t('Clear')"
          no-caps
          color="primary"
          class="rounded-borders full-width"
          />
        </div>
        <q-list dense>
          <q-item
            v-for="c in scaleItems"
            :key="c.value"
            @click="onScaleItem(c.value, c.name)"
            :class="{ 'text-primary' : scaleId === c.value }"
            clickable
            >
            <template v-if="scaleId === c.value">
              <q-item-section class="text-primary"><span><span class="mono check-menu">&check;</span>{{ $t(c.label) }}</span></q-item-section>
            </template>
            <template v-else>
              <q-item-section><span><span class="mono check-menu">&nbsp;</span>{{ $t(c.label) }}</span></q-item-section>
            </template>
          </q-item>
          <q-separator />
          <q-item
            v-for="c in scaleCalcList"
            :key="c.code"
            @click="onScaleCalc(c.value, c.code)"
            :class="{ 'text-primary' : scaleCalc === c.value }"
            clickable
            >
            <template v-if="scaleCalc === c.value">
              <q-item-section class="text-primary"><span class="mono"><span class="mono check-menu">&check;</span>{{ $t(c.label) }}</span></q-item-section>
            </template>
            <template v-else>
              <q-item-section><span class="mono"><span class="mono check-menu">&nbsp;</span>{{ $t(c.label) }}</span></q-item-section>
            </template>
          </q-item>
        </q-list>
      </q-menu>
    </q-btn>
    <!-- end of calculated scale menu -->

    <!-- chart menu -->
    <q-btn
      :disable="isScalar"
      unelevated
      dense
      color="primary"
      class="col-auto rounded-borders"
      icon="mdi-chart-bar"
      :title="$t('Chart')"
      :aria-label="$t('Chart')"
      >
      <q-menu>

        <div class=" q-px-sm q-mt-sm q-mb-none">
          <q-btn
          :disable="isScalar"
          @click="onChartToggle()"
          v-close-popup
          unelevated
          :label="isShowChart ? $t('Hide chart') : $t('Show chart')"
          no-caps
          color="primary"
          class="rounded-borders full-width"
          />
        </div>
        <q-list dense>
          <q-item
            :disable="isScalar"
            @click="showChart('col')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'col') }"
            :clickable="!isShowChart || chartType !== 'col'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="bar_chart" /></q-item-section>
            <q-item-section>{{ $t('Column chart') }}</q-item-section>
          </q-item>
          <q-item
            :disable="isScalar"
            @click="showChart('col-stack')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'col-stack') }"
            :clickable="!isShowChart || chartType !== 'col-stack'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="stacked_bar_chart" /></q-item-section>
            <q-item-section>{{ $t('Column stacked') }}</q-item-section>
          </q-item>
          <q-item
            :disable="isScalar"
            @click="showChart('col-100')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'col-100') }"
            :clickable="!isShowChart || chartType !== 'col-100'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="stacked_bar_chart" /></q-item-section>
            <q-item-section>{{ $t('Column stacked 100%') }}</q-item-section>
          </q-item>
          <q-item
          :disable="isScalar"
          @click="showChart('row')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'row') }"
            :clickable="!isShowChart || chartType !== 'row'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="bar_chart" style="rotate: 90deg;" /></q-item-section>
            <q-item-section>{{ $t('Row chart') }}</q-item-section>
          </q-item>
          <q-item
          :disable="isScalar"
          @click="showChart('row-stack')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'row-stack') }"
            :clickable="!isShowChart || chartType !== 'row-stack'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="stacked_bar_chart" style="rotate: 90deg;" /></q-item-section>
            <q-item-section>{{ $t('Row stacked') }}</q-item-section>
          </q-item>
          <q-item
          :disable="isScalar"
          @click="showChart('row-100')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'row-100') }"
            :clickable="!isShowChart || chartType !== 'row-100'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="stacked_bar_chart" style="rotate: 90deg;" /></q-item-section>
            <q-item-section>{{ $t('Row stacked 100%') }}</q-item-section>
          </q-item>
          <q-item
          :disable="isScalar"
          @click="showChart('line')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'line') }"
            :clickable="!isShowChart || chartType !== 'line'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="mdi-chart-line" /></q-item-section>
            <q-item-section>{{ $t('Line chart') }}</q-item-section>
          </q-item>
          <q-item
          :disable="isScalar"
          @click="showChart('spline')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'spline') }"
            :clickable="!isShowChart || chartType !== 'spline'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="mdi-chart-bell-curve" /></q-item-section>
            <q-item-section>{{ $t('Spline chart') }}</q-item-section>
          </q-item>
          <q-item
          :disable="isScalar"
          @click="showChart('cubic')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'cubic') }"
            :clickable="!isShowChart || chartType !== 'cubic'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="wifi_channel" /></q-item-section>
            <q-item-section>{{ $t('Cubic spline') }}</q-item-section>
          </q-item>
          <q-item
          :disable="isScalar"
          @click="showChart('pie')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'pie') }"
            :clickable="!isShowChart || chartType !== 'pie'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="pie_chart" /></q-item-section>
            <q-item-section>{{ $t('Pie chart') }}</q-item-section>
          </q-item>
          <q-item
          :disable="isScalar"
          @click="showChart('donut')"
            :class="{ 'text-primary' : (isShowChart && chartType === 'donut') }"
            :clickable="!isShowChart || chartType !== 'donut'"
            v-close-popup
            >
            <q-item-section avatar><q-icon name="mdi-chart-donut" /></q-item-section>
            <q-item-section>{{ $t('Donut chart') }}</q-item-section>
          </q-item>
          <q-item
          :disable="isScalar || !isShowChart"
            @click="onChartLabelToggle()"
            clickable
            >
            <template v-if="isChartLabels">
              <q-item-section avatar><q-icon name="mdi-numeric-off" /></q-item-section>
              <q-item-section class="text-primary">{{ $t('Hide chart values') }}</q-item-section>
            </template>
            <template v-else>
              <q-item-section avatar><q-icon name="mdi-numeric" /></q-item-section>
              <q-item-section>{{ $t('Show chart values') }}</q-item-section>
            </template>
          </q-item>
          <q-item
            :disable="isScalar || !isShowChart"
            >
            <q-item-section avatar><q-icon name="mdi-resize" /></q-item-section>
            <q-item-section class="q-pb-lg q-pr-xs q-ma-xs">
              <q-slider
                v-model="prctChartSize"
                :step="10"
                snap
                :min="20"
                :max="100"
                :label-value="prctChartSize + '%'"
                label-always
                switch-label-side
              />
            </q-item-section>
          </q-item>
        </q-list>
      </q-menu>
    </q-btn>
    <!-- end of chart menu -->

    <q-separator vertical inset spaced="sm" color="secondary" />

    <q-btn
      @click="onCopyToClipboard"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-mr-xs"
      icon="mdi-content-copy"
      :title="$t('Copy tab separated values to clipboard: ') + 'Ctrl+C'"
      />
    <q-btn
      @click="onDownload"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders"
      icon="mdi-download"
      :title="$t('Download') + ' '  + tableName + ' ' + $t('as CSV')"
      />
    <q-separator vertical inset spaced="sm" color="secondary" />

    <q-btn
      @click="onToggleRowColControls"
      :disable="!ctrl.isRowColModeToggle"
      :outline="!ctrl.isRowColControls"
      :unelevated="ctrl.isRowColControls"
      dense
      color="primary"
      :class="{ 'q-mr-xs' : ctrl.isRowColModeToggle }"
      class="col-auto rounded-borders q-mr-xs"
      :icon="ctrl.isRowColControls ? 'mdi-table-headers-eye-off' : 'mdi-table-headers-eye'"
      :title="ctrl.isRowColControls ? $t('Hide rows and columns bars') : $t('Show rows and columns bars')"
      />

    <template v-if="ctrl.isRowColModeToggle">
      <q-btn
        @click="onSetRowColMode(2)"
        :disable="pvc.rowColMode === 2"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders q-mr-xs"
        icon="mdi-view-quilt-outline"
        :title="$t('Switch to default pivot view')"
        />
      <q-btn
        @click="onSetRowColMode(1)"
        :disable="pvc.rowColMode === 1"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders q-mr-xs"
        icon="mdi-view-compact-outline"
        :title="$t('Table view: hide rows and columns name')"
        />
      <q-btn
        @click="onSetRowColMode(3)"
        :disable="pvc.rowColMode === 3"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders q-mr-xs"
        icon="mdi-view-list-outline"
        :title="$t('Table view: always show rows and columns item and name')"
        />
      <q-btn
        @click="onSetRowColMode(0)"
        :disable="pvc.rowColMode === 0"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders q-mr-xs"
        icon="mdi-view-module-outline"
        :title="$t('Table view: always show rows and columns item')"
        />
    </template>

    <q-btn
      @click="onShowMoreFormat"
      :disable="!ctrl.isMoreView"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-mr-xs"
      icon="mdi-decimal-increase"
      :title="$t('Increase precision')"
      />
    <q-btn
      @click="onShowLessFormat"
      :disable="!ctrl.isLessView"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-mr-xs"
      icon="mdi-decimal-decrease"
      :title="$t('Decrease precision')"
      />
    <q-btn
      v-if="ctrl.isRawUseView"
      @click="onToggleRawValue"
      :flat="!ctrl.isRawView"
      :outline="ctrl.isRawView"
      dense
      :class="!ctrl.isRawView ? 'bar-button-on' : 'bar-button-off'"
      class="col-auto rounded-borders q-mr-xs"
      icon="mdi-loupe"
      :title="!ctrl.isRawView ? $t('Show raw source value') : $t('Show formatted value')"
      />
    <q-btn
      @click="onShowItemNames"
      :disable="isScalar"
      :flat="!pvc.isShowNames"
      :outline="pvc.isShowNames"
      dense
      :class="!pvc.isShowNames ? 'bar-button-on' : 'bar-button-off'"
      class="col-auto rounded-borders"
      icon="mdi-label-outline"
      :title="pvc.isShowNames ? $t('Show labels') : $t('Show names')"
      />
    <q-separator vertical inset spaced="sm" color="secondary" />

    <q-btn
      @click="onSaveDefaultView"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-mr-xs"
      icon="mdi-content-save-cog"
      :title="$t('Save table view as default view of') + ' ' + tableName"
      />
    <q-btn
      @click="onReloadDefaultView"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-mr-xs"
      icon="mdi-cog-refresh-outline"
      :title="$t('Reset table view to default and reload') + ' ' + tableName"
      />

    <div
      class="col-auto"
      >
      <span>{{ tableName }}<br />
      <span class="om-text-descr">{{ tableDescr }}</span></span>
    </div>

    </q-toolbar>

  </div>
  <!-- end of output table header -->

  <q-card v-if="isShowChart && chartType === 'col'" class="q-mx-sm q-mb-sm">
    <apexchart :width="chartWidth" :options="chartOpts" :series="chartSeries"></apexchart>
  </q-card>
  <q-card v-if="isShowChart && chartType === 'col-stack'" class="q-mx-sm q-mb-sm">
    <apexchart :width="chartWidth" :options="chartOpts" :series="chartSeries"></apexchart>
  </q-card>
  <q-card v-if="isShowChart && chartType === 'col-100'" class="q-mx-sm q-mb-sm">
    <apexchart :width="chartWidth" :options="chartOpts" :series="chartSeries"></apexchart>
  </q-card>
  <q-card v-if="isShowChart && chartType === 'row'" class="q-mx-sm q-mb-sm">
    <apexchart :width="chartWidth" :options="chartOpts" :series="chartSeries"></apexchart>
  </q-card>
  <q-card v-if="isShowChart && chartType === 'row-stack'" class="q-mx-sm q-mb-sm">
    <apexchart :width="chartWidth" :options="chartOpts" :series="chartSeries"></apexchart>
  </q-card>
  <q-card v-if="isShowChart && chartType === 'row-100'" class="q-mx-sm q-mb-sm">
    <apexchart :width="chartWidth" :options="chartOpts" :series="chartSeries"></apexchart>
  </q-card>
  <q-card v-if="isShowChart && chartType === 'line'" class="q-mx-sm q-mb-sm">
    <apexchart :width="chartWidth" :options="chartOpts" :series="chartSeries"></apexchart>
  </q-card>
  <q-card v-if="isShowChart && chartType === 'spline'" class="q-mx-sm q-mb-sm">
    <apexchart :width="chartWidth" :options="chartOpts" :series="chartSeries"></apexchart>
  </q-card>
  <q-card v-if="isShowChart && chartType === 'cubic'" class="q-mx-sm q-mb-sm">
    <apexchart :width="chartWidth" :options="chartOpts" :series="chartSeries"></apexchart>
  </q-card>

  <template v-if="isShowChart && chartType === 'pie'">
    <q-card v-for="(d, k) of chartSeries" :key="(d.name || '') + '-pie-n-' + k.toString()" class="q-mx-sm q-mb-sm">
      <apexchart
        :width="chartWidth"
        :series="d.data"
        :options="{...chartOpts, ...{ title: { text: d.name } } }">
      </apexchart>
    </q-card>
  </template>

  <template v-if="isShowChart &&  chartType === 'donut'">
    <q-card v-for="(d, k) of chartSeries" :key="(d.name || '') + '-don-n-' + k.toString()" class="q-mx-sm q-mb-sm">
      <apexchart
        :width="chartWidth"
        :series="d.data"
        :options="{...chartOpts, ...{ title: { text: d.name } } }">
      </apexchart>
    </q-card>
  </template>

  <q-card
    v-if="isShowChart && (['row', 'row-stack', 'row-100', 'col', 'col-stack', 'col-100', 'line', 'spline', 'cubic', 'pie', 'donut'].indexOf(chartType) < 0)"
    class="q-ma-lg q-pa-lg"
    >
    {{ $t('No chart data') }}
  </q-card>

  <!-- pivot table controls and view -->
  <div class="q-mx-sm">

    <div v-show="ctrl.isRowColControls"
      class="other-panel"
      >
      <draggable
        v-model="otherFields"
        group="fields"
        @start="onDrag"
        @end="onDrop"
        class="other-fields other-drag"
        :class="{'drag-area-hint': isDragging}"
        >
        <div v-for="f in otherFields" :key="f.name" :id="'item-draggable-' + f.name" class="field-drag om-text-medium">
          <q-select
            :model-value="f.singleSelection"
            @update:model-value="onUpdateSelect"
            @focus="onFocusSelect"
            :options="f.options"
            :option-label="pvc.isShowNames ? 'name' : 'label'"
            @input="onSelectInput('other', f.name, f.singleSelection)"
            use-input
            hide-selected
            @filter="f.filter"
            :label="selectLabel(pvc.isShowNames, f)"
            outlined
            dense
            options-dense
            bottom-slots
            class="other-select"
            options-selected-class="option-selected"
            :title="$t('Select') + ' ' + (pvc.isShowNames ? f.name : f.label)"
            >
            <template v-slot:before>
              <q-icon
                name="mdi-hand-back-left"
                size="xs"
                class="bg-primary text-white rounded-borders select-handle-move"
                :title="$t('Drag and drop')"
                />
              <div class="column">
                <q-icon
                  name="check"
                  size="xs"
                  class="row bg-primary text-white rounded-borders select-handle-button om-bg-inactive q-mb-xs"
                  :title="$t('Select all')"
                  />
                <q-icon
                  name="cancel"
                  size="xs"
                  class="row bg-primary text-white rounded-borders select-handle-button om-bg-inactive"
                  :title="$t('Clear all')"
                  />
              </div>
            </template>
            <template v-slot:hint>
              <div class="row">
                <span class="col ellipsis">{{ pvc.isShowNames ? f.name : f.label }}</span>
                <span class="col-auto text-no-wrap">{{ f.selection ? f.selection.length : 0 }} / {{ f.enums ? f.enums.length : 0 }}</span>
              </div>
            </template>
          </q-select>
        </div>
      </draggable>
      <q-tooltip anchor="center right" content-class="om-tooltip">{{ $t('Slicer dimensions') }}</q-tooltip>
    </div>

    <div v-show="ctrl.isRowColControls"
      class="col-panel"
      >
      <draggable
        v-model="colFields"
        group="fields"
        @start="onDrag"
        @end="onDrop"
        class="col-fields col-drag"
        :class="{'drag-area-hint': isDragging}"
        >
        <div v-for="f in colFields" :key="f.name" :id="'item-draggable-' + f.name" class="field-drag om-text-medium">
          <q-select
            :model-value="f.selection"
            @update:model-value="onUpdateSelect"
            @focus="onFocusSelect"
            :options="f.options"
            :option-label="pvc.isShowNames ? 'name' : 'label'"
            @input="onSelectInput('col', f.name, f.selection)"
            use-input
            hide-selected
            @filter="f.filter"
            :label="selectLabel(pvc.isShowNames, f)"
            multiple
            outlined
            dense
            options-dense
            bottom-slots
            class="col-select"
            options-selected-class="option-selected"
            :title="$t('Select') + ' ' + (pvc.isShowNames ? f.name : f.label)"
            >
            <template v-slot:before>
              <q-icon
                name="mdi-hand-back-left"
                size="xs"
                class="bg-primary text-white rounded-borders select-handle-move"
                :title="$t('Drag and drop')"
                />
              <div class="column">
                <q-btn
                  @click.stop="onSelectAll(f.name)"
                  flat
                  dense
                  size="xs"
                  class="row bg-primary text-white rounded-borders q-mb-xs"
                  icon="check"
                  :title="$t('Select all')"
                  />
                <q-btn
                  @click.stop="onClearAll(f.name)"
                  flat
                  dense
                  size="xs"
                  class="row bg-primary text-white rounded-borders"
                  icon="cancel"
                  :title="$t('Clear all')"
                  />
              </div>
            </template>
            <template v-slot:hint>
              <div class="row">
                <span class="col ellipsis">{{ pvc.isShowNames ? f.name : f.label }}</span>
                <span class="col-auto text-no-wrap">{{ f.selection ? f.selection.length : 0 }} / {{ f.enums ? f.enums.length : 0 }}</span>
              </div>
            </template>
          </q-select>
        </div>
      </draggable>
      <q-tooltip anchor="center right" content-class="om-tooltip">{{ $t('Column dimensions') }}</q-tooltip>
    </div>

    <div class="pv-panel">

      <draggable
        v-show="ctrl.isRowColControls"
        v-model="rowFields"
        group="fields"
        @start="onDrag"
        @end="onDrop"
        class="row-fields row-drag"
        :class="{'drag-area-hint': isDragging}"
        >
        <div v-for="f in rowFields" :key="f.name" :id="'item-draggable-' + f.name" class="field-drag om-text-medium">
          <q-select
            :model-value="f.selection"
            @update:model-value="onUpdateSelect"
            @focus="onFocusSelect"
            :options="f.options"
            :option-label="pvc.isShowNames ? 'name' : 'label'"
            @input="onSelectInput('row', f.name, f.selection)"
            use-input
            hide-selected
            @filter="f.filter"
            :label="selectLabel(pvc.isShowNames, f)"
            multiple
            outlined
            dense
            options-dense
            bottom-slots
            class="row-select"
            options-selected-class="option-selected"
            :title="$t('Select') + ' ' + (pvc.isShowNames ? f.name : f.label)"
            >
            <template v-slot:before>
              <q-icon
                name="mdi-hand-back-left"
                size="xs"
                class="bg-primary text-white rounded-borders select-handle-move"
                :title="$t('Drag and drop')"
                />
              <div class="column">
                <q-btn
                  @click.stop="onSelectAll(f.name)"
                  flat
                  dense
                  size="xs"
                  class="row bg-primary text-white rounded-borders q-mb-xs"
                  icon="check"
                  :title="$t('Select all')"
                  />
                <q-btn
                  @click.stop="onClearAll(f.name)"
                  flat
                  dense
                  size="xs"
                  class="row bg-primary text-white rounded-borders"
                  icon="cancel"
                  :title="$t('Clear all')"
                  />
              </div>
            </template>
            <template v-slot:hint>
              <div class="row">
                <span class="col ellipsis">{{ pvc.isShowNames ? f.name : f.label }}</span>
                <span class="col-auto text-no-wrap">{{ f.selection ? f.selection.length : 0 }} / {{ f.enums ? f.enums.length : 0 }}</span>
              </div>
            </template>
          </q-select>
        </div>
        <q-tooltip content-class="om-tooltip">{{ $t('Row dimensions') }}</q-tooltip>
      </draggable>

      <pv-table
        ref="omPivotTable"
        :rowFields="rowFields"
        :colFields="colFields"
        :otherFields="otherFields"
        :pv-data="inpData"
        :pv-control="pvc"
        :refreshViewTickle="ctrl.isPvTickle"
        :refreshDimsTickle="ctrl.isPvDimsTickle"
        :refreshRangeTickle="ctrl.isPvRangeTickle"
        @pv-key-pos="onPvKeyPos"
        >
      </pv-table>

    </div>

  </div>
  <!-- end of pivot table controls and view -->

  <refresh-run v-if="(digest || '') !== '' && (runDigest || '') !== ''"
    :model-digest="digest"
    :run-digest="runDigest"
    :refresh-tickle="refreshTickle"
    :refresh-run-tickle="refreshRunTickle"
    @done="loadRunWait = false"
    @wait="loadRunWait = true"
    >
  </refresh-run>

  <run-info-dialog :show-tickle="runInfoTickle" :model-digest="digest" :run-digest="runDigest"></run-info-dialog>
  <table-info-dialog :show-tickle="tableInfoTickle" :table-name="tableName" :run-digest="runDigest"></table-info-dialog>

  <value-filter-dialog
    :show-tickle="valueFilterTickle"
    :update-tickle="refreshTickle"
    :measure-list="valueFilterMeasure"
    :value-filter="valueFilter"
    :is-show-names="pvc.isShowNames"
    @value-filter-apply="onValueFilterApply"
    >
  </value-filter-dialog>

  <q-inner-loading :showing="loadWait || loadRunWait">
    <q-spinner-gears size="md" color="primary" />
  </q-inner-loading>

</div>
</template>

<script src="./table-page.js"></script>

<style lang="scss" scope="local">
  /* pivot vew controls: rows, columns, other dimensions and drag-drop area */
  .flex-item {
    display: flex;
    flex: 1 1 auto;
    min-width: 0;
  }

  .col-panel, .other-panel, .pv-panel {
    @extend .flex-item;
    flex-direction: row;
    flex-wrap: nowrap;
  }

  .other-fields, .col-fields {
    flex: 1 0 auto;
  }
  .row-fields {
    min-width: 12rem;
    flex: 0 0 auto;
  }
  .row-select, .col-select, .other-select {
    min-width: 10rem;
  }
  .page-start-item {
    background: rgba(0, 0, 0, 0.1);
    min-width: 2rem;
    text-align: center;
  }

  .drag-area {
    min-height: 2.5rem;
    padding: 0.125rem;
    border: 1px dashed grey;
  }
  .drag-area-hint {
    background-color: whitesmoke;
  }
  .drag-area-disabled {
    background-color: lightgrey;
    opacity: 0.5;
    border: 1px solid red;
  }
  .sortable-ghost {
    opacity: 0.5;
  }
  .top-col-drag {
    @extend .flex-item;
    @extend .drag-area;
    flex-direction: row;
    flex-wrap: wrap;
    justify-content: flex-start;
    width: 100%;
  }
  .col-drag {
    @extend .top-col-drag;
    border-top-style: none;
  }
  .other-drag {
    @extend .top-col-drag;
  }
  .row-drag {
    @extend .flex-item;
    @extend .drag-area;
    flex-direction: column;
    flex-wrap: wrap;
    align-items: flex-start;
    border-top-style: none;
  }

  .field-drag {
    display: inline-block;
    margin-left: 0.25rem;
    padding: 0.25rem;
    &:hover {
      cursor: move;
    }
  }

  .option-selected {
    background-color: whitesmoke;
  }
  .select-handle-move {
    min-height: 2.5rem;
    &:hover {
      cursor: move;
    }
  }
  .select-handle-button {
    &:hover {
      cursor: move;
    }
  }

  .bar-button-on {
    background-color: $primary;
    color: white;
  }
  .bar-button-off {
    color: $primary;
  }
  .check-menu {
    min-width: 1.125rem;
    display: inline-block;
  }

  .upload-right {
    text-align: right;
  }
  .upload-max-width-10 {
    max-width: 10rem;
  }
</style>
