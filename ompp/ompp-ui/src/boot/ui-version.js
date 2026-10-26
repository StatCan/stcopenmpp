// Make sure the browser runs the latest deployed version of the UI.
//
// Browsers may reuse a cached copy of index.html when the UI is opened (not reloaded),
// and that old index.html loads the previous build of the UI.
// This check downloads index.html bypassing the browser cache, compares its script files
// with the scripts of the running page and if they are different then reloads the page once.

import { defineBoot } from '#q-app/wrappers'

const RELOAD_KEY = 'openmpp-ui-version-reload'

// return sorted list of module script paths from the document
const scriptsOf = (doc, baseUrl) => {
  const lst = []
  for (const s of doc.querySelectorAll('script[type="module"][src]')) {
    try {
      lst.push(new URL(s.getAttribute('src'), baseUrl).pathname)
    } catch (e) {}
  }
  return lst.sort().join('|')
}

const checkVersion = async () => {
  const pageUrl = window.location.origin + window.location.pathname // without #/route and ?query
  const current = scriptsOf(document, pageUrl)
  if (!current) return

  const rsp = await fetch(pageUrl, { cache: 'no-store' })
  if (!rsp.ok) return
  const html = await rsp.text()
  const latest = scriptsOf(new DOMParser().parseFromString(html, 'text/html'), pageUrl)

  if (!latest || latest === current) {
    try { sessionStorage.removeItem(RELOAD_KEY) } catch (e) {}
    return // running the latest version
  }

  // newer version deployed: reload once, do not loop if server keeps returning a different page
  let isReloaded = false
  try {
    isReloaded = sessionStorage.getItem(RELOAD_KEY) === latest
    sessionStorage.setItem(RELOAD_KEY, latest)
  } catch (e) {}
  if (!isReloaded) {
    console.warn('New version of UI deployed, reloading the page')
    window.location.reload()
  }
}

export default defineBoot(() => {
  if (!process.env.PROD) return // development server always serves the latest code

  // do not delay the application start: check in background
  checkVersion().catch((e) => console.warn('UI version check failed', e))
})
