/*
Shared functions for parameters or output tables tree
*/

/*
Walk tree and remove empty branches.

Input gTree[] must be an array of nodes, NOT a single top level node.
Each node expected to have following properties:
{
  key: 'any key',         // used for debug only
  isGroup: true or false, // if true then it is a group
  children: []            // children of the group
}
Empty groups are removed from the tree.
Group node is a node where if (node.isTree) is true.
*/
export const removeEmptyGroups = gTree => {
  // add top level tree nodes as starting point
  const wStack = []
  wStack.push({
    key: 'walk-tree-top-level-node',
    index: 0,
    isGroup: true,
    children: []
  })
  for (let k = 0; k < gTree.length; k++) {
    wStack[0].children.push(gTree[k])
  }

  // walk the tree until end of top level and remove empty branches
  while (wStack.length > 0) {
    const level = wStack[wStack.length - 1]

    // end of current level: pop to the parent level
    // if current level is empty group then remove it from the parent
    if (level.index >= level.children.length) {
      wStack.pop()
      if (wStack.length <= 0) break // end of tree top level

      const parent = wStack[wStack.length - 1]
      if (!level.isGroup || level.children.length > 0) {
        parent.index++ // move to the next child in parent list
      } else {
        if (parent.children.length < parent.index) parent.children.splice(parent.index, 1)
      }
      continue
    }

    // for all children of current level do:
    //   if child is not a group then skip it (go to next child)
    //   if child is empty group then remove that child
    //   if child is not empty group then push it to the stack and goto the next level down
    while (level.index < level.children.length) {
      const child = level.children[level.index]
      if (!child.isGroup) {
        level.index++
        continue
      }
      if (child.children.length <= 0) {
        level.children.splice(level.index, 1)
        continue
      }
      // else: child is not empty group, go to the one level down
      wStack.push({
        key: child.key,
        index: 0,
        isGroup: child.isGroup,
        children: child.children
      })
      break // go to the one level down
    }
  }

  // remove empty branches from top level of the tree
  return gTree.filter(g => !g.isGroup || g.children.length > 0)
}
