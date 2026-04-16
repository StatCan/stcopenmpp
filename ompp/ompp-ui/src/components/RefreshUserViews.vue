<!-- reload user views by model name if user home directory allowed on the server -->
<script>
import { mapState } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import * as Mdf from 'src/model-common'
import * as Idb from 'src/idb/idb'

export default {
  name: 'RefreshUserViews',

  props: {
    modelName: { type: String, default: '' }
  },

  render () { return null }, // no html

  data () {
    return {
      loadDone: false,
      loadWait: false
    }
  },

  computed: {
    ...mapState(useModelStore, [
      'theModel'
    ]),
    ...mapState(useServerStateStore, {
      omsUrl: 'omsUrl',
      serverConfig: 'config'
    })
  },

  emits: ['done', 'wait'],

  methods: {
    // refersh user views from the server and update inxeded db
    async doRefresh () {
      // exit on empty model name
      // exit if user views not supported by the server
      if (!this.modelName || !this.serverConfig.AllowUserHome) {
        this.$emit('done', this.loadDone, 0) // always emit result
        return
      }

      // validate model name: model must be loaded
      const mName = Mdf.modelName(this.theModel)
      if (this.modelName !== mName) {
        console.warn('Unable to restore user views due to mismatch of model name', this.modelName, mName)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to restore user views due to mismatch of model name: ') + mName })
        this.$emit('done', this.loadDone, 0) // always emit result
        return
      }

      // get user views from the server
      this.loadDone = false
      this.loadWait = true
      this.$emit('wait')

      let vs = {}
      const u = this.omsUrl + '/api/user/view/model/' + encodeURIComponent(this.modelName)
      try {
        const response = await this.$axios.get(u)
        vs = response.data
        this.loadDone = true // assume successful write into indexed db
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or read of user views failed', em)
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or read of user views failed: ') + mName })
      }

      // if model parameter views not empty then write into indexed db
      let count = 0

      if (vs && (vs?.model?.name || '') !== '') {
        // views are not empty: check model name
        if (!vs || vs?.model?.name !== this.modelName) {
          console.warn('Unable to restore user views due to mismatch of model name', this.modelName, vs?.model?.name)
          this.$q.notify({ type: 'negative', message: this.$t('Unable to restore user views due to mismatch of model name: ') + this.modelName })
          this.loadDone = false // error: model name in server json response must be same as current UI model name
        } else {
          //
          // write parameters view into indexed db
          if (Array.isArray(vs.model?.parameterViews) && vs.model?.parameterViews?.length) {
            let pName = ''
            try {
              const dbCon = await Idb.connection()
              const rw = await dbCon.openReadWrite(this.modelName)
              for (const v of vs.model.parameterViews) {
                if (!v?.name || !v?.view) continue

                pName = v.name
                if (this.theModel.ParamTxt.findIndex(p => p.Param.Name === pName) < 0) continue // parameter not exist that model version

                await rw.put(v.name, v.view)
                count++
              }
            } catch (e) {
              console.warn('Unable to save default parameter view:', pName, e)
              this.$q.notify({ type: 'negative', message: this.$t('Unable to save default parameter view: ') + pName })
              this.loadDone = false // error during view write, it can be incorrect json
            }
          }

          // write output tables view into indexed db
          if (Array.isArray(vs.model?.tableViews) && vs.model?.tableViews?.length) {
            let tName = ''
            try {
              const dbCon = await Idb.connection()
              const rw = await dbCon.openReadWrite(this.modelName)
              for (const v of vs.model.tableViews) {
                if (!v?.name || !v?.view) continue

                tName = v.name
                if (this.theModel.TableTxt.findIndex(tt => tt.Table.Name === tName) < 0) continue // table not exist in that model version

                await rw.put(v.name, v.view)
                count++
              }
            } catch (e) {
              console.warn('Unable to save default output table view:', tName, e)
              this.$q.notify({ type: 'negative', message: this.$t('Unable to save default output table view: ') + tName })
              this.loadDone = false // error during view write, it can be incorrect json
            }
          }

          // write entity microdata view into indexed db
          if (this.serverConfig.AllowMicrodata && Array.isArray(vs.model?.microdataViews) && vs.model?.microdataViews?.length) {
            let eName = ''
            try {
              const dbCon = await Idb.connection()
              const rw = await dbCon.openReadWrite(this.modelName)
              for (const v of vs.model.microdataViews) {
                if (!v?.name || !v?.view) continue

                eName = v.name
                if (this.theModel.EntityTxt.findIndex(e => e.Entity.Name === eName) < 0) continue // entity not exist that model version

                await rw.put(v.name, v.view)
                count++
              }
            } catch (e) {
              console.warn('Unable to save default microdata view: ', eName, e)
              this.$q.notify({ type: 'negative', message: this.$t('Unable to save default microdata view: ') + eName })
              this.loadDone = false // error during view write, it can be incorrect json
            }
          }
        }
      }

      this.$emit('done', this.loadDone, count)
      this.loadWait = false
    }
  },

  mounted () {
    this.doRefresh()
  }
}
</script>
