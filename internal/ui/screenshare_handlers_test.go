package ui

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ddvk/rmfakecloud/internal/app/hub"
	"github.com/ddvk/rmfakecloud/internal/screenshare"
	"github.com/gin-gonic/gin"
)

// newSendAnswerContext builds a gin context for screenshareSendAnswer: it carries the
// roomId param, the user/browser context keys the handler reads, and the given JSON body.
func newSendAnswerContext(roomID, body string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(body))
	c.Params = gin.Params{{Key: "roomId", Value: roomID}}
	c.Set(userIDContextKey, "user1")
	c.Set(browserIDContextKey, "device1")
	return c
}

// A missing or non-object signaling payload must be rejected with 400, not crash the
// handler. Before the fix the ignored json.Unmarshal error left a nil map and the
// `inner["sourceDeviceID"] = clientID` write panicked on untrusted client input.
func TestScreenshareSendAnswerRejectsNonObjectPayload(t *testing.T) {
	rm := screenshare.NewRoomManager()
	room := rm.CreateRoom("user1", "device1")
	app := &ReactAppWrapper{roomManager: rm, h: hub.NewHub()}

	cases := map[string]string{
		"non-object payload": `{"payload": 123, "targetClientId":"x"}`,
		"missing payload":    `{"targetClientId":"x"}`,
		"null payload":       `{"payload": null, "targetClientId":"x"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			c := newSendAnswerContext(room.RoomID, body) // must not panic
			app.screenshareSendAnswer(c)
			if got := c.Writer.Status(); got != 400 {
				t.Fatalf("expected 400 for %s, got %d", name, got)
			}
		})
	}
}

// A well-formed object payload must still be accepted (202) so the fix doesn't regress
// the happy path.
func TestScreenshareSendAnswerAcceptsObjectPayload(t *testing.T) {
	rm := screenshare.NewRoomManager()
	room := rm.CreateRoom("user1", "device1")
	app := &ReactAppWrapper{roomManager: rm, h: hub.NewHub()}

	c := newSendAnswerContext(room.RoomID, `{"payload":{"type":"answer"},"targetClientId":"x"}`)
	app.screenshareSendAnswer(c)
	if got := c.Writer.Status(); got != 202 {
		t.Fatalf("expected 202 for valid object payload, got %d", got)
	}
}
