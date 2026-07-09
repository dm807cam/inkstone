package fs

import (
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/ddvk/rmfakecloud/internal/config"
)

// failingReader yields one chunk of data and then fails, simulating a client
// disconnect or I/O error partway through an upload.
type failingReader struct {
	chunk []byte
	sent  bool
}

func (r *failingReader) Read(p []byte) (int, error) {
	if !r.sent {
		r.sent = true
		return copy(p, r.chunk), nil
	}
	return 0, errors.New("simulated mid-stream failure")
}

// TestLoadBlobSuccess covers the success path of LoadBlob: a stored blob is
// returned as a readable, closeable reader with the correct size and crc32c
// hash. This guards the path touched by the FD-leak fix (#12).
func TestLoadBlobSuccess(t *testing.T) {
	uid := "blobuser"
	dir := path.Join(os.TempDir(), "rmfake-loadblob")
	t.Cleanup(func() { os.RemoveAll(dir) })

	fs := NewStorage(&config.Config{DataDir: dir})
	blobDir := fs.getUserBlobPath(uid)
	if err := os.MkdirAll(blobDir, 0700); err != nil {
		t.Fatal(err)
	}

	const blobID = "abc123"
	const content = "hello blob"
	if err := os.WriteFile(path.Join(blobDir, blobID), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	reader, _, size, hash, err := fs.LoadBlob(uid, blobID)
	if err != nil {
		t.Fatalf("LoadBlob returned error: %v", err)
	}
	if reader == nil {
		t.Fatal("LoadBlob returned a nil reader")
	}
	defer reader.Close()

	if size != int64(len(content)) {
		t.Errorf("size = %d, want %d", size, len(content))
	}
	if !strings.HasPrefix(hash, "crc32c=") {
		t.Errorf("hash = %q, want a crc32c= prefix", hash)
	}

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading blob: %v", err)
	}
	if string(got) != content {
		t.Errorf("content = %q, want %q", got, content)
	}
}

// TestLoadBlobNotFound covers the missing-blob path.
func TestLoadBlobNotFound(t *testing.T) {
	uid := "blobuser"
	dir := path.Join(os.TempDir(), "rmfake-loadblob-missing")
	t.Cleanup(func() { os.RemoveAll(dir) })

	fs := NewStorage(&config.Config{DataDir: dir})
	if err := os.MkdirAll(fs.getUserBlobPath(uid), 0700); err != nil {
		t.Fatal(err)
	}

	reader, _, _, _, err := fs.LoadBlob(uid, "does-not-exist")
	if err != ErrorNotFound {
		t.Errorf("err = %v, want ErrorNotFound", err)
	}
	if reader != nil {
		t.Error("expected nil reader for a missing blob")
		reader.Close()
	}
}

// TestStoreBlobTornWriteLeavesNoFinalFile guards issue #30: if the upload stream
// fails partway through, StoreBlob must return an error and leave NO file at the
// canonical (content-addressed) blob path — a truncated blob there would be
// permanently served as corrupt bytes for a name that promises exact contents.
func TestStoreBlobTornWriteLeavesNoFinalFile(t *testing.T) {
	uid := "blobuser"
	dir := path.Join(os.TempDir(), "rmfake-storeblob-torn")
	t.Cleanup(func() { os.RemoveAll(dir) })

	fs := NewStorage(&config.Config{DataDir: dir})
	blobDir := fs.getUserBlobPath(uid)
	if err := os.MkdirAll(blobDir, 0700); err != nil {
		t.Fatal(err)
	}

	const blobID = "deadbeef"
	r := &failingReader{chunk: []byte("partial-content-before-failure")}
	if _, err := fs.StoreBlob(uid, blobID, r, -1); err == nil {
		t.Fatal("StoreBlob returned nil error on a failing stream; expected an error")
	}

	if _, err := os.Stat(path.Join(blobDir, blobID)); !os.IsNotExist(err) {
		t.Errorf("a file exists at the canonical blob path after a torn write (err=%v); "+
			"StoreBlob must be atomic", err)
	}

	// The temp file must also be cleaned up on failure.
	entries, err := os.ReadDir(blobDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("blob dir not empty after a torn write, leftover: %v", names)
	}
}

// TestStoreBlobRoundTrip verifies the atomic write path preserves contents
// exactly: a stored blob reads back byte-for-byte.
func TestStoreBlobRoundTrip(t *testing.T) {
	uid := "blobuser"
	dir := path.Join(os.TempDir(), "rmfake-storeblob-roundtrip")
	t.Cleanup(func() { os.RemoveAll(dir) })

	fs := NewStorage(&config.Config{DataDir: dir})
	if err := os.MkdirAll(fs.getUserBlobPath(uid), 0700); err != nil {
		t.Fatal(err)
	}

	const blobID = "cafebabe"
	const content = "the exact bytes that must survive a temp+rename"
	if _, err := fs.StoreBlob(uid, blobID, strings.NewReader(content), -1); err != nil {
		t.Fatalf("StoreBlob returned error: %v", err)
	}

	reader, _, size, _, err := fs.LoadBlob(uid, blobID)
	if err != nil {
		t.Fatalf("LoadBlob returned error: %v", err)
	}
	defer reader.Close()
	if size != int64(len(content)) {
		t.Errorf("size = %d, want %d", size, len(content))
	}
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Errorf("round-trip content = %q, want %q", got, content)
	}
}
