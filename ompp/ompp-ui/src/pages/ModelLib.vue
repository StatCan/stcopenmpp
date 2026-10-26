<template>
<q-page class="text-body1">

  <div class="row items-center full-width q-pt-sm q-px-sm">

    <q-btn
      v-if="isAnyModelGroup"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-mr-xs om-tree-control-button"
      :icon="!isTreeExpanded ? 'keyboard_arrow_down' :'keyboard_arrow_up'"
      :title="!isTreeExpanded ? $t('Expand all') : $t('Collapse all')"
      @click="doToogleExpandTree"
      />
    <q-btn
      v-if="!!isSortModelTree && !!treeData && treeData?.length > 0"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-mr-xs om-tree-control-button"
      :icon="isDescModelTree ? 'mdi-sort-alphabetical-descending' :'mdi-sort-alphabetical-ascending'"
      :title="isDescModelTree ? $t('Sort Descending') : $t('Sort Ascending')"
      @click="doToogleTreeSort"
      />
    <span class="col-grow">
      <q-input
        ref="filterInput"
        debounce="500"
        v-model="treeFilter"
        outlined
        dense
        :placeholder="$t('Find model...')"
        >
        <template v-slot:append>
          <q-icon v-if="treeFilter !== ''" name="cancel" class="cursor-pointer" @click="resetFilter" />
          <q-icon v-else name="search" />
        </template>
      </q-input>
    </span>

  </div>

  <div class="q-pa-sm">
    <q-tree
      ref="theTree"
      :nodes="treeData"
      node-key="key"
      default-expand-all
      v-model:expanded="expandedKeys"
      :filter="treeFilter"
      :filter-method="doTreeFilter"
      no-transition
      :no-results-label="$t('No models found')"
      :no-nodes-label="$t('Server offline or no models published')"
      >
      <template v-slot:default-header="prop">
        <div
          v-if="!prop.node.digest"
          class="row no-wrap items-center"
          :class="{'om-tree-found-node': treeWalk.keysFound[prop.node.key]}"
          >
          <div class="col">
            <template v-if="treeLabelKind == 'name-only'"><span>{{ prop.node.label }}</span></template>
            <template v-if="treeLabelKind == 'descr-only'"><span>{{ prop.node.descr || prop.node.label }}</span></template>
            <template v-if="treeLabelKind !== 'name-only' && treeLabelKind !== 'descr-only'">
              <span>{{ prop.node.label }}<br />
              <span class="om-text-descr">{{ prop.node.descr }}</span></span>
            </template>
          </div>
        </div>
        <div v-else
          class="row no-wrap items-center full-width om-tree-leaf"
          :class="{'om-tree-found-node': treeWalk.keysFound[prop.node.key]}"
          >
          <q-btn
            @click.stop="doShowModelNote(prop.node.digest)"
            flat
            round
            dense
            padding="xs"
            color="primary"
            class="col-auto"
            icon="mdi-information-outline"
            :title="$t('About') + ' ' + prop.node.label"
            />
          <q-btn
            @click.stop="doModelDocLink(prop.node.digest)"
            :disable="!isModelDocLink(prop.node.digest)"
            flat
            round
            dense
            padding="xs"
            :color="isModelDocLink(prop.node.digest) ? 'primary' : 'secondary'"
            class="col-auto"
            icon="mdi-book-open-outline"
            :title="$t('Model Documentation') + ' ' + prop.node.label"
            />
          <q-btn
            @click.stop="onCopyModel(prop.node.digest, prop.node.label, prop.node.ver)"
            :disable="!serverConfig.ModelLib.IsCopy || serverConfig.IsReadonly || !prop.node?.pubLst"
            flat
            round
            dense
            padding="xs"
            :color="!serverConfig.ModelLib.IsCopy || serverConfig.IsReadonly || !prop.node?.pubLst ? 'secondary' : 'primary'"
            class="col-auto"
            icon="mdi-database-plus"
            :title="$t('Copy') + ' ' + prop.node.label + (prop.node.ver ? ('-' + prop.node.ver) : '')"
            />
          <div class="col om-tree-leaf-link q-ml-xs">
            <template v-if="treeLabelKind == 'name-only'"><span>{{ prop.node.label }}</span></template>
            <template v-if="treeLabelKind == 'descr-only'"><span>{{ prop.node.descr || prop.node.label }}</span></template>
            <template v-if="treeLabelKind !== 'name-only' && treeLabelKind !== 'descr-only'">
              <span>{{ prop.node.label }}<br />
              <span class="om-text-descr">{{ prop.node.descr }}</span></span>
            </template>
          </div>
        </div>
      </template>
    </q-tree>
  </div>

  <div
    ref="logListBox"
    class="q-px-sm q-pt-lg q-pb-sm"
    >
    <q-expansion-item
      v-model="isShowCopyLogs"
      switch-toggle-side
      expand-separator
      header-class="om-section-header"
      class="q-my-sm"
      >

      <template v-slot:header>
        <q-item-section>
          <div class="row no-wrap items-center full-width">
            <q-btn
              @click.stop="refreshLogList()"
              :disable="loadLogListWait"
              flat
              outline
              dense
              no-caps
              :label="logList.length ? logListRefreshTs : ''"
              color="primary"
              icon="mdi-refresh-circle"
              class="col-auto bg-white rounded-borders q-py-none q-px-xs"
              :title="$t('Refresh copy models history')"
              />
            <span
              class="col-auto q-pl-md">{{ $t('Copy Models History') }} {{ logList.length ? (': ' + logList.length.toLocaleString(locale)) : '' }}</span>
          </div>
        </q-item-section>
      </template>

      <q-card>
        <q-card-section>
          <table class="om-p-table">

            <thead>
              <tr>
                <th class="om-p-head-center text-weight-medium"></th>
                <th class="om-p-head-center text-weight-medium">{{ $t('Copy Stamp') }}</th>
                <th class="om-p-head-center text-weight-medium">{{ $t('Model Name-Version') }}</th>
                <th class="om-p-head-center text-weight-medium">{{ $t('Status') }}</th>
                <th class="om-p-head-center text-weight-medium">{{ $t('Log File') }}</th>
              </tr>
            </thead>

            <tbody>
              <template v-for="lg of logList" :key="lg.LogStamp + '-lg-' + lg.BaseName">

                <tr>
                  <td class="om-p-cell-center">
                    <button
                      @click="doToggleCopyLog(lg.LogFileName)"
                      :disabled="loadLogWait"
                      :title="lg.LogFileName === copyLogPath ? $t('Hide model copy log') : $t('Show model copy log')"
                      >
                      <q-icon
                        :name="lg.LogFileName === copyLogPath ? 'mdi-close-circle-outline' : 'mdi-text-long'"
                        size="sm"
                        color="primary" />
                    </button>
                  </td>
                  <td class="om-p-cell-left">{{ fromUnderscoreTs(lg.LogStamp) }}</td>
                  <td class="om-p-cell-left">{{ lg.BaseName }}</td>
                  <td class="om-p-cell-center" :class="!lg.IsError ? '' : 'text-negative'">
                    <span v-if="!lg.IsError">&#x2714;</span>
                    <span v-else>{{ $t('Failed') }}</span>
                  </td>
                  <td class="om-p-cell-left om-text-descr">{{ lg.LogFileName }}</td>
                </tr>

                <tr v-if="lg.LogFileName === copyLogPath">
                  <td colspan="5" class="om-p-cell-left">
                    <div class="row no-wrap items-center full-width log-hdr">
                      <button
                        @click="onToogleRefreshCopyLog(lg.LogFileName)"
                        :disabled="copyLogStat.IsError"
                        :title="isRefreshCopyLog ? $t('Pause auto refresh') : $t('Auto refresh model copy log')"
                        >
                        <q-icon
                          :name="isRefreshCopyLog ? (((refreshCount % 2) === 1) ? 'mdi-autorenew' : 'mdi-sync') : 'mdi-play-circle-outline'"
                          size="sm"
                          color="primary" />
                      </button>
                      <span class="col-auto mono q-pl-md"><i>{{ copyLogStat.LogFileName ? copyLogStat.LogFileName : lg.LogFileName }}<span v-if="copyLogStat.ModTs" class="om-text-descr mono q-pl-sm">[{{ (!copyLogStat.IsError ? $t('updated at:') : $t('failed at:')) + ' ' + fromModTime(copyLogStat.ModTs) }}]</span></i></span>
                    </div>
                    <div v-if="copyLogStat.Lines.length <= 0"
                      class="q-pl-md">
                      <span class="mono">{{ $t('Log file not found or empty') }}</span>
                    </div>
                    <div v-else
                      class="q-pl-md">
                      <pre>{{copyLogStat.Lines.join('\n')}}</pre>
                    </div>
                  </td>
                </tr>

              </template>
            </tbody>

          </table>
        </q-card-section>
      </q-card>

    </q-expansion-item>
  </div>

  <model-info-base-dialog
    :show-tickle="modelInfoTickle"
    :model="modelInfo"
    :doc-link="modelInfoDocLink"
    >
  </model-info-base-dialog>

  <confirm-dialog
    @confirm-yes="onYesCopyModel"
    :show-tickle="showCopyModelDialogTickle"
    :item-id="copyDigest"
    :item-name="copyNameVer"
    :dialog-title="!isModelExist ? $t('Copy model') : $t('Overwrite existing model')"
    :body-text="!isModelExist ? $t('Copy') : $t('Model already exist')"
    :body-note="isModelExist ? $t('Do you want to overwrite existing model?') : ''"
    :icon-name="'mdi-database-plus'"
    >
  </confirm-dialog>

  <q-inner-loading :showing="loadWait || loadLibWait || loadLogListWait || loadLogWait || loadModelListWait">
    <q-spinner-gears size="lg" color="primary" />
  </q-inner-loading>

</q-page>
</template>

<script src="./model-lib.js"></script>

<style lang="scss" scope="local">
  .log-hdr {
    padding-left: 1rem;
  }
</style>
