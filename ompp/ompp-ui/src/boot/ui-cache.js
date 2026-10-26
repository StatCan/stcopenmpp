// UI cache: restore persisted UI state before app is mounted, cache server responses, save scroll positions
import { defineBoot } from '#q-app/wrappers'
import { api } from './axios'
import { installHttpCache } from 'src/cache/http-cache'
import { restoreState, startStateSave } from 'src/cache/persist-state'
import { saveScroll, startScrollWatch } from 'src/cache/scroll'

const RESTORE_TIMEOUT_MS = 3000 // do not wait longer than that for indexedDB restore

export default defineBoot(async ({ router, store }) => {
  installHttpCache(api)

  // restore UI state, if indexedDB is slow or blocked then start without restored state
  await Promise.race([
    restoreState(store),
    new Promise((resolve) => setTimeout(resolve, RESTORE_TIMEOUT_MS))
  ])

  // save scroll position of the page before navigation to other page
  router.beforeEach((to, from) => {
    if (from.matched.length > 0 && to.path !== from.path) saveScroll(from.path)
  })
  startScrollWatch()

  // on page close or reload save current page scroll position, it must be done before state is saved
  const onHide = () => saveScroll(router.currentRoute.value?.path)
  window.addEventListener('pagehide', onHide)
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'hidden') onHide()
  })

  startStateSave(store)
})
