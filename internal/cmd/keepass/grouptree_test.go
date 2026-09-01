package keepass

import (
	"testing"

	"github.com/passbolt/go-passbolt/api"
	"github.com/tobischo/gokeepasslib/v3"
)

func entryTitled(title string) gokeepasslib.Entry {
	e := gokeepasslib.NewEntry()
	e.Values = append(e.Values, gokeepasslib.ValueData{
		Key: "Title", Value: gokeepasslib.V{Content: title},
	})
	return e
}

func groupNames(groups []gokeepasslib.Group) []string {
	names := make([]string, len(groups))
	for i, g := range groups {
		names[i] = g.Name
	}
	return names
}

func TestBuildFolderTree_Nested(t *testing.T) {
	folders := []api.Folder{
		{ID: "c", Name: "child", FolderParentID: "a"},
		{ID: "a", Name: "top"},
		{ID: "g", Name: "grandchild", FolderParentID: "c"},
	}
	root := &folderNode{name: "root"}
	nodes := buildFolderTree(root, folders)

	nodes["g"].entries = append(nodes["g"].entries, entryTitled("deep"))

	got := root.toGroup()
	if len(got.Groups) != 1 || got.Groups[0].Name != "top" {
		t.Fatalf("root groups = %v, want [top]", groupNames(got.Groups))
	}
	top := got.Groups[0]
	if len(top.Groups) != 1 || top.Groups[0].Name != "child" {
		t.Fatalf("top groups = %v, want [child]", groupNames(top.Groups))
	}
	child := top.Groups[0]
	if len(child.Groups) != 1 || child.Groups[0].Name != "grandchild" {
		t.Fatalf("child groups = %v, want [grandchild]", groupNames(child.Groups))
	}
	grand := child.Groups[0]
	if len(grand.Entries) != 1 || grand.Entries[0].GetTitle() != "deep" {
		t.Fatalf("grandchild entries = %+v, want one entry titled deep", grand.Entries)
	}
}

func TestBuildFolderTree_SameNameSiblings(t *testing.T) {
	// Passbolt allows same-named folders on the same level; matching by name
	// would merge them. Each must stay a distinct group with its own entries.
	folders := []api.Folder{
		{ID: "f1", Name: "dup"},
		{ID: "f2", Name: "dup"},
	}
	root := &folderNode{name: "root"}
	nodes := buildFolderTree(root, folders)

	nodes["f1"].entries = append(nodes["f1"].entries, entryTitled("in-first"))
	nodes["f2"].entries = append(nodes["f2"].entries, entryTitled("in-second"))

	got := root.toGroup()
	if len(got.Groups) != 2 {
		t.Fatalf("root groups = %v, want two dup groups", groupNames(got.Groups))
	}
	for i, wantTitle := range []string{"in-first", "in-second"} {
		g := got.Groups[i]
		if g.Name != "dup" {
			t.Errorf("group %d name = %q, want dup", i, g.Name)
		}
		if len(g.Entries) != 1 || g.Entries[0].GetTitle() != wantTitle {
			t.Errorf("group %d entries = %+v, want one entry titled %q", i, g.Entries, wantTitle)
		}
	}
}

func TestBuildFolderTree_OrphanAttachesToRoot(t *testing.T) {
	// A folder can reference a parent the exporting user cannot see; it must
	// fall back to the root group instead of disappearing.
	folders := []api.Folder{
		{ID: "o", Name: "orphan", FolderParentID: "invisible"},
	}
	root := &folderNode{name: "root"}
	buildFolderTree(root, folders)

	got := root.toGroup()
	if len(got.Groups) != 1 || got.Groups[0].Name != "orphan" {
		t.Fatalf("root groups = %v, want [orphan]", groupNames(got.Groups))
	}
}

func TestBuildFolderTree_CycleAttachesToRoot(t *testing.T) {
	// Broken parent data must not hang the export: cyclic folders (and their
	// descendants) flatten to root, and toGroup terminates.
	folders := []api.Folder{
		{ID: "a", Name: "alpha", FolderParentID: "b"},
		{ID: "b", Name: "beta", FolderParentID: "a"},
		{ID: "s", Name: "selfie", FolderParentID: "s"},
		{ID: "c", Name: "below-cycle", FolderParentID: "a"},
	}
	root := &folderNode{name: "root"}
	buildFolderTree(root, folders)

	got := root.toGroup()
	if len(got.Groups) != 4 {
		t.Fatalf("root groups = %v, want all four cyclic folders flattened", groupNames(got.Groups))
	}
}

func TestSubtreeIDs(t *testing.T) {
	folders := []api.Folder{
		{ID: "a", Name: "top"},
		{ID: "b", Name: "child", FolderParentID: "a"},
		{ID: "c", Name: "grandchild", FolderParentID: "b"},
		{ID: "x", Name: "other"},
	}

	got := subtreeIDs(folders, "a")
	for _, id := range []string{"a", "b", "c"} {
		if !got[id] {
			t.Errorf("subtree of a should contain %q, got %v", id, got)
		}
	}
	if got["x"] {
		t.Errorf("subtree of a must not contain unrelated folder x, got %v", got)
	}

	if got := subtreeIDs(folders, "c"); len(got) != 1 || !got["c"] {
		t.Errorf("subtree of leaf c = %v, want just c", got)
	}
}

func TestSubtreeIDs_UnknownFolder(t *testing.T) {
	folders := []api.Folder{{ID: "a", Name: "top"}}
	if got := subtreeIDs(folders, "missing"); got != nil {
		t.Errorf("unknown folder = %v, want nil", got)
	}
}

func TestSubtreeIDs_CycleTerminates(t *testing.T) {
	folders := []api.Folder{
		{ID: "a", Name: "alpha", FolderParentID: "b"},
		{ID: "b", Name: "beta", FolderParentID: "a"},
	}
	got := subtreeIDs(folders, "a")
	if !got["a"] || !got["b"] {
		t.Errorf("cyclic subtree = %v, want both members", got)
	}
}

func TestToGroup_SortsChildrenByName(t *testing.T) {
	folders := []api.Folder{
		{ID: "z", Name: "zeta"},
		{ID: "a", Name: "alpha"},
		{ID: "m", Name: "mu"},
	}
	root := &folderNode{name: "root"}
	buildFolderTree(root, folders)

	got := root.toGroup()
	want := []string{"alpha", "mu", "zeta"}
	names := groupNames(got.Groups)
	if len(names) != len(want) {
		t.Fatalf("root groups = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("root groups = %v, want %v (sorted by name)", names, want)
		}
	}
}
