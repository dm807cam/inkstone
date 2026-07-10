package fs

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/ddvk/rmfakecloud/internal/config"
	"github.com/ddvk/rmfakecloud/internal/storage"
	"github.com/ddvk/rmfakecloud/internal/storage/models"
	"github.com/google/uuid"
)

// buildRmDoc returns a valid .rmdoc archive: a <docid>.metadata entry plus any
// extra entries. extra maps entry name -> uncompressed contents (Deflate is
// used so a large, highly-compressible entry stays small on the wire, which is
// exactly the decompression-bomb shape being guarded against).
func buildRmDoc(t *testing.T, docid, name string, extra map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	metaJSON, err := json.Marshal(models.MetadataFile{DocumentName: name, Version: 1})
	if err != nil {
		t.Fatal(err)
	}
	mw, err := zw.Create(docid + storage.MetadataFileExt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mw.Write(metaJSON); err != nil {
		t.Fatal(err)
	}

	for entryName, content := range extra {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: entryName, Method: zip.Deflate})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func newRmDocTestStorage(t *testing.T) (*FileSystemStorage, string) {
	t.Helper()
	fs := NewStorage(&config.Config{DataDir: t.TempDir()})
	uid := "rmdocuser"
	if err := os.MkdirAll(fs.getUserBlobPath(uid), 0700); err != nil {
		t.Fatal(err)
	}
	return fs, uid
}

// TestCreateFromRmDocRoundTrip is the happy path: a legitimate small .rmdoc is
// accepted and turned into a document. This guards against the size bounds (#32)
// rejecting normal uploads.
func TestCreateFromRmDocRoundTrip(t *testing.T) {
	fs, uid := newRmDocTestStorage(t)
	docid := uuid.New().String()
	archive := buildRmDoc(t, docid, "My Notebook", nil)

	doc, err := fs.createFromRmDoc(uid, "", bytes.NewReader(archive))
	if err != nil {
		t.Fatalf("createFromRmDoc returned error on a valid archive: %v", err)
	}
	if doc == nil || doc.ID != docid {
		t.Fatalf("expected document id %q, got %+v", docid, doc)
	}
	if doc.Name != "My Notebook" {
		t.Errorf("document name = %q, want %q", doc.Name, "My Notebook")
	}
}

// TestCreateFromRmDocRejectsOversizedArchive verifies the raw upload is bounded
// before it is fully buffered, so an arbitrarily large body cannot exhaust
// memory. (#32)
func TestCreateFromRmDocRejectsOversizedArchive(t *testing.T) {
	fs, uid := newRmDocTestStorage(t)

	orig := maxRmDocArchiveBytes
	maxRmDocArchiveBytes = 1024
	t.Cleanup(func() { maxRmDocArchiveBytes = orig })

	body := bytes.NewReader(make([]byte, maxRmDocArchiveBytes+512))
	_, err := fs.createFromRmDoc(uid, "", body)
	if err == nil {
		t.Fatal("expected an error for an oversized archive, got nil")
	}
	if !strings.Contains(err.Error(), "archive exceeds") {
		t.Errorf("error = %q, want it to mention the archive size limit", err)
	}
}

// TestCreateFromRmDocRejectsDecompressionBomb verifies a small archive whose
// entries expand past the total-uncompressed cap is rejected mid-extraction,
// without reading the entry in full. (#32)
func TestCreateFromRmDocRejectsDecompressionBomb(t *testing.T) {
	fs, uid := newRmDocTestStorage(t)

	orig := maxRmDocTotalUncompressedBytes
	maxRmDocTotalUncompressedBytes = 64 * 1024
	t.Cleanup(func() { maxRmDocTotalUncompressedBytes = orig })

	// 8 MiB of zeros: tiny once Deflated, but far past the 64 KiB cap.
	bomb := make([]byte, 8<<20)
	docid := uuid.New().String()
	archive := buildRmDoc(t, docid, "bomb", map[string][]byte{docid + ".rm": bomb})

	_, err := fs.createFromRmDoc(uid, "", bytes.NewReader(archive))
	if err == nil {
		t.Fatal("expected an error for a decompression bomb, got nil")
	}
	if !strings.Contains(err.Error(), "uncompressed size exceeds") {
		t.Errorf("error = %q, want it to mention the uncompressed size limit", err)
	}

	// The bomb must not have been committed to the tree.
	tree, err := fs.GetCachedTree(uid)
	if err != nil {
		t.Fatalf("GetCachedTree: %v", err)
	}
	if _, err := tree.FindDoc(docid); err == nil {
		t.Errorf("bomb document %q should not have been added to the tree", docid)
	}
}
