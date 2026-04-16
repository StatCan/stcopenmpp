<template>
<q-page class="text-body1">

  <q-card
    class="q-ma-sm"
    >
    <q-card-section>

      <table class="om-p-table">
        <thead>
          <tr>
            <th colspan="6" class="om-p-title text-weight-medium">{{ modelNameVer }}</th>
          </tr>
          <tr>
            <th class="om-p-cell">
              <q-btn
                v-if="isDownloadEnabled || isUploadEnabled"
                @click="logRefreshPauseToggle"
                flat
                dense
                class="col-auto bg-primary text-white rounded-borders q-mr-xs"
                :icon="!isLogRefreshPaused ? (((logRefreshCount % 2) === 1) ? 'mdi-autorenew' : 'mdi-sync') : 'mdi-play-circle-outline'"
                :title="!isLogRefreshPaused ? $t('Pause') : $t('Refresh')"
                />
              <q-btn
                v-else
                disable
                flat
                dense
                class="col-auto bg-primary text-white rounded-borders q-mr-xs"
                icon="mdi-autorenew"
                :title="$t('Refresh')"
                />
            </th>
            <th class="om-p-cell-left mono">{{ lastLogTimeStamp }}</th>
            <th class="om-p-head-center text-weight-medium">{{ $t('Ready') }}</th>
            <th class="om-p-head-center text-weight-medium">{{ $t('In progress') }}</th>
            <th class="om-p-head-center text-weight-medium">{{ $t('Failed') }}</th>
            <th class="om-p-head-center text-weight-medium">{{ $t('Total') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="isDownloadEnabled">
            <td colspan="2" class="om-p-head-left">{{ $t('Downloads') }}</td>
            <td class="om-p-cell-right mono">{{ readyDownCount }}</td>
            <td class="om-p-cell-right mono">{{ progressDownCount }}</td>
            <td class="om-p-cell-right mono">{{ errorDownCount }}</td>
            <td class="om-p-cell-right mono">{{ totalDownCount }}</td>
          </tr>
          <tr v-if="isDownloadEnabled">
            <td colspan="2" class="om-p-head-left">{{ $t('Uploads') }}</td>
            <td class="om-p-cell-right mono">{{ readyUpCount }}</td>
            <td class="om-p-cell-right mono">{{ progressUpCount }}</td>
            <td class="om-p-cell-right mono">{{ errorUpCount }}</td>
            <td class="om-p-cell-right mono">{{ totalUpCount }}</td>
          </tr>

          <tr>
            <td colspan="6" class="om-p-cell">
              <q-radio v-model="fastDownload" val="yes" :label="$t('Do fast downloads, only to analyze output values')" />
              <br />
              <q-radio v-model="fastDownload" val="no" :label="$t('Do full downloads, compatible with desktop model')" />
              <br />
              <q-checkbox v-model="isMicroDownload" :disable="fastDownload === 'yes' || !serverConfig.AllowMicrodata" :label="$t('Do full downloads, including microdata')" />
              <br />
              <q-checkbox v-model="isIdCsvDownload" :disable="!serverConfig.AllowDownload" :label="$t('Download with dimension items ID instead of code')" />
            </td>
          </tr>

          <tr>
            <td colspan="6" class="bg-primary text-white om-p-cell-left text-weight-medium">
              <span class="q-pl-xs">{{ $t('Upload model run .zip') }}</span>
            </td>
          </tr>
          <tr v-if="isUploadEnabled">
            <td class="om-p-cell">
              <q-btn
                v-if="!isDiskOver"
                @click="onUploadRun"
                :disable="!isUploadEnabled || !runFileSelected"
                flat
                dense
                class="col-auto bg-primary text-white rounded-borders"
                icon="mdi-upload"
                :title="$t('Upload model run .zip')"
                />
              <q-btn
                v-else
                to="/disk-use"
                outline
                rounded
                dense
                icon="mdi-database-alert"
                color="negative"
                class="col-auto rounded-borders"
                :title="$t('View and cleanup storage space')"
                />
            </td>
            <td colspan="5" class="om-p-cell">
              <q-file
                v-model="runUploadFile"
                :disable="!isUploadEnabled || !digest || isDiskOver"
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
            </td>
          </tr>

          <tr>
            <td colspan="6" class="bg-primary text-white om-p-cell-left text-weight-medium">
              <span class="q-pl-xs">{{ $t('Upload scenario .zip') }}</span>
            </td>
          </tr>
          <tr v-if="isUploadEnabled">
            <td class="om-p-cell">
              <q-btn
                v-if="!isDiskOver"
                @click="onUploadWorkset"
                :disable="!isUploadEnabled || !wsFileSelected"
                flat
                dense
                class="col-auto bg-primary text-white rounded-borders"
                icon="mdi-upload"
                :title="$t('Upload scenario .zip')"
                />
              <q-btn
                v-else
                to="/disk-use"
                outline
                rounded
                dense
                icon="mdi-database-alert"
                color="negative"
                class="col-auto rounded-borders"
                :title="$t('View and cleanup storage space')"
                />
            </td>
            <td colspan="5" class="om-p-cell">
              <q-file
                v-model="wsUploadFile"
                :disable="!isUploadEnabled || !digest || isDiskOver"
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
            </td>
          </tr>
          <tr v-if="isUploadEnabled">
            <td class="om-p-cell">
              <q-checkbox
                v-model="isNoDigestCheck"
                :disable="!isUploadEnabled || !wsFileSelected"
                :title="$t('Ignore input scenario model digest (model version)')"
                />
            </td>
            <td colspan="6" class="om-p-cell-left"><span class="q-pl-sm">{{ $t('Ignore input scenario model digest (model version)') }}</span></td>
          </tr>

          <tr>
            <td colspan="6" class="om-p-title text-weight-medium">{{ $t('Cleanup downloads or uploads of all models') }}</td>
          </tr>
          <tr>
            <td colspan="1" class="om-p-cell-center">
              <q-btn
                :disable="!serverConfig.AllowDownload || totalDownCount <= 0"
                @click.stop="onAllUpDownDeleteClick('download')"
                unelevated
                dense
                color="primary"
                icon="mdi-delete-outline"
                :title="$t('Delete all download files')"
                />
            </td>
            <td colspan="5" class="om-p-cell">{{ $t('Delete all files of all models in download folder') }}</td>
          </tr>
          <tr>
            <td colspan="1" class="om-p-cell-center">
                <q-btn
                  :disable="!serverConfig.AllowUpload || totalUpCount <= 0"
                  @click.stop="onAllUpDownDeleteClick('upload')"
                  unelevated
                  dense
                  color="primary"
                  icon="mdi-delete-outline"
                  :title="$t('Delete all upload files')"
                  />
            </td>
            <td colspan="5" class="om-p-cell">{{ $t('Delete all files of all models in upload folder') }}</td>
          </tr>
        </tbody>
      </table>

    </q-card-section>
  </q-card>

  <q-expansion-item
    :disable="!isDownloadEnabled"
    v-model="downloadExpand"
    group="up-down-expand"
    switch-toggle-side
    expand-separator
    :label="$t('Downloads of') + (modelNameVer ? ' ' + modelNameVer : '')"
    header-class="bg-primary text-white"
    class="q-ma-sm"
    >
  <q-card
    v-for="uds in downStatusLst" :key="'down-' + (uds.LogFileName || 'no-log')"
    class="up-down-card q-my-sm"
    >

    <q-card-section class="q-pb-sm">
      <div
        class="row items-center"
        >
        <q-btn
          v-if="!isDeleteKind(uds.Kind)"
          @click="onFolderTreeClick('down', uds.Folder)"
          :unelevated="((folderSelected || '') !== uds.Folder) || ((upDownSelected || '') !== 'down')"
          :outline="((folderSelected || '') === uds.Folder) && ((upDownSelected || '') === 'down')"
          dense
          color="primary"
          class="col-auto rounded-borders q-mr-xs"
          icon="mdi-file-tree"
          :title="(((folderSelected || '') !== uds.Folder || (upDownSelected || '') !== 'down') ? $t('Expand') : $t('Collapse')) + ' ' + uds.Folder"
          />
        <q-btn
          v-if="!isDeleteKind(uds.Kind) && !isProgress(uds.Status)"
          @click="onUpDownDeleteClick('down', uds.Folder)"
          flat
          dense
          class="col-auto bg-primary text-white rounded-borders q-mr-xs"
          icon="mdi-delete"
          :title="$t('Delete: ') + uds.Folder"
          />
        <q-btn
          v-else
          disable
          flat
          dense
          class="col-auto bg-primary text-white rounded-borders q-mr-xs"
          icon="mdi-delete-clock"
          :title="$t('Deleting: ') + uds.Folder"
          />
        <span class="col-auto no-wrap q-mr-xs">
          <q-btn
            :disable="!uds.Lines.length"
            @click="uds.isShowLog = !uds.isShowLog"
            no-caps
            unelevated
            dense
            color="primary"
            class="rounded-borders q-pr-xs"
            :class="isReady(uds.Status) || isProgress(uds.Status) ? 'bg-primary' : 'bg-warning'"
            >
            <q-icon :name="uds.isShowLog ? 'keyboard_arrow_up' : 'keyboard_arrow_down'" />
            <span>{{ uds.LogFileName }}</span>
          </q-btn>
        </span>
        <model-bar
          v-if="isModelKind(uds.Kind)"
          :model-digest="uds.ModelDigest"
          @model-info-click="doShowModelNote(uds.ModelDigest)"
          >
        </model-bar>
        <run-bar
          v-if="isRunKind(uds.Kind)"
          :model-digest="uds.ModelDigest"
          :run-digest="uds.RunDigest"
          @run-info-click="doShowRunNote(uds.ModelDigest, uds.RunDigest)"
          >
        </run-bar>
        <workset-bar
          v-if="isWorksetKind(uds.Kind)"
          :model-digest="uds.ModelDigest"
          :workset-name="uds.WorksetName"
          @set-info-click="doShowWorksetNote(uds.ModelDigest, uds.WorksetName)"
          >
        </workset-bar>
      </div>

    </q-card-section>
    <q-separator inset />

    <template v-if="uds.isShowLog">
      <q-card-section class="q-py-sm">
        <div>
          <pre>{{uds.Lines.join('\n')}}</pre>
        </div>
      </q-card-section>
      <q-separator inset />
    </template>

    <q-card-section class="q-pt-sm">

      <q-list>
        <q-item
          v-if="uds.IsZip"
          clickable
          tag="a"
          target="_blank"
          :href="'/download/' + encodeURIComponent(uds.ZipFileName)"
          class="q-pl-none"
          :title="$t('Download') + ' ' + uds.ZipFileName"
          >
          <q-item-section avatar>
            <q-avatar rounded icon="mdi-download-circle" size="md" font-size="1.25rem" color="primary" text-color="white" />
          </q-item-section>
          <q-item-section>
            <q-item-label>{{ uds.ZipFileName }}</q-item-label>
            <q-item-label caption>{{ fileTimeStamp(uds.ZipModTime) }}<span>&nbsp;&nbsp;</span>{{ fileSizeStr(uds.ZipSize) }}</q-item-label>
          </q-item-section>
        </q-item>
      </q-list>

      <template v-if="uds.IsFolder && uds.Folder === folderSelected && upDownSelected === 'down'">
        <div
          class="row no-wrap items-center full-width"
          >
          <q-btn
            v-if="isAnyFolderDir"
            flat
            dense
            class="col-auto bg-primary text-white rounded-borders q-mr-xs om-tree-control-button"
            :icon="isFolderTreeExpanded ? 'keyboard_arrow_up' : 'keyboard_arrow_down'"
            :title="isFolderTreeExpanded ? $t('Collapse all') : $t('Expand all')"
            @click="doToogleExpandFolderTree"
            />
          <span class="col-grow">
            <q-input
              ref="folderTreeFilterInput"
              debounce="500"
              v-model="folderTreeFilter"
              outlined
              dense
              :placeholder="$t('Find files...')"
              >
              <template v-slot:append>
                <q-icon v-if="folderTreeFilter !== ''" name="cancel" class="cursor-pointer" @click="resetFolderFilter" />
                <q-icon v-else name="search" />
              </template>
            </q-input>
          </span>
        </div>

        <div class="q-px-sm">
          <q-tree
            ref="folderTree"
            :nodes="folderTreeData"
            node-key="key"
            no-transition
            :filter="folderTreeFilter"
            :filter-method="doFolderTreeFilter"
            :no-results-label="$t('No files found')"
            :no-nodes-label="$t('Server offline or no files found')"
            >
            <template v-slot:default-header="prop">

              <div
                v-if="prop.node.isGroup"
                class="row no-wrap items-center full-width"
                >
                <div class="col">
                  <span>{{ prop.node.label }}<br />
                  <span class="mono om-text-descr">{{ prop.node.descr }}</span></span>
                </div>
              </div>

              <a
                v-if="!prop.node.isGroup"
                @click="onFolderLeafClick(prop.node.label, '/download/' + prop.node.link)"
                :href="'/download/' + prop.node.link"
                target="_blank"
                :download="prop.node.label"
                class="row no-wrap items-center full-width cursor-pointer om-tree-leaf file-link"
                :title="$t('Download') + ' ' + prop.node.label"
                >
                <span class="text-primary">{{ prop.node.label }}<br />
                <span class="mono om-text-descr">{{ prop.node.descr }}</span></span>
              </a>

            </template>
          </q-tree>
        </div>

      </template>

    </q-card-section>

  </q-card>
  </q-expansion-item>

  <q-expansion-item
    :disable="!isUploadEnabled"
    v-model="uploadExpand"
    group="up-down-expand"
    switch-toggle-side
    expand-separator
    :label="$t('Uploads of') + (modelNameVer ? ' ' + modelNameVer : '')"
    header-class="bg-primary text-white"
    class="q-ma-sm"
    >
  <q-card
    v-for="uds in upStatusLst" :key="'up-' + (uds.LogFileName || 'no-log')"
    class="up-down-card q-my-sm"
    >

    <q-card-section class="q-pb-sm">
      <div
        class="row items-center"
        >
        <q-btn
          v-if="!isDeleteKind(uds.Kind)"
          @click="onFolderTreeClick('up', uds.Folder)"
          :unelevated="((folderSelected || '') !== uds.Folder) || ((upDownSelected || '') !== 'up')"
          :outline="((folderSelected || '') === uds.Folder) && ((upDownSelected || '') === 'up')"
          dense
          color="primary"
          class="col-auto rounded-borders q-mr-xs"
          icon="mdi-file-tree"
          :title="(((folderSelected || '') !== uds.Folder || (upDownSelected || '') !== 'up') ? $t('Expand') : $t('Collapse')) + ' ' + uds.Folder"
          />
        <q-btn
          v-if="!isDeleteKind(uds.Kind) && !isProgress(uds.Status)"
          @click="onUpDownDeleteClick('up', uds.Folder)"
          flat
          dense
          class="col-auto bg-primary text-white rounded-borders q-mr-xs"
          icon="mdi-delete"
          :title="$t('Delete: ') + uds.Folder"
          />
        <q-btn
          v-else
          disable
          flat
          dense
          class="col-auto bg-primary text-white rounded-borders q-mr-xs"
          icon="mdi-delete-clock"
          :title="$t('Deleting: ') + uds.Folder"
          />
        <span class="col-auto no-wrap q-mr-xs">
          <q-btn
            :disable="!uds.Lines.length"
            @click="uds.isShowLog = !uds.isShowLog"
            no-caps
            unelevated
            dense
            color="primary"
            class="rounded-borders q-pr-xs"
            :class="isReady(uds.Status) || isProgress(uds.Status) ? 'bg-primary' : 'bg-warning'"
            >
            <q-icon :name="uds.isShowLog ? 'keyboard_arrow_up' : 'keyboard_arrow_down'" />
            <span>{{ uds.LogFileName }}</span>
          </q-btn>
        </span>
        <workset-bar
          v-if="isUploadKind(uds.Kind)"
          :model-digest="uds.ModelDigest"
          :workset-name="uds.WorksetName"
          @set-info-click="doShowWorksetNote(uds.ModelDigest, uds.WorksetName)"
          >
        </workset-bar>
      </div>
    </q-card-section>
    <q-separator inset />

    <template v-if="uds.isShowLog">
      <q-card-section class="q-py-sm">
        <div>
          <pre>{{uds.Lines.join('\n')}}</pre>
        </div>
      </q-card-section>
      <q-separator inset />
    </template>

    <q-card-section class="q-pt-sm">

      <q-list>
        <q-item
          v-if="uds.IsZip"
          clickable
          tag="a"
          target="_blank"
          :href="'/upload/' + encodeURIComponent(uds.ZipFileName)"
          class="q-pl-none"
          :title="$t('Download') + ' ' + uds.ZipFileName"
          >
          <q-item-section avatar>
            <q-avatar rounded icon="mdi-download-circle" size="md" font-size="1.25rem" color="primary" text-color="white" />
          </q-item-section>
          <q-item-section>
            <q-item-label>{{ uds.ZipFileName }}</q-item-label>
            <q-item-label caption>{{ fileTimeStamp(uds.ZipModTime) }}<span>&nbsp;&nbsp;</span>{{ fileSizeStr(uds.ZipSize) }}</q-item-label>
          </q-item-section>
        </q-item>
      </q-list>

      <template v-if="uds.IsFolder && uds.Folder === folderSelected && upDownSelected === 'up'">
        <div
          class="row no-wrap items-center full-width"
          >
          <q-btn
            v-if="isAnyFolderDir"
            flat
            dense
            class="col-auto bg-primary text-white rounded-borders q-mr-xs om-tree-control-button"
            :icon="isFolderTreeExpanded ? 'keyboard_arrow_up' : 'keyboard_arrow_down'"
            :title="isFolderTreeExpanded ? $t('Collapse all') : $t('Expand all')"
            @click="doToogleExpandFolderTree"
            />
          <span class="col-grow">
            <q-input
              ref="folderTreeFilterInput"
              debounce="500"
              v-model="folderTreeFilter"
              outlined
              dense
              :placeholder="$t('Find files...')"
              >
              <template v-slot:append>
                <q-icon v-if="folderTreeFilter !== ''" name="cancel" class="cursor-pointer" @click="resetFolderFilter" />
                <q-icon v-else name="search" />
              </template>
            </q-input>
          </span>
        </div>

        <div class="q-px-sm">
          <q-tree
            ref="folderTree"
            :nodes="folderTreeData"
            node-key="key"
            no-transition
            :filter="folderTreeFilter"
            :filter-method="doFolderTreeFilter"
            :no-results-label="$t('No files found')"
            :no-nodes-label="$t('Server offline or no files found')"
            >
            <template v-slot:default-header="prop">

              <div
                v-if="prop.node.isGroup"
                class="row no-wrap items-center full-width"
                >
                <div class="col">
                  <span>{{ prop.node.label }}<br />
                  <span class="mono om-text-descr">{{ prop.node.descr }}</span></span>
                </div>
              </div>

              <a
                v-if="!prop.node.isGroup"
                @click="onFolderLeafClick(prop.node.label, '/upload/' + prop.node.link)"
                :href="'/upload/' + prop.node.link"
                target="_blank"
                :download="prop.node.label"
                class="row no-wrap items-center full-width cursor-pointer om-tree-leaf file-link"
                :title="$t('Download') + ' ' + prop.node.label"
                >
                <span class="text-primary">{{ prop.node.label }}<br />
                <span class="mono om-text-descr">{{ prop.node.descr }}</span></span>
              </a>

            </template>
          </q-tree>
        </div>

      </template>

    </q-card-section>

  </q-card>
  </q-expansion-item>

  <q-expansion-item
    :disable="!serverConfig.AllowFiles"
    v-model="filesExpand"
    group="up-down-expand"
    @show="doUserFilesRefresh"
    switch-toggle-side
    expand-separator
    :label="$t('Files')"
    header-class="bg-primary text-white"
    class="q-ma-sm"
    >
  <q-card
    v-if="filesExpand"
    class="up-down-card q-my-sm"
    >
    <q-card-section class="q-pb-sm">

      <div
        class="row no-wrap items-center full-width"
        >
        <q-btn
          v-if="isAnyFolderDir"
          @click="doToogleExpandFilesTree"
          dense
          color="primary"
          class="col-auto rounded-borders q-mr-xs om-tree-control-button"
          :icon="isFilesTreeExpanded ? 'keyboard_arrow_up' : 'keyboard_arrow_down'"
          :title="isFilesTreeExpanded ? $t('Collapse all') : $t('Expand all')"
          />
        <q-btn
          @click="doUserFilesRefresh"
          dense
          color="primary"
          class="col-auto rounded-borders q-mr-xs om-tree-control-button"
          icon="refresh"
          :title="$t('Refresh')"
          :aria-label="$t('Refresh')"
          />
        <span class="col-grow">
          <q-input
            ref="filesTreeFilterInput"
            debounce="500"
            v-model="filesTreeFilter"
            outlined
            dense
            :placeholder="$t('Find files...')"
            >
            <template v-slot:append>
              <q-icon v-if="filesTreeFilter !== ''" name="cancel" class="cursor-pointer" @click="resetFilesFilter" />
              <q-icon v-else name="search" />
            </template>
          </q-input>
        </span>
      </div>

      <div class="q-px-sm">
        <q-tree
          ref="filesTree"
          :nodes="filesTreeData"
          node-key="key"
          no-transition
          v-model:expanded="filesTreeExpanded"
          :filter="filesTreeFilter"
          :filter-method="doFilesTreeFilter"
          :no-results-label="$t('No files found')"
          :no-nodes-label="$t('Server offline or no files found')"
          >
          <template v-slot:default-header="prop">

            <div
              v-if="prop.node.isGroup"
              class="row no-wrap items-center full-width"
              >
              <template v-if="serverConfig.AllowUpload">
                <q-btn
                @click.stop="onFilesCreateFolderClick(prop.node.label, prop.node.Path)"
                flat
                round
                dense
                color="primary"
                padding="xs"
                class="col-auto"
                icon="mdi-folder-plus-outline"
                :title="$t('Create new folder')"
                />
              <q-btn
                @click.stop="onUploadFileClick(prop.node.label, prop.node.Path)"
                flat
                round
                dense
                color="primary"
                padding="xs"
                class="col-auto"
                icon="mdi-upload"
                :title="$t('Upload file')"
                />
              <q-btn
                v-if="!isUpDownPath(prop.node.Path)"
                :disable="!isFilesDeleteEnabled(prop.node.Path) || (prop.node.children.length || 0) <= 0"
                @click.stop="onFilesDeleteClick(prop.node.label, true, prop.node.Path)"
                flat
                round
                dense
                :color="isFilesDeleteEnabled(prop.node.Path) ? 'primary' : 'secondary'"
                padding="xs"
                class="col-auto"
                icon="mdi-delete-outline"
                :title="($t('Delete: ') + ' ' + prop.node.label)"
                />
                <q-btn
                  v-else
                  :disable="!isFilesDeleteEnabled(prop.node.Path) || (prop.node.children.length || 0) <= 0"
                  @click.stop="onAllUpDownDeleteClick(prop.node.Path)"
                  unelevated
                  dense
                  :color="isFilesDeleteEnabled(prop.node.Path) ? 'primary' : 'secondary'"
                  class="col-auto"
                  icon="mdi-delete-outline"
                  :title="(prop.node.Path === 'upload') ? $t('Delete all upload files') : $t('Delete all download files')"
                  />
              </template>
              <div class="col q-pl-xs">
                <span>{{ (prop.node.label !== '/' || !prop.node.descr) ? prop.node.label : '' }} <br v-if="!!prop.node.label && prop.node.label !== '/'"/>
                <span class="mono om-text-descr">{{ prop.node.descr + ' : ' + ((prop.node.children.length || 0).toString() + ' ' + $t('file(s)')) }}</span></span>
              </div>
            </div>

            <div v-else
              @click="onFilesDownloadClick(prop.node.label, prop.node.link)"
              class="row no-wrap items-center full-width cursor-pointer om-tree-leaf file-link"
              >
              <q-btn
                v-if="serverConfig.AllowUpload"
                :disable="!isFilesDeleteEnabled(prop.node.Path)"
                @click.stop="onFilesDeleteClick(prop.node.label, false, prop.node.Path)"
                flat
                round
                dense
                :color="isFilesDeleteEnabled(prop.node.Path) ? 'primary' : 'secondary'"
                padding="xs"
                class="col-auto"
                icon="mdi-delete-outline"
                :title="$t('Delete: ') + ' ' + prop.node.label"
                />
              <q-btn
                flat
                round
                dense
                color="primary"
                padding="xs"
                class="col-auto"
                icon="mdi-download"
                :title="$t('Download') + ' ' + prop.node.label"
                />
              <div class="col">
                <span class="text-primary">{{ prop.node.label }}<br />
                <span class="mono om-text-descr">{{ prop.node.descr  + ' : ' + fileSizeStr(prop.node?.Size || 0) }}</span></span>
              </div>
            </div>

            </template>
        </q-tree>
      </div>

    </q-card-section>
  </q-card>
  </q-expansion-item>

  <model-info-dialog :show-tickle="modelInfoTickle" :digest="modelInfoDigest"></model-info-dialog>
  <run-info-dialog :show-tickle="runInfoTickle" :model-digest="modelInfoDigest" :run-digest="runInfoDigest"></run-info-dialog>
  <workset-info-dialog :show-tickle="worksetInfoTickle" :model-digest="modelInfoDigest" :workset-name="worksetInfoName"></workset-info-dialog>

  <refresh-workset-list
    v-if="digest"
    :digest="digest"
    :refresh-tickle="refreshTickle"
    :refresh-workset-list-tickle="refreshWsListTickle"
    @done="loadWsListWait = false"
    @wait="loadWsListWait = true">
  </refresh-workset-list>

  <delete-confirm-dialog
    @delete-yes="onYesUpDownDelete"
    :show-tickle="showUpDownDeleteDialogTickle"
    :item-name="folderToDelete"
    :kind="upDownToDelete"
    :dialog-title="dialogTitleToDelete"
    >
  </delete-confirm-dialog>

  <delete-confirm-dialog
    @delete-yes="onYesFilesDelete"
    :show-tickle="showFilesDeleteDialogTickle"
    :item-name="nameToDelete"
    :item-id="pathToDelete"
    :kind="upDownToDelete"
    :dialog-title="dialogTitleToDelete"
    >
  </delete-confirm-dialog>

  <q-dialog v-model="showFolderPrompt">
    <q-card class="file-prompt text-body1">

      <q-card-section class="text-h6 bg-primary text-white">{{ $t('Enter new folder name') }}</q-card-section>

      <q-card-section horizontal class="items-center q-pa-sm">
        <q-btn
          @click="onYesNewFolderClick"
          :disable="!isUploadEnabled || !newFolderName"
          flat
          dense
          class="col-auto bg-primary text-white rounded-borders"
          icon="mdi-folder-plus"
          :title="$t('Create new folder')"
          />
        <q-input
          v-model="newFolderName"
          autofocus
          dense
          outlined
          hide-bottom-space
          maxlength="255"
          size="80"
          color="primary"
          class="q-ml-xs"
          @keyup.enter="onYesNewFolderClick"
          />
      </q-card-section>

      <q-card-actions align="right">
        <q-btn :label="$t('Cancel')" flat color="primary" v-close-popup />
        <q-btn
          @click="onYesNewFolderClick"
          :label="$t('OK')" flat color="primary" v-close-popup />
        </q-card-actions>

    </q-card>
  </q-dialog>

  <q-dialog v-model="showUploadFillePrompt">
    <q-card class="file-prompt text-body1">

      <q-card-section class="text-h6 bg-primary text-white">{{ $t('Upload file') }}</q-card-section>

      <q-card-section horizontal class="items-center q-pa-sm">
        <q-btn
          @click="onYesUploadFileClick"
          :disable="isDiskOver || !isUploadEnabled || !uploadFileSelected"
          flat
          dense
          class="col-auto bg-primary text-white rounded-borders"
          icon="mdi-upload"
          :title="$t('Upload file')"
          />
        <q-file
          v-model="uploadFile"
          :disable="!isUploadEnabled"
          outlined
          dense
          autofocus
          clearable
          hide-bottom-space
          maxlength="255"
          size="80"
          color="primary"
          class="col-grow q-ml-xs"
          :label="$t('Select file to upload')"
          >
        </q-file>
      </q-card-section>

      <q-card-actions align="right">
        <q-btn :label="$t('Cancel')" flat color="primary" v-close-popup />
        <q-btn
          :disable="isDiskOver || !isUploadEnabled || !uploadFileSelected"
          @click="onYesUploadFileClick"
          :label="$t('OK')" flat color="primary" v-close-popup />
        </q-card-actions>

    </q-card>
  </q-dialog>

  <q-inner-loading :showing="loadWsListWait || loadConfig || loadDiskUse">
    <q-spinner-gears size="lg" color="primary" />
  </q-inner-loading>

</q-page>
</template>

<script src="./updown-list.js"></script>

<style lang="scss" scope="local">
  .file-link {
    text-decoration: none;
    // display: inline-block;
  }
  .file-prompt {
    min-width: 40rem;
  }
  // override card shadow inside of expansion item
  .q-expansion-item__content > div.up-down-card {
    box-shadow: $shadow-1;
  }
</style>
