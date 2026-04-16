<template>
<q-page class="text-body1">

  <q-card
    class="q-mx-sm"
    >
    <q-card-section>

      <table class="om-p-table">
        <thead>
          <tr>
            <th class="om-p-head-center">{{ $t('Storage Space') }}</th>
            <th colspan="2" class="om-p-head-center text-weight-medium mono">{{ fileTimeStamp(diskUseState.DiskUse.UpdateTs) }}</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td class="om-p-head-left">{{ $t('Used by You') }}</td>
            <td colspan="2" class="om-p-cell-right mono" :class="isOver ? 'text-negative text-weight-bold' : ''">{{ fileSizeStr(diskUseState.DiskUse.TotalSize) }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Your Limit') }}</td>
            <td
              colspan="2"
              class="om-p-cell-right mono"
              :class="(diskUseState.DiskUse.Limit > 0 && diskUseState.DiskUse.TotalSize >= diskUseState.DiskUse.Limit) ? 'text-negative text-weight-bold' : ''"
              >{{ diskUseState.DiskUse.Limit > 0 ? fileSizeStr(diskUseState.DiskUse.Limit) : 'unlimited' }}</td>
          </tr>
          <tr v-if="diskUseState.DiskUse.AllLimit > 0">
            <td class="om-p-head-left">{{ $t('All Users Total') }}</td>
            <td
              colspan="2"
              class="om-p-cell-right mono"
              :class="(diskUseState.DiskUse.AllSize >= diskUseState.DiskUse.AllLimit) ? 'text-negative text-weight-bold' : ''"
              >{{ fileSizeStr(diskUseState.DiskUse.AllSize) }}</td>
          </tr>
          <tr v-if="diskUseState.DiskUse.AllLimit > 0">
            <td class="om-p-head-left">{{ $t('All Users Limit') }}</td>
            <td
              colspan="2"
              class="om-p-cell-right mono"
              :class="(diskUseState.DiskUse.AllSize >= diskUseState.DiskUse.AllLimit) ? 'text-negative text-weight-bold' : ''"
              >{{ !!diskUseState.DiskUse.AllLimit ? fileSizeStr(diskUseState.DiskUse.AllLimit) : 'unlimited' }}</td>
          </tr>
          <tr v-if="isDownloadEnabled">
            <td class="om-p-head-left">{{ $t('Downloads') }}</td>
            <td class="om-p-cell-center">
              <q-btn
                :disable="!diskUseState.DiskUse.DownSize"
                @click="onAllDownloadDelete"
                unelevated
                round
                color="primary"
                :icon="diskUseState.DiskUse.DownSize > 0 ? 'mdi-delete' : 'mdi-delete-outline'"
                :title="$t('Delete all download files')"
                />
            </td>
            <td class="om-p-cell-right mono">{{ fileSizeStr(diskUseState.DiskUse.DownSize) }}</td>
          </tr>
          <tr v-if="isUploadEnabled">
            <td class="om-p-head-left">{{ $t('Uploads') }}</td>
            <td class="om-p-cell-center">
              <q-btn
                :disable="!diskUseState.DiskUse.UpSize"
                @click="onAllUploadDelete"
                unelevated
                round
                color="primary"
                :icon="diskUseState.DiskUse.UpSize > 0 ? 'mdi-delete' : 'mdi-delete-outline'"
                :title="$t('Delete all upload files')"
                />
            </td>
            <td class="om-p-cell-right mono">{{ fileSizeStr(diskUseState.DiskUse.UpSize) }}</td>
          </tr>
        </tbody>
      </table>

    </q-card-section>

    <q-card-section v-if="dbUseLst.length > 0" class="q-pt-sm">

      <table class="om-p-table">
        <thead>
          <tr>
            <th class="om-p-head-center">{{ $t('Status') }}</th>
            <th class="om-p-head-center">{{ $t('Open / Close') }}</th>
            <th v-if="isCleanupEnabled" class="om-p-head-center">{{ $t('Cleanup') }}</th>
            <th v-if="isCleanupEnabled" class="om-p-head-center">{{ $t('Delete') }}</th>
            <th class="om-p-head-center text-weight-medium">{{ $t('Size') }}</th>
            <th class="om-p-head-center text-weight-medium">{{ $t('Updated') }}</th>
            <th class="om-p-head-center text-weight-medium">{{ $t('Model Database') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="dbu in dbUseLst" :key="'md-' + (dbu.path || 'no-path')">
            <td class="om-p-cell-center">
              <q-icon v-if="dbu.isOn" size="lg" color="primary" name="mdi-database-check-outline" :title="$t('model database') + ' ' + dbu.nameVer" />
              <q-icon v-if="dbu.isOff" size="lg" color="primary" name="mdi-database-lock" :title="$t('model database offline')" />
              <q-icon v-if="dbu.isCleanup" size="lg" color="primary" name="mdi-database-clock" :title="$t('model database maintenance in progress')" />
            </td>
            <td class="om-p-cell-center">
              <q-btn
                v-if="dbu.isOn"
                @click="onCloseDb(dbu)"
                unelevated
                round
                color="primary"
                icon="mdi-lock-open-variant"
                :title="$t('Close model database') + ' ' + dbu.nameVer"
                />
              <q-btn
                v-if="dbu.isOff"
                @click="onOpenDb(dbu)"
                outline
                round
                color="primary"
                icon="mdi-lock"
                :title="$t('Open model database')"
                />
            </td>
            <td v-if="isCleanupEnabled" class="om-p-cell-center">
              <q-btn
                v-if="dbu.isOff || dbu.isOn"
                :disable="dbu.isOn || !isCleanupEnabled"
                @click="onCleanupDb(dbu)"
                unelevated
                round
                color="primary"
                icon="mdi-vacuum"
                :title="$t('Cleanup model database')"
                />
            </td>
            <td v-if="isCleanupEnabled" class="om-p-cell-center">
              <q-btn
                v-if="dbu.isOn"
                :disable="dbu.isOff || !isCleanupEnabled"
                @click="onDeleteModel(dbu)"
                unelevated
                round
                color="primary"
                icon="mdi-delete-forever"
                :title="$t('Delete the model')"
                />
            </td>
            <td class="om-p-cell-right mono">{{ fileSizeStr(dbu.size) }}</td>
            <td class="om-p-cell-left mono">{{ fileTimeStamp(dbu.modTs) }}</td>
            <td class="om-p-cell-left"
              ><span class="om-text-descr">{{ dbu.path  }}</span>
              <template v-if="dbu.nameVer !== ''"><br /><span>{{ dbu.nameVer  }}</span></template>
              <template v-if="dbu.descr !== ''"><br /><span class="om-text-descr">{{ dbu.descr }}</span></template></td>
          </tr>
        </tbody>
      </table>

    </q-card-section>

    <q-card-section v-if="isCleanupLogList" class="q-pt-sm">

      <table class="om-p-table">
        <thead>
          <tr>
            <th class="om-p-head-center text-weight-medium"></th>
            <th class="om-p-head-center text-weight-medium">{{ $t('Started') }}</th>
            <th class="om-p-head-center text-weight-medium">{{ $t('Model Database') }}</th>
            <th class="om-p-head-center text-weight-medium">{{ $t('Cleanup Log File') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="cf in cleanupLogLst" :key="'cf-' + (cf.LogStamp || 'no-stamp')">
            <td class="om-p-cell-left mono">
              <q-btn
                @click="doShowCleanupLog(cf.LogFileName)"
                unelevated
                round
                color="primary"
                icon="mdi-text-long"
                :title="$t('Log File') + ' ' + cf.LogFileName"
                />
            </td>
            <td class="om-p-cell-left mono">{{ fromTimeStamp(cf.LogStamp) }}</td>
            <td class="om-p-cell-left">{{ cf.DbName }}</td>
            <td class="om-p-cell-left">{{ cf.LogFileName }}</td>
          </tr>
        </tbody>
      </table>

    </q-card-section>

  </q-card>

  <delete-confirm-dialog
    @delete-yes="onYesAllUpDownDelete"
    :show-tickle="showAllDeleteDialogTickle"
    :item-name="(upDownToDelete === 'up' ? $t('All upload files') :  $t('All download files'))"
    :kind="upDownToDelete"
    :bodyText="upDownToDelete === 'up' ? $t('Delete all files in upload folder.') : $t('Delete all files in download folder.')"
    :dialog-title="(upDownToDelete === 'up' ? $t('Delete all upload files') :  $t('Delete all download files')) + '?'"
    >
  </delete-confirm-dialog>

  <confirm-dialog
    @confirm-yes="onYesCloseDb"
    :show-tickle="showCloseDbDialogTickle"
    :item-id="digestModelDb"
    :item-name="nameVerModelDb"
    :dialog-title="$t('Close model database' + '?')"
    :body-text="$t('Close')"
    :body-note="$t('You would not be able to use the model if database is closed.')"
    :icon-name="'mdi-lock'"
    >
  </confirm-dialog>
  <confirm-dialog
    @confirm-yes="onYesCleanupDb"
    :show-tickle="showCleanupDbDialogTickle"
    :item-name="pathCleanupDb"
    :dialog-title="$t('Cleanup model database' + '?')"
    :body-text="$t('Cleanup')"
    :body-note="$t('Cleanup may take very long time. It is recommended to delete old model runs and old input scenarios before cleanup.')"
    :icon-name="'mdi-vacuum'"
    >
  </confirm-dialog>
  <confirm-dialog
    @confirm-yes="onYesDeleteModel"
    :show-tickle="showDeleteModelDialogTickle"
    :item-id="digestModelDb"
    :item-name="nameVerModelDb"
    :dialog-title="$t('Delete the model' + '?')"
    :body-text="$t('Delete')"
    :body-note="$t('All model data will be deleted permanently')"
    :icon-name="'mdi-delete-forever'"
    >
  </confirm-dialog>

  <q-dialog full-width v-model="showCleanupLog">
    <q-card>

      <q-card-section class="row text-h6 bg-primary text-white">
        <div>{{ titleCleanupLog }}</div><q-space /><q-btn icon="mdi-close" flat dense round v-close-popup />
      </q-card-section>

      <q-card-section class="q-pb-none">
        <span class="mono">{{dtCleanupLog}} <i>[{{nameCleanupLog}}]</i></span>
        <div>
          <pre>{{linesCleanupLog.join('\n')}}</pre>
        </div>
      </q-card-section>

      <q-card-actions align="right">
        <q-btn flat :label="$t('OK')" color="primary" v-close-popup />
      </q-card-actions>

    </q-card>
  </q-dialog>

  <q-inner-loading :showing="loadWait">
    <q-spinner-gears size="lg" color="primary" />
  </q-inner-loading>

</q-page>
</template>

<script src="./disk-use.js"></script>

<style lang="scss" scope="local">
</style>
