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

  <!-- microdata header -->
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
            @click="doShowEntityNote"
            v-close-popup
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-information-outline" />
            </q-item-section>
            <q-item-section>{{ $t('About') + ' '  + entityName }}</q-item-section>
          </q-item>
          <q-item
            clickable
            v-close-popup
            tag="a"
            target="_blank"
            :href="'/?model=' + encodeURIComponent(digest) + '&run=' + encodeURIComponent(runDigest) + '&entity=' + encodeURIComponent(entityName)"
            >
            <q-item-section avatar>
              <q-icon color="primary" name="open_in_new" />
            </q-item-section>
            <q-item-section>{{ $t('Open in new window') }}</q-item-section>
          </q-item>

          <q-separator />

          <q-item
            @click="doMicroPage"
            v-close-popup
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-microscope" />
            </q-item-section>
            <q-item-section>{{ $t('Microdata view') }}</q-item-section>
          </q-item>
          <!-- calculated measures menu -->
          <q-item
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-function-variant" />
            </q-item-section>
            <q-item-section>{{ $t('Aggregate microdata') }}</q-item-section>
            <q-item-section side>
              <q-icon name="mdi-menu-right" />
            </q-item-section>
            <q-menu anchor="top end" self="top start">
              <q-list dense>
                <q-item
                  @click="onCalcPage()"
                  :disable="!groupDimCalc?.length || (!calcEnums.length && (!aggrCalc || !attrCalc.length))"
                  v-close-popup
                  clickable
                  >
                  <q-item-section>{{ $t('Apply') }}</q-item-section>
                </q-item>
                <q-item
                  @click="onCalcEdit()"
                  :disable="!groupDimCalc?.length || (!calcEnums.length && (!aggrCalc || !attrCalc.length))"
                  v-close-popup
                  clickable
                  >
                  <q-item-section>{{ $t('Edit') + '\u2026' }}</q-item-section>
                </q-item>

                <q-item v-if="isRunCompare" clickable>
                  <q-item-section>{{ $t('Comparison') }}</q-item-section>
                  <q-item-section side><span class="text-no-wrap mono">{{ cmpCalc }} &#x25B6;</span></q-item-section>
                  <q-menu anchor="top end" self="top start">
                    <q-list dense>
                      <q-item
                        v-for="c in compareCalcList"
                        :key="c.code"
                        @click="onCompareToogle(c.code)"
                        clickable
                        >
                        <template v-if="cmpCalc === c.code">
                          <q-item-section class="text-primary"><span><span class="mono check-menu">&check;</span>{{ $t(c.label) }}</span></q-item-section>
                          <q-item-section class="mono text-primary" side>{{ c.code }}</q-item-section>
                        </template>
                        <template v-else>
                          <q-item-section><span><span class="mono check-menu">&nbsp;</span>{{ $t(c.label) }}</span></q-item-section>
                          <q-item-section class="mono" side>{{ c.code }}</q-item-section>
                        </template>
                      </q-item>
                    </q-list>
                  </q-menu>
                </q-item>

                <q-item clickable>
                  <q-item-section>{{ $t('Aggregation') }}</q-item-section>
                  <q-item-section side><span class="text-no-wrap mono">{{ aggrCalc }} &#x25B6;</span></q-item-section>
                  <q-menu anchor="top end" self="top start">
                    <q-list dense>
                      <q-item
                        v-for="c in aggrCalcList"
                        :key="c.code"
                        @click="onAggregateSet(c.code)"
                        clickable
                        >
                        <template v-if="aggrCalc === c.code">
                          <q-item-section class="text-primary"><span><span class="mono check-menu">&check;</span>{{ $t(c.label) }}</span></q-item-section>
                          <q-item-section class="mono text-primary" side>{{ c.code }}</q-item-section>
                        </template>
                        <template v-else>
                          <q-item-section><span><span class="mono check-menu">&nbsp;</span>{{ $t(c.label) }}</span></q-item-section>
                          <q-item-section class="mono" side>{{ c.code }}</q-item-section>
                        </template>
                      </q-item>
                    </q-list>
                  </q-menu>
                </q-item>

                <q-item clickable>
                  <q-item-section>{{ $t('Dimensions') }}</q-item-section>
                  <q-item-section side><span class="text-no-wrap mono">{{ ((groupDimCalc?.length) || 0).toString() + '/' + (rank - 1 > 0 ? rank - 1 : 0).toString() }} &#x25B6;</span></q-item-section>
                  <q-menu anchor="top end" self="top start">
                    <q-list dense>
                      <template v-for="(d, n) in dimProp">
                        <template v-if="n > 0 && n < rank">
                          <q-item :key="'by-dim-' + d.name"
                            @click="onGroupByToogle(d.name)"
                            clickable
                            >
                            <template v-if="isGroupBy(d.name)">
                              <q-item-section v-if="!pvc.isShowNames" class="text-primary"><span><span class="mono check-menu">&check;</span>{{ d.label }}</span></q-item-section>
                              <q-item-section v-if="pvc.isShowNames" class="text-primary mono"><span><span class="mono check-menu">&check;</span>{{ d.name }}</span></q-item-section>
                            </template>
                            <template v-else>
                              <q-item-section v-if="!pvc.isShowNames"><span><span class="mono check-menu">&nbsp;</span>{{ d.label }}</span></q-item-section>
                              <q-item-section v-if="pvc.isShowNames" class="mono"><span><span class="mono check-menu">&nbsp;</span>{{ d.name }}</span></q-item-section>
                            </template>
                          </q-item>
                        </template>
                      </template>
                    </q-list>
                  </q-menu>
                </q-item>

                <q-item clickable>
                  <q-item-section>{{ $t('Measures') }}</q-item-section>
                  <q-item-section side><span class="text-no-wrap mono">{{ (attrCalc?.length || 0).toString() + '/' + attrCount.toString() }} &#x25B6;</span></q-item-section>
                  <q-menu anchor="top end" self="top start">
                    <q-list dense>
                      <template v-for="a in attrEnums">
                        <template v-if="a?.isFloat || a?.isInt">
                          <q-item :key="'c-attr-' + a.name"
                            @click="onCalcAttrToogle(a.name)"
                            clickable
                            >
                            <template v-if="isAttrCalc(a.name)">
                              <q-item-section v-if="!pvc.isShowNames" class="text-primary"><span><span class="mono check-menu">&check;</span>{{ a.label }}</span></q-item-section>
                              <q-item-section v-if="pvc.isShowNames" class="text-primary mono"><span><span class="mono check-menu">&check;</span>{{ a.name }}</span></q-item-section>
                            </template>
                            <template v-else>
                              <q-item-section v-if="!pvc.isShowNames"><span><span class="mono check-menu">&nbsp;</span>{{ a.label }}</span></q-item-section>
                              <q-item-section v-if="pvc.isShowNames" class="mono"><span><span class="mono check-menu">&nbsp;</span>{{ a.name }}</span></q-item-section>
                            </template>
                          </q-item>
                        </template>
                      </template>
                    </q-list>
                  </q-menu>
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

          <q-separator />

          <q-item
            @click="onCopyToClipboard"
            v-close-popup
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-content-copy" />
            </q-item-section>
            <q-item-section>{{ $t('Copy tab separated values to clipboard: ') + 'Ctrl+C' }}</q-item-section>
          </q-item>
          <q-item
            @click="onDownload"
            v-close-popup
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-download" />
            </q-item-section>
            <q-item-section>{{ $t('Download') + ' '  + entityName + ' ' + $t('as CSV') }}</q-item-section>
          </q-item>

          <q-separator />

          <q-item
            @click="onToggleRowColControls"
            :disable="!ctrl.isRowColModeToggle"
            v-close-popup
            clickable
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
              v-close-popup
              clickable
              >
              <q-item-section avatar>
                <q-icon color="primary" name="mdi-view-quilt-outline" />
              </q-item-section>
              <q-item-section>{{ $t('Switch to default pivot view') }}</q-item-section>
            </q-item>
            <q-item
              @click="onSetRowColMode(1)"
              :disable="pvc.rowColMode === 1"
              v-close-popup
              clickable
              >
              <q-item-section avatar>
                <q-icon color="primary" name="mdi-view-list-outline" />
              </q-item-section>
              <q-item-section>{{ $t('Table view: hide rows and columns name') }}</q-item-section>
            </q-item>
            <q-item
              @click="onSetRowColMode(3)"
              :disable="pvc.rowColMode === 3"
              v-close-popup
              clickable
              >
              <q-item-section avatar>
                <q-icon color="primary" name="mdi-view-compact-outline" />
              </q-item-section>
              <q-item-section>{{ $t('Table view: always show rows and columns item and name') }}</q-item-section>
            </q-item>
            <q-item
              @click="onSetRowColMode(0)"
              :disable="pvc.rowColMode === 0"
              v-close-popup
              clickable
              >
              <q-item-section avatar>
                <q-icon color="primary" name="mdi-view-module-outline" />
              </q-item-section>
              <q-item-section>{{ $t('Table view: always show rows and columns item') }}</q-item-section>
            </q-item>
          </template>

          <template v-if="ctrl.isFloatView">
            <q-item
              @click="onShowMoreFormat"
              :disable="!ctrl.isMoreView"
              v-close-popup
              clickable
              >
              <q-item-section avatar>
                <q-icon color="primary" name="mdi-decimal-increase" />
              </q-item-section>
              <q-item-section>{{ $t('Increase precision') }}</q-item-section>
            </q-item>
            <q-item
              @click="onShowLessFormat"
              :disable="!ctrl.isLessView"
              v-close-popup
              clickable
              >
              <q-item-section avatar>
                <q-icon color="primary" name="mdi-decimal-decrease" />
              </q-item-section>
              <q-item-section>{{ $t('Decrease precision') }}</q-item-section>
            </q-item>
          </template>
          <q-item
            v-if="ctrl.isRawUseView"
            @click="onToggleRawValue"
            v-close-popup
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-loupe" />
            </q-item-section>
            <q-item-section>{{ !ctrl.isRawView ? $t('Show raw source value') : $t('Show formatted value') }}</q-item-section>
          </q-item>
          <q-item
            @click="onShowItemNames"
            :disable="isScalar"
            v-close-popup
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-label-outline" />
            </q-item-section>
            <q-item-section>{{ pvc.isShowNames ? $t('Show labels') : $t('Show names') }}</q-item-section>
          </q-item>

          <q-separator />

          <q-item
            @click="onSaveDefaultView"
            v-close-popup
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-content-save-cog" />
            </q-item-section>
            <q-item-section>{{ $t('Save microdata view as default view of') + ' ' + entityName }}</q-item-section>
          </q-item>
          <q-item
            @click="onReloadDefaultView"
            v-close-popup
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-cog-refresh-outline" />
            </q-item-section>
            <q-item-section>{{ $t('Reset microdata view to default and reload') + ' ' + entityName }}</q-item-section>
          </q-item>

        </q-list>
      </q-menu>
    </q-btn>

    <!-- end of menu -->

    <q-btn
      @click="doShowEntityNote"
      unelevated
      dense
      color="primary"
      class="col-auto rounded-borders"
      icon="mdi-information"
      :title="$t('About') + ' ' + entityName"
      />
    <q-btn
      tag="a"
      target="_blank"
      :href="'/?model=' + encodeURIComponent(digest) + '&run=' + encodeURIComponent(runDigest) + '&entity=' + encodeURIComponent(entityName)"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-ml-xs"
      icon="open_in_new"
      :title="$t('Open in new window')"
      />
    <q-separator vertical inset spaced="sm" color="secondary" />

    <template v-if="isPages">
      <q-btn
        @click="isHidePageControls = !isHidePageControls"
        :unelevated="isHidePageControls"
        :outline="!isHidePageControls"
        dense
        color="primary"
        class="col-auto rounded-borders q-mr-xs"
        icon="mdi-unfold-more-vertical"
        :title="!isHidePageControls ? $t('Hide pagination controls') : $t('Show pagination controls')"
        />
      <template v-if="!isHidePageControls">
        <q-btn
          @click="onFirstPage"
          :disable="pageStart === 0"
          unelevated
          dense
          color="primary"
          class="col-auto rounded-borders q-mr-xs"
          icon="mdi-page-first"
          :title="$t('First page')"
          />
        <q-btn
          @click="onPrevPage"
          :disable="pageStart === 0"
          unelevated
          dense
          color="primary"
          class="col-auto rounded-borders q-mr-xs"
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
          unelevated
          dense
          color="primary"
          class="col-auto rounded-borders q-mr-xs"
          icon="mdi-chevron-right"
          :title="$t('Next page')"
          />
        <q-btn
          @click="onLastPage"
          :disable="isLastPage"
          unelevated
          dense
          color="primary"
          class="col-auto rounded-borders q-mr-xs"
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
      @click="doMicroPage()"
      :unelevated="ctrl.kind === 1 || ctrl.kind === 2"
      :disable="ctrl.kind !== 1 && ctrl.kind !== 2"
      dense
      color="primary"
      class="col-auto rounded-borders q-mr-xs"
      icon="mdi-microscope"
      :title="$t('Microdata view')"
      :aria-label="$t('Microdata view')"
      />
    <!-- calculated measures menu -->
    <q-btn
      :outline="ctrl.kind === 1 || ctrl.kind === 2"
      :unelevated="ctrl.kind !== 1 && ctrl.kind !== 2"
      dense
      color="primary"
      class="col-auto rounded-borders q-mr-xs"
      icon="mdi-function-variant"
      :title="$t('Aggregate microdata')"
      :aria-label="$t('Aggregate microdata')"
      >
      <q-menu>
        <div class="full-width q-px-sm q-mt-sm q-mb-none">
          <q-btn
            @click="onCalcPage()"
            :disable="!groupDimCalc?.length || (!calcEnums.length && (!aggrCalc || !attrCalc.length))"
            v-close-popup
            unelevated
            :label="$t('Apply')"
            no-caps
            color="primary q-mr-sm"
            class="rounded-borders"
            />
          <q-btn
            @click="onCalcEdit()"
            :disable="!groupDimCalc?.length || (!calcEnums.length && (!aggrCalc || !attrCalc.length))"
            v-close-popup
            unelevated
            :label="$t('Edit') + '\u2026'"
            no-caps
            color="primary"
            class="rounded-borders"
            />
        </div>
        <q-list dense>

          <q-item v-if="isRunCompare" clickable>
            <q-item-section>{{ $t('Comparison') }}</q-item-section>
            <q-item-section side><span class="text-no-wrap mono">{{ cmpCalc }} &#x25B6;</span></q-item-section>
            <q-menu anchor="top end" self="top start">
              <q-list dense>
                <q-item
                  v-for="c in compareCalcList"
                  :key="c.code"
                  @click="onCompareToogle(c.code)"
                  clickable
                  >
                  <template v-if="cmpCalc === c.code">
                    <q-item-section class="text-primary"><span><span class="mono check-menu">&check;</span>{{ $t(c.label) }}</span></q-item-section>
                    <q-item-section class="mono text-primary" side>{{ c.code }}</q-item-section>
                  </template>
                  <template v-else>
                    <q-item-section><span><span class="mono check-menu">&nbsp;</span>{{ $t(c.label) }}</span></q-item-section>
                    <q-item-section class="mono" side>{{ c.code }}</q-item-section>
                  </template>
                </q-item>
              </q-list>
            </q-menu>
          </q-item>

          <q-item clickable>
            <q-item-section>{{ $t('Aggregation') }}</q-item-section>
            <q-item-section side><span class="text-no-wrap mono">{{ aggrCalc }} &#x25B6;</span></q-item-section>
            <q-menu anchor="top end" self="top start">
              <q-list dense>
                <q-item
                  v-for="c in aggrCalcList"
                  :key="c.code"
                  @click="onAggregateSet(c.code)"
                  clickable
                  >
                  <template v-if="aggrCalc === c.code">
                    <q-item-section class="text-primary"><span><span class="mono check-menu">&check;</span>{{ $t(c.label) }}</span></q-item-section>
                    <q-item-section class="mono text-primary" side>{{ c.code }}</q-item-section>
                  </template>
                  <template v-else>
                    <q-item-section><span><span class="mono check-menu">&nbsp;</span>{{ $t(c.label) }}</span></q-item-section>
                    <q-item-section class="mono" side>{{ c.code }}</q-item-section>
                  </template>
                </q-item>
              </q-list>
            </q-menu>
          </q-item>

          <q-item clickable>
            <q-item-section>{{ $t('Dimensions') }}</q-item-section>
            <q-item-section side><span class="text-no-wrap mono">{{ ((groupDimCalc?.length) || 0).toString() + '/' + (rank - 1 > 0 ? rank - 1 : 0).toString() }} &#x25B6;</span></q-item-section>
            <q-menu anchor="top end" self="top start">
              <q-list dense>
                <template v-for="(d, n) in dimProp">
                  <template v-if="n > 0 && n < rank">
                    <q-item :key="'by-dim-' + d.name"
                      @click="onGroupByToogle(d.name)"
                      clickable
                      >
                      <template v-if="isGroupBy(d.name)">
                        <q-item-section v-if="!pvc.isShowNames" class="text-primary"><span><span class="mono check-menu">&check;</span>{{ d.label }}</span></q-item-section>
                        <q-item-section v-if="pvc.isShowNames" class="text-primary mono"><span><span class="mono check-menu">&check;</span>{{ d.name }}</span></q-item-section>
                      </template>
                      <template v-else>
                        <q-item-section v-if="!pvc.isShowNames"><span><span class="mono check-menu">&nbsp;</span>{{ d.label }}</span></q-item-section>
                        <q-item-section v-if="pvc.isShowNames" class="mono"><span><span class="mono check-menu">&nbsp;</span>{{ d.name }}</span></q-item-section>
                      </template>
                    </q-item>
                  </template>
                </template>
              </q-list>
            </q-menu>
          </q-item>

          <q-item clickable>
            <q-item-section>{{ $t('Measures') }}</q-item-section>
            <q-item-section side><span class="text-no-wrap mono">{{ (attrCalc?.length || 0).toString() + '/' + attrCount.toString() }} &#x25B6;</span></q-item-section>
            <q-menu anchor="top end" self="top start">
              <q-list dense>
                <template v-for="a in attrEnums">
                  <template v-if="a?.isFloat || a?.isInt">
                    <q-item :key="'c-attr-' + a.name"
                      @click="onCalcAttrToogle(a.name)"
                      clickable
                      >
                      <template v-if="isAttrCalc(a.name)">
                        <q-item-section v-if="!pvc.isShowNames" class="text-primary"><span><span class="mono check-menu">&check;</span>{{ a.label }}</span></q-item-section>
                        <q-item-section v-if="pvc.isShowNames" class="text-primary mono"><span><span class="mono check-menu">&check;</span>{{ a.name }}</span></q-item-section>
                      </template>
                      <template v-else>
                        <q-item-section v-if="!pvc.isShowNames"><span><span class="mono check-menu">&nbsp;</span>{{ a.label }}</span></q-item-section>
                        <q-item-section v-if="pvc.isShowNames" class="mono"><span><span class="mono check-menu">&nbsp;</span>{{ a.name }}</span></q-item-section>
                      </template>
                    </q-item>
                  </template>
                </template>
              </q-list>
            </q-menu>
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
      class="col-auto rounded-borders"
      :icon="!!valueFilter.length ? 'mdi-filter-outline' : 'mdi-filter'"
      :label="!!valueFilter.length ? '[ ' + valueFilter.length.toLocaleString() + ' ]' : ''"
      :title="$t('Filter by values') + (!!valueFilter.length ? ' [ ' + valueFilter.length.toLocaleString() + ' ]' : '\u2026')"
      :aria-label="$t('Filter by values')"
      />

    <q-separator vertical inset spaced="sm" color="secondary" />

    <q-btn
      @click="onCopyToClipboard"
      unelevated
      dense
      color="primary"
      class="col-auto rounded-borders q-mr-xs"
      icon="mdi-content-copy"
      :title="$t('Copy tab separated values to clipboard: ') + 'Ctrl+C'"
      />
    <q-btn
      @click="onDownload"
      unelevated
      dense
      color="primary"
      class="col-auto rounded-borders"
      icon="mdi-download"
      :title="$t('Download') + ' '  + entityName + ' ' + $t('as CSV')"
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
          unelevated
          dense
          color="primary"
          class="col-auto rounded-borders q-mr-xs"
          icon="mdi-view-quilt-outline"
          :title="$t('Switch to default pivot view')"
          />
        <q-btn
          @click="onSetRowColMode(1)"
          :disable="pvc.rowColMode === 1"
          unelevated
          dense
          color="primary"
          class="col-auto rounded-borders q-mr-xs"
          icon="mdi-view-list-outline"
          :title="$t('Table view: hide rows and columns name')"
          />
        <q-btn
          @click="onSetRowColMode(3)"
          :disable="pvc.rowColMode === 3"
          unelevated
          dense
          color="primary"
          class="col-auto rounded-borders q-mr-xs"
          icon="mdi-view-compact-outline"
          :title="$t('Table view: always show rows and columns item and name')"
          />
        <q-btn
          @click="onSetRowColMode(0)"
          :disable="pvc.rowColMode === 0"
          unelevated
          dense
          color="primary"
          class="col-auto rounded-borders q-mr-xs"
          icon="mdi-view-module-outline"
          :title="$t('Table view: always show rows and columns item')"
          />
      </template>

    <template v-if="ctrl.isFloatView">
      <q-btn
        @click="onShowMoreFormat"
        :disable="!ctrl.isMoreView"
        unelevated
        dense
        color="primary"
        class="col-auto rounded-borders q-mr-xs"
        icon="mdi-decimal-increase"
        :title="$t('Increase precision')"
        />
      <q-btn
        @click="onShowLessFormat"
        :disable="!ctrl.isLessView"
        unelevated
        dense
        color="primary"
        class="col-auto rounded-borders q-mr-xs"
        icon="mdi-decimal-decrease"
        :title="$t('Decrease precision')"
        />
    </template>
    <q-btn
      v-if="ctrl.isRawUseView"
      @click="onToggleRawValue"
      :unelevated="!ctrl.isRawView"
      :outline="ctrl.isRawView"
      dense
      color="primary"
      class="col-auto rounded-borders"
      icon="mdi-loupe"
      :title="!ctrl.isRawView ? $t('Show raw source value') : $t('Show formatted value')"
      />
    <q-btn
      @click="onShowItemNames"
      :disable="isScalar"
      :unelevated="!pvc.isShowNames"
      :outline="pvc.isShowNames"
      dense
      color="primary"
      class="col-auto rounded-borders q-ml-xs"
      icon="mdi-label-outline"
      :title="pvc.isShowNames ? $t('Show labels') : $t('Show names')"
      />
    <q-separator vertical inset spaced="sm" color="secondary" />

    <q-btn
      @click="onSaveDefaultView"
      unelevated
      dense
      color="primary"
      class="col-auto rounded-borders q-mr-xs"
      icon="mdi-content-save-cog"
      :title="$t('Save microdata view as default view of') + ' ' + entityName"
      />
    <q-btn
      @click="onReloadDefaultView"
      unelevated
      dense
      color="primary"
      class="col-auto rounded-borders q-mr-xs"
      icon="mdi-cog-refresh-outline"
      :title="$t('Reset microdata view to default and reload') + ' ' + entityName"
      />

    <div
      class="col-auto"
      >
      <span>{{ entityName }}<br />
      <span class="om-text-descr">{{ entityDescr }}</span></span>
    </div>

    </q-toolbar>

  </div>
  <!-- end of output table header -->

  <!-- pivot table controls and view -->
  <div class="q-mx-sm">

    <div v-show="ctrl.isRowColControls"
      class="other-panel"
      >
      <draggable
        v-model="otherFields"
        group="fields"
        :disabled="!!isOtherDropDisabled ? true : null"
        @start="onDrag"
        @end="onDrop"
        @choose="onChoose"
        @unchoose="onUnchoose"
        class="other-fields other-drag"
        :class="{'drag-area-hint': isDragging, 'drag-area-disabled': isOtherDropDisabled}"
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
        @choose="onChoose"
        @unchoose="onUnchoose"
        class="col-fields col-drag"
        :class="{'drag-area-hint': isDragging}"
        >
        <div v-for="f in colFields" :key="f.name" :id="'item-draggable-' + f.name" class="field-drag om-text-medium">
          <template v-if="f.name == 'ENTITY_KEY_DIM'">
            <div
              class="row nowrap col-select"
              :title="pvc.isShowNames ? f.name : f.label"
              >
                <q-icon
                  name="mdi-hand-back-left"
                  size="md"
                  class="col-1 q-px-sm q-mr-sm bg-primary text-white rounded-borders select-handle-move"
                  :title="$t('Drag and drop')"
                  />
                <div id="key-dim-box" class="col-auto om-text-secondary om-text-body2 q-px-sm rounded-borders">
                  <span>{{ (pvc.isShowNames ? f.name : f.label) }}: </span><span class="key-dim-count">{{ f.selection.length.toLocaleString() }}</span>
                </div>
            </div>
          </template>
          <template v-else>
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
          </template>
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
        @choose="onChoose"
        @unchoose="onUnchoose"
        class="row-fields row-drag"
        :class="{'drag-area-hint': isDragging}"
        >
        <div v-for="f in rowFields" :key="f.name" :id="'item-draggable-' + f.name" class="field-drag om-text-medium">
          <template v-if="f.name == 'ENTITY_KEY_DIM'">
            <div
              class="row nowrap col-select"
              :title="pvc.isShowNames ? f.name : f.label"
              >
                <q-icon
                  name="mdi-hand-back-left"
                  size="md"
                  class="col-1 q-px-sm q-mr-sm bg-primary text-white rounded-borders select-handle-move"
                  :title="$t('Drag and drop')"
                  />
                <div id="key-dim-box" class="col-auto om-text-secondary om-text-body2 q-px-sm rounded-borders">
                  <span>{{ (pvc.isShowNames ? f.name : f.label) }}: </span><span class="key-dim-count">{{ f.selection.length.toLocaleString() }}</span>
                </div>
            </div>
          </template>
          <template v-else>
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
          </template>
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
  <entity-info-dialog :show-tickle="entityInfoTickle" :entity-name="entityName" :run-digest="runDigest"></entity-info-dialog>

  <entity-calc-dialog
    :show-tickle="calcEditTickle"
    :update-tickle="calcUpdateTickle"
    :calc-enums="calcEnums"
    @calc-list-apply="onCalcEditApply"
    >
  </entity-calc-dialog>

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

<script src="./entity-page.js"></script>

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

  #key-dim-box {
    min-width: 14rem;
    line-height: 2.5rem;
    border: 1px solid lightgray;
  }
  .key-dim-count  {
    float: right;
    line-height: 2.5rem;
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

  .check-menu {
    min-width: 1.125rem;
    display: inline-block;
  }
  .bar-button-on {
    background-color: $primary;
    color: white;
  }
  .bar-button-off {
    color: $primary;
  }

  .upload-right {
    text-align: right;
  }
  .upload-max-width-10 {
    max-width: 10rem;
  }
</style>
