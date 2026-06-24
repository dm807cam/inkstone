package integrations

import (
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ddvk/rmfakecloud/internal/messages"
	"github.com/ddvk/rmfakecloud/internal/model"
)

// TestWebhookSendMessage verifies that SendMessage posts the expected
// multipart request: a JSON "data" field and a PNG "attachment" file, and
// that it returns the endpoint's response body.
func TestWebhookSendMessage(t *testing.T) {
	var gotData string
	var gotAttachment []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		gotData = r.FormValue("data")

		file, _, err := r.FormFile("attachment")
		if err != nil {
			t.Errorf("missing attachment: %v", err)
			http.Error(w, "no attachment", http.StatusBadRequest)
			return
		}
		defer file.Close()
		if _, err := png.Decode(file); err != nil {
			t.Errorf("attachment is not a valid PNG: %v", err)
		}
		gotAttachment = []byte("read")

		w.Write([]byte("accepted"))
	}))
	defer srv.Close()

	wh := newWebhook(model.IntegrationConfig{Endpoint: srv.URL})

	data := messages.IntegrationMessageData{
		Destinations: []json.RawMessage{json.RawMessage(`{"id":"abc"}`)},
	}
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))

	resp, err := wh.SendMessage(data, img)
	if err != nil {
		t.Fatalf("SendMessage returned error: %v", err)
	}

	if resp != "accepted" {
		t.Errorf("response = %q, want %q", resp, "accepted")
	}
	if gotData == "" {
		t.Error("server received empty data field")
	}
	if !json.Valid([]byte(gotData)) {
		t.Errorf("data field is not valid JSON: %q", gotData)
	}
	if gotAttachment == nil {
		t.Error("server received no attachment")
	}
}
