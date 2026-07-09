package fs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// v6OCRFixture returns a known-good v6 .rm page so renderRmPageToPNG produces an image to
// hand to the (fake) transcriber.
func v6OCRFixture(t *testing.T) []byte {
	t.Helper()
	p := filepath.Join("..", "..", "..", "third_party", "rmc-go", "tests",
		"pen_with_shapes_and_text_boxes_bullets.rm")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return data
}

// fakeOCRTranscriber satisfies exporter.Transcriber: it returns canned text or an error
// without contacting a vision model.
type fakeOCRTranscriber struct {
	reply string
	err   error
}

func (f fakeOCRTranscriber) Transcribe(png []byte, prompt string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.reply, nil
}

// A transcription failure must propagate as an error (and a nil reader) so the HTTP handler
// can return a proper error status, instead of being swallowed behind an already-committed
// 200 with an empty body. This guards the fix for #20.
func TestOCRPagesToReaderPropagatesError(t *testing.T) {
	page := v6OCRFixture(t)
	reader, err := ocrPagesToReader([][]byte{page}, "txt", fakeOCRTranscriber{err: errors.New("vision endpoint down")})
	if err == nil {
		t.Fatal("expected an error when transcription fails")
	}
	if reader != nil {
		t.Error("expected a nil reader on failure")
		reader.Close()
	}
}

// On success the returned reader yields the concatenated transcription.
func TestOCRPagesToReaderSuccess(t *testing.T) {
	page := v6OCRFixture(t)
	reader, err := ocrPagesToReader([][]byte{page}, "txt", fakeOCRTranscriber{reply: "transcribed page"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reader == nil {
		t.Fatal("expected a non-nil reader on success")
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading OCR output: %v", err)
	}
	if !strings.Contains(string(got), "transcribed page") {
		t.Errorf("OCR output missing transcription: %q", got)
	}
}
