package app

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// newTestContext builds a gin context whose request body is the given string.
func newTestContext(body string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(body))
	return c
}

func TestReadLimitedBodyUnderLimit(t *testing.T) {
	const payload = "small payload"
	c := newTestContext(payload)

	got, err := readLimitedBody(c, maxControlBodySize)
	if err != nil {
		t.Fatalf("unexpected error for under-limit body: %v", err)
	}
	if string(got) != payload {
		t.Fatalf("body altered: got %q, want %q", got, payload)
	}
}

func TestReadLimitedBodyOverLimit(t *testing.T) {
	const limit = 16
	// One byte past the limit must be rejected so the handler never buffers
	// an unbounded body (memory-exhaustion DoS guard).
	c := newTestContext(strings.Repeat("a", limit+1))

	if _, err := readLimitedBody(c, limit); err == nil {
		t.Fatal("expected error for over-limit body, got nil")
	}
}
