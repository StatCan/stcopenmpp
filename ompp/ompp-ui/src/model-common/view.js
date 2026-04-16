// user views common functions: model view, model tabs, parameters view

// return empty model view
/* eslint-disable no-multi-spaces */
export const emptyModelView = () => {
  return {
    runDigest: '',         // digest of selected run
    digestCompareList: [], // array of run digests to compare with selected run
    worksetName: '',       // name of selected workset
    taskName: '',          // name of selected task
    tabs: []               // route parts of open tabs
  }
}
/* eslint-enable no-multi-spaces */
