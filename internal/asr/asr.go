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
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xlsama/meetkeep/internal/audio"
)

// Transcriber turns speech into text.
type Transcriber interface {
	// Transcribe returns what is said in 16 kHz mono speech.
	Transcribe(ctx context.Context, pcm []int16) (string, error)
	// TranscribeFile returns the sentences of an audio file with when they
	// are said.
	TranscribeFile(ctx context.Context, path string) ([]Phrase, error)
}

// Phrase is a sentence and when it is said.
type Phrase struct {
	StartMs int64
	EndMs   int64
	Text    string
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
		Client:   &http.Client{},
	}, nil
}

type definition struct {
	EnhancedMode struct {
		Enabled      bool          `json:"enabled"`
		Model        string        `json:"model"`
		ModelOptions *modelOptions `json:"modelOptions,omitempty"`
	} `json:"enhancedMode"`
	PhraseList *phraseList `json:"phraseList,omitempty"`
}

type modelOptions struct {
	Timestamps string `json:"timestamps"`
}

type result struct {
	CombinedPhrases []struct {
		Text string `json:"text"`
	} `json:"combinedPhrases"`
	Phrases []struct {
		Offset   int64  `json:"offsetMilliseconds"`
		Duration int64  `json:"durationMilliseconds"`
		Text     string `json:"text"`
		Words    []struct {
			Offset   int64 `json:"offsetMilliseconds"`
			Duration int64 `json:"durationMilliseconds"`
		} `json:"words"`
	} `json:"phrases"`
}

type phraseList struct {
	Phrases []string `json:"phrases"`
}

func (m *MAI) Transcribe(ctx context.Context, pcm []int16) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	result, err := m.send(ctx, "audio.wav", audio.EncodeWAV(pcm), false)
	if err != nil {
		return "", err
	}
	var parts []string
	for _, p := range result.CombinedPhrases {
		parts = append(parts, strings.Fields(p.Text)...)
	}
	return strings.Join(parts, " "), nil
}

func (m *MAI) TranscribeFile(ctx context.Context, path string) ([]Phrase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	result, err := m.send(ctx, filepath.Base(path), data, true)
	if err != nil {
		return nil, err
	}
	phrases := make([]Phrase, 0, len(result.Phrases))
	for _, p := range result.Phrases {
		text := strings.Join(strings.Fields(p.Text), " ")
		if text == "" {
			continue
		}
		phrase := Phrase{StartMs: p.Offset, EndMs: p.Offset + p.Duration, Text: text}
		if n := len(p.Words); n > 0 {
			phrase.StartMs = p.Words[0].Offset
			phrase.EndMs = p.Words[n-1].Offset + p.Words[n-1].Duration
		}
		phrases = append(phrases, phrase)
	}
	return phrases, nil
}

func (m *MAI) send(ctx context.Context, name string, data []byte, timestamps bool) (result, error) {
	var def definition
	def.EnhancedMode.Enabled, def.EnhancedMode.Model = true, "MAI-Transcribe-2"
	if timestamps {
		def.EnhancedMode.ModelOptions = &modelOptions{Timestamps: "word"}
	}
	if len(m.Hotwords) > 0 {
		def.PhraseList = &phraseList{Phrases: m.Hotwords}
	}
	spec, err := json.Marshal(def)
	if err != nil {
		return result{}, err
	}

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("audio", name)
	if err != nil {
		return result{}, err
	}
	if _, err := file.Write(data); err != nil {
		return result{}, err
	}
	if err := form.WriteField("definition", string(spec)); err != nil {
		return result{}, err
	}
	if err := form.Close(); err != nil {
		return result{}, err
	}

	url := m.Endpoint + "/speechtotext/transcriptions:transcribe?api-version=2025-10-15"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return result{}, err
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("Ocp-Apim-Subscription-Key", m.Key)

	res, err := m.Client.Do(req)
	if err != nil {
		return result{}, err
	}
	defer res.Body.Close()
	answer, err := io.ReadAll(io.LimitReader(res.Body, 256<<20))
	if err != nil {
		return result{}, err
	}
	if res.StatusCode != http.StatusOK {
		return result{}, fmt.Errorf("MAI-Transcribe-2 返回 %s：%s", res.Status, snippet(answer))
	}
	var parsed result
	return parsed, json.Unmarshal(answer, &parsed)
}

func snippet(data []byte) string {
	s := strings.TrimSpace(string(data))
	if r := []rune(s); len(r) > 160 {
		return string(r[:160])
	}
	return s
}
