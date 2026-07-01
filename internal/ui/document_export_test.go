package ui

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ddvk/rmfakecloud/internal/common"
	"github.com/ddvk/rmfakecloud/internal/storage"
	"github.com/ddvk/rmfakecloud/internal/ui/viewmodel"
	"github.com/gin-gonic/gin"
)

// fakeExportBackend implements `backend`; only Export/GetDocumentTree carry behaviour.
type fakeExportBackend struct{ tree *viewmodel.DocumentTree }

func (f *fakeExportBackend) GetDocumentTree(string) (*viewmodel.DocumentTree, error) {
	return f.tree, nil
}
func (f *fakeExportBackend) Export(_, _, _ string, _ storage.ExportOption) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("body")), nil
}
func (f *fakeExportBackend) CreateDocument(_, _, _ string, _ io.Reader) (*storage.Document, error) {
	return nil, nil
}
func (f *fakeExportBackend) CreateFolder(_, _, _ string) (*storage.Document, error) { return nil, nil }
func (f *fakeExportBackend) UpdateDocument(_, _, _, _ string) error                 { return nil }
func (f *fakeExportBackend) DeleteDocument(_, _ string) error                       { return nil }
func (f *fakeExportBackend) Sync(string)                                            {}

// exportDisposition runs getDocument against a tree and returns the header it set.
func exportDisposition(tree *viewmodel.DocumentTree, docid, typ string) string {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?type="+typ, nil)
	c.Params = gin.Params{{Key: docIDParam, Value: docid}}
	c.Set(userIDContextKey, "user1")
	c.Set(backendVersionKey, common.Sync15)
	app := &ReactAppWrapper{backends: map[common.SyncVersion]backend{common.Sync15: &fakeExportBackend{tree: tree}}}
	app.getDocument(c)
	return c.Writer.Header().Get("Content-Disposition")
}

// A resolvable docid names the download by its visible name: sanitized ASCII
// filename= plus RFC 5987 filename* for the non-ASCII original; never the raw docid.
func TestGetDocumentNamesDownloadByVisibleName(t *testing.T) {
	const docid = "a1b2c3d4-0000"
	tree := &viewmodel.DocumentTree{Entries: []viewmodel.Entry{
		&viewmodel.Directory{ID: "f1", Name: "Notes", Entries: []viewmodel.Entry{
			&viewmodel.Document{ID: docid, Name: "Café Meeting"},
		}},
	}}
	cd := exportDisposition(tree, docid, "txt")
	if !strings.Contains(cd, `filename="Caf_ Meeting.txt"`) ||
		!strings.Contains(cd, "filename*=UTF-8''Caf%C3%A9%20Meeting.txt") ||
		strings.Contains(cd, docid) {
		t.Fatalf("unexpected disposition: %q", cd)
	}
}

// An unresolvable docid falls back to the docid so the download stays stable.
func TestGetDocumentFallsBackToDocidWhenNameUnresolved(t *testing.T) {
	const docid = "unknown-doc-id"
	cd := exportDisposition(&viewmodel.DocumentTree{}, docid, "md")
	if !strings.Contains(cd, `filename="`+docid+`.md"`) ||
		!strings.Contains(cd, "filename*=UTF-8''"+docid+".md") {
		t.Fatalf("expected docid fallback, got: %q", cd)
	}
}
