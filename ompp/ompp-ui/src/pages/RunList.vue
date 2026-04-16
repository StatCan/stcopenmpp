<template>

<div class="text-body1">

  <q-card v-if="isNotEmptyRunCurrent" class="q-ma-sm">

    <div
      class="row reverse-wrap items-center"
      >

      <q-btn
        outline
        dense
        class="col-auto text-primary rounded-borders q-ml-sm"
        icon="menu"
        :title="$t('Menu')"
        :aria-label="$t('Menu')"
        >
        <q-menu auto-close>
          <q-list>

            <q-item
              @click="doShowRunNote(runDigestSelected)"
              clickable
              >
              <q-item-section avatar>
                <q-icon
                  :color="isRunDeleted(runCurrent.Status, runCurrent.Name) ? 'negative' : ((isSuccess(runCurrent.Status) || isInProgress(runCurrent.Status)) ? 'primary' : 'warning')"
                  :name="isSuccess(runCurrent.Status) ? 'mdi-information-outline' : (isInProgress(runCurrent.Status) ? 'mdi-run' : 'mdi-alert-circle-outline')"
                  />
              </q-item-section>
              <q-item-section>{{ isRunDeleted(runCurrent.Status, runCurrent.Name) ? $t('Deleted') : ($t('About')) + ' ' + runCurrent.Name }}</q-item-section>
            </q-item>
            <q-separator />

            <q-item
              :disable="isShowNoteEditor || isCompare || uploadFileSelect || isRunDeleted(runCurrent.Status, runCurrent.Name)"
              @click="onEditRunNote(runDigestSelected)"
              clickable
              >
              <q-item-section avatar>
                <q-icon color="primary" name="mdi-file-document-edit-outline" />
              </q-item-section>
              <q-item-section>{{ $t('Edit notes for') + ' ' + runCurrent.Name }}</q-item-section>
            </q-item>

            <q-item
              :disable="isNewWorksetDisabled()"
              @click="onNewWorksetClick"
              clickable
              >
              <q-item-section avatar>
                <q-icon color="primary" name="mdi-notebook-plus-outline" />
              </q-item-section>
              <q-item-section>{{ paramDiff.length > 0 ? ($t('Create new input scenario with {count} parameter(s) from: ', { count: paramDiff.length }) + firstCompareName) : $t('Create new input scenario') }}</q-item-section>
            </q-item>

            <template v-if="serverConfig.AllowUpload">
              <q-item
                :disable="isShowNoteEditor || isDiskOver"
                @click="doShowFileSelect()"
                v-show="!uploadFileSelect"
                clickable
                >
                <q-item-section avatar>
                  <q-icon color="primary" name="mdi-cloud-upload-outline" />
                </q-item-section>
                <q-item-section>{{ $t('Upload model run .zip') }}</q-item-section>
              </q-item>
              <q-item
                :disable="isShowNoteEditor"
                @click="doCancelFileSelect()"
                v-show="uploadFileSelect"
                clickable
                >
                <q-item-section avatar>
                  <q-icon color="primary" name="mdi-close-circle-outline" />
                </q-item-section>
                <q-item-section>{{ $t('Cancel upload') }}</q-item-section>
              </q-item>
            </template>

          </q-list>
        </q-menu>
      </q-btn>

      <q-btn
        @click="doShowRunNote(runDigestSelected)"
        flat
        dense
        class="col-auto text-white rounded-borders q-ml-xs"
        :class="isRunDeleted(runCurrent.Status, runCurrent.Name) ? 'bg-negative' : ((isSuccess(runCurrent.Status) || isInProgress(runCurrent.Status)) ? 'bg-primary' : 'bg-warning')"
        :icon="isSuccess(runCurrent.Status) ? 'mdi-information' : (isInProgress(runCurrent.Status) ? 'mdi-run' : 'mdi-alert-circle-outline')"
        :title="(isRunDeleted(runCurrent.Status, runCurrent.Name) ? $t('Deleted') : $t('About')) + ' ' + runCurrent.Name"
        />
      <q-separator vertical inset spaced="sm" color="secondary" />

      <q-btn
        :disable="isShowNoteEditor || isCompare || uploadFileSelect || isRunDeleted(runCurrent.Status, runCurrent.Name)"
        @click="onEditRunNote(runDigestSelected)"
        flat
        dense
        class="col-auto text-white rounded-borders"
        :class="(isSuccess(runCurrent.Status) || isInProgress(runCurrent.Status)) ? 'bg-primary' : 'bg-warning'"
        icon="mdi-file-document-edit-outline"
        :title="$t('Edit notes for') + ' ' + runCurrent.Name"
        />
      <q-btn
        :disable="isNewWorksetDisabled()"
        @click="onNewWorksetClick"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders q-ml-xs"
        icon="mdi-notebook-plus"
        :title="paramDiff.length > 0 ? ($t('Create new input scenario with {count} parameter(s) from: ', { count: paramDiff.length }) + firstCompareName) : $t('Create new input scenario')"
        />

      <template v-if="serverConfig.AllowUpload">
        <q-btn
          :disable="isShowNoteEditor || isDiskOver"
          @click="doShowFileSelect()"
          v-show="!uploadFileSelect"
          flat
          dense
          class="col-auto text-white rounded-borders q-ml-xs bg-primary text-white rounded-borders"
          icon='mdi-cloud-upload'
          :title="$t('Upload model run .zip')"
          />
        <q-btn
          @click="doCancelFileSelect()"
          v-show="uploadFileSelect"
          flat
          dense
          class="col-auto text-white rounded-borders q-ml-xs bg-primary text-white rounded-borders"
          icon='mdi-close-circle'
          :title="$t('Cancel upload')"
          />
      </template>

      <span class="col-auto no-wrap tab-switch-container q-ml-xs">
        <q-btn
          @click="onToogleShowParamTree"
          no-caps
          unelevated
          dense
          color="primary"
          class="rounded-borders tab-switch-button"
          :class="{ 'om-bg-inactive' : !isParamTreeShow }"
          >
          <q-icon :name="isParamTreeShow ? 'keyboard_arrow_up' : 'keyboard_arrow_down'" />
          <span>{{ $t('Parameters') }}</span>
          <q-badge outline class="q-ml-sm q-mr-xs">{{ paramVisibleCount }}</q-badge>
          <q-separator
            vertical dark v-if="isCompare && paramDiff.length > 0"
            />
          <span
            v-if="isCompare && paramDiff.length > 0"
            >
            <q-icon name="mdi-not-equal-variant"/><q-badge outline class="q-mx-xs">{{ paramDiff.length }}</q-badge>
          </span>
        </q-btn>
      </span>

      <span class="col-auto no-wrap tab-switch-container">
        <q-btn
          :disable="!isSuccess(runCurrent.Status)"
          @click="onToogleShowTableTree"
          no-caps
          unelevated
          dense
          color="primary"
          class="rounded-borders tab-switch-button"
          :class="{ 'om-bg-inactive' : !isTableTreeShow }"
          >
          <q-icon :name="isTableTreeShow ? 'keyboard_arrow_up' : 'keyboard_arrow_down'" />
          <span>{{ $t('Output Tables') }}</span>
          <q-badge outline class="q-ml-sm q-mr-xs">{{ tableVisibleCount }}</q-badge>
          <q-separator
            vertical dark v-if="isCompare && tableDiff.length > 0"
            />
          <span
            v-if="isCompare && tableDiff.length > 0"
            >
            <q-icon name="mdi-not-equal-variant"/><q-badge outline class="q-mx-xs">{{ tableDiff.length }}</q-badge>
          </span>
        </q-btn>
      </span>

      <span v-if="isMicrodata" v-show="runCurrent.Entity.length > 0" class="col-auto no-wrap tab-switch-container">
        <q-btn
          :disable="!isSuccess(runCurrent.Status)"
          @click="onToogleShowEntityTree"
          no-caps
          unelevated
          dense
          color="primary"
          class="rounded-borders tab-switch-button"
          :class="{ 'om-bg-inactive' : !isEntityTreeShow }"
          >
          <q-icon :name="isEntityTreeShow ? 'keyboard_arrow_up' : 'keyboard_arrow_down'" />
          <span>{{ $t('Microdata') }}</span>
          <q-badge outline class="q-ml-sm q-mr-xs">{{ entityVisibleCount }}</q-badge>
          <q-separator
            vertical dark v-if="isCompare && entityDiff.length > 0"
            />
          <span
            v-if="isCompare && entityDiff.length > 0"
            >
            <q-icon name="mdi-not-equal-variant"/><q-badge outline class="q-mx-xs">{{ entityDiff.length }}</q-badge>
          </span>
        </q-btn>
      </span>

      <transition
        enter-active-class="animated fadeIn"
        leave-active-class="animated fadeOut"
        mode="out-in"
        >
        <div
          :key="runDigestSelected"
          class="col-auto q-ml-xs"
          >
          <span>{{ runCurrent.Name }}<br />
          <span class="om-text-descr"><span class="mono q-pr-sm">{{ dateTimeStr(runCurrent.UpdateDateTime) }}</span>{{ descrRunCurrent }}</span></span>
        </div>
      </transition>

    </div>

    <q-card-section v-show="isParamTreeShow" class="q-px-sm q-pt-none">

      <run-parameter-list
        :run-digest="runDigestSelected"
        :refresh-tickle="refreshTickle"
        :refresh-param-tree-tickle="refreshParamTreeTickle"
        :name-filter="paramDiff"
        :is-param-leaf-click="true"
        :is-on-off-not-in-list="true"
        :is-param-download="true"
        in-list-icon="mdi-not-equal-variant"
        in-list-on-label="Show only different parameters"
        in-list-off-label="Show all parameters"
        @run-parameter-select="onRunParamClick"
        @run-parameter-info-show="doShowParamNote"
        @run-parameter-group-info-show="doShowGroupNote"
        @run-parameter-tree-updated="onParamTreeUpdated"
        >
      </run-parameter-list>

    </q-card-section>

    <q-card-section v-show="isTableTreeShow" class="q-px-sm q-pt-none">

      <table-list
        :run-digest="runDigestSelected"
        :refresh-tickle="refreshTickle"
        :refresh-table-tree-tickle="refreshTableTreeTickle"
        :name-filter="tableDiff"
        :is-table-leaf-click="true"
        :is-on-off-not-in-list="true"
        :is-table-download="true"
        in-list-icon="mdi-not-equal-variant"
        in-list-on-label="Show only different output tables"
        in-list-off-label="Show all output tables"
        @table-select="onTableLeafClick"
        @table-info-show="doShowTableNote"
        @table-group-info-show="doShowGroupNote"
        @table-tree-updated="onTableTreeUpdated"
        >
      </table-list>

    </q-card-section>

    <q-card-section v-if="isMicrodata" v-show="isEntityTreeShow" class="q-px-sm q-pt-none">

      <entity-list
        :run-digest="runDigestSelected"
        :refresh-tickle="refreshTickle"
        :refresh-entity-tree-tickle="refreshEntityTreeTickle"
        :name-filter="entityAttrsUse"
        :is-entity-click="true"
        :is-in-list-enable="isCompare"
        :is-entity-download="true"
        in-list-icon="mdi-not-equal-variant"
        in-list-on-label="Show only different microdata entities"
        in-list-off-label="Show all microdata entities"
        @entity-select="onEntityClick"
        @entity-info-show="doShowEntityRunNote"
        @entity-attr-info-show="doShowEntityAttrNote"
        @entity-group-info-show="doShowEntityGroupNote"
        @entity-tree-updated="onEntityTreeUpdated"
        >
      </entity-list>

    </q-card-section>

  </q-card>

  <q-card
    v-if="isNewWorksetShow"
    bordered
    class="primary-border-025 q-ma-sm"
    >
    <new-workset
      @save-new-set="onNewWorksetSave"
      @cancel-new-set="onNewWorksetCancel"
      class="q-pa-sm"
      >
    </new-workset>
  </q-card>

  <q-card v-if="uploadFileSelect">
    <div class="row q-mt-xs q-pa-sm">
      <q-btn
        @click="onUploadRun"
        v-if="uploadFileSelect"
        :disable="!fileSelected"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders"
        icon="mdi-upload"
        :title="$t('Upload model run .zip')"
        />
      <q-file
        v-model="uploadFile"
        v-if="uploadFileSelect"
        accept='.zip'
        outlined
        dense
        clearable
        hide-bottom-space
        class="col q-pl-xs"
        color="primary"
        :label="$t('Select model run .zip for upload')"
        >
      </q-file>
    </div>
  </q-card>

  <q-card
    v-if="isShowNoteEditor"
    bordered
    class="primary-border-025 q-ma-sm"
    >
    <q-card-section class="q-pa-sm">

      <div class="row items-center full-width">
        <div class="col section-title bg-primary text-white q-pl-md"><span>{{ $t('Description and Notes') }}</span></div>
      </div>

      <markdown-editor
        v-if="isShowNoteEditor"
        :the-key="noteEditorLangCode"
        :the-name="runCurrent.Name"
        :the-descr="runCurrentDescr()"
        :the-note="runCurrentNote()"
        :description-editable="true"
        :notes-editable="true"
        :lang-code="noteEditorLangCode"
        @cancel-note="onCancelRunNote"
        @save-note="onSaveRunNote"
        class="q-pa-sm"
      >
      </markdown-editor>

    </q-card-section>
  </q-card>

  <q-card class="q-ma-sm">
    <div class="q-pa-sm">
      <q-tree
        ref="runTree"
        default-expand-all
        :nodes="runTreeData"
        node-key="key"
        tick-strategy="leaf-filtered"
        no-transition
        no-connectors
        v-model:expanded="runTreeExpanded"
        v-model:ticked="runTreeTicked"
        :filter="runFilter"
        :filter-method="doRunTreeFilter"
        :no-results-label="$t('No model runs found')"
        :no-nodes-label="$t('No model runs published (or server offline)')"
        >
        <template v-slot:default-header="prop">

          <div
            v-if="prop.node.key === 'rtl-top-node'"
            class="row no-wrap items-center full-width"
            >
            <div class="col-auto">
              <q-btn
                :disable="!runTreeTicked.length"
                @click.stop="onRunMultipleDelete"
                :round="!runTreeTicked.length"
                :outline="!runTreeTicked.length"
                :rounded="!!runTreeTicked.length"
                no-caps
                color="primary"
                padding="xs"
                class="col-auto"
                :icon="!!runTreeTicked.length ? 'mdi-delete' : 'mdi-delete-outline'"
                :label="!!runTreeTicked.length ? '[ ' + runTreeTicked.length.toLocaleString() + ' ]' : ''"
                :title="$t('Delete') + (!!runTreeTicked.length ? ' [ ' + runTreeTicked.length.toLocaleString() + ' ]' : '\u2026')"
                />
            </div>
            <span
              @click.stop="() => {}"
              class="col-grow q-pl-sm"
              >
              <q-input
                ref="runFilterInput"
                debounce="500"
                v-model="runFilter"
                outlined
                dense
                :placeholder="$t('Find model run...')"
                >
                <template v-slot:append>
                  <q-icon v-if="runFilter !== ''" name="cancel" class="cursor-pointer" @click="resetRunFilter" />
                  <q-icon v-else name="search" />
                </template>
              </q-input>
            </span>
          </div>

          <div v-else
            @click="onRunLeafClick(prop.node.digest)"
            class="row no-wrap items-center full-width cursor-pointer om-tree-leaf"
            :class="{ 'text-primary' : prop.node.digest === runDigestSelected }"
            >
            <q-btn
              @click.stop="doShowRunNote(prop.node.digest)"
              :flat="prop.node.digest !== runDigestSelected"
              :outline="prop.node.digest === runDigestSelected"
              round
              dense
              :color="isRunDeleted(prop.node.status, prop.node.label) ? 'negative' : (isSuccess(prop.node.status) || isInProgress(prop.node.status) ? 'primary' : 'warning')"
              padding="xs"
              class="col-auto"
              :icon="isSuccess(prop.node.status) ? (prop.node.digest === runDigestSelected ? 'mdi-information' : 'mdi-information-outline') : (isInProgress(prop.node.status) ? 'mdi-run' : 'mdi-alert-circle-outline')"
              :title="(isRunDeleted(prop.node.status, prop.node.label) ? $t('Deleted') : $t('About')) + ' ' + prop.node.label"
              />
            <q-btn
              :disable="!prop.node.stamp"
              @click.stop="doRunLogClick(prop.node.stamp)"
              flat
              round
              dense
              :color="!!prop.node.stamp ? 'primary' : 'secondary'"
              padding="xs"
              class="col-auto"
              icon="mdi-text-long"
              :title="$t('Run Log: ') + prop.node.label"
              />
            <q-btn
              :disable="!isSuccess(prop.node.status) || prop.node.digest === runDigestSelected || isRunDeleted(prop.node.status, prop.node.label)"
              @click.stop="onRunCompareClick(prop.node.digest)"
              flat
              round
              dense
              padding="xs"
              class="col-auto"
              :class="(!isSuccess(prop.node.status) || prop.node.digest === runDigestSelected) ? 'text-secondary' : (isDigestCompare(prop.node.digest) ? 'text-white bg-primary' : 'text-primary')"
              icon="mdi-not-equal-variant"
              :title="(isDigestCompare(prop.node.digest) ? $t('Clear run comparison with') : $t('Compare this model run with')) + ' ' + runCurrent.Name"
              />
            <q-btn
              :disable="!prop.node.digest"
              @click.stop="onRunDelete(prop.node.label, prop.node.digest, prop.node.status)"
              flat
              round
              dense
              :color="prop.node.digest ? 'primary' : 'secondary'"
              padding="xs"
              class="col-auto"
              icon="mdi-delete-outline"
              :title="$t('Delete: ') + prop.node.label"
              />
            <q-btn
              v-if="serverConfig.AllowDownload"
              :disable="!isSuccess(prop.node.status) || isRunDeleted(prop.node.status, prop.node.label)"
              @click.stop="onDownloadRun(prop.node.digest)"
              flat
              round
              dense
              :color="isSuccess(prop.node.status) ? 'primary' : 'secondary'"
              padding="xs"
              class="col-auto"
              icon="mdi-download-circle-outline"
              :title="$t('Download') + ' ' + prop.node.label"
              />
            <div class="col q-ml-xs">
              <span><span :class="{ 'text-bold': prop.node.digest === runDigestSelected }">{{ prop.node.label }}</span><br />
              <span
                :class="prop.node.digest === runDigestSelected ? 'om-text-descr-selected' : 'om-text-descr'"
                >
                <span class="mono q-pr-sm">{{ prop.node.lastTime }}</span>{{ prop.node.descr }}</span></span>
            </div>
          </div>

        </template>
      </q-tree>
    </div>
  </q-card>

  <run-info-dialog :show-tickle="runInfoTickle" :model-digest="digest" :run-digest="runInfoDigest"></run-info-dialog>
  <parameter-info-dialog :show-tickle="paramInfoTickle" :param-name="paramInfoName" :run-digest="runDigestSelected"></parameter-info-dialog>
  <table-info-dialog :show-tickle="tableInfoTickle" :table-name="tableInfoName" :run-digest="runDigestSelected"></table-info-dialog>
  <group-info-dialog :show-tickle="groupInfoTickle" :group-name="groupInfoName"></group-info-dialog>
  <entity-info-dialog :show-tickle="entityInfoTickle"  :entity-name="entityInfoName" :run-digest="runDigestSelected"></entity-info-dialog>
  <entity-attr-info-dialog :show-tickle="attrInfoTickle" :entity-name="entityInfoName" :attr-name="attrInfoName"></entity-attr-info-dialog>
  <entity-group-info-dialog :show-tickle="entityGroupInfoTickle"  :entity-name="entityInfoName" :group-name="entityGroupInfoName"></entity-group-info-dialog>

  <refresh-run v-if="(digest || '') !== ''"
    :model-digest="digest"
    :run-digest="runDigestRefresh"
    :refresh-tickle="refreshTickle"
    :refresh-run-tickle="refreshRunTickle"
    @done="doneRunLoad"
    @wait="loadRunWait = true"
    >
  </refresh-run>
  <refresh-run-array v-if="(digest || '') !== ''"
    :model-digest="digest"
    :run-digest-array="compareDigestArray"
    :refresh-tickle="refreshTickle"
    :refresh-all-tickle="refreshRunArrayTickle"
    @done="doneRunArrayLoad"
    @wait="loadRunArrayWait = true"
    >
  </refresh-run-array>
  <delete-confirm-dialog
    @delete-yes="onYesRunDelete"
    :show-tickle="showDeleteDialogTickle"
    :item-name="runNameToDelete"
    :item-id="runDigestToDelete"
    :bodyText="isInProgress(runStatusToDelete) ? $t('This model run is NOT completed.') : ''"
    :dialog-title="$t('Delete model run') + '?'"
    >
  </delete-confirm-dialog>
  <delete-confirm-dialog
    @delete-yes="onYesRunMultipleDelete"
    :show-tickle="showDeleteMultipleDialogTickle"
    :item-name="'[ ' + runMultipleCount.toLocaleString() + ' ] ' + $t('model runs to be deleted')"
    :item-id="!!runMultipleCount ? runMultipleCount.toLocaleString() : 'empty'"
    :bodyText="$t('Model may be unavaliable until delete is completed.')"
    :dialog-title="$t('Delete multiple model runs') + '?'"
    >
  </delete-confirm-dialog>

  <create-workset
    :create-now="isCreateWorksetNow"
    :model-digest="digest"
    :new-name="nameOfNewWorkset"
    :current-run-digest="copyDigestNewWorkset"
    :copy-from-run="copyParamNewWorkset"
    :descr-notes="newDescrNotes"
    @done="doneWorksetCreate"
    @wait="loadWorksetCreate = true"
    >
  </create-workset>

  <q-inner-loading :showing="loadWait || loadRunWait || loadRunArrayWait || loadWorksetCreate">
    <q-spinner-gears size="md" color="primary" />
  </q-inner-loading>

</div>
</template>

<script src="./run-list.js"></script>

<style lang="scss" scope="local">
  .tab-switch-container {
    margin-right: 1px;
  }
  .tab-switch-button {
    border-top-right-radius: 1rem;
  }
  .primary-border-025 {
    border: 0.25rem solid $primary;
  }
</style>
