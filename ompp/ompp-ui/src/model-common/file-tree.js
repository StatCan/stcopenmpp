// files tree common methods

import * as Hlpr from './helper'

/* eslint-disable no-multi-spaces */
/*
// PathItem contain basic file info after tree walk: relative path, size and modification time
type PathItem struct {
  Path    string // file path in / slash form
  IsDir   bool   // if true then it is a directory
  Size    int64  // file size (may be zero for directories)
  ModTime int64  // file modification time in milliseconds since epoch
}
*/
// return empty  emptyFilePathItem
export const emptyFilePathItem = () => {
  return {
    Path: '',     // file path in / slash form
    IsDir: false, // if true then it is a directory
    Size: 0,      // file size (may be zero for directories)
    ModTime: 0    // file modification time in milliseconds since epoch
  }
}
/* eslint-enable no-multi-spaces */

// return true if this is download-or-upload file item
export const isFilePathItem = (fi) => {
  if (!fi) return false
  if (!fi.hasOwnProperty('Path') || typeof fi.Path !== typeof 'string' ||
    !fi.hasOwnProperty('IsDir') || typeof fi.IsDir !== typeof true ||
    !fi.hasOwnProperty('Size') || typeof fi.Size !== typeof 1 ||
    !fi.hasOwnProperty('ModTime') || typeof fi.ModTime !== typeof 1) {
    return false
  }
  return true
}

// return true if this is not empty download-or-upload file item
export const isNotEmptyFilePathItem = (fi) => {
  if (!isFilePathItem(fi)) return false
  return (fi.Path || '') !== ''
}

// return true if each array element isFilePathItem()
export const isFilePathTree = (pLst) => {
  if (!pLst) return false
  if (!Array.isArray(pLst)) return false
  for (let k = 0; k < pLst.length; k++) {
    if (!isFilePathItem(pLst[k])) return false
  }
  return true
}

// return tree of files: download folder, upload folder or user files tree.
// keyPrefix by default 'fi' is prepended to neach node key
/*
 each node: {
      key:      node key string
      Path:     file path
      link:     url encoded file path,
      label:    last path element (file name or folder name)
      descr:    modification time string
      children: []
      isGroup:  true if it is directory
      Size:     size in bytes
    }
*/
export const makeFileTree = (fLst, keyPrefix) => {
  if (!fLst || !Array.isArray(fLst) || fLst.length <= 0) { // empty file list
    return { isRootDir: false, isAnyDir: false, tree: [] }
  }
  const kpr = (!keyPrefix || typeof keyPrefix !== typeof 'string') ? 'fi' : keyPrefix

  // make files (and folders) map: map file path to folder name and item name (file name or sub-folder name)
  const fPath = {}
  let isAny = false
  let isRoot = false

  for (let k = 0; k < fLst.length; k++) {
    if (!fLst[k].Path || fLst[k].Path === '.' || fLst[k].Path === '..') continue

    isAny = isAny || fLst[k].IsDir

    // if root folder then push path as is
    if (fLst[k].Path === '/') {
      isRoot = true
      fPath[fLst[k].Path] = { base: '', name: '/', label: '/' }
      continue
    }

    // remove trailing / from path
    if (fLst[k].Path.endsWith('/')) fLst[k].Path = fLst[k].Path.substr(0, fLst[k].Path.length - 1)

    // split path to the base folder and name, use name without leading / as label
    const n = fLst[k].Path.lastIndexOf('/')

    fPath[fLst[k].Path] = {
      base: n >= 0 ? fLst[k].Path.substr(0, n) : '',
      name: n >= 0 ? fLst[k].Path.substr(n) : fLst[k].Path,
      label: n >= 0 ? fLst[k].Path.substr(n + 1) : fLst[k].Path
    }
  }

  // if root / folder exists then make all top folders and files children of root folder
  if (isRoot) {
    for (const pk in fPath) {
      if (pk === '/' || pk === '.' || pk === '..') continue // skip root and skip invlaid paths
      if (fPath[pk].base === '') {
        fPath[pk].base = '/' // top level folder or file is a child or root / folder
      }
    }
  }

  // make href link: for each part of the path do encodeURIComponent and keep / as is
  const pathEncode = (path) => {
    if (!path || typeof path !== typeof 'string') return ''

    const ps = path.split('/')
    for (let k = 0; k < ps.length; k++) {
      ps[k] = encodeURIComponent(ps[k])
    }
    return ps.join('/')
  }

  // add top level folders and files as starting point into the tree
  const fTree = []
  const fProc = []
  const fDone = {}
  const fTopFiles = []

  for (const fi of fLst) {
    if (!fi.Path || fi.Path === '.' || fi.Path === '..') continue

    if (fPath[fi.Path].base !== '') continue // not a top level folder or file

    // make tree node
    const fn = {
      key: kpr + '-' + fi.Path + '-' + (fi.ModTime || 0).toString(),
      Path: fi.Path,
      link: pathEncode(fi.Path),
      label: fPath[fi.Path].label,
      descr: Hlpr.modTsToTimeStamp(fi.ModTime),
      children: [],
      isGroup: fi.IsDir,
      Size: fi.Size
    }
    fDone[fi.Path] = fn

    // if this is top level folder then add it to list of root folders
    if (fi.IsDir) {
      fTree.push(fn)
      fProc.push(fn)
    } else { // this is top level file
      fTopFiles.push(fn)
    }
  }

  // build folders and files tree
  while (fProc.length > 0) {
    const fNow = fProc.pop()

    // make all children of current item
    for (const fi of fLst) {
      if (!fi.Path || fi.Path === '.' || fi.Path === '..') continue
      if (fDone[fi.Path]) continue

      if (fPath[fi.Path].base !== fNow.Path) continue

      const fn = {
        key: kpr + '-' + fi.Path + '-' + (fi.ModTime || 0).toString(),
        Path: fi.Path,
        link: pathEncode(fi.Path),
        label: fPath[fi.Path].label,
        descr: Hlpr.modTsToTimeStamp(fi.ModTime),
        children: [],
        isGroup: fi.IsDir,
        Size: fi.Size
      }
      fNow.children.push(fn)
      fDone[fi.Path] = fn

      if (fi.IsDir) fProc.push(fn)
    }
  }

  // push top level files after top level folders
  fTree.push(...fTopFiles)

  return { isRootDir: isRoot, isAnyDir: isAny, tree: fTree }
}
