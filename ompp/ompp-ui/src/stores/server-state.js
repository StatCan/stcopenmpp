// server state and configuration
import { defineStore, acceptHMRUpdate } from 'pinia'

import * as Mdf from '../model-common'

export const useServerStateStore = defineStore('server-state', {
  state: () => ({
    omsUrl: (process.env.OMS_URL || ''), // oms web-service url
    config: Mdf.emptyConfig(), // server state and configuration
    diskUse: Mdf.emptyDiskUseState() // disk space usage
  }),

  getters: {
  },

  actions: {
    dispatchServerConfig (cfg) {
      if (Mdf.isConfig(cfg)) {
        cfg.Env ??= {}
        let uiex = {}
        try {
          uiex = JSON.parse((cfg?.UiExtra || '{}'))
        } catch {
          uiex = {}
        }
        this.config = cfg
        this.config.UiExtra = uiex
      }
    },

    dispatchDiskUse (du) {
      if (Mdf.isDiskUseState(du)) this.diskUse = du
    }
  }
})


if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useServerStateStore, import.meta.hot))
}
