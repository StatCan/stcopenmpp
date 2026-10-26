# UI caching

The UI remembers what the user did and caches server data in the browser (IndexedDB database `openmpp_ui_cache`, settings in `localStorage`).
Both can be switched off or cleared in **Settings → Browser cache**.

## UI state (survives page reload)

| What | Where it lives |
|---|---|
| UI language, selected model, selected run / input scenario per model, runs to compare | `ui-state` store, persisted by `src/cache/persist-state.js` |
| Open tabs and the last active tab of each model (opening a model returns to that tab) | `modelView[digest].tabs` / `.activeTab` |
| Parameter, output table and microdata views (pivot layout, filters, page, chart), including unsaved parameter edits | `paramViews`, `tableViews`, `mdViews` |
| Per-tab inputs: run filter and selected runs, scenario filter, open sections, **New Run form** (run options, description/notes, tables, microdata), downloads page filters | `tabStates`, via `src/cache/tab-state.js` mixin |
| Search filter of every parameter / table / microdata tree | `tabStates`, in `OmTableTree.vue` |
| Scroll position of each page / tab | `scrollPos`, via `src/cache/scroll.js` (router `scrollBehavior`) |
| Model list, current model metadata, run list, scenario list, model words and languages | `model` store, persisted; lists are shown at once and refreshed from the server in background |

The New Run form drops the run name, description and notes after a run is submitted and keeps the other options.

## Server responses (`src/cache/http-cache.js`, axios adapter)

| Request | Strategy |
|---|---|
| Model text, word list and language list | Cache first (the model digest identifies the content), kept for 30 days |
| Parameter, table and microdata pages of **completed** runs (success, error, exit), including run comparison | Cache first, only when every run involved is completed |
| Model list, run list, scenario list, run and scenario text, server config, user views, scenario parameter pages | Always from the server; the cached copy is used only when the server cannot be reached, with a warning |

Deleting a run removes its cached data, and deleting a model removes all cached data for that model. The total size has a limit (256 MB by default, configurable), and the least recently used responses are evicted first.

## Adding cached state to another tab page

```js
import { tabStateMixin } from 'src/cache/tab-state'

export default {
  mixins: [tabStateMixin],
  tabState: {
    fields: ['myFilter', 'isSectionOpen'],     // data properties to remember
    key () { return 'my-tab:' + this.digest }  // cache key, derived from props
  },
  ...
}
```
