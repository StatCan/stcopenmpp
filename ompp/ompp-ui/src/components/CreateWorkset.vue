<!-- create new workset: verify workset data and send request to the server -->
<script>
import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import * as Mdf from 'src/model-common'

export default {
  name: 'CreateWorkset',

  props: {
    modelDigest: { type: String, default: '' },
    newName: { type: String, default: '' },
    createNow: { type: Boolean, default: false },
    currentWorksetName: { type: String, default: '' },
    currentRunDigest: { type: String, default: '' },
    isBasedOnRun: { type: Boolean, default: false },
    descrNotes: { type: Array, default: () => [] },
    copyFromRun: { type: Array, default: () => [] },
    copyFromWorkset: { type: Array, default: () => [] }
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
      omsUrl: 'omsUrl'
    })
  },

  watch: {
    createNow () { if (this.createNow) this.doCreate() }
  },

  emits: ['done', 'wait'],

  methods: {
    ...mapActions(useModelStore, [
      'worksetTextByName',
      'isExistInWorksetTextList'
    ]),

    // validate and create new workset
    doCreate () {
      if (!this.modelDigest) {
        console.warn('Invalid (empty) model digest')
        this.$emit('done', false, this.modelDigest, this.newName)
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) model digest') })
        return
      }
      if (!this.newName || (this.newName || '') === '' || typeof this.newName !== typeof 'string') {
        console.warn('Invalid (empty) workset name')
        this.$emit('done', false, this.modelDigest, this.newName)
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) input scenario name') })
        return
      }
      if (Mdf.cleanFileNameInput(this.newName) !== this.newName || this.newName.length > 255) {
        console.warn('Invalid workset name:', this.newName)
        this.$emit('done', false, this.modelDigest, this.newName)
        this.$q.notify({ type: 'negative', message: this.$t('Invalid input scenario name: ') + (this.newName || '') })
        return
      }
      // check if the workset with the same name already exist in the model
      if (this.isExistInWorksetTextList({ ModelDigest: this.modelDigest, Name: this.newName })) {
        this.$q.notify({ type: 'negative', message: this.$t('Error: input scenario name must be unique: ') + (this.newName || '') })
        this.$emit('done', false, this.modelDigest, this.newName)
        return
      }

      // new workset header
      const ws = {
        ModelDigest: this.modelDigest,
        Name: this.newName,
        IsReadonly: false,
        BaseRunDigest: (this.isBasedOnRun && (this.currentRunDigest || '') !== '') ? this.currentRunDigest : '',
        Txt: (this.descrNotes || []),
        Param: []
      }

      // return true if parameter exist in current workset
      const wsCurrent = this.worksetTextByName({ ModelDigest: this.modelDigest, Name: this.currentWorksetName })

      const isInCurrentWs = (name) => {
        if (wsCurrent.BaseRunDigest !== '') return true // workset based on the run: all parameters included
        return wsCurrent.Param.findIndex((p) => p.Name === name) >= 0
      }

      // expand all parameter groups
      const pUse = {}
      const gUse = {}
      const gs = []

      // starting point: append all groups from copy lists
      for (const pn of this.copyFromRun) {
        if (pUse[pn.name]) continue

        if (pn.isGroup) {
          gs.push({
            name: pn.name,
            isFromRun: true
          })
        } else {
          ws.Param.push({
            Name: pn.name,
            Kind: 'run',
            From: this.currentRunDigest
          })
          pUse[pn.name] = true
        }
      }
      for (const pn of this.copyFromWorkset) {
        if (pUse[pn.name]) continue

        if (pn.isGroup) {
          gs.push({
            name: pn.name,
            isFromRun: false
          })
        } else {
          if (isInCurrentWs(pn.name)) {
            ws.Param.push({
              Name: pn.name,
              Kind: 'set',
              From: this.currentWorksetName
            })
            pUse[pn.name] = true
          }
        }
      }

      // expand each group
      while (gs.length > 0) {
        const g = gs.pop()

        if (gUse[g.name]) continue // group already processed
        gUse[g.name] = true

        const gt = Mdf.groupTextByName(this.theModel, g.name)
        if (!gt.Group.IsParam) continue // not a parameter group

        for (const pc of gt.Group.GroupPc) {
          // if this is a child group
          if (pc.ChildGroupId >= 0) {
            const cg = Mdf.groupTextById(this.theModel, pc.ChildGroupId)
            if (cg.Group.Name && !gUse[cg.Group.Name]) {
              gs.push({
                name: cg.Group.Name,
                isFromRun: g.isFromRun
              })
            }
          }

          // if this is a child leaf parameter
          if (pc.ChildLeafId >= 0) {
            const cp = Mdf.paramTextById(this.theModel, pc.ChildLeafId)
            if (cp.Param.Name && !pUse[cp.Param.Name] && (g.isFromRun || isInCurrentWs(cp.Param.Name))) {
              ws.Param.push({
                Name: cp.Param.Name,
                Kind: g.isFromRun ? 'run' : 'set',
                From: g.isFromRun ? this.currentRunDigest : this.currentWorksetName
              })
              pUse[cp.Param.Name] = true
            }
          }
        }
      }

      // send request to create new workset
      this.doCreateNewWorkset(ws)
    },

    // create new workset
    async doCreateNewWorkset (ws) {
      if (!ws.ModelDigest || !ws.Name) {
        console.warn('Unable to create workset: model digest or workset name is empty')
        this.$emit('done', false, ws.ModelDigest, ws.Name)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to create new input scenario: model digest or scenario name is empty') })
        return
      }

      this.loadDone = false
      this.loadWait = true
      this.$emit('wait')
      this.$q.notify({ type: 'info', message: this.$t('Creating: ') + ws.Name })

      let nm = ''
      let em = ''
      const u = this.omsUrl + '/api/workset-create'
      try {
        const response = await this.$axios.put(u, ws)
        nm = response.data?.Name
        this.loadDone = true
      } catch (e) {
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Error at create workset', ws.Name, em)
      }
      this.loadWait = false

      if (this.loadDone) {
        this.$q.notify({ type: 'info', message: this.$t('Created: ') + nm })
      } else {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to create new input scenario: ') + ws.Name })
        if (em && typeof em === typeof 'string') {
          this.$q.notify({ type: 'negative', message: em })
        }
      }

      this.$emit('done', this.loadDone, ws.ModelDigest, this.loadDone ? nm : ws.Name)
    }
  },

  mounted () {
  }
}
</script>
