<template>
<div class="text-body1">

  <div class="q-pa-sm">
    <q-toolbar class="shadow-1 rounded-borders">

      <template v-if="isFromRun">
        <run-bar
          :model-digest="digest"
          :run-digest="runDigest"
          @run-info-click="doShowRunNote"
          >
        </run-bar>
      </template>

      <template v-else>
        <workset-bar
          :model-digest="digest"
          :workset-name="worksetName"
          :is-readonly-button="!isReadonlyServer"
          :is-new-run-button="isModelTab && !isReadonlyServer"
          :is-show-menu="true"
          @set-info-click="doShowWorksetNote"
          @set-update-readonly="onWorksetReadonlyToggle"
          @new-run-select="onNewRunClick"
          >
        </workset-bar>
      </template>

    </q-toolbar>
  </div>

  <!-- parameter header -->
  <div class="q-mx-sm q-mb-sm">
    <q-toolbar class="row reverse-wrap items-center shadow-1 rounded-borders">

    <!-- menu -->
    <q-btn
      outline
      dense
      class="col-auto text-primary rounded-borders"
      icon="menu"
      :title="$t('Menu')"
      :aria-label="$t('Menu')"
      >
      <q-menu auto-close>
        <q-list dense>

          <q-item
            @click="doShowParamNote"
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-information-outline" />
            </q-item-section>
            <q-item-section>{{ $t('About') + ' ' + parameterName }}</q-item-section>
          </q-item>
          <q-item
            clickable
            v-close-popup
            tag="a"
            target="_blank"
            :href="'/?model=' + encodeURIComponent(digest) + (isFromRun ? '&run=' + encodeURIComponent(runDigest) : '&set=' + encodeURIComponent(worksetName)) + '&parameter=' + encodeURIComponent(parameterName)"
            >
            <q-item-section avatar>
              <q-icon color="primary" name="open_in_new" />
            </q-item-section>
            <q-item-section>{{ $t('Open in new window') }}</q-item-section>
          </q-item>

          <q-separator />

          <template v-if="!isFromRun">
            <q-item
              @click="doEditToogle"
              :disable="!edt.isEnabled || noteEditorShow || isReadonlyServer"
              clickable
              >
              <q-item-section avatar>
                <q-icon color="primary" :name="edt.isEdit ? 'mdi-close-circle-outline' : 'mdi-table-edit'" />
              </q-item-section>
              <q-item-section>{{ (edt.isEdit ? $t('Discard changes of') : $t('Edit')) + ' ' + parameterName }}</q-item-section>
            </q-item>
            <q-item
              @click="onEditSave"
              :disable="!edt.isEnabled || !edt.isUpdated || isReadonlyServer"
              clickable
              >
              <q-item-section avatar>
                <q-icon color="primary" name="mdi-content-save-edit" />
              </q-item-section>
              <q-item-section>{{ $t('Save') + ' ' + parameterName }}</q-item-section>
            </q-item>
            <q-item
              @click="onUndo"
              :disable="!edt.isEnabled || edt.lastHistory <= 0"
              clickable
              >
              <q-item-section avatar>
                <q-icon color="primary" name="mdi-undo-variant" />
              </q-item-section>
              <q-item-section>{{ $t('Undo') + ': Ctrl+Z' }}</q-item-section>
            </q-item>
            <q-item
              @click="onRedo"
              :disable="!edt.isEnabled || edt.lastHistory >= edt.history.length"
              clickable
              >
              <q-item-section avatar>
                <q-icon color="primary" name="mdi-redo-variant" />
              </q-item-section>
              <q-item-section>{{ $t('Redo') + ': Ctrl+Y' }}</q-item-section>
            </q-item>
            <q-separator />
          </template>

          <q-item
            v-if="!noteEditorShow"
            @click="onEditParamNote()"
            :disable="!isFromRun && (!edt.isEnabled || edt.isEdit) || isReadonlyServer"
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-file-document-edit-outline" />
            </q-item-section>
            <q-item-section>{{ $t('Edit notes for') + ' ' + parameterName }}</q-item-section>
          </q-item>
          <q-item
            v-if="noteEditorShow"
            @click="onCancelParamNote()"
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-close-circle-outline" />
            </q-item-section>
            <q-item-section>{{ $t('Discard changes to notes for') + ' ' + parameterName }}</q-item-section>
          </q-item>
          <q-item
            @click="onSaveParamNote()"
            :disable="!noteEditorShow || (!isFromRun && (!edt.isEnabled || edt.isEdit))"
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-content-save-edit" />
            </q-item-section>
            <q-item-section>{{ $t('Save notes for') + ' ' + parameterName }}</q-item-section>
          </q-item>

          <q-separator />

          <q-item
            @click="onCopyToClipboard"
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-content-copy" />
            </q-item-section>
            <q-item-section>{{ $t('Copy tab separated values to clipboard: ') + 'Ctrl+C' }}</q-item-section>
          </q-item>
          <q-item
            @click="onDownload"
            :disable="!isFromRun && edt.isEdit"
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-download" />
            </q-item-section>
            <q-item-section>{{ $t('Download') + ' '  + parameterName + ' ' + $t('as CSV') }}</q-item-section>
          </q-item>
          <q-item
            v-if="!isFromRun && !isReadonlyServer"
            @click="doShowFileSelect()"
            v-show="!uploadFileSelect"
            :disable="!isUploadEnabled || edt.isEdit"
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-upload" />
            </q-item-section>
            <q-item-section>{{ $t('Upload') + ' ' + parameterName + '.csv' }}</q-item-section>
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

          <q-separator />

          <q-item
            @click="onToggleRowColControls"
            :disable="!ctrl.isRowColModeToggle"
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
            :disable="isReadonlyServer || edt.isEdit"
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-content-save-cog" />
            </q-item-section>
            <q-item-section>{{ $t('Save parameter view as default view of') + ' ' + parameterName }}</q-item-section>
          </q-item>
          <q-item
            @click="onReloadDefaultView"
            :disable="edt.isEdit"
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-cog-refresh-outline" />
            </q-item-section>
            <q-item-section>{{ $t('Reset parameter view to default and reload') + ' ' + parameterName }}</q-item-section>
          </q-item>

        </q-list>
      </q-menu>
    </q-btn>
    <!-- end of menu -->

    <q-btn
      @click="doShowParamNote"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-ml-xs"
      icon="mdi-information"
      :title="$t('About') + ' ' + parameterName"
      />
    <q-btn
      tag="a"
      target="_blank"
      :href="'/?model=' + encodeURIComponent(digest) + (isFromRun ? '&run=' + encodeURIComponent(runDigest) : '&set=' + encodeURIComponent(worksetName)) + '&parameter=' + encodeURIComponent(parameterName)"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-ml-xs"
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

    <template v-if="!isFromRun">
      <q-btn
        @click="doEditToogle"
        :disable="!edt.isEnabled || uploadFileSelect || noteEditorShow || isReadonlyServer"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders"
        :icon="edt.isEdit ? 'cancel' : 'mdi-table-edit'"
        :title="(edt.isEdit ? $t('Discard changes of') : $t('Edit')) + ' ' + parameterName"
        />
      <q-btn
        @click="onEditSave"
        :disable="!edt.isEnabled || !edt.isUpdated || isReadonlyServer"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders q-ml-xs"
        icon="mdi-content-save-edit"
        :title="$t('Save') + ' ' + parameterName"
        />
      <q-btn
        @click="onUndo"
        :disable="!edt.isEnabled || edt.lastHistory <= 0"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders q-ml-xs"
        icon="mdi-undo-variant"
        :title="$t('Undo') + ': Ctrl+Z'"
        />
      <q-btn
        @click="onRedo"
        :disable="!edt.isEnabled || edt.lastHistory >= edt.history.length"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders q-ml-xs"
        icon="mdi-redo-variant"
        :title="$t('Redo') + ': Ctrl+Y'"
        />
      <q-separator vertical inset spaced="sm" color="secondary" />
    </template>

    <q-btn
      v-if="!noteEditorShow && !isReadonlyServer"
      @click="onEditParamNote()"
      :disable="!isFromRun && (!edt.isEnabled || edt.isEdit)"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders"
      icon="mdi-file-document-edit-outline"
      :title="$t('Edit notes for') + ' ' + parameterName"
      />
    <q-btn
      v-if="noteEditorShow"
      @click="onCancelParamNote()"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-ml-xs"
      icon="cancel"
      :title="$t('Discard changes to notes for') + ' ' + parameterName"
      />
    <q-btn
      @click="onSaveParamNote()"
      :disable="!noteEditorShow || (!isFromRun && (!edt.isEnabled || edt.isEdit))"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-ml-xs"
      icon="mdi-content-save-edit"
      :title="$t('Save notes for') + ' ' + parameterName"
      />
    <q-separator vertical inset spaced="sm" color="secondary" />

    <q-btn
      @click="onCopyToClipboard"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders"
      icon="mdi-content-copy"
      :title="$t('Copy tab separated values to clipboard: ') + 'Ctrl+C'"
      />
    <q-btn
      @click="onDownload"
      :disable="!isFromRun && edt.isEdit"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-ml-xs"
      icon="mdi-download"
      :title="$t('Download') + ' '  + parameterName + ' ' + $t('as CSV')"
      />

    <q-btn
      v-if="!isFromRun && !isReadonlyServer"
      @click="doShowFileSelect()"
      v-show="!uploadFileSelect"
      :disable="!isUploadEnabled || edt.isEdit || noteEditorShow"
      flat
      dense
      class="col-auto text-white rounded-borders q-ml-xs bg-primary text-white rounded-borders"
      icon="mdi-upload"
      :title="$t('Upload') + ' ' + parameterName + '.csv'"
      />
    <q-btn
      @click="doCancelFileSelect()"
      v-show="uploadFileSelect"
      flat
      dense
      class="col-auto text-white rounded-borders q-ml-xs bg-primary text-white rounded-borders"
      icon="mdi-close-circle"
      :title="$t('Cancel upload')"
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
        icon="mdi-view-list-outline"
        :title="$t('Table view: hide rows and columns name')"
        />
      <q-btn
        @click="onSetRowColMode(3)"
        :disable="pvc.rowColMode === 3"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders q-mr-xs"
        icon="mdi-view-compact-outline"
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

    <template v-if="ctrl.isFloatView">
      <q-btn
        @click="onShowMoreFormat"
        :disable="!ctrl.isMoreView || edt.isEdit"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders q-mr-xs"
        icon="mdi-decimal-increase"
        :title="$t('Increase precision')"
        />
      <q-btn
        @click="onShowLessFormat"
        :disable="!ctrl.isLessView || edt.isEdit"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders q-mr-xs"
        icon="mdi-decimal-decrease"
        :title="$t('Decrease precision')"
        />
    </template>
    <q-btn
      v-if="ctrl.isRawUseView"
      @click="onToggleRawValue"
      :disable="edt.isEdit"
      :flat="!ctrl.isRawView || edt.isEdit"
      :outline="ctrl.isRawView && !edt.isEdit"
      dense
      :class="(!ctrl.isRawView || edt.isEdit) ? 'bar-button-on' : 'bar-button-off'"
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
      :disable="edt.isEdit || isReadonlyServer"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-mr-xs"
      icon="mdi-content-save-cog"
      :title="$t('Save table view as default view of') + ' ' + parameterName"
      />
    <q-btn
      @click="onReloadDefaultView"
      :disable="edt.isEdit"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-mr-xs"
      icon="mdi-cog-refresh-outline"
      :title="$t('Reset table view to default and reload') + ' ' + parameterName"
      />

    <div
      class="col-auto"
      >
      <span>{{ parameterName }}<br />
      <span class="om-text-descr">{{ paramDescr }}</span></span>
    </div>

    </q-toolbar>

    <q-card v-if="uploadFileSelect">

      <div class="row q-mt-xs q-pa-sm">
        <q-btn
          @click="onUploadParameter"
          v-if="uploadFileSelect"
          :disable="!fileSelected"
          flat
          dense
          class="col-auto bg-primary text-white rounded-borders"
          icon="mdi-upload"
          :title="$t('Upload selected file')"
          />
        <q-file
          v-model="uploadFile"
          v-if="uploadFileSelect"
          accept='.csv'
          outlined
          dense
          clearable
          hide-bottom-space
          class="col q-pl-xs"
          color="primary"
          :label="$t('Select') + ' ' + parameterName + '.csv'"
          >
        </q-file>
      </div>

      <div class="row items-center q-mt-xs q-pa-sm">
        <span class="col-auto q-px-md"></span>
        <span class="col-auto q-px-xs">{{ $t('Sub-values Count') }}:</span>
        <q-input
          v-model="subCountUpload"
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
          class="upload-max-width-10"
          input-class="col-auto upload-right"
          :title="$t('Number of sub-values (a.k.a. members or replicas or sub-samples)')"
          >
        </q-input>

        <span class="col-auto q-pl-md q-pr-xs">{{ $t('Default Sub-value') }}:</span>
        <q-input
          v-model="defaultSubUpload"
          :disable="subCountUpload < 2"
          type="number"
          maxlength="4"
          outlined
          dense
          hide-bottom-space
          class="upload-max-width-10"
          input-class="col-auto upload-right"
          :title="$t('Default sub-value, if parameter has more than 1 sub-value')"
          >
        </q-input>
      </div>
    </q-card>
  </div>
  <!-- end of parameter header -->

  <div class="q-mx-sm q-mb-sm">

  <markdown-editor
    v-if="noteEditorShow"
    ref="param-note-editor"
    class="q-px-none q-py-xs"
    :the-note="noteEditorNotes"
    :lang-code="noteEditorLangCode"
    :description-editable="false"
    :notes-editable="true"
    :is-hide-save="true"
    :is-hide-cancel="true"
    >
  </markdown-editor>

  </div>

  <!-- pivot table controls and view -->
  <div class="q-mx-sm">

    <div v-show="ctrl.isRowColControls"
      class="other-panel"
      >
      <draggable
        v-model="otherFields"
        :disabled="!!edt.isEdit ? true : null"
        group="fields"
        @start="onDrag"
        @end="onDrop"
        class="other-fields other-drag"
        :class="{'drag-area-hint': isDragging, 'drag-area-disabled': edt.isEdit}"
        >
        <div v-for="f in otherFields" :key="f.name" :id="'item-draggable-' + f.name" class="field-drag om-text-medium">
          <q-select
            :model-value="f.singleSelection"
            @update:model-value="onUpdateSelect"
            @focus="onFocusSelect"
            :disable="edt.isEdit"
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
                :class="{'disabled': edt.isEdit}"
                :title="$t('Drag and drop')"
                />
              <div class="column">
                <q-icon
                  name="check"
                  size="xs"
                  class="row bg-primary text-white rounded-borders select-handle-button om-bg-inactive q-mb-xs"
                  :class="{'disabled': edt.isEdit}"
                  :title="$t('Select all')"
                  />
                <q-icon
                  name="cancel"
                  size="xs"
                  class="row bg-primary text-white rounded-borders select-handle-button om-bg-inactive"
                  :class="{'disabled': edt.isEdit}"
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
        :pv-edit="edt"
        @pv-key-pos="onPvKeyPos"
        @pv-edit="onPvEdit"
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
  <refresh-workset v-if="(digest || '') !== '' && (worksetName || '') !== ''"
    :model-digest="digest"
    :workset-name="worksetName"
    :refresh-tickle="refreshTickle"
    :refresh-workset-tickle="refreshWsTickle"
    @done="loadWsWait = false"
    @wait="loadWsWait = true"
    >
  </refresh-workset>

  <run-info-dialog v-if="isFromRun" :show-tickle="runInfoTickle" :model-digest="digest" :run-digest="runDigest"></run-info-dialog>
  <workset-info-dialog v-if="!isFromRun" :show-tickle="worksetInfoTickle" :model-digest="digest" :workset-name="worksetName"></workset-info-dialog>
  <parameter-info-dialog v-if="isFromRun" :show-tickle="paramInfoTickle" :param-name="parameterName" :run-digest="runDigest"></parameter-info-dialog>
  <parameter-info-dialog v-if="!isFromRun" :show-tickle="paramInfoTickle" :param-name="parameterName" :workset-name="worksetName"></parameter-info-dialog>
  <edit-discard-dialog
    @discard-changes-yes="onYesDiscardChanges"
    :show-tickle="showEditDiscardTickle"
    :dialog-title="$t('Discard all changes') + '?'"
    >
  </edit-discard-dialog>
  <edit-discard-dialog
    @discard-changes-yes="onYesDiscardParamNote"
    :show-tickle="showDiscardParamNoteTickle"
    :dialog-title="$t('Discard changes to notes for') + ' ' + parameterName + '?'"
    >
  </edit-discard-dialog>

  <q-inner-loading :showing="loadWait || loadRunWait || loadWsWait">
    <q-spinner-gears size="md" color="primary" />
  </q-inner-loading>

</div>
</template>

<script src="./parameter-page.js"></script>

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

  .upload-right {
    text-align: right;
  }
  .upload-max-width-10 {
    max-width: 10rem;
  }
</style>
