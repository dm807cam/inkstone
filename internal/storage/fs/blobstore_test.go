package fs

import (
	"io"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/ddvk/rmfakecloud/internal/config"
	"github.com/juju/fslock"
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

// TestStoreBlobRootLockContention guards issue #29: when the per-user
// .root.history lock cannot be acquired, StoreBlob must fail closed — return an
// error and leave .root.history and the root blob untouched — rather than log
// the failure and overwrite the root without the lock (which silently loses a
// concurrent sync's documents).
func TestStoreBlobRootLockContention(t *testing.T) {
	uid := "blobuser"
	dir := path.Join(os.TempDir(), "rmfake-storeblob-lock")
	t.Cleanup(func() { os.RemoveAll(dir) })

	fs := NewStorage(&config.Config{DataDir: dir})
	blobDir := fs.getUserBlobPath(uid)
	if err := os.MkdirAll(blobDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Hold the root-history lock from a second handle, as a concurrent sync
	// would. flock treats distinct file descriptors independently, so this
	// contends with StoreBlob's own lock even within one process.
	historyPath := path.Join(blobDir, historyFile)
	held := fslock.New(historyPath)
	if err := held.LockWithTimeout(time.Second); err != nil {
		t.Fatalf("could not take the contended lock: %v", err)
	}
	defer held.Unlock()

	// Pre-state: the lock above created .root.history empty.
	before, err := os.Stat(historyPath)
	if err != nil {
		t.Fatalf("stat history: %v", err)
	}

	// StoreBlob cannot obtain the lock, so (after its 5s internal timeout) it
	// must return an error and mutate nothing.
	gen, err := fs.StoreBlob(uid, rootBlob, strings.NewReader("new-root-generation"), 0)
	if err == nil {
		t.Fatalf("StoreBlob succeeded despite lock contention (gen=%d); expected an error", gen)
	}

	after, err := os.Stat(historyPath)
	if err != nil {
		t.Fatalf("stat history after: %v", err)
	}
	if after.Size() != before.Size() {
		t.Errorf(".root.history size changed from %d to %d; StoreBlob mutated it without the lock",
			before.Size(), after.Size())
	}

	if _, err := os.Stat(path.Join(blobDir, rootBlob)); !os.IsNotExist(err) {
		t.Errorf("root blob was created despite lock contention (err=%v)", err)
	}
}
