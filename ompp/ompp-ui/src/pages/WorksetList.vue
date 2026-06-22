<template>
<div class="text-body1">

  <q-card v-if="isNotEmptyWorksetCurrent" class="q-ma-sm">

    <div
      class="row reverse-wrap items-center"
      >

      <!-- menu -->
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
              @click="doShowWorksetNote(worksetNameSelected)"
              clickable
              >
              <q-item-section avatar>
                <q-icon
                  color="primary"
                  name="mdi-information-outline"
                  />
              </q-item-section>
              <q-item-section>{{ $t('About') + ' ' + worksetNameSelected }}</q-item-section>
            </q-item>
            <q-separator />

            <q-item
              :disable="isReadonlyServer || uploadFileSelect || isReadonlyWorksetCurrent || isEdit()"
              @click="onShowNoteEditor"
              clickable
              >
              <q-item-section avatar>
                <q-icon color="primary" name="mdi-file-document-edit-outline" />
              </q-item-section>
              <q-item-section>{{ $t('Edit notes for') + ' ' + worksetNameSelected }}</q-item-section>
            </q-item>
            <q-item
              :disable="isReadonlyServer || uploadFileSelect || isEdit()"
              @click="onNewWorksetClick"
              clickable
              >
              <q-item-section avatar>
                <q-icon color="primary" name="mdi-notebook-plus-outline" />
              </q-item-section>
              <q-item-section>{{ $t('Create new input scenario') }}</q-item-section>
            </q-item>

            <template v-if="serverConfig.AllowUpload && !isReadonlyServer">
              <q-item
                :disable="isNewWorksetShow || isShowNoteEditor || isDiskOver"
                @click="doShowFileSelect()"
                v-show="!uploadFileSelect"
                clickable
                >
                <q-item-section avatar>
                  <q-icon color="primary" name="mdi-cloud-upload-outline" />
                </q-item-section>
                <q-item-section>{{ $t('Upload scenario .zip') }}</q-item-section>
              </q-item>
              <q-item
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
            <q-separator />

            <q-item
              :disable="isReadonlyServer && isEdit() || isReadonlyWorksetCurrent || !isRunSuccess"
              @click="onFromRunShow"
              clickable
              >
              <q-item-section avatar>
                <q-icon color="primary" name="mdi-table-arrow-left" />
              </q-item-section>
              <q-item-section>{{ $t('Copy parameters from model run') + (isRunSuccess ? ' ' + runCurrent.Name : '') }}</q-item-section>
            </q-item>
            <q-item
              :disable="isReadonlyServer || isEdit() || isReadonlyWorksetCurrent || !isNotEmptyFrom || !isReadonlyFrom"
              @click="onFromWorksetShow"
              clickable
              >
              <q-item-section avatar>
                <q-icon color="primary" name="mdi-table-plus" />
              </q-item-section>
              <q-item-section>{{ $t('Copy parameters from scenario') + ' ' + ((worksetNameFrom && worksetNameFrom !== worksetNameSelected) ? worksetNameFrom : $t('Source scenario not selected')) }}</q-item-section>
            </q-item>
            <q-separator />

            <q-item
              @click="onNewRunClick(worksetNameSelected)"
              :disable="!isReadonlyWorksetCurrent || isReadonlyServer"
              clickable
              >
              <q-item-section avatar>
                <q-icon color="primary" name="mdi-run" />
              </q-item-section>
              <q-item-section>{{ $t('Run the Model using Scenario') + ' ' +  worksetNameSelected }}</q-item-section>
            </q-item>
            <q-item
              @click="onWorksetReadonlyToggle"
              :disable="isReadonlyServer || isEdit() || !isNotEmptyWorksetCurrent"
              clickable
              >
              <q-item-section avatar>
                <q-icon color="primary" :name="(!isNotEmptyWorksetCurrent || isReadonlyWorksetCurrent) ? 'mdi-lock' : 'mdi-lock-open-variant'" />
              </q-item-section>
              <q-item-section>{{ ((!isNotEmptyWorksetCurrent || isReadonlyWorksetCurrent) ? $t('Open to edit scenario') : $t('Close to run scenario')) + ' ' + worksetCurrent.Name }}</q-item-section>
            </q-item>

          </q-list>
        </q-menu>
      </q-btn>
      <!-- end of menu -->

      <q-btn
        @click="doShowWorksetNote(worksetNameSelected)"
        flat
        dense
        class="col-auto text-white bg-primary rounded-borders q-ml-xs"
        icon="mdi-information"
        :title="$t('About') + ' ' + worksetNameSelected"
        />
      <q-separator vertical inset spaced="sm" color="secondary" />

      <q-btn
        :disable="isReadonlyServer || uploadFileSelect || isReadonlyWorksetCurrent || isEdit()"
        @click="onShowNoteEditor"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders"
        icon="mdi-file-document-edit-outline"
        :title="$t('Edit notes for') + ' ' + worksetNameSelected"
        />

      <q-btn
        :disable="isReadonlyServer || uploadFileSelect || isEdit()"
        @click="onNewWorksetClick"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders q-ml-xs"
        icon="mdi-notebook-plus"
        :title="$t('Create new input scenario')"
       />

      <template v-if="serverConfig.AllowUpload && !isReadonlyServer">
        <q-btn
          :disable="isNewWorksetShow || isShowNoteEditor || isDiskOver"
          @click="doShowFileSelect()"
          v-show="!uploadFileSelect"
          flat
          dense
          class="col-auto text-white rounded-borders q-ml-xs bg-primary text-white rounded-borders"
          icon='mdi-cloud-upload'
          :title="$t('Upload scenario .zip')"
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
      <q-separator vertical inset spaced="sm" color="secondary" />

      <q-btn
        :disable="isReadonlyServer && isEdit() || isReadonlyWorksetCurrent || !isRunSuccess"
        @click="onFromRunShow"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders"
        icon="mdi-table-arrow-left"
        :title="$t('Copy parameters from model run') + (isRunSuccess ? ' ' + runCurrent.Name : '')"
       />
      <q-btn
        :disable="isReadonlyServer && isEdit() || isReadonlyWorksetCurrent || !isNotEmptyFrom || !isReadonlyFrom"
        @click="onFromWorksetShow"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders q-ml-xs"
        icon="mdi-table-plus"
        :title="$t('Copy parameters from scenario') + ' ' + ((worksetNameFrom && worksetNameFrom !== worksetNameSelected) ? worksetNameFrom : $t('Source scenario not selected'))"
       />
      <q-separator vertical inset spaced="sm" color="secondary" />

      <q-btn
        @click="onNewRunClick(worksetNameSelected)"
        :disable="!isReadonlyWorksetCurrent || isReadonlyServer"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders"
        icon="mdi-run"
        :title="$t('Run the Model using Scenario') + ' ' +  worksetNameSelected"
        />
      <q-btn
        @click="onWorksetReadonlyToggle"
        :disable="isReadonlyServer || isEdit() || !isNotEmptyWorksetCurrent"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders q-ml-xs"
        :icon="(!isNotEmptyWorksetCurrent || isReadonlyWorksetCurrent) ? 'mdi-lock' : 'mdi-lock-open-variant'"
        :title="((!isNotEmptyWorksetCurrent || isReadonlyWorksetCurrent) ? $t('Open to edit scenario') : $t('Close to run scenario')) + ' ' + worksetCurrent.Name"
        />

      <span class="col-auto no-wrap q-ma-xs">
        <q-btn
          :disable="isShowNoteEditor || isNewWorksetShow"
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
          <q-badge outline class="q-ml-sm q-mr-xs">{{ paramTreeCount }}</q-badge>
        </q-btn>
      </span>

      <transition
        enter-active-class="animated fadeIn"
        leave-active-class="animated fadeOut"
        mode="out-in"
        >
        <div
          :key="worksetNameSelected"
          class="col-auto"
          >
          <span>{{ worksetNameSelected }}<br />
          <span class="om-text-descr"><span class="mono q-pr-sm">{{ dateTimeStr(worksetCurrent.UpdateDateTime) }}</span>{{ descrWorksetCurrent }}</span></span>
        </div>
      </transition>
    </div>

    <q-card-section
      v-show="isParamTreeShow"
      class="q-px-sm q-pt-none"
      >
      <workset-parameter-list
        :workset-name="worksetNameSelected"
        :refresh-tickle="refreshTickle"
        :refresh-param-tree-tickle="refreshParamTreeTickle"
        :is-param-leaf-click="true"
        :is-remove="!isReadonlyServer"
        :is-remove-group="!isReadonlyServer"
        :is-remove-disabled="isReadonlyWorksetCurrent || isReadonlyServer"
        :is-param-download="true"
        :is-param-download-disabled="!isReadonlyWorksetCurrent || isFromWorksetShow"
        @set-parameter-select="onWorksetParamClick"
        @set-parameter-info-show="doParamNoteWorksetCurrent"
        @set-parameter-group-info-show="doShowGroupNote"
        @set-parameter-tree-updated="onParamTreeUpdated"
        @set-parameter-remove="onParamDelete"
        @set-parameter-group-remove="onParamGroupDelete"
        >
      </workset-parameter-list>

    </q-card-section>

  </q-card>

  <q-card
    v-if="isFromRunShow"
    bordered
    class="primary-border-025 q-ma-sm"
    >
    <div class="row items-center full-width q-pa-sm">
      <q-btn
        @click="onFromRunHide"
        flat
        dense
        class="col-auto section-title-button bg-primary text-white rounded-borders q-mr-xs"
        icon="mdi-close-circle"
        :title="$t('Close')"
        />
      <div
        class="col-grow section-title bg-primary text-white q-px-md"
        :class="{ 'om-bg-inactive': isReadonlyWorksetCurrent || !isRunSuccess }"
        >
        <span>{{ $t('Copy parameters from model run') + (isRunSuccess ? ' ' + runCurrent.Name : '') }}</span>
      </div>
    </div>

    <q-card-section
      v-show="!isReadonlyWorksetCurrent && isRunSuccess"
      class="q-pa-sm"
      >
      <run-parameter-list
        :run-digest="runDigestSelected"
        :refresh-tickle="refreshTickle"
        :is-add="true"
        :is-add-group="true"
        @run-parameter-add="onParamRunCopy"
        @run-parameter-group-add="onParamGroupRunCopy"
        @run-parameter-info-show="doParamNoteRun"
        @run-parameter-group-info-show="doShowGroupNote"
        >
      </run-parameter-list>
    </q-card-section>

  </q-card>

  <q-card
    v-if="isFromWorksetShow"
    bordered
    class="primary-border-025 q-ma-sm"
    >

    <div class="row items-center full-width q-pa-sm">
      <q-btn
        @click="onFromWorksetHide"
        flat
        dense
        class="col-auto section-title-button bg-primary text-white rounded-borders q-mr-xs"
        icon="mdi-close-circle"
        :title="$t('Close')"
        />
      <div
        class="col-grow section-title bg-primary text-white q-px-md"
        :class="{ 'om-bg-inactive': isReadonlyWorksetCurrent || !isNotEmptyFrom || !isReadonlyFrom || !worksetNameFrom || worksetNameFrom === worksetNameSelected }"
        >
        <span>{{ $t('Copy parameters from input scenario') + ((worksetNameFrom && worksetNameFrom !== worksetNameSelected) ? ' ' + worksetNameFrom : $t('Source scenario not selected')) }}</span>
      </div>
    </div>

    <q-card-section
      v-show="!isReadonlyWorksetCurrent && isNotEmptyFrom && isReadonlyFrom && worksetNameFrom && worksetNameFrom !== worksetNameSelected"
      class="q-pa-sm"
      >
      <workset-parameter-list
        :workset-name="worksetNameFrom"
        :refresh-tickle="refreshTickle"
        :refresh-param-tree-tickle="refreshParamTreeFromTickle"
        :is-add="!isReadonlyServer"
        :is-add-group="!isReadonlyServer"
        @set-parameter-add="onParamWorksetCopy"
        @set-parameter-group-add="onParamGroupWorksetCopy"
        @set-parameter-info-show="doParamNoteWorksetFrom"
        @set-parameter-group-info-show="doShowGroupNote"
        >
      </workset-parameter-list>
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
        @click="onUploadWorkset"
        v-if="uploadFileSelect"
        :disable="!fileSelected"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders"
        icon="mdi-upload"
        :title="$t('Upload scenario .zip')"
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
        :label="$t('Select input scenario .zip for upload')"
        >
      </q-file>
    </div>
    <div class="row q-pl-sm q-pb-sm">
      <q-checkbox
        v-model="isNoDigestCheck"
        v-if="uploadFileSelect"
        :disable="!fileSelected"
        :label="$t('Ignore input scenario model digest (model version)')"
        />
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
        :the-name="worksetNameSelected"
        :description-editable="!isReadonlyWorksetCurrent"
        :the-descr="descrWorksetCurrent"
        :descr-prompt="$t('Input scenario description')"
        :notes-editable="!isReadonlyWorksetCurrent"
        :the-note="noteCurrent"
        :note-prompt="$t('Input scenario notes')"
        :lang-code="noteEditorLangCode"
        @save-note="onSaveNoteEditor"
        @cancel-note="onCancelNoteEditor"
        class="q-pa-sm"
        >
      </markdown-editor>

    </q-card-section>
  </q-card>

  <q-card class="q-ma-sm">
    <div class="q-pa-sm">
      <q-tree
        ref="worksetTree"
        default-expand-all
        :nodes="wsTreeData"
        node-key="key"
        tick-strategy="leaf-filtered"
        no-transition
        no-connectors
        v-model:expanded="wsTreeExpanded"
        v-model:ticked="wsTreeTicked"
        :filter="wsTreeFilter"
        :filter-method="doWsTreeFilter"
        :no-results-label="$t('No input scenarios found')"
        :no-nodes-label="$t('No input scenarios published (or server offline)')"
        >
        <template v-slot:default-header="prop">

          <div
            v-if="prop.node.key === 'wsl-top-node'"
            class="row no-wrap items-center full-width"
            >
            <div class="col-auto">
              <q-btn
                v-if="!isReadonlyServer"
                :disable="!wsTreeTicked.length || isEdit() || isUnsavedTicked()"
                @click.stop="onWsMultipleDelete"
                :round="!wsTreeTicked.length"
                :outline="!wsTreeTicked.length || isEdit() || isUnsavedTicked()"
                :rounded="!!wsTreeTicked.length || isEdit() || isUnsavedTicked()"
                no-caps
                color="primary"
                class="col-auto"
                :icon="!!wsTreeTicked.length ? 'mdi-delete' : 'mdi-delete-outline'"
                :label="!!wsTreeTicked.length ? '[ ' + wsTreeTicked.length.toLocaleString() + ' ]' : ''"
                :title="$t('Delete') + (!!wsTreeTicked.length ? ' [ ' + wsTreeTicked.length.toLocaleString() + ' ]' : '\u2026')"
                />
            </div>
            <span
              @click.stop="() => {}"
              class="col-grow q-pl-sm"
              >
              <q-input
                ref="filterInput"
                debounce="500"
                v-model="wsTreeFilter"
                outlined
                dense
                :placeholder="$t('Find input scenario...')"
                >
                <template v-slot:append>
                  <q-icon v-if="wsTreeFilter !== ''" name="cancel" class="cursor-pointer" @click="resetFilter" />
                  <q-icon v-else name="search" />
                </template>
              </q-input>
            </span>
          </div>

          <div v-else
            @click="onWorksetLeafClick(prop.node.label)"
            class="row no-wrap items-center full-width om-tree-leaf"
            :class="[{ 'text-primary' : prop.node.label === worksetNameSelected }, isEdit() ? 'cursor-not-allowed' : 'cursor-pointer']"
            >
            <q-btn
              @click.stop="doShowWorksetNote(prop.node.label)"
              :flat="prop.node.label !== worksetNameSelected"
              :outline="prop.node.label === worksetNameSelected"
              round
              dense
              color="primary"
              padding="xs"
              class="col-auto"
              :icon="prop.node.label === worksetNameSelected ? 'mdi-information' : 'mdi-information-outline'"
              :title="$t('About') + ' ' + prop.node.label"
              />
            <q-btn
              v-if="prop.node.label && !isReadonlyServer"
              :disable="!prop.node.isReadonly || prop.node.label === worksetNameSelected || isReadonlyWorksetCurrent"
              @click.stop="onWorksetFromClick(prop.node.label)"
              flat
              round
              dense
              padding="xs"
              class="col-auto"
              :class="(!prop.node.isReadonly || prop.node.label === worksetNameSelected || isReadonlyWorksetCurrent) ? 'text-secondary' : (prop.node.label !== worksetNameFrom ? 'text-primary' : 'text-white bg-primary')"
              icon="mdi-table-plus"
              :title="$t('Copy parameters from') + ' ' + prop.node.label"
              />
            <q-btn
              v-if="prop.node.label"
              :disable="isReadonlyServer || !prop.node.isReadonly || isEdit()"
              @click.stop="onNewRunClick(prop.node.label)"
              flat
              round
              dense
              :color="(prop.node.isReadonly && !isEdit())? 'primary' : 'secondary'"
              padding="xs"
              class="col-auto"
              icon="mdi-run"
              :title="$t('Run the Model using Scenario') + ' ' +  prop.node.label"
              />
            <q-btn
              v-if="prop.node.label && !isReadonlyServer"
              :disable="prop.node.isReadonly || isEdit()"
              @click.stop="onDeleteWorkset(prop.node.label)"
              flat
              round
              dense
              :color="(!prop.node.isReadonly && !isEdit())? 'primary' : 'secondary'"
              padding="xs"
              class="col-auto"
              icon="mdi-delete-outline"
              :title="$t('Delete') + ' ' + prop.node.label"
              />
            <q-btn
              v-if="serverConfig.AllowDownload && !isReadonlyServer"
              :disable="!prop.node.isReadonly || isEdit()"
              @click.stop="onDownloadWorkset(prop.node.label)"
              flat
              round
              dense
              :color="(prop.node.isReadonly && !isEdit()) ? 'primary' : 'secondary'"
              padding="xs"
              class="col-auto"
              icon="mdi-download-circle-outline"
              :title="$t('Download') + ' ' + prop.node.label"
              />
            <div class="col q-ml-xs">
              <span><span :class="{ 'text-bold': prop.node.label === worksetNameSelected }">{{ prop.node.label }}</span><br />
              <span
                :class="prop.node.label === worksetNameSelected ? 'om-text-descr-selected' : 'om-text-descr'"
                >
                <span class="mono q-pr-sm">{{ prop.node.lastTime }}</span>{{ prop.node.descr }}</span></span>
            </div>
          </div>

        </template>
      </q-tree>
    </div>
  </q-card>

  <refresh-workset v-if="(digest || '') !== '' && (worksetNameSelected || '') !== ''"
    :model-digest="digest"
    :workset-name="worksetNameSelected"
    :refresh-tickle="refreshTickle"
    :refresh-workset-tickle="refreshWsTickle"
    @done="doneWsLoad"
    @wait="loadWsWait = true"
    >
  </refresh-workset>
  <refresh-workset v-if="(digest || '') !== '' && (worksetNameFrom || '') !== ''"
    :model-digest="digest"
    :workset-name="worksetNameFrom"
    :refresh-tickle="refreshTickle"
    :refresh-workset-tickle="refreshWsFromTickle"
    @done="doneWsFromLoad"
    @wait="loadWsWait = true"
    >
  </refresh-workset>

  <workset-info-dialog :show-tickle="worksetInfoTickle" :model-digest="digest" :workset-name="worksetInfoName"></workset-info-dialog>
  <parameter-info-dialog :show-tickle="paramInfoTickle" :param-name="paramInfoName" :workset-name="worksetInfoName" :run-digest="runInfoDigest"></parameter-info-dialog>
  <group-info-dialog :show-tickle="groupInfoTickle" :group-name="groupInfoName"></group-info-dialog>

  <delete-workset
    :delete-now="isDeleteWorksetNow"
    :model-digest="digest"
    :workset-name="worksetNameToDelete"
    @done="doneDeleteWorkset"
    @wait="loadWorksetDelete = true"
    >
  </delete-workset>
  <delete-confirm-dialog
    @delete-yes="onYesDeleteWorkset"
    :show-tickle="showDeleteWorksetTickle"
    :item-name="worksetNameToDelete"
    :dialog-title="$t('Delete input scenario') + '?'"
    >
  </delete-confirm-dialog>

  <delete-confirm-dialog
    @delete-yes="onYesDeleteParameter"
    :show-tickle="showDeleteParameterTickle"
    :item-name="paramInfoName"
    :dialog-title="$t('Delete parameter from input scenario') + '?'"
    >
  </delete-confirm-dialog>
  <delete-confirm-dialog
    @delete-yes="onYesDeleteGroup"
    :show-tickle="showDeleteGroupTickle"
    :item-name="groupInfoName"
    :dialog-title="$t('Delete group from input scenario') + '?'"
    >
  </delete-confirm-dialog>
  <delete-confirm-dialog
    @delete-yes="onYesWsMultipleDelete"
    :show-tickle="showDeleteMultipleDialogTickle"
    :item-name="'[ ' + wsMultipleCount.toString() + ' ] ' + $t('input scenarios to be deleted')"
    :item-id="!!wsMultipleCount ? wsMultipleCount.toString() : 'empty'"
    :bodyText="isWsDeleteAll ? $t('Do you really want to delete ALL input scenarios?') : $t('Model may be unavaliable until delete is completed.')"
    :dialog-title="(isWsDeleteAll ? $t('Delete ALL input scenarios') : $t('Delete multiple input scenarios')) + '?'"
    >
  </delete-confirm-dialog>

  <confirm-dialog
    @confirm-yes="onYesParamFromRun"
    :show-tickle="showParamFromRunTickle"
    :item-name="paramInfoName"
    :dialog-title="$t('Parameter already exist')"
    :body-text="$t('Replace')"
    :icon-name="'mdi-content-copy'"
    >
  </confirm-dialog>
  <confirm-dialog
    @confirm-yes="onYesParamFromWorkset"
    :show-tickle="showParamFromWorksetTickle"
    :item-name="paramInfoName"
    :dialog-title="$t('Parameter already exist')"
    :body-text="$t('Replace')"
    :icon-name="'mdi-content-copy'"
    >
  </confirm-dialog>
  <confirm-dialog
    @confirm-yes="onYesGroupFromRun"
    :show-tickle="showGroupFromRunTickle"
    :item-name="groupInfoName"
    :dialog-title="$t('Parameter(s) already exist')"
    :body-text="confirmMsg"
    :icon-name="'mdi-content-copy'"
    >
  </confirm-dialog>
  <confirm-dialog
    @confirm-yes="onYesGroupFromWorkset"
    :show-tickle="showGroupFromWorkTickle"
    :item-name="groupInfoName"
    :dialog-title="$t('Parameter(s) already exist')"
    :body-text="confirmMsg"
    :icon-name="'mdi-content-copy'"
    >
  </confirm-dialog>

  <create-workset
    :create-now="isCreateWorksetNow"
    :model-digest="digest"
    :new-name="nameOfNewWorkset"
    :descr-notes="newDescrNotes"
    @done="doneWorksetCreate"
    @wait="loadWorksetCreate = true"
    >
  </create-workset>

  <q-inner-loading :showing="loadWait || loadWsWait || loadWorksetDelete || loadWsMultipletDelete || loadWorksetCreate">
    <q-spinner-gears size="md" color="primary" />
  </q-inner-loading>

</div>

</template>

<script src="./workset-list.js"></script>

<style lang="scss" scope="local">
  .tab-switch-container {
    margin-right: 1px;
  }
  .tab-switch-button {
    border-top-right-radius: 1rem;
  }
  .section-title-button {
    height: 2.5rem;
  }
  .section-title {
    line-height: 2.5rem;
  }
  .primary-border-025 {
    border: 0.25rem solid $primary;
  }
</style>
