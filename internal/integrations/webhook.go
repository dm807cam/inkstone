package integrations

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/ddvk/rmfakecloud/internal/messages"
	"github.com/ddvk/rmfakecloud/internal/model"
)

// webhookTimeout bounds the outbound webhook request so a slow or hung
// endpoint cannot block the calling goroutine indefinitely. Mirrors the
// 30s convention used by the ICS integration (see ics.go).
const webhookTimeout = 30 * time.Second

type Webhook struct {
	Endpoint string
}

func newWebhook(i model.IntegrationConfig) *Webhook {
	return &Webhook{
		Endpoint: i.Endpoint,
	}
}

func (i *Webhook) SendMessage(data messages.IntegrationMessageData, img image.Image) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add data field
	if mdata, err := json.Marshal(data); err != nil {
		return "", err
	} else if err := writer.WriteField("data", string(mdata)); err != nil {
		return "", err
	}

	// Add attachment
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}

	part, err := writer.CreateFormFile("attachment", "reMarkable.png")
	if err != nil {
		return "", err
	}
	if _, err := part.Write(buf.Bytes()); err != nil {
		return "", err
	}

	// Close
	if err := writer.Close(); err != nil {
		return "", err
	}

	// Do the request
	client := &http.Client{Timeout: webhookTimeout}
	resp, err := client.Post(i.Endpoint, writer.FormDataContentType(), body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(responseData), nil
}
