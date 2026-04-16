import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'
import languages from 'quasar/lang/index.json'
import * as Mdf from 'src/model-common'
import RefreshModel from 'components/RefreshModel.vue'
import RefreshRunList from 'components/RefreshRunList.vue'
import RefreshRun from 'components/RefreshRun.vue'
import RefreshWorksetList from 'components/RefreshWorksetList.vue'
import RefreshWorkset from 'components/RefreshWorkset.vue'
import RefreshUserViews from 'components/RefreshUserViews.vue'
import UploadUserViews from 'components/UploadUserViews.vue'
import UpdateWorksetStatus from 'components/UpdateWorksetStatus.vue'
import ModelInfoDialog from 'components/ModelInfoDialog.vue'

const DISK_USE_MIN_REFRESH_TIME = (17 * 1000) // msec, minimum disk space usage refresh interval
const DISK_USE_MAX_ERR = 5 // max error count to stop disk use retrival and block model runs
const NO_REDIRECT = 0
const WS_PARAM_REDIRECT = 1 // redirect by url to workset parameter
const RUN_PARAM_REDIRECT = 2 // redirect by url to run parameter
const TABLE_REDIRECT = 3 // redirect by url to output table
const ENTITY_REDIRECT = 4 // redirect by url to run parameter

export default {
  name: 'MainLayout',
  components: {
    RefreshModel,
    RefreshRunList,
    RefreshRun,
    RefreshWorksetList,
    RefreshWorkset,
    RefreshUserViews,
    UploadUserViews,
    UpdateWorksetStatus,
    ModelInfoDialog
  },

  data () {
    return {
      leftDrawerOpen: false,
      refreshTickle: false,
      refreshRunTickle: false,
      refreshWsTickle: false,
      loadConfigWait: false,
      loadDiskUseDone: false,
      nDiskUseErr: 0, // disk use error count
      loginUrl: '',
      logoutUrl: '',
      orgTitle: 'OpenM++',
      orgLink: 'https://openmpp.org',
      isLoginCatch: false,
      modelInfoTickle: false,
      toUpDownSection: 'down',
      isDiskUse: false,
      diskUseRefreshInt: '',
      diskUseMs: DISK_USE_MIN_REFRESH_TIME,
      //
      isRedirect: false, // redirect by url
      redirectTo: NO_REDIRECT,
      toModelDigest: '',
      toRunDns: '',
      runDnsCurrent: '',
      toWsName: '',
      wsNameCurrent: '',
      toParamName: '',
      toTableName: '',
      toEntityName: '',
      loadModelWait: false,
      loadWsListWait: false,
      loadWsWait: false,
      loadRunListWait: false,
      loadRunWait: false,
      loadUserViewsWait: false,
      //
      wsStatusUpadteWait: false,
      isReadonlyWsStatus: false,
      nameWsStatus: '',
      updateWsStatusTickle: false,
      uploadUserViewsTickle: false,
      uploadUserViewsWait: false,
      //
      langCode: this.$q.lang.getLocale(),
      appLanguages: languages.filter(lang => ['fr', 'en-US'].includes(lang.isoName)) // list of languages: code and label
    }
  },

  computed: {
    mainTitle () {
      const t = (this.theModelDir ? this.theModelDir + '/' : '') + Mdf.modelTitle(this.theModel)
      return (t !== '') ? t : this.orgTitle
    },
    isModel () { return Mdf.isModel(this.theModel) },
    modelDigest () { return Mdf.modelDigest(this.theModel) },
    modelName () { return Mdf.modelName(this.theModel) },
    modelDocLink () {
      return this.serverConfig.IsModelDoc ? Mdf.modelDocLinkByDigest(this.modelDigest, this.modelList, this.uiLang, this.modelLanguage) : ''
    },
    runTextCount () { return Mdf.runTextCount(this.runTextList) },
    worksetTextCount () { return Mdf.worksetTextCount(this.worksetTextList) },

    ...mapState(useModelStore, [
      'theModel',
      'theModelDir',
      'modelList',
      'runTextList',
      'worksetTextList',
      'modelCount',
      'modelLanguage'
    ]),
    ...mapState(useServerStateStore, {
      omsUrl: 'omsUrl',
      serverConfig: 'config',
      diskUseState: 'diskUse'
    }),
    ...mapState(useUiStateStore, [
      'uiLang',
      'defaultTitle',
      'runDigestSelected',
      'worksetNameSelected',
      'taskNameSelected',
     'paramViewWorksetUpdatedCount'
    ])
  },

  watch: {
    isDiskUse () { this.restartDiskUseRefresh() },
    diskUseMs () { this.restartDiskUseRefresh() },
    modelCount () { this.doRedirectByUrl() },
    defaultTitle () { if (this.$route.path === '/' && this.defaultTitle) document.title = this.defaultTitle },

    // language updated outside of main menu
    uiLang () {
      if (!this.uiLang) {
        let lc = this.$q.lang.getLocale()
        if (this.appLanguages.findIndex((ln) => ln.isoName === lc) < 0) { // language not included in translation pack, use default en-US
          lc = 'en-US'
        }
        this.langCode = lc
        this.$i18n.locale = lc
      }
      this.doOrgTitle()
    },

    // switch app language: Quasar and vue i18n language
    langCode (lc) {
      if (this.appLanguages.findIndex((ln) => ln.isoName === lc) < 0) { // language not included in translation pack, use default en-US
          lc = 'en-US'
          this.langCode = lc
      }
      this.$i18n.locale = lc

      // find Quasar language translation
      try {
        if (Array.isArray(this.$quasarLangs)) {
          const j = this.$quasarLangs.findIndex((ln) => ln.default.isoName === lc)
          if (j >= 0) {
            this.$q.lang.set(this.$quasarLangs[j].default)
          } else {
            console.warn('Quasar language not found:', lc)
          }
        }
      } catch (err) {
        console.warn('Error at loading language:', lc, err)
      }
    },

    // set 401 and 403 interceptor: open login page
    loginUrl () {
      if (this.isLoginCatch || !this.loginUrl || typeof this.loginUrl !== typeof 'string') return // use only first non-empty login url

      this.$nextTick(() => {
        this.$axios.interceptors.response.use(
          (response) => response,
          (error) => {
            if (error.response) {
              if (error.response.status === 401 || error.response.status === 403) { // open login URL
                console.warn('Authentication session expired, status:', error.response.status)
                this.$q.notify({ type: 'negative', message: this.$t('Login session expired or authorization error.') })
                setTimeout(
                  () => window.location.assign(this.loginUrl),
                  5000)
              }
            }
            return Promise.reject(error)
          })
      })
      this.isLoginCatch = true
    }
  },

  methods: {
    ...mapActions(useServerStateStore, [
      'dispatchServerConfig',
      'dispatchDiskUse'
    ]),
    ...mapActions(useUiStateStore, [
      'dispatchUiLang',
      'dispatchDefaultTitle',
      'dispatchSortModelTree',
      'dispatchRunDigestSelected',
      'dispatchWorksetNameSelected'
    ]),

    // show model notes dialog
    doShowModelNote () {
      this.modelInfoTickle = !this.modelInfoTickle
    },
    // new selected language in the menu
    onLangMenu (lc) {
      this.langCode = lc // switch app language by watch
      if (lc) { // use selected language for model text metadata
        this.dispatchUiLang(lc)
        this.refreshTickle = !this.refreshTickle
      }
    },
    // update org title and link on language switch
    doOrgTitle () {
      if (!this.serverConfig?.UiExtra || !Array.isArray(this.serverConfig.UiExtra?.DefaultTitle)) return

      const ulc = this.uiLang || this.$q.lang.getLocale()

      for (const m of this.serverConfig.UiExtra.DefaultTitle) {
        if (m.LangCode === ulc) {
          this.orgTitle = m?.Label || 'OpenM++'
          this.orgLink = m?.Link || 'https://openmpp.org'
          break
        }
        const m2p = Mdf.splitLangCode(m.LangCode)
        const ui2p = Mdf.splitLangCode(ulc)
        if (m2p.isEmpty || ui2p.isEmpty) {
          continue
        }
        if (m2p.lower === ui2p.lower) {
          this.orgTitle = m?.Label || 'OpenM++'
          this.orgLink = m?.Link || 'https://openmpp.org'
          break
        }
        if (m2p.first === ui2p.first) {
          this.orgTitle = m?.Label || 'OpenM++'
          this.orgLink = m?.Link || 'https://openmpp.org'
          break
        }
      }

      this.dispatchDefaultTitle(this.orgTitle)
    },
    // return more menu extra items array for current UI language
    moreMenu () {
      if (!this.serverConfig?.UiExtra || !Array.isArray(this.serverConfig.UiExtra?.MoreMenu)) return []

      const ulc = this.uiLang || this.$q.lang.getLocale()
      const mm = []
      for (const m of this.serverConfig.UiExtra.MoreMenu) {
        if (m.LangCode === ulc) {
          mm.push(m)
          continue
        }
        const m2p = Mdf.splitLangCode(m.LangCode)
        const ui2p = Mdf.splitLangCode(ulc)
        if (m2p.isEmpty || ui2p.isEmpty) {
          continue
        }
        if (m2p.lower === ui2p.lower) {
          mm.push(m)
          continue
        }
        if (m2p.first === ui2p.first) {
          mm.push(m)
          continue
        }
      }
      return mm
    },

    // refresh views and backend data
    onRefresh () {
      this.clearRedirect()
      this.doRefresh()
    },
    doRefresh () {
      this.doConfigRefresh()
      this.restartDiskUseRefresh()
      this.refreshTickle = !this.refreshTickle
    },
    restartDiskUseRefresh () {
      this.nDiskUseErr = 0
      clearInterval(this.diskUseRefreshInt)
      // refersh disk space usage now and setup refresh by timer
      if (this.isDiskUse) {
        this.doRefreshDiskUse()
        this.diskUseRefreshInt = setInterval(this.doGetDiskUse, this.diskUseMs)
      }
    },
    // refresh disk use by request
    onDiskUseRefresh () {
      if (this.isDiskUse) {
        this.doRefreshDiskUse()
      }
    },

    // view downloads page section
    onDownloadSelect (digest) {
      if (!digest) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to view downloads') })
        return
      }
      // show downloads for model selected from model list
      this.toUpDownSection = 'down'
      this.$router.push('/updown-list/model/' + encodeURIComponent(digest))
    },
    // view uploads page section
    onUploadSelect (digest) {
      if (!digest) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to view uploads') })
        return
      }
      // show uploads for model selected from model list
      this.toUpDownSection = 'up'
      this.$router.push('/updown-list/model/' + encodeURIComponent(digest))
    },

    // receive server configuration, including configuration of model catalog and run catalog
    async doConfigRefresh () {
      this.loadConfigWait = true

      const u = this.omsUrl + '/api/service/config'
      try {
        // send request to the server
        const response = await this.$axios.get(u)
        this.dispatchServerConfig(response.data) // update server config in store
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or configuration retrieve failed.', em)
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or configuration retrieve failed.') })
      }
      this.loadConfigWait = false

      this.loginUrl = Mdf.configEnvValue(this.serverConfig, 'OM_CFG_LOGIN_URL')
      this.logoutUrl = Mdf.configEnvValue(this.serverConfig, 'OM_CFG_LOGOUT_URL')

      // update disk space usage if necessary
      this.isDiskUse = !!this?.serverConfig?.IsDiskUse
      this.diskUseMs = this.getDiskUseRefreshMs(this?.serverConfig?.DiskScanMs)

      // update UI poperties
      const isTsort = (typeof this.serverConfig.UiExtra?.ModelTreeDescending === typeof true)
      const isTdesc = isTsort && !!this.serverConfig.UiExtra?.ModelTreeDescending
      this.dispatchSortModelTree({ isSort: isTsort, isDesc: isTdesc })

      this.doOrgTitle()
    },
    // get interval of disk use configuration refresh
    getDiskUseRefreshMs (ms) {
      return (!!ms && typeof ms === typeof 1 && ms >= DISK_USE_MIN_REFRESH_TIME) ? ms : DISK_USE_MIN_REFRESH_TIME
    },
    // return file size as translated string for example: 12 MB or 34 GB
    fileSizeStr (size) {
      const fs = Mdf.fileSizeParts(size)
      return fs.val + ' ' + this.$t(fs.name)
    },

    // receive disk space usage from server
    async doGetDiskUse () {
      if (this.nDiskUseErr > DISK_USE_MAX_ERR) return // get disk use failed

      this.loadDiskUseDone = false
      let isOk = false

      const u = this.omsUrl + '/api/service/disk-use'
      try {
        // send request to the server
        const response = await this.$axios.get(u)
        this.dispatchDiskUse(response.data) // update disk space usage in store
        isOk = Mdf.isDiskUseState(response.data) // validate disk usage info
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or disk usage retrieve failed.', em)
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or disk space usage retrieve failed.') })
      }
      this.loadDiskUseDone = true

      // update disk space usage to notify user
      if (isOk) {
        this.isDiskUse = this.diskUseState.IsDiskUse
        this.diskUseMs = this.getDiskUseRefreshMs(this.diskUseState.DiskUse.DiskScanMs)
        this.nDiskUseErr = 0
      } else {
        this.nDiskUseErr++
      }

      if (this.nDiskUseErr > DISK_USE_MAX_ERR) {
        clearInterval(this.diskUseRefreshInt)

        const du = Mdf.emptyDiskUseState()
        du.DiskUse.IsOver = true
        this.dispatchDiskUse(du) // block model runs

        console.warn('Disk usage retrieve failed:', this.nDiskUseErr)
        this.$q.notify({ type: 'negative', message: this.$t('Disk space usage retrieve failed') })
      }
    },

    // send request to refersh disk space usage
    async doRefreshDiskUse () {
      if (this.nDiskUseErr > DISK_USE_MAX_ERR) return // get disk use failed

      const u = this.omsUrl + '/api/service/disk-use/refresh'
      try {
        await this.$axios.post(u) // ignore response on success
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or disk usage refresh failed.', em)
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or disk space usage refresh failed.') })
      }

      // get disk usage from the server
      setTimeout(() => this.doGetDiskUse(), 1231)
    },

    //
    // redirect by url to open parameter, table or entity page:
    //   http://host/?model=DigestOrName&set=Default&parameter=ageSex
    //   http://host/?model=DigestOrName&run=NameOrDigestOrStamp&parameter=ageSex
    //   http://host/?model=DigestOrName&run=NameOrDigestOrStamp&table=salarySex
    //   http://host/?model=DigestOrName&run=NameOrDigestOrStamp&entity=Person
    async doRedirectByUrl () {
      this.clearRedirect()

      if (!this.modelCount || this.$route.path !== '/') {
        return // exit: no models or it is not a redirect url path
      }
      const mdl = this.$route.query?.model || '' // model digest or name
      if (!mdl) {
        return // model digest is empty: it is not a redirect url
      }
      const wsName = this.$route.query?.set || '' // workset name
      const rdn = this.$route.query?.run || '' // run digest or name
      const pName = this.$route.query?.parameter || '' // paramter name
      const tName = this.$route.query?.table || '' // output table name
      const eName = this.$route.query?.entity || '' // microdata entity name

      if (!!wsName && !!pName && !rdn && !tName && !eName) { // redirect to workset parameter
        this.redirectTo = WS_PARAM_REDIRECT
        this.toWsName = wsName
        this.toParamName = pName
      }
      if (!!rdn && !!pName && !wsName && !tName && !eName) { // redirect to run parameter
        this.redirectTo = RUN_PARAM_REDIRECT
        this.toRunDns = rdn
        this.toParamName = pName
      }
      if (!!rdn && !!tName && !wsName && !pName && !eName) { // redirect to output table
        this.redirectTo = TABLE_REDIRECT
        this.toRunDns = rdn
        this.toTableName = tName
      }
      if (!!rdn && !!eName && !wsName && !pName && !tName) { // redirect to microdata entity
        this.redirectTo = ENTITY_REDIRECT
        this.toRunDns = rdn
        this.toEntityName = eName
      }
      if (this.redirectTo === NO_REDIRECT) return // it is not a redirect url

      // load the model, if it is exists in models list
      let mdgst = ''
      for (const m of this.modelList) {
        if (m.Model.Digest === mdl || m.Model.Name === mdl) {
          mdgst = m.Model.Digest
          break
        }
      }
      if (!mdgst) {
        console.warn('Model not found', mdl)
        this.$q.notify({ type: 'negative', message: this.$t('Model not found') + ' : ' + mdl })
        return
      }
      this.toModelDigest = mdgst
      this.isRedirect = true
    },
    // clear redirect state
    clearRedirect () {
      this.isRedirect = false
      this.redirectTo = NO_REDIRECT
    },

    // model loaded
    doneModelLoad (isSuccess, dgst) {
      this.loadModelWait = false
      if (!isSuccess) {
        console.warn('Model load failed:', this.toModelDigest)
        return
      }
      this.$q.notify({ type: 'info', message: this.$t('Model') +' : '+ this.modelName })
    },
    // list of model runs loaded
    doneRunListLoad (isSuccess) {
      this.loadRunListWait = false
      if (!isSuccess || !Mdf.isLength(this.runTextList)) { // do not refresh run: run list empty
        this.runDnsCurrent = ''
        return
      }
      // run by digest, name or run stamp
      this.runDnsCurrent = this.toRunDns
      this.refreshRunTickle = !this.refreshRunTickle
    },
    // current model run loaded
    doneRunLoad (isSuccess, dgst) {
      this.loadRunWait = false
      if (!isSuccess && (dgst || '') === '') { // model run not found
        this.runDnsCurrent = ''
        return
      }
      this.$q.notify({ type: 'info', message: this.$t('Model run') +' : '+ this.toRunDns })

      // redirect to run parameter page, output table or microdata entity
      this.dispatchRunDigestSelected({ digest: this.modelDigest, runDigest: dgst })

      // parameter/:parameterName/model/:digest/run/:runDigest
      if (this.redirectTo === RUN_PARAM_REDIRECT) {
        this.$q.notify({ type: 'info', message: this.$t('Open') +' : '+ this.toParamName })
        this.$router.push(
          'parameter/' + encodeURIComponent(this.toParamName || '-') +
          '/model/' + encodeURIComponent(this.modelDigest || '-') +
          '/run/' + encodeURIComponent(dgst || '-')
        )
        this.clearRedirect()
      }
      // table/:toTableName/model/:digest/run/:runDigest
      if (this.redirectTo === TABLE_REDIRECT) {
        this.$q.notify({ type: 'info', message: this.$t('Open') +' : '+ this.toTableName })
        this.$router.push(
          'table/' + encodeURIComponent(this.toTableName || '-') +
          '/model/' + encodeURIComponent(this.modelDigest || '-') +
          '/run/' + encodeURIComponent(dgst || '-')
        )
        this.clearRedirect()
      }
      // entity/:toEntityName/model/:digest/run/:runDigest
      if (this.redirectTo === ENTITY_REDIRECT) {
        this.$q.notify({ type: 'info', message: this.$t('Open') +' : '+ this.toEntityName })
        this.$router.push(
          'entity/' + encodeURIComponent(this.toEntityName || '-') +
          '/model/' + encodeURIComponent(this.modelDigest || '-') +
          '/run/' + encodeURIComponent(dgst || '-')
        )
        this.clearRedirect()
      }
    },
    // list of model worksets loaded
    doneWsListLoad (isSuccess) {
      this.loadWsListWait = false
      if (!isSuccess || !Mdf.isLength(this.worksetTextList)) { // do not refresh workset: workset list empty
        this.wsNameCurrent = ''
        return
      }
      // refresh workset by name
      this.wsNameCurrent = this.toWsName
      this.refreshWsTickle = !this.refreshWsTickle
    },
    // current workset loaded
    doneWsLoad (isSuccess, name, isNewRun) {
      this.loadWsWait = false
      if (!isSuccess && (name || '') === '') { // workset not found
        this.wsNameCurrent = ''
        return
      }
      this.$q.notify({ type: 'info', message: this.$t('Input Scenario') +' : '+ this.wsNameCurrent })

      // redirect to workset parameter page
      this.dispatchWorksetNameSelected({ digest: this.modelDigest, worksetName: name })

      // parameter/:parameterName/model/:digest/set/:worksetName
      if (this.redirectTo === WS_PARAM_REDIRECT) {
        this.$q.notify({ type: 'info', message: this.$t('Open') +' : '+ this.toParamName })
        this.$router.push(
          'parameter/' + encodeURIComponent(this.toParamName || '-') +
          '/model/' + encodeURIComponent(this.modelDigest || '-') +
          '/set/' + encodeURIComponent(this.toWsName || '-')
        )
        this.clearRedirect()
      }
    },

    // user views for current model are loaded
    doneUserViewsLoad (isSuccess, nViews) {
      this.loadUserViewsWait = false
      if (nViews > 0) {
        this.$q.notify({ type: 'info', message: this.$t('User views count: ') + nViews.toString() })
      }
    },
    // start uploading views into user home directory
    onUserViewUpdated (dgst, mName) {
      this.uploadUserViewsTickle = !this.uploadUserViewsTickle
    },
    // user views uploaded to the server
    doneUserViewsUpload (isSuccess, nViews) {
      this.uploadUserViewsWait = false
      if (isSuccess && nViews > 0) {
        this.$q.notify({ type: 'info', message: this.$t('User views uploaded: ') + nViews.toString() })
      }
    },

    // update workset readonly status
    onWorksetReadonlyUpdate (dgst, name, isReadonly) {
      if (dgst !== this.modelDigest || !name) return

      if (isReadonly) {
        // if there are any edited and unsaved parameters for this workset
        const n = this.paramViewWorksetUpdatedCount({ digest: this.modelDigest, worksetName: name })
        if (n > 0) {
          console.warn('Unable to save input scenario: unsaved parameters count:', name, n)
          this.$q.notify({
            type: 'negative',
            message: this.$t('Unable to save input scenario {setName} because you have {count} unsaved parameter(s)', { setName: name, count: n })
          })
          return
        }
      }
      // else: update workset status
      this.isReadonlyWsStatus = isReadonly
      this.nameWsStatus = name
      this.updateWsStatusTickle = !this.updateWsStatusTickle
    }
  },

  mounted () {
    this.doConfigRefresh()
  },

  beforeUnmount () {
    clearInterval(this.diskUseRefreshInt)
  },

  created () {
    // if locale for current language not avaliable then
    // find fallback locale (assuming fallback is available)
    let ln = this.langCode

    // match first part of lanuage code to avaliable locales
    if (this.$i18n.availableLocales.indexOf(ln) < 0) {
      const ui2p = Mdf.splitLangCode(ln)

      if (ui2p.first !== '') {
        for (const lcIdx in this.$i18n.availableLocales) {
          const lc = this.$i18n.availableLocales[lcIdx]

          const av2p = Mdf.splitLangCode(lc)
          if (av2p.first === ui2p.first) {
            ln = lc
            break
          }
        }
      }
    }

    if (ln && this.langCode !== ln) {
      this.langCode = ln // switch app language
    }
  }
}
