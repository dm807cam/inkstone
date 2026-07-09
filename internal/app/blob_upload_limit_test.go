package app

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ddvk/rmfakecloud/internal/storage"
	"github.com/gin-gonic/gin"
)

// consumingBlobStorer is a BlobStorage stub whose StoreBlob fully reads the
// stream (like the real file-system store), so an http.MaxBytesReader wrapping
// the body actually trips when the client sends too much.
type consumingBlobStorer struct{ read int64 }

func (s *consumingBlobStorer) StoreBlob(uid, blobID string, r io.Reader, matchGeneration int64) (int64, error) {
	n, err := io.Copy(io.Discard, r)
	s.read = n
	return 0, err
}
func (s *consumingBlobStorer) LoadBlob(uid, blobID string) (io.ReadCloser, int64, int64, string, error) {
	return nil, 0, 0, "", nil
}
func (s *consumingBlobStorer) GetBlobURL(uid, docid string, write bool) (string, time.Time, error) {
	return "", time.Time{}, nil
}
func (s *consumingBlobStorer) CreateBlobDocument(uid, name, parent string, stream io.Reader) (*storage.Document, error) {
	return nil, nil
}

func newBlobWriteContext(body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/blob", strings.NewReader(body))
	c.Set(userIDKey, "u1")
	return c, rec
}

// TestBlobStorageWriteRejectsOversizedBody guards #32: a body over the cap must
// be rejected with 413 instead of streaming unbounded onto disk.
func TestBlobStorageWriteRejectsOversizedBody(t *testing.T) {
	const limit = 16
	orig := blobUploadLimit
	blobUploadLimit = limit
	t.Cleanup(func() { blobUploadLimit = orig })

	store := &consumingBlobStorer{}
	app := &App{blobStorer: store}

	c, _ := newBlobWriteContext(strings.Repeat("a", limit+1))
	app.blobStorageWrite(c)

	if got := c.Writer.Status(); got != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body should yield %d, got %d", http.StatusRequestEntityTooLarge, got)
	}
}

// TestBlobStorageWriteAcceptsBodyAtLimit verifies a conforming (<= limit) body
// still succeeds and is stored in full.
func TestBlobStorageWriteAcceptsBodyAtLimit(t *testing.T) {
	const limit = 16
	orig := blobUploadLimit
	blobUploadLimit = limit
	t.Cleanup(func() { blobUploadLimit = orig })

	store := &consumingBlobStorer{}
	app := &App{blobStorer: store}

	body := strings.Repeat("a", limit)
	c, _ := newBlobWriteContext(body)
	app.blobStorageWrite(c)

	if got := c.Writer.Status(); got != http.StatusOK {
		t.Fatalf("at-limit body should yield %d, got %d", http.StatusOK, got)
	}
	if store.read != int64(len(body)) {
		t.Fatalf("stored %d bytes, want %d", store.read, len(body))
	}
}
