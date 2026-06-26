package fs

import (
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/ddvk/rmfakecloud/internal/config"
)

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
