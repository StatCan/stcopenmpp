<template>
<q-layout view="hHh Lpr lFf" class="text-body1">

  <q-header>
    <q-toolbar>

      <q-btn
        @click="leftDrawerOpen = !leftDrawerOpen"
        flat
        round
        dense
        icon="menu"
        :title="$t('Menu')"
        :aria-label="$t('Menu')"
        />
      <q-btn
        v-if="!!isDiskUse && !!diskUseState?.DiskUse?.IsOver"
        to="/disk-use"
        flat
        round
        dense
        icon="mdi-database-alert"
        color="negative"
        class="bg-white q-ml-sm"
        :title="$t('View and cleanup storage space')"
        />
      <q-btn
        v-if="isModel"
        @click="doShowModelNote"
        flat
        round
        dense
        icon="mdi-information-outline"
        class="q-ml-sm"
        :title="$t('About') + ' ' + modelName"
        />
      <q-btn
        v-if="isModel && modelDocLink"
        :href="'/doc/' + modelDocLink"
        target="_blank"
        flat
        round
        dense
        icon="mdi-book-open-outline"
        class="text-white"
        :title="$t('Model Documentation') + ' ' + modelName"
        />

      <q-toolbar-title class="q-pl-xs">
        <router-link
          v-if="isModel"
          :to="'/model/' + modelDigest"
          :title="modelName"
          class="full-width ellipsis title-link"
          >
          {{ mainTitle }}
        </router-link>
        <a v-else
          class="title-link"
          :href="orgLink" target="_blank"
          :title="orgTitle"
          >
          <span class="full-width ellipsis text-white q">{{ orgTitle }}<q-icon name="3p" right/></span>
        </a>
      </q-toolbar-title>

      <q-btn
        v-if="isModel"
        type="a"
        :href="orgLink" target="_blank"
        flat
        round
        dense
        :title="orgTitle"
        :aria-label="orgTitle"
        icon="3p"
        />
      <q-btn
        v-if="loginUrl"
        type="a"
        :href="loginUrl"
        flat
        round
        dense
        :title="$t('Home')"
        :aria-label="$t('Home')"
        icon="mdi-home-account"
        />
      <q-btn
        v-if="logoutUrl"
        type="a"
        :href="logoutUrl"
        flat
        round
        dense
        :title="$t('Logout')"
        :aria-label="$t('Logout')"
        icon="mdi-logout"
        />
      <q-btn
        @click="onRefresh"
        flat
        round
        dense
        icon="refresh"
        :title="$t('Refresh')"
        :aria-label="$t('Refresh')"
      />

      <q-btn
        flat
        round
        dense
        icon="more_vert"
        :title="$t('More')"
        :aria-label="$t('More')"
        >
        <q-menu auto-close>
          <q-list>

            <q-item
              v-for="ln in appLanguages"
              :key="ln.isoName"
              @click="onLangMenu(ln.isoName)"
              clickable
              :active="langCode === ln.isoName"
              active-class="primary"
              >
              <q-item-section avatar>{{ ln.isoName }}</q-item-section>
              <q-item-section>{{ ln.nativeName }}</q-item-section>
            </q-item>
            <q-separator />

            <q-item
              clickable
              to="/license"
              >
              <q-item-section avatar>
                <q-icon name="mdi-license" />
              </q-item-section>
              <q-item-section>{{ $t('Licence') }}</q-item-section>
            </q-item>

            <template v-if="moreMenu()?.length">
              <q-separator />
              <q-item
                v-for="m in moreMenu()"
                :key="m.Link"
                clickable
                tag="a"
                target="_blank"
                :href="m.Link"
                >
                <q-item-section avatar>
                  <q-icon name="mdi-link" />
                </q-item-section>
                <q-item-section>{{ m.Label }}</q-item-section>
              </q-item>
            </template>

          </q-list>
        </q-menu>
      </q-btn>

    </q-toolbar>
  </q-header>

  <q-drawer
    v-model="leftDrawerOpen"
    bordered
    content-class="bg-grey-1"
    >
    <q-list>

      <q-item
        clickable
        to="/"
        exact
        >
        <q-item-section avatar>
          <q-icon name="mdi-folder-home-outline" />
        </q-item-section>
        <q-item-section>
          <q-item-label>{{ $t('Models') }}</q-item-label>
          <q-item-label caption>{{ $t('Models list') }}</q-item-label>
        </q-item-section>
        <q-item-section avatar>
          <q-badge v-if="modelCount" color="secondary" :label="modelCount" />
        </q-item-section>
      </q-item>
      <q-separator />

      <q-item
        clickable
        :disable="!isModel || !runTextCount"
        :to="'/model/' + modelDigest + '/run-list'"
        >
        <q-item-section avatar>
          <q-icon name="mdi-folder-table-outline" />
        </q-item-section>
        <q-item-section>
          <q-item-label>{{ $t('Model Runs') }}</q-item-label>
          <q-item-label caption>{{ $t('List of model runs') }}</q-item-label>
        </q-item-section>
        <q-item-section avatar>
          <q-badge v-if="isModel" color="secondary" :label="runTextCount" />
        </q-item-section>
      </q-item>

      <q-item
        clickable
        :disable="!isModel || !worksetTextCount"
        :to="'/model/' + modelDigest + '/set-list'"
        >
        <q-item-section avatar>
          <q-icon name="mdi-folder-edit-outline" />
        </q-item-section>
        <q-item-section>
          <q-item-label>{{ $t('Input Scenarios') }}</q-item-label>
          <q-item-label caption>{{ $t('List of input scenarios') }}</q-item-label>
        </q-item-section>
        <q-item-section avatar>
          <q-badge v-if="isModel" color="secondary" :label="worksetTextCount" />
        </q-item-section>
      </q-item>

      <q-item
        clickable
        :disable="!runDigestSelected && !worksetNameSelected && !taskNameSelected"
        :to="'/model/' + modelDigest + '/new-run'"
        >
        <q-item-section avatar>
          <q-icon name="mdi-run" />
        </q-item-section>
        <q-item-section>
          <q-item-label>{{ $t('Run the Model') }}</q-item-label>
        </q-item-section>
      </q-item>

      <q-item
        clickable
        :disable="!isModel || (!serverConfig.AllowDownload && !serverConfig.AllowUpload)"
        :to="'/model/' + modelDigest + '/updown-list'"
        >
        <q-item-section avatar>
          <q-icon name="mdi-download-circle" />
        </q-item-section>
        <q-item-section>
          <q-item-label>{{ $t('Downloads and Uploads') }}</q-item-label>
          <q-item-label caption>{{ $t('View downloads and uploads') }}</q-item-label>
        </q-item-section>
      </q-item>

      <q-separator />

      <q-item
        clickable
        to="/settings"
        >
        <q-item-section avatar>
          <q-icon name="settings" />
        </q-item-section>
        <q-item-section>
          <q-item-label>{{ $t('Settings') }}</q-item-label>
          <q-item-label caption class="ellipsis secondary">{{ $t('Session state and settings') }}</q-item-label>
        </q-item-section>
      </q-item>

      <q-item
        clickable
        :disable="!serverConfig.IsJobControl"
        to="/service-state"
        >
        <q-item-section avatar>
          <q-icon name="mdi-server-network" />
        </q-item-section>
        <q-item-section>
          <q-item-label>{{ $t('Service Status') }}</q-item-label>
          <q-item-label caption>{{ $t('Service status and model(s) run queue') }}</q-item-label>
        </q-item-section>
      </q-item>

      <q-item
        clickable
        :disable="!isDiskUse"
        to="/disk-use"
        >
        <q-item-section avatar>
          <q-icon
            :name="!!diskUseState?.DiskUse?.IsOver ? 'mdi-database-alert' : 'mdi-database'"
            :color="!!diskUseState?.DiskUse?.IsOver ? 'negative' : ''"
            />
        </q-item-section>
        <q-item-section>
          <q-item-label>{{ $t('Disk Usage') }}</q-item-label>
          <q-item-label caption>{{ $t('View and cleanup storage space') }}</q-item-label>
        </q-item-section>
        <q-item-section avatar>
          <q-badge v-if="!!diskUseState?.DiskUse?.TotalSize" color="secondary" :label="fileSizeStr(diskUseState?.DiskUse?.TotalSize)" />
        </q-item-section>
      </q-item>

      <q-item v-if="serverConfig.IsAdminAll"
        clickable
        :disable="!serverConfig.IsJobControl"
        to="/admin-state"
        >
        <q-item-section avatar>
          <q-icon name="mdi-server-security" />
        </q-item-section>
        <q-item-section>
          <q-item-label>{{ $t('Admin Status') }}</q-item-label>
          <q-item-label caption>{{ $t('Admin status and model(s) run') }}</q-item-label>
        </q-item-section>
      </q-item>

    </q-list>
  </q-drawer>

  <q-page-container>
    <router-view
      :refresh-tickle="refreshTickle"
      :to-up-down-section="toUpDownSection"
      @download-select="onDownloadSelect"
      @upload-select="onUploadSelect"
      @disk-use-refresh="onDiskUseRefresh"
      @set-update-readonly="onWorksetReadonlyUpdate"
      @user-view-updated="onUserViewUpdated"
      >
    </router-view>
  </q-page-container>

  <model-info-dialog :show-tickle="modelInfoTickle" :digest="modelDigest"></model-info-dialog>

  <template v-if="isRedirect">
    <refresh-model
      :digest="toModelDigest"
      :refresh-tickle="refreshTickle"
      @done="doneModelLoad"
      @wait="loadModelWait = true"
      >
    </refresh-model>
    <template v-if="modelDigest !== '' && (redirectTo === 2 || redirectTo === 3 || redirectTo === 4)">
      <refresh-run-list
        :digest="modelDigest"
        :refresh-tickle="refreshTickle"
        @done="doneRunListLoad"
        @wait="loadRunListWait = true"
        >
      </refresh-run-list>
      <refresh-run v-if="runDnsCurrent !== ''"
        :model-digest="modelDigest"
        :run-digest="runDnsCurrent"
        :refresh-tickle="refreshTickle"
        :refresh-run-tickle="refreshRunTickle"
        @done="doneRunLoad"
        @wait="loadRunWait = true"
        >
    </refresh-run>
    </template>
    <template v-if="modelDigest !== '' && redirectTo === 1">
      <refresh-workset-list
        :digest="modelDigest"
        :refresh-tickle="refreshTickle"
        @done="doneWsListLoad"
        @wait="loadWsListWait = true">
      </refresh-workset-list>
      <refresh-workset v-if="wsNameCurrent !== ''"
        :model-digest="modelDigest"
        :workset-name="wsNameCurrent"
        :refresh-tickle="refreshTickle"
        :refresh-workset-tickle="refreshWsTickle"
        @done="doneWsLoad"
        @wait="loadWsWait = true">
      </refresh-workset>
    </template>
  </template>

  <update-workset-status
    :model-digest="modelDigest"
    :workset-name="nameWsStatus"
    :is-readonly="isReadonlyWsStatus"
    :update-status-tickle="updateWsStatusTickle"
    @done="wsStatusUpadteWait = false"
    @wait="wsStatusUpadteWait = true">
  </update-workset-status>

  <template v-if="modelName !== ''">
    <refresh-user-views
      :model-name="modelName"
      @done="doneUserViewsLoad"
      @wait="loadUserViewsWait = true"
      >
    </refresh-user-views>
    <upload-user-views
      :model-name="modelName"
      :upload-views-tickle="uploadUserViewsTickle"
      @done="doneUserViewsUpload"
      @wait="uploadUserViewsWait = true"
      >
    </upload-user-views>
  </template>

  <q-inner-loading :showing="loadConfigWait || loadModelWait || loadRunListWait || loadRunWait || loadWsListWait || loadWsWait || loadUserViewsWait || uploadUserViewsWait">
    <q-spinner-gears size="xl" color="primary" />
  </q-inner-loading>

</q-layout>
</template>

<script src="./main-layout.js"></script>

<style lang="scss" scope="local">
  .title-link {
    text-decoration: none;
    color: white;
    // display: inline-block;
  }
</style>
