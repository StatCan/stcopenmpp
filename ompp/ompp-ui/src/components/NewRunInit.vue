<!-- initiate (start) new model run: send request to the server -->
<script>
import { mapState } from 'pinia'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'
import * as Mdf from 'src/model-common'

export default {
  name: 'NewRunInit',

  props: {
    modelDigest: { type: String, default: '' },
    runOpts: {
      type: Object,
      default: () => ({
        runName: '',
        worksetName: '',
        baseRunDigest: '',
        runDescr: {}, // run description[language code]
        subCount: 1,
        threadCount: 1,
        progressPercent: 1,
        progressStep: 0,
        workDir: '',
        useCsvDir: false,
        csvDir: '',
        csvId: false,
        iniName: '',
        useIni: false,
        iniAnyKey: false,
        profile: '',
        sparseOutput: false,
        runTmpl: '',
        mpiNpCount: 0,
        mpiUseJobs: false,
        mpiOnRoot: false,
        mpiTmpl: ''
      })
    },
    tablesRetain: { type: Array, default: () => [] },
    microdataOpts: {
      type: Object,
      default: () => { return Mdf.emptyRunRequestMicrodata() }
    },
    runNotes: {
      type: Object,
      default: () => ({})
    }
  },

  render () { return null }, // no html

  data () {
    return {
      loadDone: false,
      loadWait: false
    }
  },

  computed: {
    ...mapState(useServerStateStore, {
      omsUrl: 'omsUrl',
      serverConfig: 'config'
    }),
    ...mapState(useUiStateStore, ['uiLang'])
  },

  emits: ['done', 'wait'],

  methods: {
    // initiate new model run: send request to the server
    async doNewRunInit () {
      if (!this.modelDigest) {
        this.$emit('done', false, '')
        console.warn('Unable to run the model: model digest is empty')
        this.$q.notify({ type: 'negative', message: this.$t('Unable to run the model: model digest is empty') })
        return
      }

      this.loadDone = false
      this.loadWait = true
      this.$emit('wait')

      // set run options
      const rv = {
        ModelDigest: (this.modelDigest || ''),
        Dir: '',
        Opts: { },
        Threads: 1,
        IsMpi: false,
        Mpi: {
          Np: 0,
          IsNotOnRoot: false,
          IsNotByJob: !this.serverConfig.IsJobControl
        },
        Template: '',
        Tables: [],
        Microdata: Mdf.emptyRunRequestMicrodata(),
        RunNotes: []
      }

      if ((this.runOpts.runName || '') !== '') rv.Opts['OpenM.RunName'] = this.runOpts.runName
      if ((this.runOpts.worksetName || '') !== '') rv.Opts['OpenM.SetName'] = this.runOpts.worksetName
      if ((this.runOpts.baseRunDigest || '') !== '') rv.Opts['OpenM.BaseRunDigest'] = this.runOpts.baseRunDigest
      if ((this.runOpts.subCount || 1) !== 1) rv.Opts['OpenM.SubValues'] = this.runOpts.subCount.toString()
      if ((this.runOpts.workDir || '') !== '') rv.Dir = this.runOpts.workDir
      if ((this.runOpts.progressPercent || 1) !== 1) rv.Opts['OpenM.ProgressPercent'] = this.runOpts.progressPercent.toString()
      if (this.runOpts.progressStep) rv.Opts['OpenM.ProgressStep'] = this.runOpts.progressStep.toString()
      if (this.runOpts.useCsvDir && (this.runOpts.csvDir || '') !== '') rv.Opts['OpenM.ParamDir'] = this.runOpts.csvDir
      if (this.runOpts.useCsvDir && this.runOpts.csvId) rv.Opts['OpenM.IdCsv'] = 'true'
      if (this.runOpts.useIni && ((this.runOpts.iniName || '') !== '')) rv.Opts['OpenM.IniFile'] = this.runOpts.iniName
      if (this.runOpts.useIni && this.runOpts.iniAnyKey) rv.Opts['OpenM.IniAnyKey'] = 'true'
      if ((this.runOpts.profile || '') !== '') rv.Opts['OpenM.Profile'] = this.runOpts.profile
      if (this.runOpts.sparseOutput) rv.Opts['OpenM.SparseOutput'] = 'true'
      if (this.uiLang) rv.Opts['OpenM.MessageLanguage'] = this.uiLang

      if ((this.runOpts.threadCount || 1) >= 1) {
        rv.Threads = this.runOpts.threadCount
        rv.Opts['OpenM.Threads'] = this.runOpts.threadCount.toString() // for backward compatibility only
      }

      rv.Tables = Array.from(this.tablesRetain)

      if (this.serverConfig.AllowMicrodata) rv.Microdata = this.microdataOpts

      if ((this.runOpts.mpiNpCount || 0) <= 0) {
        if (this.runOpts.runTmpl) rv.Template = this.runOpts.runTmpl
      } else {
        rv.IsMpi = true
        rv.Mpi.Np = this.runOpts.mpiNpCount
        if (!this.runOpts.mpiOnRoot) {
          rv.Mpi.IsNotOnRoot = true
          rv.Opts['OpenM.NotOnRoot'] = 'true' // for backward compatibility only
        }
        if (this.runOpts.mpiTmpl) {
          rv.Template = this.runOpts.mpiTmpl
        }
        rv.Mpi.IsNotByJob = !this.serverConfig.IsJobControl || !this.runOpts.mpiUseJobs
        rv.Opts['OpenM.LogRank'] = 'true'
      }

      for (const lcd in this.runOpts.runDescr) {
        if ((lcd || '') !== '' && (this.runOpts.runDescr[lcd] || '') !== '') rv.Opts[lcd + '.RunDescription'] = this.runOpts.runDescr[lcd]
      }

      for (const lcd in this.runNotes) {
        if ((lcd || '') !== '' && (this.runNotes[lcd] || '') !== '') {
          rv.RunNotes.push({
            LangCode: lcd,
            Note: this.runNotes[lcd]
          })
        }
      }

      let rst = Mdf.emptyRunState()
      const u = this.omsUrl + '/api/run'

      try {
        // send model run request to the server, response expected to contain run stamp
        const response = await this.$axios.post(u, rv)
        rst = response.data
        this.loadDone = true
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or model run failed to start', em)
        // this.$q.notify({ type: 'negative', message: this.$t('Server offline or model run failed to start: ') + this.modelDigest })
      }
      this.loadWait = false

      // return run status, empty on error
      if (!Mdf.isRunState(rst)) rst = Mdf.emptyRunState()

      this.$emit('done', this.loadDone, rst.RunStamp, rst.SubmitStamp)
    }
  },

  mounted () {
    this.doNewRunInit()
  }
}
</script>
