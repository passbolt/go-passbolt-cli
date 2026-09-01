package keepass

import (
	"sort"

	"github.com/passbolt/go-passbolt/api"
	"github.com/tobischo/gokeepasslib/v3"
)

// folderNode is one group of the KeePass tree under construction.
// gokeepasslib groups are value types, so children and entries are collected
// on pointer nodes first and converted into nested Groups in one pass once
// the tree is complete.
type folderNode struct {
	name     string
	children []*folderNode
	entries  []gokeepasslib.Entry
}

// buildFolderTree arranges the Passbolt folders as folderNodes below root and
// returns them indexed by folder ID. Folders are matched by ID because
// sibling names are not unique in Passbolt. A folder whose parent is not in
// the listing (not shared with this user) is attached to root, as is any
// folder whose ancestry loops — a healthy server never produces a cycle, but
// the recursive conversion in toGroup would not terminate on one.
//
// Folder names come from the plaintext v4 Name field; unlike v5 resources,
// the folder API still serves cleartext names.
func buildFolderTree(root *folderNode, folders []api.Folder) map[string]*folderNode {
	nodes := make(map[string]*folderNode, len(folders))
	parentOf := make(map[string]string, len(folders))
	for _, f := range folders {
		nodes[f.ID] = &folderNode{name: f.Name}
		parentOf[f.ID] = f.FolderParentID
	}
	for _, f := range folders {
		parent, ok := nodes[f.FolderParentID]
		if !ok || hasCycle(f.ID, parentOf) {
			parent = root
		}
		parent.children = append(parent.children, nodes[f.ID])
	}
	return nodes
}

// hasCycle reports whether following the parent chain from id ever revisits
// a folder. parentOf returns "" for IDs outside the listing, which ends the
// walk at the root.
func hasCycle(id string, parentOf map[string]string) bool {
	seen := make(map[string]bool)
	for cur := id; cur != ""; cur = parentOf[cur] {
		if seen[cur] {
			return true
		}
		seen[cur] = true
	}
	return false
}

// subtreeIDs returns the IDs of folderID and every folder below it, or nil
// when folderID is not in the listing. The visited check doubles as cycle
// protection against broken parent data.
func subtreeIDs(folders []api.Folder, folderID string) map[string]bool {
	childrenOf := make(map[string][]string, len(folders))
	known := false
	for _, f := range folders {
		childrenOf[f.FolderParentID] = append(childrenOf[f.FolderParentID], f.ID)
		if f.ID == folderID {
			known = true
		}
	}
	if !known {
		return nil
	}

	subtree := map[string]bool{folderID: true}
	queue := []string{folderID}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, child := range childrenOf[cur] {
			if !subtree[child] {
				subtree[child] = true
				queue = append(queue, child)
			}
		}
	}
	return subtree
}

// toGroup converts the node and its subtree into a gokeepasslib.Group.
// Children are sorted by name (stable, so same-named siblings keep their API
// order) to make the export deterministic regardless of listing order.
func (n *folderNode) toGroup() gokeepasslib.Group {
	group := gokeepasslib.NewGroup()
	group.Name = n.name
	group.Entries = n.entries
	sort.SliceStable(n.children, func(i, j int) bool {
		return n.children[i].name < n.children[j].name
	})
	for _, child := range n.children {
		group.Groups = append(group.Groups, child.toGroup())
	}
	return group
}
