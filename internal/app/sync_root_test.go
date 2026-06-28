package app

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ddvk/rmfakecloud/internal/storage"
	"github.com/ddvk/rmfakecloud/internal/storage/fs"
	"github.com/gin-gonic/gin"
)

// spyReadCloser counts Close calls so a test can assert the caller honours the
// io.ReadCloser ownership contract of LoadBlob.
type spyReadCloser struct {
	io.Reader
	closed int
}

func (s *spyReadCloser) Close() error { s.closed++; return nil }

// fakeBlobStorer is a minimal storage.BlobStorage returning a canned reader (or
// error) from LoadBlob; the root handlers don't touch the other methods.
type fakeBlobStorer struct {
	reader  *spyReadCloser
	gen     int64
	loadErr error
}

func (f *fakeBlobStorer) LoadBlob(uid, blobID string) (io.ReadCloser, int64, int64, string, error) {
	if f.loadErr != nil {
		return nil, 0, 0, "", f.loadErr
	}
	return f.reader, f.gen, 0, "", nil
}
func (f *fakeBlobStorer) GetBlobURL(uid, d string, w bool) (string, time.Time, error) {
	return "", time.Time{}, nil
}
func (f *fakeBlobStorer) StoreBlob(uid, b string, s io.Reader, g int64) (int64, error) { return 0, nil }
func (f *fakeBlobStorer) CreateBlobDocument(uid, n, p string, s io.Reader) (*storage.Document, error) {
	return nil, nil
}

func newRootCtx(rec *httptest.ResponseRecorder) *gin.Context {
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set(userIDKey, "user1")
	return c
}

// TestSyncGetRootClosesReader pins the FD-leak fix: both root handlers must
// close the reader LoadBlob hands them and still return the hash. Before the
// fix the reader was read with io.ReadAll but never closed, leaking a file
// descriptor on every device sync.
func TestSyncGetRootClosesReader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handlers := map[string]func(*App, *gin.Context){"v3": (*App).syncGetRootV3, "v4": (*App).syncGetRootV4}
	for name, call := range handlers {
		t.Run(name, func(t *testing.T) {
			spy := &spyReadCloser{Reader: strings.NewReader("roothash-abc")}
			rec := httptest.NewRecorder()
			call(&App{blobStorer: &fakeBlobStorer{reader: spy, gen: 7}}, newRootCtx(rec))

			if spy.closed != 1 {
				t.Fatalf("reader Close() called %d times, want 1 (FD leak guard)", spy.closed)
			}
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if !strings.Contains(rec.Body.String(), "roothash-abc") {
				t.Fatalf("response missing root hash: %q", rec.Body.String())
			}
		})
	}
}

// TestSyncGetRootNotFound keeps new-account behaviour intact: a missing root
// blob yields V3 404 and V4 200 (the helper must pass fs.ErrorNotFound through).
func TestSyncGetRootNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name       string
		call       func(*App, *gin.Context)
		wantStatus int
	}{
		{"v3", (*App).syncGetRootV3, http.StatusNotFound},
		{"v4", (*App).syncGetRootV4, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.call(&App{blobStorer: &fakeBlobStorer{loadErr: fs.ErrorNotFound}}, newRootCtx(rec))
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
		})
	}
}
