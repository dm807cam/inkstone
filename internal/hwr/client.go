package hwr

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ddvk/rmfakecloud/internal/config"
)

const (
	defaultHost = "https://cloud.myscript.com"
	apiPath     = "/api/v4.0/iink/batch"

	// JIIX jiix type
	JIIX = "application/vnd.myscript.jiix"
)

// Recognizer turns the tablet's iink-batch handwriting payload into a JIIX response.
// The reMarkable "Convert to text" button POSTs MyScript iink-batch JSON and renders the
// "label" field of the returned JIIX, so every backend must speak that same contract.
type Recognizer interface {
	SendRequest(data []byte) (body []byte, err error)
}

// NewRecognizer selects the handwriting recognition backend from configuration.
// "llm" routes to a self-hosted vision model; anything else uses MyScript (the default).
func NewRecognizer(cfg *config.Config) Recognizer {
	if cfg != nil && cfg.HWRProvider == "llm" {
		return &LLMClient{
			URL:    cfg.HWRLLMURL,
			Key:    cfg.HWRLLMKey,
			Model:  cfg.HWRLLMModel,
			Prompt: cfg.HWRLLMPrompt,
			Lang:   cfg.HWRLangOverride,
		}
	}
	return &HWRClient{Cfg: cfg}
}

// HWRClient forwards the iink-batch payload to the MyScript cloud verbatim.
type HWRClient struct {
	Cfg *config.Config
}

func DoLangOverride(originalData []byte, overrideLang string) ([]byte, error) {
	var jsonData map[string]interface{}
	if err := json.Unmarshal(originalData, &jsonData); err != nil {
		return nil, fmt.Errorf("failed to parse json: %w", err)
	}

	config, ok := jsonData["configuration"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("configuration schema missing in json")
	}

	config["lang"] = overrideLang

	modifiedData, err := json.Marshal(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate modified json: %w", err)
	}

	return modifiedData, nil
}

// SendRequest sends the request
func (hwr *HWRClient) SendRequest(data []byte) (body []byte, err error) {
	if hwr.Cfg.HWRLangOverride != "" {
		overrideLang := hwr.Cfg.HWRLangOverride
		modifiedData, err := DoLangOverride(data, overrideLang)
		if err != nil {
			return nil, fmt.Errorf("failed to override language: %w", err)
		}
		data = modifiedData
	}

	if hwr.Cfg == nil || hwr.Cfg.HWRApplicationKey == "" || hwr.Cfg.HWRHmac == "" {
		return nil, fmt.Errorf("no hwr key set")
	}
	appKey := hwr.Cfg.HWRApplicationKey
	fullkey := appKey + hwr.Cfg.HWRHmac
	mac := hmac.New(sha512.New, []byte(fullkey))
	mac.Write(data)
	result := hex.EncodeToString(mac.Sum(nil))

	// Bound the request so a slow or unreachable MyScript host cannot block
	// the calling goroutine indefinitely (matches the ICS integration's 30s).
	client := http.Client{Timeout: 30 * time.Second}

	host := defaultHost
	if hwr.Cfg.HWRHost != "" {
		host = hwr.Cfg.HWRHost
	}

	req, err := http.NewRequest("POST", host+apiPath, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", JIIX)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("applicationKey", appKey)
	req.Header.Set("hmac", result)

	res, err := client.Do(req)

	if err != nil {
		return
	}
	defer res.Body.Close()
	body, err = io.ReadAll(res.Body)
	if err != nil {
		return
	}

	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("not ok, Status: %d", res.StatusCode)
		return
	}

	return body, nil
}
