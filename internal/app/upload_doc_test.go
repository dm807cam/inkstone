package app

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ddvk/rmfakecloud/internal/common"
	"github.com/gin-gonic/gin"
)

// TestUploadDocMissingMetaReturns400 verifies that a valid multipart upload that
// omits the "meta" form field is rejected with 400 rather than panicking on an
// out-of-range index (form.Value["meta"][0] on an absent field). The sibling
// "file" field and uploadDocV2's meta header already guard their inputs this way;
// this asserts uploadDoc's "meta" field does too. Pre-fix this test panics.
func TestUploadDocMissingMetaReturns400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// A well-formed multipart form that carries a file part but no "meta" field,
	// so the multipart parse succeeds and the handler reaches the meta guard.
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, err := w.CreateFormFile("file", "note.pdf")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write([]byte("%PDF-1.4")); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/sync/v3/files", &body)
	c.Request.Header.Set("Content-Type", w.FormDataContentType())
	// uploadDoc reads the sync version before parsing the form; getSyncVersion
	// panics if it is unset.
	c.Set(syncVersionKey, common.Sync15)

	app := &App{}
	app.uploadDoc(c)

	// gin buffers the status; c.AbortWithStatus records it on c.Writer.
	if got := c.Writer.Status(); got != http.StatusBadRequest {
		t.Fatalf("missing 'meta' should yield %d, got %d", http.StatusBadRequest, got)
	}
}
