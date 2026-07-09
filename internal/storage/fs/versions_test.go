package fs

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/ddvk/rmfakecloud/internal/config"
)

// TestListDocVersionsAndRestore drives the sync15 version history end to end:
// create a document (generation 1), rename it (generation 2), list the two
// resulting versions, and restore the oldest to prove the content/name revert.
func TestListDocVersionsAndRestore(t *testing.T) {
	uid := "versionuser"
	dir := path.Join(os.TempDir(), "rmfake-versions")
	t.Cleanup(func() { os.RemoveAll(dir) })

	fs := NewStorage(&config.Config{DataDir: dir})
	if err := os.MkdirAll(fs.getUserBlobPath(uid), 0700); err != nil {
		t.Fatal(err)
	}

	// Generation 1: create the document (visible name derived from the filename).
	doc, err := fs.CreateBlobDocument(uid, "notes.pdf", "", strings.NewReader("v1 body"))
	if err != nil {
		t.Fatalf("CreateBlobDocument: %v", err)
	}

	// Generation 2: rename it. A metadata change gives the document a new hash,
	// which is what makes it a distinct restorable version.
	if err := fs.UpdateBlobDocument(uid, doc.ID, "renamed", ""); err != nil {
		t.Fatalf("UpdateBlobDocument: %v", err)
	}

	versions, err := fs.ListDocVersions(uid, doc.ID)
	if err != nil {
		t.Fatalf("ListDocVersions: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("want 2 versions, got %d (%+v)", len(versions), versions)
	}
	// Newest first.
	if versions[0].Generation <= versions[1].Generation {
		t.Errorf("versions not newest-first: %+v", versions)
	}

	// Restore the oldest version; the original name should come back.
	oldest := versions[len(versions)-1]
	if err := fs.RestoreVersion(uid, doc.ID, oldest.RootHash); err != nil {
		t.Fatalf("RestoreVersion: %v", err)
	}

	tree, err := fs.GetCachedTree(uid)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := tree.FindDoc(doc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored.DocumentName != "notes" {
		t.Errorf("restored name = %q, want %q", restored.DocumentName, "notes")
	}
}
