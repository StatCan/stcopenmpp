// Remember page scroll position for each route path and restore it when user returns to the page (tab).
//
// Data on tab pages loaded asynchronously, so scroll restore waits until page is high enough,
// but no longer than RESTORE_WAIT_MS and only if user did not scroll the page by himself.

import { useUiStateStore } from 'src/stores/ui-state'
import { getCacheSettings } from './cache-settings'

const RESTORE_WAIT_MS = 4000
const CHECK_INTERVAL_MS = 100

let restoreSeq = 0 // sequence number of current restore, increased on each navigation and user scroll

// cancel pending scroll restore if user scroll the page
const onUserScroll = () => { restoreSeq++ }

// save current scroll position of the page by route path
export const saveScroll = (path) => {
  if (!path || !getCacheSettings().isStateCache) return
  try {
    useUiStateStore().dispatchScrollPos(path, window.scrollY || 0)
  } catch (e) {}
}

// router scroll behavior: use browser saved position on back/forward, else saved position of the route path
export const scrollBehavior = (to, from, savedPosition) => {
  const seq = ++restoreSeq

  if (to.path === from.path && from.matched.length > 0) return false // only query or hash changed: keep scroll

  let y = 0
  if (savedPosition && typeof savedPosition.top === typeof 1) {
    y = savedPosition.top
  } else if (getCacheSettings().isStateCache) {
    try {
      y = useUiStateStore().scrollPosition(to.path)
    } catch (e) {
      y = 0
    }
  }
  if (!y || y <= 0) return { left: 0, top: 0 }

  // wait until page content is loaded and high enough to scroll
  return new Promise((resolve) => {
    const start = Date.now()

    const check = () => {
      if (seq !== restoreSeq) {
        resolve(false) // user scrolled or navigated to other page
        return
      }
      const maxY = (document.documentElement?.scrollHeight || 0) - (window.innerHeight || 0)
      if (maxY >= y) {
        resolve({ left: 0, top: y })
        return
      }
      if (Date.now() - start >= RESTORE_WAIT_MS) {
        resolve(maxY > 0 ? { left: 0, top: maxY } : false)
        return
      }
      setTimeout(check, CHECK_INTERVAL_MS)
    }
    check()
  })
}

// start listening user scroll actions to cancel pending scroll restore
export const startScrollWatch = () => {
  try {
    window.addEventListener('wheel', onUserScroll, { passive: true })
    window.addEventListener('touchmove', onUserScroll, { passive: true })
    window.addEventListener('keydown', (e) => {
      if (['PageUp', 'PageDown', 'Home', 'End', 'ArrowUp', 'ArrowDown', ' '].includes(e.key)) onUserScroll()
    }, { passive: true })
  } catch (e) {}
}
