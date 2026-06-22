<template>
<div class="text-body1">

  <q-card class="q-ma-sm">

    <q-card-section>

      <div class="row items-center full-width">

        <span class="col-auto q-mr-sm">
          <q-btn
           v-if="!isDiskOver"
            @click="onModelRunClick"
            :disable="isInitRun || !runOpts.runName || isNoTables || isDiskOver || isReadonlyServer"
            color="primary"
            class="rounded-borders q-pr-xs"
            >
            <q-icon name="mdi-run" class="q-mr-xs"/>
            <span>{{ $t('Run the Model') }}</span>
          </q-btn>
          <q-btn
            v-else
            to="/disk-use"
            outline
            rounded
            color="negative"
            class="rounded-borders q-pr-xs"
            :title="$t('View and cleanup storage space')"
            >
            <q-icon name="mdi-database-alert" color="negative" class="q-mr-xs"/>
            <span>{{ $t('Cleanup Required') }}</span>
          </q-btn>
        </span>

        <span class="col-grow">
          <q-btn
            @click="isRunOptsShow = !isRunOptsShow"
            no-caps
            unelevated
            :ripple="false"
            color="primary"
            align="left"
            class="full-width"
            >
            <q-icon :name="isRunOptsShow ? 'keyboard_arrow_up' : 'keyboard_arrow_down'" />
            <span class="text-body1">{{ $t('Model Run Options') }}</span>
          </q-btn>
        </span>

      </div>

      <table v-show="isRunOptsShow">
      <tbody>
        <tr>
          <td class="q-pr-xs"><span v-if="!runOpts.runName" class="text-negative text-weight-bold">* </span><span>{{ $t('Run Name') }}:</span></td>
          <td>
            <q-input
              v-model="runOpts.runName"
              maxlength="255"
              size="80"
              required
              @focus="onRunNameFocus"
              @blur="onRunNameBlur"
              :rules="[ val => (val || '') !== '' ]"
              outlined
              dense
              clearable
              hide-bottom-space
              :placeholder="$t('Name of the new model run') + ' (* ' + $t('Required') + ')'"
              :title="$t('Name of the new model run')"
              >
            </q-input>
          </td>
        </tr>

        <tr>
          <td class="q-pr-xs">{{ $t('Sub-Values (Sub-Samples)') }}:</td>
          <td>
            <q-input
              v-model="runOpts.subCount"
              type="number"
              maxlength="4"
              min="1"
              max="8192"
              :rules="[
                val => val !== void 0,
                val => val >= 1 && val <= 8192
              ]"
              outlined
              dense
              hide-bottom-space
              class="tc-max-width-10"
              input-class="tc-right"
              :title="$t('Number of sub-values (a.k.a. members or replicas or sub-samples)')"
              />
          </td>
        </tr>

        <tr>
          <td
            :disable="!isReadonlyWorksetCurrent"
            class="q-pr-xs"
            >
            <q-checkbox v-model="useWorkset" :disable="!isReadonlyWorksetCurrent" :label="$t('Use Scenario:')"/>
          </td>
          <td>
            <workset-bar
              :model-digest="digest"
              :workset-name="worksetCurrent.Name"
              @set-info-click="doShowWorksetNote"
              >
            </workset-bar>
          </td>
        </tr>

        <tr>
          <td
            :disable="!isCompletedRunCurrent || isRunDeleted"
            class="q-pr-xs"
            >
            <q-checkbox
              v-model="useBaseRun"
              @click="onUseBaseRunClick"
              :disable="!isCompletedRunCurrent"
              :label="$t('Use Base Run:')"/>
          </td>
          <td>
            <run-bar
              :model-digest="digest"
              :run-digest="runCurrent.RunDigest"
              @run-info-click="doShowRunNote"
              >
            </run-bar>
          </td>
        </tr>

        <tr
          v-for="(p, idx) in presetLst" :key="p.name"
          >
          <td class="q-pr-xs">
            <q-btn
              @click="doPresetSelected(presetLst[idx])"
              no-caps
              unelevated
              color="primary"
              align="between"
              class="rounded-borders full-width"
              >
              <span>{{ p.label }}</span>
              <q-icon name="mdi-menu-right" />
            </q-btn>
          </td>
          <td class="om-text-descr-title">{{ p.descr }}</td>
        </tr>
      </tbody>
      </table>

    </q-card-section>

  </q-card>

  <q-card class="q-ma-sm">

    <q-expansion-item
      switch-toggle-side
      expand-separator
      header-class="bg-primary text-white"
      >
      <template v-slot:header>
        <q-icon v-if="isNoTables" name="star" color="red" />
        <span>{{ $t('Output Tables: ') + (tablesRetain.length !== tableCount ? (tablesRetain.length.toLocaleString() + ' / ' + tableCount.toLocaleString()) : $t('All')) }}</span>
      </template>

      <q-card-section
        class="q-pa-sm"
        >
        <template v-if="isNoTables">
          <span class="text-negative text-weight-bold">*&nbsp;</span><span>{{ $t('Please select output tables in order to run the model') }}</span>
        </template>
        <template v-else>
          <table-list
            :run-digest="''"
            :refresh-tickle="refreshTickle"
            :refresh-table-tree-tickle="refreshTableTreeTickle"
            :name-filter="tablesRetain"
            :is-in-list-clear="true"
            in-list-clear-icon="mdi-expand-all"
            :in-list-clear-label="$t('Retain all output tables')"
            :is-remove="true"
            :is-remove-group="true"
            @table-remove="onTableRemove"
            @table-group-remove="onTableGroupRemove"
            @table-info-show="doShowTableNote"
            @table-group-info-show="doShowGroupNote"
            @table-clear-in-list="onRetainAllTables"
            >
          </table-list>
        </template>
      </q-card-section>

      <q-card-section
        class="primary-border-025 shadow-up-1 q-pa-sm"
        >
        <table-list
          :run-digest="''"
          :refresh-tickle="refreshTickle"
          :is-add="true"
          :is-add-group="true"
          @table-add="onTableAdd"
          @table-group-add="onTableGroupAdd"
          @table-info-show="doShowTableNote"
          @table-group-info-show="doShowGroupNote"
          >
        </table-list>
      </q-card-section>

    </q-expansion-item>

  </q-card>

  <q-card v-if="entityAttrCount > 0" class="q-ma-sm">

    <q-expansion-item
      switch-toggle-side
      expand-separator
      header-class="bg-primary text-white"
      >
      <template v-slot:header>
        <span>{{ $t('Microdata: ') + (entityAttrsUse.length !== entityAttrCount ? (entityAttrsUse.length.toLocaleString() + ' / ' + entityAttrCount.toLocaleString()) : $t('All')) }}</span>
        <span v-if="isTooManyEntityAttrs()">
          <q-icon name="mdi-exclamation-thick" color="red" class="bg-white q-pa-xs q-ml-md q-mr-xs"/><span>{{ $t('Excessive use of microdata may slow down model run or lead to failure') }}</span>
        </span>
      </template>

      <q-card-section
        class="q-pa-sm"
        >
        <template v-if="isNoEntityAttrsUse">
          <span>{{ $t('No entity microdata included into model run results') }}</span>
        </template>
        <template v-else>
          <entity-list
            :run-digest="''"
            :refresh-tickle="refreshTickle"
            :refresh-entity-tree-tickle="refreshEntityTreeTickle"
            :name-filter="entityAttrsUse"
            :is-in-list-enable="true"
            :is-in-list-clear="true"
            :in-list-clear-label="$t('Do not use entity microdata')"
            in-list-clear-icon="mdi-collapse-all"
            :is-remove-entity-attr="true"
            :is-remove-entity="true"
            @entity-attr-remove="onAttrRemove"
            @entity-remove="onEntityRemove"
            @entity-group-remove="onEntityGroupRemove"
            @entity-clear-in-list="onClearEntityAttrs"
            @entity-info-show="doShowEntityNote"
            @entity-attr-info-show="doShowEntityAttrNote"
            @entity-group-info-show="doShowEntityGroupNote"
            >
          </entity-list>
        </template>
      </q-card-section>

      <q-card-section
        class="primary-border-025 shadow-up-1 q-pa-sm"
        >
        <entity-list
          :run-digest="''"
          :refresh-tickle="refreshTickle"
          :is-add-entity-attr="true"
          :is-add-entity="true"
          @entity-attr-add="onAttrAdd"
          @entity-add="onEntityAdd"
          @entity-group-add="onEntityGroupAdd"
          @entity-info-show="doShowEntityNote"
          @entity-attr-info-show="doShowEntityAttrNote"
          @entity-group-info-show="doShowEntityGroupNote"
          >
        </entity-list>
      </q-card-section>

    </q-expansion-item>

  </q-card>

  <q-card class="q-ma-sm">

    <q-expansion-item
      v-model="langOptsExpanded"
      switch-toggle-side
      expand-separator
      header-class="bg-primary text-white"
      :label="$t('Description and Notes')"
      >

      <markdown-editor
        v-for="t in txtNewRun"
        :key="t.LangCode"
        :ref="'new-run-note-editor-' + t.LangCode"
        :the-key="t.LangCode"
        :the-descr="t.Descr"
        :descr-prompt="$t('Model run description') + ' (' + t.LangName + ')'"
        :the-note="t.Note"
        :note-prompt="$t('Model run notes') + ' (' + t.LangName + ')'"
        :description-editable="true"
        :notes-editable="true"
        :is-hide-save="true"
        :is-hide-cancel="true"
        class="q-px-sm q-py-xs"
      >
      </markdown-editor>

    </q-expansion-item>

  </q-card>

  <q-card class="q-ma-sm">
    <q-expansion-item
      v-model="advOptsExpanded"
      switch-toggle-side
      expand-separator
      header-class="bg-primary text-white"
      :label="$t('Advanced Run Options')"
      >
      <q-card-section>
        <table>
        <tbody>
          <tr>
            <td class="q-pr-xs">{{ $t('Modelling Threads max') }}:</td>
            <td>
              <q-input
                v-model="runOpts.threadCount"
                type="number"
                maxlength="4"
                min="1"
                max="8192"
                :rules="[
                  val => val !== void 0,
                  val => val >= 1 && val <= 8192
                ]"
                outlined
                dense
                hide-bottom-space
                class="tc-max-width-10"
                input-class="tc-right"
                :title="$t('Maximum number of modelling threads')"
                />
            </td>
          </tr>

          <tr>
            <td class="q-pr-xs">{{ $t('Log Progress Percent') }}:</td>
            <td>
              <q-input
                v-model.number="runOpts.progressPercent"
                type="number"
                maxlength="3"
                min="1"
                max="100"
                :rules="[
                  val => val !== void 0,
                  val => val >= 1 && val <= 100
                ]"
                outlined
                dense
                hide-bottom-space
                class="tc-max-width-10"
                input-class="tc-right"
                :title="$t('Percent completed to log model progress')"
                />
            </td>
          </tr>

          <tr>
            <td class="q-pr-xs">{{ $t('Log Progress Step') }}:</td>
            <td>
              <q-input
                v-model.number="runOpts.progressStep"
                type="number"
                maxlength="8"
                min=0
                :rules="[
                  val => val !== void 0,
                  val => val >= 0
                ]"
                outlined
                dense
                hide-bottom-space
                class="tc-max-width-10"
                input-class="tc-right"
                :title="$t('Step to log model progress: number of cases or time passed')"
                />
            </td>
          </tr>

          <!-- ini file -->
          <template v-if="enableIni">
            <tr>
              <td class="q-pr-xs">
                <div class="row items-center">
                  <q-btn
                    v-if="serverConfig.AllowFiles"
                    @click="showIniFilesTree = !showIniFilesTree"
                    unelevated
                    dense
                    color="primary"
                    class="col-auto rounded-borders"
                    :icon="showIniFilesTree ? 'mdi-file-tree-outline' : 'mdi-file-tree'"
                    :title="$t('Select INI-file')"
                    />
                  <div class="col"><q-checkbox v-model="runOpts.useIni" :disable="(runOpts.iniName || '') === ''" :label="$t('Use INI-file:')" /></div>
                </div>
              </td>
              <td :class="runOpts.useIni ? ['om-light-border', 'rounded-borders'] : []">
                <div class="q-pl-xs"><span v-if="!!runOpts.iniName">{{ iniFileLabel(runOpts.iniName) }}</span><span v-else class="om-text-descr">{{ '(' + $t('None') + ')' }}</span></div>
              </td>
            </tr>

            <template v-if="showIniFilesTree">
              <tr>
                <td colspan="2" class="om-primary-border">
                  <q-input
                    ref="iniTreeFilterInput"
                    debounce="500"
                    v-model="iniTreeFilter"
                    outlined
                    dense
                    :placeholder="$t('Find files...')"
                    >
                    <template v-slot:append>
                      <q-icon v-if="iniTreeFilter !== ''" name="cancel" class="cursor-pointer" @click="resetIniFilter" />
                      <q-icon v-else name="search" />
                    </template>
                  </q-input>

                  <q-tree
                    :nodes="iniTreeData"
                    node-key="key"
                    default-expand-all
                    no-transition
                    :filter="iniTreeFilter"
                    :filter-method="doIniTreeFilter"
                    :no-results-label="$t('No files found')"
                    :no-nodes-label="$t('Server offline or no files found')"
                    >
                    <template v-slot:default-header="prop">
                      <div
                        v-if="prop.node.isGroup"
                        class="row no-wrap items-center full-width"
                        >
                        <div class="col">
                          <span>{{ prop.node.label }}
                            <template v-if="!!prop.node.descr"><br /><span class="mono om-text-descr">{{ prop.node.descr }}</span></template>
                          </span>
                        </div>
                      </div>
                      <div v-else
                        @click="onIniLeafClick(prop.node.path)"
                        class="row no-wrap items-center full-width cursor-pointer om-tree-leaf"
                        :class="{ 'text-primary' : (runOpts.useIni && prop.node.path === runOpts.iniName) }"
                        >
                        <q-btn
                          v-if="serverConfig.AllowDownload"
                          :disable="!prop.node.link"
                          @click.stop="onFileDownloadClick(prop.node.label, prop.node.link)"
                          flat
                          round
                          dense
                          :color="!!prop.node.link ? 'primary' : 'secondary'"
                          padding="xs"
                          class="col-auto"
                          icon="mdi-download-circle-outline"
                          :title="$t('Download') + ' ' + prop.node.label"
                          />
                        <div class="col q-ml-xs">
                          <span><span :class="{ 'text-bold': (runOpts.useIni && prop.node.path === runOpts.iniName) }">{{ prop.node.label }}</span><br />
                          <span
                            class="mono"
                            :class="(runOpts.useIni && prop.node.path === runOpts.iniName) ? 'om-text-descr-selected' : 'om-text-descr'"
                            >{{ prop.node.descr }}</span>
                          </span>
                        </div>
                      </div>
                    </template>
                  </q-tree>
                </td>
              </tr>
            </template>

            <tr v-if="enableIniAnyKey">
              <td
                :disable="!runOpts.useIni"
                class="q-pr-xs"
                >
                <q-checkbox v-model="runOpts.iniAnyKey" :disable="!runOpts.useIni" :label="$t('Development options:')"/>
              </td>
              <td :class="(runOpts.useIni && runOpts.iniAnyKey) ? ['om-light-border', 'rounded-borders'] : []">
                <span v-if="!!runOpts.iniName" class="q-pl-xs">{{ iniFileLabel(runOpts.iniName) }}</span><span v-else class="om-text-descr">{{ '(' + $t('None') + ')' }}</span>
              </td>
            </tr>

          </template>
          <!-- end of ini file -->

          <!-- csv files -->
          <template v-if="serverConfig.AllowFiles">
            <tr>
              <td class="q-pr-xs">
                <div class="row items-center">
                  <q-btn
                    @click="showCsvFilesTree = !showCsvFilesTree"
                    unelevated
                    dense
                    color="primary"
                    class="col-auto rounded-borders"
                    :icon="showCsvFilesTree ? 'mdi-file-tree-outline' : 'mdi-file-tree'"
                    :title="$t('Select CSV Directory')"
                    />
                  <div class="col"><q-checkbox v-model="runOpts.useCsvDir" :disable="(runOpts.csvDir || '') === ''" :label="$t('Use CSV Directory:')" /></div>
                </div>
              </td>
              <td :class="runOpts.useCsvDir ? ['om-light-border', 'rounded-borders'] : []">
                <div class="q-pl-xs"><span v-if="!!runOpts.csvDir">{{ csvDirLabel(runOpts.csvDir) }}</span><span v-else class="om-text-descr">{{ '(' + $t('None') + ')' }}</span></div>
              </td>
            </tr>

            <template v-if="showCsvFilesTree">
              <tr>
                <td colspan="2" class="om-primary-border">
                  <q-input
                    ref="csvTreeFilterInput"
                    debounce="500"
                    v-model="csvTreeFilter"
                    outlined
                    dense
                    :placeholder="$t('Find files...')"
                    >
                    <template v-slot:append>
                      <q-icon v-if="csvTreeFilter !== ''" name="cancel" class="cursor-pointer" @click="resetCsvFilter" />
                      <q-icon v-else name="search" />
                    </template>
                  </q-input>

                  <q-tree
                    :nodes="csvTreeData"
                    node-key="key"
                    no-transition
                    v-model:expanded="csvTreeExpanded"
                    :filter="csvTreeFilter"
                    :filter-method="doCsvTreeFilter"
                    :no-results-label="$t('No files found')"
                    :no-nodes-label="$t('Server offline or no files found')"
                    >
                    <template v-slot:default-header="prop">
                      <div
                        v-if="prop.node.isGroup"
                        @click.stop="onCsvDirClick(prop.node.Path)"
                        class="row no-wrap items-center full-width"
                        :class="isCsvDirPath(prop.node.Path) ? ['text-primary', 'text-weight-bold'] : []"
                        >
                        <div class="col">
                          <span>{{ prop.node.label }} <br v-if="!!prop.node.label && prop.node.Path !== '/'"/>
                            <span class="mono om-text-descr">{{ prop.node.descr + ' : ' + csvFileCount(prop.node.children).toString() + ' ' + $t('file(s)') }}</span>
                          </span>
                        </div>
                      </div>

                      <div v-else
                        :disable="!serverConfig.AllowDownload || !prop.node.link"
                        @click="onFileDownloadClick(prop.node.label, prop.node.link)"
                        class="row no-wrap items-center full-width cursor-pointer om-tree-leaf"
                        >
                        <q-btn
                          v-if="serverConfig.AllowDownload"
                          :disable="!prop.node.link"
                          flat
                          round
                          dense
                          :color="!!prop.node.link ? 'primary' : 'secondary'"
                          padding="xs"
                          class="col-auto"
                          icon="mdi-download-circle-outline"
                          :title="$t('Download') + ' ' + prop.node.label"
                          />
                        <div class="col q-ml-xs">
                          <span class="text-primary">{{ prop.node.label }}<br />
                          <span class="mono om-text-descr">{{ prop.node.descr  + ' : ' + fileSizeStr(prop.node?.Size || 0) }}</span></span>
                        </div>
                      </div>
                    </template>
                  </q-tree>
                </td>
              </tr>
            </template>

            <tr>
              <td
                :disable="!runOpts.useCsvDir"
                class="q-pr-xs"
                >
                <q-checkbox v-model="runOpts.csvId" :disable="!runOpts.useCsvDir" :label="$t('CSV file(s) contains Id:')"/>
              </td>
              <td :class="(runOpts.useCsvDir && runOpts.csvId) ? ['om-light-border', 'rounded-borders'] : []">
                <div class="q-pl-xs"><span v-if="!!runOpts.csvDir">{{ csvDirLabel(runOpts.csvDir) }}</span><span v-else class="om-text-descr">{{ '(' + $t('None') + ')' }}</span></div>
              </td>
            </tr>

          </template>
          <!-- end of csv files -->

          <tr
            :disable="isEmptyRunTemplateList || runOpts.mpiNpCount > 0"
            >
            <td class="q-pr-xs">{{ $t('Model Run Template') }}:</td>
            <td>
              <q-select
                v-model="runOpts.runTmpl"
                :options="runTemplateLst"
                :disable="isEmptyRunTemplateList || runOpts.mpiNpCount > 0"
                outlined
                dense
                clearable
                hide-bottom-space
                class="tc-min-width-10"
                :title="$t('Template to run the model')"
                />
            </td>
          </tr>

          <tr>
            <td class="q-pr-xs">{{ $t('Working Directory') }}:</td>
            <td>
              <q-input
                v-model="runOpts.workDir"
                maxlength="2048"
                size="80"
                @blur="onWorkDirBlur"
                outlined
                dense
                clearable
                hide-bottom-space
                :placeholder="$t('Relative path to working directory to run the model')"
                :title="$t('Path to working directory to run the model')"
                >
              </q-input>
            </td>
          </tr>

          <tr
            :disable="isEmptyProfileList"
            >
            <td class="q-pr-xs">{{ $t('Profile Name') }}:</td>
            <td>
              <q-select
                v-model="runOpts.profile"
                :options="profileLst"
                :disable="isEmptyProfileList"
                outlined
                dense
                clearable
                hide-bottom-space
                class="tc-min-width-10"
                :title="$t('Profile name in database to get model run options')"
                />
            </td>
          </tr>
        </tbody>
        </table>
      </q-card-section>
    </q-expansion-item>
  </q-card>

  <q-card class="q-ma-sm">
    <q-expansion-item
      v-model="mpiOptsExpanded"
      switch-toggle-side
      expand-separator
      header-class="bg-primary text-white"
      :label="$t('Cluster Run Options')"
      >
      <q-card-section>
        <table>
        <tbody>
          <tr>
            <td class="q-pr-xs">{{ $t('MPI Number of Processes') }}:</td>
            <td>
              <q-input
                v-model.number="runOpts.mpiNpCount"
                @change="onMpiNpCount"
                type="number"
                maxlength="5"
                min="0"
                max="65536"
                :rules="[
                  val => val !== void 0,
                  val => val >= 0 && val <= 65536
                ]"
                outlined
                dense
                hide-bottom-space
                class="tc-max-width-10"
                input-class="tc-right"
                :title="$t('Number of parallel processes to run')"
                />
            </td>
          </tr>

          <tr
            :disable="!serverConfig.IsJobControl"
            >
            <td class="q-pr-xs">{{ $t('Use Jobs Service') }}:</td>
            <td class="tc-max-width-10 row om-light-border rounded-borders">
              <q-space />
              <q-toggle
                @click="onMpiUseJobs"
                v-model="runOpts.mpiUseJobs"
                :disable="!serverConfig.IsJobControl"
                :title="runOpts.mpiUseJobs ? $t('Use jobs service to run the model') : $t('Do not use jobs service to run the model')"
                />
            </td>
          </tr>

          <tr
            :disable="runOpts.mpiNpCount <= 0 && !runOpts.mpiUseJobs"
            >
            <td class="q-pr-xs">{{ $t('Use MPI Root for Modelling') }}:</td>
            <td class="tc-max-width-10 row om-light-border rounded-borders">
              <q-space />
              <q-toggle
                v-model="runOpts.mpiOnRoot"
                :disable="runOpts.mpiNpCount <= 0 && !runOpts.mpiUseJobs"
                :title="runOpts.mpiOnRoot ? $t('Use MPI root process to run the model') : $t('Do not use MPI root process to run the model')"
                />
            </td>
          </tr>

          <tr
            :disable="runOpts.mpiNpCount <= 0 && !runOpts.mpiUseJobs"
            >
            <td class="q-pr-xs">{{ $t('MPI Model Run Template') }}:</td>
            <td>
              <q-select
                v-model="runOpts.mpiTmpl"
                :options="mpiTemplateLst"
                :disable="runOpts.mpiNpCount <= 0 && !runOpts.mpiUseJobs"
                outlined
                dense
                clearable
                hide-bottom-space
                class="tc-min-width-10"
                :title="$t('Template to run the model')"
                />
            </td>
          </tr>
        </tbody>
        </table>
      </q-card-section>
    </q-expansion-item>
  </q-card>

  <new-run-init
    v-if="isInitRun"
    :model-digest="digest"
    :run-opts="runOpts"
    :tables-retain="retainTablesGroups"
    :microdata-opts="microOpts"
    :run-notes="newRunNotes"
    @done="doneNewRunInit"
    @wait="()=>{}">
  </new-run-init>

  <run-info-dialog :show-tickle="runInfoTickle" :model-digest="digest" :run-digest="runCurrent.RunDigest"></run-info-dialog>
  <workset-info-dialog :show-tickle="worksetInfoTickle" :model-digest="digest" :workset-name="worksetCurrent.Name"></workset-info-dialog>
  <table-info-dialog :show-tickle="tableInfoTickle" :table-name="tableInfoName" :run-digest="''"></table-info-dialog>
  <group-info-dialog :show-tickle="groupInfoTickle" :group-name="groupInfoName"></group-info-dialog>
  <entity-info-dialog :show-tickle="entityInfoTickle" :entity-name="entityInfoName"></entity-info-dialog>
  <entity-attr-info-dialog :show-tickle="attrInfoTickle" :entity-name="entityInfoName" :attr-name="attrInfoName"></entity-attr-info-dialog>
  <entity-group-info-dialog :show-tickle="entityGroupInfoTickle"  :entity-name="entityInfoName" :group-name="entityGroupInfoName"></entity-group-info-dialog>

  <q-dialog v-model="showOptChangeDialog">
    <q-card class="text-body1">

      <q-card-section class="text-h6 bg-primary text-white">
        <q-avatar icon="error" color="warning" text-color="white" /><span class="q-ml-sm">{{ $t('Run Options may not be optimal') }}</span>
      </q-card-section>

      <q-card-section>
        <table class="om-p-table">
        <tbody>
          <tr>
            <td colspan="2" class="om-p-cell-left">{{ $t('Sub-Values (Sub-Samples)') }}:</td>
            <td class="om-p-cell-right mono">{{ optChange.nSub }}</td>
          </tr>
          <tr>
            <td rowspan="2" class="om-p-cell-left">{{ $t('Limits') }}</td>
            <td class="om-p-cell-left">{{ $t('Modelling Threads') }}:</td>
            <td class="om-p-cell-right mono">{{ optChange.maxThread }}</td>
          </tr>
          <tr>
            <td class="om-p-cell-left">{{ $t('CPU Cores') }}:</td>
            <td class="om-p-cell-right mono">{{ optChange.maxCpu }}</td>
          </tr>
          <tr>
            <td colspan="3" class="om-p-head-center text-weight-bold">{{ $t('Recommended Run Options') }}</td>
          </tr>
          <tr>
            <td colspan="2" class="om-p-head-left">{{ $t('Modelling Threads max') }}:</td>
            <td class="om-p-cell-right mono">{{ optChange.nThread }}</td>
          </tr>
          <tr>
            <td colspan="2" class="om-p-head-left">{{ $t('MPI Number of Processes') }}:</td>
            <td class="om-p-cell-right mono">{{ optChange.nProc }}</td>
          </tr>
          <tr v-if="serverConfig.IsJobControl">
            <td colspan="2" class="om-p-head-left">{{ $t('Use Jobs Service') }}:</td>
            <td class="om-p-cell-right tc-max-width-10 row">
              <q-toggle
                v-model="optChange.isUseJob"
                disable
                :title="optChange.isUseJob ? $t('Use jobs service to run the model') : $t('Do not use jobs service to run the model')"
                />
            </td>
          </tr>
          <tr v-if="serverConfig.IsJobControl">
            <td colspan="2" class="om-p-head-left">{{ $t('Use MPI Root for Modelling') }}:</td>
            <td class="om-p-cell-right tc-max-width-10 row">
              <q-toggle
                v-model="optChange.isOnRoot"
                disable
                :title="optChange.isOnRoot ? $t('Use MPI root process to run the model') : $t('Do not use MPI root process to run the model')"
                />
            </td>
          </tr>
        </tbody>
        </table>
      </q-card-section>

      <q-card-actions vertical>
        <q-btn outline autofocus no-caps v-close-popup :label="$t('Use recommended options and run the model')" color="primary" @click="onYesOptsChange" />
        <q-btn outline no-caps v-close-popup :label="$t('Ignore recommended options and run the model')" color="primary" @click="onNoOptsChange" />
        <q-btn outline no-caps v-close-popup :label="$t('Let me change run options')" color="primary" />
      </q-card-actions>

    </q-card>
  </q-dialog>

  <q-inner-loading :showing="loadWait || loadConfig || loadDiskUse || loadIni || loadCsv || loadProfile">
    <q-spinner-gears size="md" color="primary" />
  </q-inner-loading>

</div>
</template>

<script src="./new-run.js"></script>

<style lang="scss" scope="local">
  .tc-right {
    text-align: right;
  }
  .tc-max-width-10 {
    max-width: 10rem;
  }
  .tc-min-width-10 {
    min-width: 10rem;
  }
  .primary-border-025 {
    border: 0.25rem solid $primary;
  }
  .file-link {
    text-decoration: none;
    // display: inline-block;
  }
</style>
