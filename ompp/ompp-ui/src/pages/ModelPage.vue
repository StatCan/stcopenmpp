<template>
<q-page class="text-body1 q-pt-xs">

  <q-card
    v-if="runDigestSelected || worksetNameSelected"
    class="row q-mb-sm q-mx-sm q-pl-sm"
    >
      <transition
        enter-active-class="animated fadeIn"
        leave-active-class="animated fadeOut"
        mode="out-in"
        >
        <span :key="runDigestSelected">
        <run-bar
          v-if="runDigestSelected"
          :model-digest="digest"
          :run-digest="runDigestSelected"
          :refresh-run-tickle="refreshTickle"
          @run-info-click="doShowRunNote"
          >
        </run-bar>
        </span>
      </transition>
    <q-separator v-if="runDigestSelected && worksetNameSelected" vertical inset spaced="lg" color="secondary" />
      <transition
        enter-active-class="animated fadeIn"
        leave-active-class="animated fadeOut"
        mode="out-in"
        >
        <span :key="worksetNameSelected">
        <workset-bar
          v-if="worksetNameSelected"
          :model-digest="digest"
          :workset-name="worksetNameSelected"
          :refresh-workset-tickle="refreshTickle"
          @set-info-click="doShowWorksetNote"
          >
        </workset-bar>
        </span>
      </transition>
  </q-card>

  <div class="row no-wrap full-width q-pl-sm">

    <q-tabs
      outside-arrows
      no-caps
      align="left"
      dense
      inline-label
      shrink
      indicator-color="transparent"
      class="col inline"
      >
      <span
        v-for="tb in tabItems"
        :key="tb.path"
        class="row no-wrap om-tab-hdr text-white"
        :class="{'om-bg-inactive': tb.path !== activeTabKey}"
        >
        <q-route-tab
          :name="tb.path"
          :to="tb.path"
          exact
          class="full-width q-pa-xs"
          >
          <span class="row inline full-width no-wrap justify-start">
            <q-icon v-if="tb.kind === 'run-list'" name="mdi-folder-table" size="sm" class="self-center q-pr-xs"/>
            <q-icon v-if="tb.kind === 'set-list'" name="mdi-folder-edit" size="sm" class="self-center q-pr-xs"/>
            <q-icon v-if="tb.kind === 'new-run'" name="mdi-run" size="sm" class="self-center q-pr-xs"/>
            <q-icon v-if="tb.kind === 'run-parameter'" name="mdi-table-arrow-left" size="sm" class="self-center q-pr-xs"/>
            <q-icon v-if="!tb.updated && tb.kind === 'set-parameter'" name="mdi-table-edit" size="sm" class="self-center q-pr-xs"/>
            <q-icon v-if="tb.updated && tb.kind === 'set-parameter'" name="mdi-content-save-edit" size="sm" class="self-center q-pr-xs"/>
            <q-icon v-if="tb.kind === 'table'" name="mdi-table-large" size="sm" class="self-center q-pr-xs"/>
            <q-icon v-if="tb.kind === 'entity'" name="mdi-microscope" size="sm" class="self-center q-pr-xs"/>
            <q-icon v-if="tb.kind === 'run-log'" name="mdi-text-long" size="sm" class="self-center q-pr-xs"/>
            <q-icon v-if="tb.kind === 'updown-list'" name="mdi-download-circle-outline" size="sm" class="self-center q-pr-xs"/>
            <span class="col-shrink om-tab-title" :title="tabTitle(tb.kind, tb.routeParts)">{{ tabTitle(tb.kind, tb.routeParts) }}</span>
            <q-badge v-if="tb.kind === 'run-list'" transparent outline class="q-ml-xs">{{ runTextCount }}</q-badge>
            <q-badge v-if="tb.kind === 'set-list'" transparent outline class="q-ml-xs">{{ worksetTextCount }}</q-badge>
            <q-icon
              v-if="!tb.updated && !!tabItems && tabItems.length > 1"
              @click="onTabCloseClick(tb.path, $event)"
              name="mdi-close"
              class="q-ml-md"
              :title="$t('close')"
              />
          </span>
        </q-route-tab>
      </span>

    </q-tabs>

    <q-btn
      :disable="isEmptyTabList"
      flat
      dense
      class="col-auto bg-primary text-white"
      icon="more_vert"
      :title="$t('More')"
      :aria-label="$t('More')"
      >
      <q-menu auto-close>
        <q-list dense>
          <q-item
            v-for="tb in tabItems"
            :key="tb.path"
            :name="tb.path"
            :to="tb.path"
            clickable
            >
            <q-item-section avatar>
              <q-icon v-if="tb.kind === 'run-list'" name="mdi-folder-table-outline" />
              <q-icon v-if="tb.kind === 'set-list'" name="mdi-folder-edit-outline" />
              <q-icon v-if="tb.kind === 'new-run'" name="mdi-run" />
              <q-icon v-if="tb.kind === 'run-parameter'" name="mdi-table-arrow-left" />
              <q-icon v-if="!tb.updated && tb.kind === 'set-parameter'" name="mdi-table-edit" />
              <q-icon v-if="tb.updated && tb.kind === 'set-parameter'" name="mdi-content-save-edit" />
              <q-icon v-if="tb.kind === 'table'" name="mdi-table-large" />
              <q-icon v-if="tb.kind === 'entity'" name="mdi-microscope" />
              <q-icon v-if="tb.kind === 'run-log'" name="mdi-text-long" />
              <q-icon v-if="tb.kind === 'updown-list'" name="mdi-download-circle-outline" />
            </q-item-section>
            <q-item-section>
              <q-item-label lines="1" :title="tabTitle(tb.kind, tb.routeParts)">{{ tabTitle(tb.kind, tb.routeParts) }}</q-item-label>
            </q-item-section>
          </q-item>
        </q-list>
      </q-menu>
    </q-btn>

  </div>

  <router-view
    :refresh-tickle="refreshTickle"
    :to-up-down-section="toUpDownSection"
    @tab-mounted="onTabMounted"
    @run-select="onRunSelect"
    @set-select="onWorksetSelect"
    @run-parameter-select="onRunParamSelect"
    @set-parameter-select="onSetParamSelect"
    @table-select="onTableSelect"
    @entity-select="onEntitySelect"
    @set-update-readonly="onWorksetReadonlyUpdate"
    @edit-updated="onEditUpdated"
    @run-log-select="onRunLogSelect"
    @run-job-select="onRunJobSelect"
    @new-run-select="onNewRunSelect"
    @download-select="onDownloadSelect"
    @upload-select="onUploadSelect"
    @run-completed-list="onRunCompletedList"
    @user-view-updated="onUserViewUpdated"
    @run-list-refresh="onRunListRefresh"
    @set-list-refresh="onWorksetListRefresh"
    @run-list-delete="onRunListDelete"
    @set-list-delete="onWorksetListDelete"
    @set-parameter-delete="onWorksetParamDelete"
    >
  </router-view>

  <refresh-model
    :digest="digest"
    :refresh-tickle="refreshTickle"
    @done="doneModelLoad"
    @wait="loadModelDone = false"
    >
  </refresh-model>
  <refresh-run-list
    :digest="digest"
    :refresh-tickle="refreshTickle"
    :refresh-run-list-tickle="refreshRunListTickle"
    @done="doneRunListLoad"
    @wait="loadRunListDone = false"
    >
  </refresh-run-list>
  <refresh-run
    :model-digest="digest"
    :run-digest="runDnsCurrent"
    :refresh-tickle="refreshTickle"
    :refresh-run-tickle="refreshRunTickle"
    @done="doneRunLoad"
    @wait="loadRunDone = false"
    >
  </refresh-run>
  <refresh-run-array v-if="(digest || '') !== ''"
    :model-digest="digest"
    :run-digest-array="runViewsArray"
    :refresh-tickle="refreshTickle"
    :refresh-all-tickle="refreshRunViewsTickle"
    @done="doneRunViewsLoad"
    @wait="loadRunViewsDone = false"
    >
  </refresh-run-array>
  <refresh-workset-list
    :digest="digest"
    :refresh-tickle="refreshTickle"
    :refresh-workset-list-tickle="refreshWsListTickle"
    @done="doneWsListLoad"
    @wait="loadWsListDone = false">
  </refresh-workset-list>
  <refresh-workset
    :model-digest="digest"
    :workset-name="wsNameCurrent"
    :refresh-tickle="refreshTickle"
    :refresh-workset-tickle="refreshWsTickle"
    :is-new-run="refreshWsToRun"
    @done="doneWsLoad"
    @wait="loadWsDone = false">
  </refresh-workset>
  <refresh-workset-array v-if="(digest || '') !== ''"
    :model-digest="digest"
    :workset-name-array="wsViewsArray"
    :refresh-tickle="refreshTickle"
    :refresh-all-tickle="refreshWsViewsTickle"
    @done="doneWsViewsLoad"
    @wait="loadWsViewsDone = false"
    >
  </refresh-workset-array>
  <refresh-user-views v-if="loadModelDone"
    :model-name="modelName"
    @done="doneUserViewsLoad"
    @wait="loadUserViewsDone = false"
    >
  </refresh-user-views>

  <run-info-dialog :show-tickle="runInfoTickle" :model-digest="digest" :run-digest="runDnsCurrent"></run-info-dialog>
  <workset-info-dialog :show-tickle="worksetInfoTickle" :model-digest="digest" :workset-name="wsNameCurrent"></workset-info-dialog>

  <q-dialog v-model="showAllDiscardDlg">
    <q-card>

      <q-card-section class="text-h6 bg-primary text-white">{{ modelName }}</q-card-section>

      <q-card-section class="q-pt-none text-body1">
        <div>{{ $t('Discard all changes?') }}</div>
        <div>{{ $t('You have {count} unsaved parameter(s)', { count: paramEditCount }) }}</div>
      </q-card-section>

      <q-card-actions align="right">
        <q-btn flat :label="$t('No')" color="primary" v-close-popup autofocus />
        <q-btn flat :label="$t('Yes')" color="primary" v-close-popup @click="onYesDiscardChanges" />
      </q-card-actions>

    </q-card>
  </q-dialog>

  <q-inner-loading
    :showing="!loadModelDone || !loadRunDone || !loadRunListDone || !loadRunViewsDone || !loadWsDone || !loadWsListDone || !loadWsViewsDone || !loadUserViewsDone"
    >
    <q-spinner-gears size="lg" color="primary" />
  </q-inner-loading>

</q-page>
</template>

<script src="./model-page.js"></script>

<style lang="scss" scope="local">
  .om-tab-title {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .om-tab-hdr {
    border-top-right-radius: 1rem;
    background-color: $primary;
    margin-right: 1px;
  }
</style>
