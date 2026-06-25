package hwr

import (
	"bytes"
	"encoding/json"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ddvk/rmfakecloud/internal/config"
)

// sampleIink is a tiny iink-batch payload with two strokes, shaped like what the tablet POSTs.
const sampleIink = `{
  "contentType": "Text",
  "width": 100, "height": 100, "xDPI": 96, "yDPI": 96,
  "configuration": {"lang": "en_US"},
  "strokeGroups": [
    {"strokes": [
      {"x": [10,20,30,40], "y": [10,12,11,10], "t": [0,1,2,3], "p": [1,1,1,1]},
      {"x": [10,40], "y": [30,30]}
    ]}
  ]
}`

func TestRenderStrokesPNG(t *testing.T) {
	out, err := renderStrokesPNG([]byte(sampleIink))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	if b := img.Bounds(); b.Dx() == 0 || b.Dy() == 0 {
		t.Fatalf("empty image: %v", b)
	}

	// There must be some ink: at least one non-white pixel.
	inked := false
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y && !inked; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if r, g, bl, _ := img.At(x, y).RGBA(); r == 0 && g == 0 && bl == 0 {
				inked = true
				break
			}
		}
	}
	if !inked {
		t.Fatal("rendered image has no ink")
	}
}

func TestRenderStrokesPNGEmpty(t *testing.T) {
	if _, err := renderStrokesPNG([]byte(`{"strokeGroups": []}`)); err == nil {
		t.Fatal("expected error for empty strokes")
	}
	if _, err := renderStrokesPNG([]byte(`not json`)); err == nil {
		t.Fatal("expected error for invalid json")
	}
}

func TestLLMSendRequest(t *testing.T) {
	const want = "hello\nworld"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("missing/bad auth header: %q", got)
		}
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "test-model" {
			t.Errorf("model = %q", req.Model)
		}
		// The request must carry the rendered image as a data URI.
		var sawImage bool
		for _, part := range req.Messages[0].Content {
			if part.Type == "image_url" && part.ImageURL != nil &&
				strings.HasPrefix(part.ImageURL.URL, "data:image/png;base64,") {
				sawImage = true
			}
		}
		if !sawImage {
			t.Error("request did not include a png image_url")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{{Message: struct {
				Content string `json:"content"`
			}{Content: want}}},
		})
	}))
	defer srv.Close()

	cfg := &config.Config{
		HWRProvider: "llm",
		HWRLLMURL:   srv.URL,
		HWRLLMKey:   "secret",
		HWRLLMModel: "test-model",
	}

	rec := NewRecognizer(cfg)
	if _, ok := rec.(*LLMClient); !ok {
		t.Fatalf("expected *LLMClient, got %T", rec)
	}

	out, err := rec.SendRequest([]byte(sampleIink))
	if err != nil {
		t.Fatalf("SendRequest: %v", err)
	}

	var doc jiixDoc
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("unmarshal jiix: %v", err)
	}
	if doc.Type != "Text" {
		t.Errorf("jiix type = %q", doc.Type)
	}
	if doc.Label != want {
		t.Errorf("jiix label = %q, want %q", doc.Label, want)
	}
}

func TestLLMNotConfigured(t *testing.T) {
	rec := &LLMClient{}
	if _, err := rec.SendRequest([]byte(sampleIink)); err == nil {
		t.Fatal("expected error when url/model unset")
	}
}

func TestNewRecognizerDefaultsToMyScript(t *testing.T) {
	if rec := NewRecognizer(&config.Config{}); func() bool { _, ok := rec.(*HWRClient); return !ok }() {
		t.Fatalf("expected *HWRClient by default, got %T", rec)
	}
}
