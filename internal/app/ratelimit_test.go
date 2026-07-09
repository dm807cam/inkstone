package app

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestIPRateLimiterPerKey(t *testing.T) {
	// A nil limiter must allow everything (so an App built without one is fine).
	var nilLimiter *ipRateLimiter
	if !nilLimiter.allow("x") {
		t.Fatal("a nil limiter must allow all requests")
	}

	r := newIPRateLimiter(2, time.Hour, 100)
	if !r.allow("a") || !r.allow("a") {
		t.Fatal("first two requests for a source should be allowed")
	}
	if r.allow("a") {
		t.Fatal("third request for the same source should be throttled")
	}
	if !r.allow("b") {
		t.Fatal("a different source should have its own budget")
	}
}

// TestIPRateLimiterFailsOpenWhenFull verifies the tracked-source table is
// hard-capped: once maxKeys sources are seen, a new source is allowed rather
// than growing the map without bound (so the limiter is not itself a memory DoS).
func TestIPRateLimiterFailsOpenWhenFull(t *testing.T) {
	const maxKeys = 8
	r := newIPRateLimiter(1, time.Hour, maxKeys)
	for i := 0; i < maxKeys; i++ {
		if !r.allow("src" + strconv.Itoa(i)) {
			t.Fatalf("source %d should be allowed while the table has room", i)
		}
	}
	if !r.allow("overflow") {
		t.Fatal("new source should fail open when the table is full")
	}
}

// TestNewDeviceRateLimited guards #34 item 3: repeated pairing attempts from one
// source are throttled with 429 rather than allowed to brute-force unbounded.
func TestNewDeviceRateLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := &App{deviceCodeLimiter: newIPRateLimiter(1, time.Hour, 100)}

	newReq := func() *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		// Invalid JSON so the handler returns right after the limiter check.
		c.Request = httptest.NewRequest(http.MethodPost, "/token/json/2/device/new", strings.NewReader("not json"))
		c.Request.RemoteAddr = "203.0.113.7:5555"
		return c
	}

	first := newReq()
	app.newDevice(first)
	if got := first.Writer.Status(); got == http.StatusTooManyRequests {
		t.Fatalf("first attempt should pass the limiter, got %d", got)
	}

	second := newReq()
	app.newDevice(second)
	if got := second.Writer.Status(); got != http.StatusTooManyRequests {
		t.Fatalf("second attempt from the same source should be 429, got %d", got)
	}
}
