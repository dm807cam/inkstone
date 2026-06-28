package exporter

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// v6Fixture returns the bytes of a known-good v6 .rm test page shipped with the rmc-go fork.
func v6Fixture(t *testing.T) []byte {
	t.Helper()
	p := filepath.Join("..", "..", "..", "third_party", "rmc-go", "tests",
		"pen_with_shapes_and_text_boxes_bullets.rm")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return data
}

// fakeTranscriber records the prompts it is given and returns canned text per call so the
// OCRDocument orchestration can be tested without a live vision model.
type fakeTranscriber struct {
	prompts  []string
	replies  []string
	calls    int
	errAfter int // return an error on this 1-based call (0 = never)
}

func (f *fakeTranscriber) Transcribe(png []byte, prompt string) (string, error) {
	f.calls++
	f.prompts = append(f.prompts, prompt)
	if f.errAfter != 0 && f.calls == f.errAfter {
		return "", errors.New("boom")
	}
	if f.calls-1 < len(f.replies) {
		return f.replies[f.calls-1], nil
	}
	return "", nil
}

func TestRenderRmPageToPNG(t *testing.T) {
	png, err := renderRmPageToPNG(v6Fixture(t))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !bytes.HasPrefix(png, []byte("\x89PNG")) {
		t.Fatalf("output is not a PNG (prefix %x)", png[:min(8, len(png))])
	}
}

func TestRenderRmPageNoStrokes(t *testing.T) {
	_, err := renderRmPageToPNG([]byte("not an rm file"))
	if err == nil {
		t.Fatal("expected error for invalid rm data")
	}
}

func TestOCRDocumentTxt(t *testing.T) {
	page := v6Fixture(t)
	fake := &fakeTranscriber{replies: []string{"first page", "second page"}}

	var buf bytes.Buffer
	if err := OCRDocument([][]byte{page, page}, OCRFormatTxt, fake, &buf); err != nil {
		t.Fatalf("OCRDocument: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "first page") || !strings.Contains(out, "second page") {
		t.Errorf("output missing page text: %q", out)
	}
	// Plain text export must use the plain-text prompt, not the Markdown one.
	for _, p := range fake.prompts {
		if p != ocrTxtPrompt {
			t.Errorf("unexpected prompt for txt export: %q", p)
		}
	}
	// A thematic break is Markdown-only; plain text must not contain it.
	if strings.Contains(out, "---") {
		t.Errorf("plain text export should not contain a thematic break: %q", out)
	}
}

func TestOCRDocumentMarkdownSeparator(t *testing.T) {
	page := v6Fixture(t)
	fake := &fakeTranscriber{replies: []string{"# Page one", "# Page two"}}

	var buf bytes.Buffer
	if err := OCRDocument([][]byte{page, page}, OCRFormatMarkdown, fake, &buf); err != nil {
		t.Fatalf("OCRDocument: %v", err)
	}
	if !strings.Contains(buf.String(), "\n---\n") {
		t.Errorf("markdown export should separate pages with a thematic break: %q", buf.String())
	}
	for _, p := range fake.prompts {
		if p != ocrMarkdownPrompt {
			t.Errorf("unexpected prompt for md export: %q", p)
		}
	}
}

// Empty/blank pages must be skipped (not transcribed) and not consume a page slot.
func TestOCRDocumentSkipsBlankPages(t *testing.T) {
	page := v6Fixture(t)
	fake := &fakeTranscriber{replies: []string{"only real page"}}

	var buf bytes.Buffer
	if err := OCRDocument([][]byte{nil, page, {}}, OCRFormatTxt, fake, &buf); err != nil {
		t.Fatalf("OCRDocument: %v", err)
	}
	if fake.calls != 1 {
		t.Errorf("expected 1 transcription call, got %d", fake.calls)
	}
	if strings.TrimSpace(buf.String()) != "only real page" {
		t.Errorf("unexpected output: %q", buf.String())
	}
}

func TestOCRDocumentNothingTranscribed(t *testing.T) {
	fake := &fakeTranscriber{}
	var buf bytes.Buffer
	if err := OCRDocument([][]byte{nil, {}}, OCRFormatTxt, fake, &buf); err == nil {
		t.Fatal("expected error when no pages could be transcribed")
	}
}

func TestOCRDocumentTranscriberError(t *testing.T) {
	page := v6Fixture(t)
	fake := &fakeTranscriber{replies: []string{"ok"}, errAfter: 1}
	var buf bytes.Buffer
	if err := OCRDocument([][]byte{page}, OCRFormatTxt, fake, &buf); err == nil {
		t.Fatal("expected error to propagate from transcriber")
	}
}
