package hwr

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"
)

const (
	defaultPrompt = "Transcribe the handwriting in this image to plain text. " +
		"Preserve line breaks. Output only the transcribed text, with no commentary, labels, or code fences."
	llmTimeout = 120 * time.Second
)

// LLMClient recognizes handwriting by rendering the strokes to an image and asking an
// OpenAI-compatible vision model (Ollama, OpenRouter, OpenAI, ...) to transcribe it. Its
// fields are populated either from the instance config or from a user's per-user override.
type LLMClient struct {
	URL    string // OpenAI-compatible base URL, e.g. http://localhost:11434/v1
	Key    string // bearer token (optional for local Ollama)
	Model  string // vision model id
	Prompt string // optional prompt override
	Lang   string // optional language hint appended to the prompt
}

// chatRequest is the minimal OpenAI-compatible chat-completions request with a vision part.
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	Stream      bool          `json:"stream"`
}

type chatMessage struct {
	Role    string        `json:"role"`
	Content []contentPart `json:"content"`
}

type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			// Content is a plain string for most models, but some return an array of
			// content parts ([{type,text},...]); RawMessage lets us handle both.
			Content json.RawMessage `json:"content"`
			// Reasoning models sometimes leave Content empty and put the answer here.
			Reasoning string `json:"reasoning"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// extractText pulls the transcription out of a chat message, tolerating content delivered
// as a plain string, as an array of {type,text} parts, or (for reasoning models) only in
// the reasoning field.
func extractText(content json.RawMessage, reasoning string) string {
	if len(content) > 0 {
		var s string
		if json.Unmarshal(content, &s) == nil {
			if t := strings.TrimSpace(s); t != "" {
				return t
			}
		} else {
			var parts []struct {
				Text string `json:"text"`
			}
			if json.Unmarshal(content, &parts) == nil {
				var b strings.Builder
				for _, p := range parts {
					b.WriteString(p.Text)
				}
				if t := strings.TrimSpace(b.String()); t != "" {
					return t
				}
			}
		}
	}
	return strings.TrimSpace(reasoning)
}

// SendRequest renders the iink payload, runs vision OCR, and returns a JIIX document so the
// response is indistinguishable (to the tablet) from the MyScript backend.
func (l *LLMClient) SendRequest(data []byte) ([]byte, error) {
	if l.URL == "" || l.Model == "" {
		return nil, fmt.Errorf("llm hwr not configured: a base URL and model are required")
	}

	var batch iinkBatch
	if err := json.Unmarshal(data, &batch); err != nil {
		return nil, fmt.Errorf("parse iink batch: %w", err)
	}

	pngBytes, err := renderStrokes(&batch)
	if err != nil {
		return nil, fmt.Errorf("render strokes: %w", err)
	}

	text, err := l.transcribe(pngBytes)
	if err != nil {
		return nil, err
	}

	var bbox *boundingBox
	if bb, ok := batch.boundingBoxMM(); ok {
		bbox = &bb
	}
	return buildJIIX(text, batch.ContentType, bbox)
}

// prompt returns the transcription instruction, applying any configured overrides.
func (l *LLMClient) prompt() string {
	p := defaultPrompt
	if l.Prompt != "" {
		p = l.Prompt
	}
	if l.Lang != "" {
		p += " The handwriting is in language: " + l.Lang + "."
	}
	return p
}

// transcribe sends the rendered image to the vision model and returns the recognized text.
func (l *LLMClient) transcribe(pngBytes []byte) (string, error) {
	dataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	reqBody := chatRequest{
		Model: l.Model,
		Messages: []chatMessage{{
			Role: "user",
			Content: []contentPart{
				{Type: "text", Text: l.prompt()},
				{Type: "image_url", ImageURL: &imageURL{URL: dataURI}},
			},
		}},
		Temperature: 0,
		Stream:      false,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	endpoint := strings.TrimRight(l.URL, "/") + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if l.Key != "" {
		req.Header.Set("Authorization", "Bearer "+l.Key)
	}

	client := http.Client{Timeout: llmTimeout}
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("vision request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("vision endpoint status %d: %s", res.StatusCode, string(body))
	}

	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("parse vision response: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("vision endpoint error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("vision endpoint returned no choices")
	}

	text := extractText(parsed.Choices[0].Message.Content, parsed.Choices[0].Message.Reasoning)
	if text == "" {
		return "", fmt.Errorf("vision endpoint returned empty text (the model may not support image input); raw response: %s", truncate(string(body), 800))
	}
	return text, nil
}

// truncate caps a string for safe inclusion in error/log messages.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}

// JIIX response shapes. The reMarkable "Convert to text" sends contentType "Raw Content"
// and expects a "Raw Content" document whose elements carry the recognized text. Older
// "Text" requests expect a flat Text document. We mirror whichever the device asked for.
//
// The device's export config requests words but not bounding-box, so word entries are bare
// {"label": "..."} with whitespace separators; we omit bounding-box/candidates/id/version,
// matching what MyScript returns for this request.
type jiixRawContent struct {
	Type     string        `json:"type"`
	Elements []jiixElement `json:"elements"`
}

type jiixElement struct {
	Type  string     `json:"type"`
	Label string     `json:"label"`
	Words []jiixWord `json:"words"`
}

type jiixText struct {
	Type  string     `json:"type"`
	Label string     `json:"label"`
	Words []jiixWord `json:"words"`
}

type jiixWord struct {
	Label string `json:"label"`
	// BoundingBox positions the word on the page (in mm). Real words carry it so the device
	// places each conversion where its handwriting was; whitespace separators omit it.
	BoundingBox *boundingBox `json:"bounding-box,omitempty"`
}

type boundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// buildJIIX wraps recognized text in the JIIX shape the device asked for, keyed off the
// iink request's contentType ("Raw Content" for the tablet's Convert to text). Every real
// word shares bbox (the stroke extent), which the device uses to position the text so
// successive conversions stack instead of overwriting each other.
func buildJIIX(text, contentType string, bbox *boundingBox) ([]byte, error) {
	words := tokenizeWords(text, bbox)
	if contentType == "Text" {
		return json.Marshal(jiixText{Type: "Text", Label: text, Words: words})
	}
	return json.Marshal(jiixRawContent{
		Type:     "Raw Content",
		Elements: []jiixElement{{Type: "Text", Label: text, Words: words}},
	})
}

// tokenizeWords splits text into JIIX word entries: maximal non-whitespace runs are words
// (each carrying bbox), and each whitespace character is its own separator entry (no bbox),
// so concatenating every label reproduces the original text exactly.
func tokenizeWords(text string, bbox *boundingBox) []jiixWord {
	words := []jiixWord{}
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			words = append(words, jiixWord{Label: cur.String(), BoundingBox: bbox})
			cur.Reset()
		}
	}
	for _, r := range text {
		if unicode.IsSpace(r) {
			flush()
			words = append(words, jiixWord{Label: string(r)})
		} else {
			cur.WriteRune(r)
		}
	}
	flush()
	return words
}
