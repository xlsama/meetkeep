// Package asr turns speech into text with a cloud service.
package asr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/xlsama/meetkeep/internal/audio"
)

// Transcriber turns 16 kHz mono speech into text.
type Transcriber interface {
	Transcribe(ctx context.Context, pcm []int16) (string, error)
}

// ErrNotConfigured means the service's address or key is missing.
var ErrNotConfigured = errors.New("转写服务还没有配置")

// MAI is Azure's MAI-Transcribe-2, through the fast transcription API.
type MAI struct {
	Endpoint string
	Key      string
	Hotwords []string
	Client   *http.Client
}

func NewMAI(endpoint, key string, hotwords []string) (*MAI, error) {
	if endpoint == "" || key == "" {
		return nil, ErrNotConfigured
	}
	return &MAI{
		Endpoint: strings.TrimRight(endpoint, "/"),
		Key:      key,
		Hotwords: hotwords,
		Client:   &http.Client{Timeout: 60 * time.Second},
	}, nil
}

type definition struct {
	EnhancedMode struct {
		Enabled bool   `json:"enabled"`
		Model   string `json:"model"`
	} `json:"enhancedMode"`
	PhraseList *phraseList `json:"phraseList,omitempty"`
}

type phraseList struct {
	Phrases []string `json:"phrases"`
}

func (m *MAI) Transcribe(ctx context.Context, pcm []int16) (string, error) {
	var def definition
	def.EnhancedMode.Enabled, def.EnhancedMode.Model = true, "MAI-Transcribe-2"
	if len(m.Hotwords) > 0 {
		def.PhraseList = &phraseList{Phrases: m.Hotwords}
	}
	spec, err := json.Marshal(def)
	if err != nil {
		return "", err
	}

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("audio", "audio.wav")
	if err != nil {
		return "", err
	}
	if _, err := file.Write(audio.EncodeWAV(pcm)); err != nil {
		return "", err
	}
	if err := form.WriteField("definition", string(spec)); err != nil {
		return "", err
	}
	if err := form.Close(); err != nil {
		return "", err
	}

	url := m.Endpoint + "/speechtotext/transcriptions:transcribe?api-version=2025-10-15"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("Ocp-Apim-Subscription-Key", m.Key)

	res, err := m.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("MAI-Transcribe-2 返回 %s：%s", res.Status, snippet(data))
	}
	var result struct {
		CombinedPhrases []struct {
			Text string `json:"text"`
		} `json:"combinedPhrases"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	var parts []string
	for _, p := range result.CombinedPhrases {
		parts = append(parts, strings.Fields(p.Text)...)
	}
	return strings.Join(parts, " "), nil
}

func snippet(data []byte) string {
	s := strings.TrimSpace(string(data))
	if r := []rune(s); len(r) > 160 {
		return string(r[:160])
	}
	return s
}
