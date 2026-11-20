package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
)

const sampleRate = 16000

type Transcriber interface {
	Transcribe(ctx context.Context, pcm []int16) (string, error)
}

type azureMAI struct {
	endpoint string
	key      string
	client   *http.Client
}

func newAzureMAI() (*azureMAI, error) {
	endpoint, key := os.Getenv("AZURE_SPEECH_ENDPOINT"), os.Getenv("AZURE_SPEECH_KEY")
	if endpoint == "" || key == "" {
		return nil, errors.New("AZURE_SPEECH_ENDPOINT and AZURE_SPEECH_KEY are not set")
	}
	return &azureMAI{
		endpoint: strings.TrimRight(endpoint, "/"),
		key:      key,
		client:   &http.Client{Timeout: 60 * time.Second},
	}, nil
}

func (a *azureMAI) Transcribe(ctx context.Context, pcm []int16) (string, error) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	audio, err := form.CreateFormFile("audio", "audio.wav")
	if err != nil {
		return "", err
	}
	if err := writeWAV(audio, pcm); err != nil {
		return "", err
	}
	if err := form.WriteField("definition", `{"enhancedMode":{"enabled":true,"model":"MAI-Transcribe-2"}}`); err != nil {
		return "", err
	}
	if err := form.Close(); err != nil {
		return "", err
	}

	url := a.endpoint + "/speechtotext/transcriptions:transcribe?api-version=2025-10-15"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("Ocp-Apim-Subscription-Key", a.key)

	res, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("MAI-Transcribe-2: %s: %s", res.Status, truncate(string(data), 200))
	}
	var result struct {
		CombinedPhrases []struct {
			Text string `json:"text"`
		} `json:"combinedPhrases"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	parts := make([]string, 0, len(result.CombinedPhrases))
	for _, p := range result.CombinedPhrases {
		parts = append(parts, strings.Fields(p.Text)...)
	}
	return strings.Join(parts, " "), nil
}

func writeWAV(w io.Writer, pcm []int16) error {
	size := uint32(len(pcm) * 2)
	header := struct {
		RIFF       [4]byte
		Size       uint32
		WAVE       [4]byte
		Fmt        [4]byte
		FmtSize    uint32
		Format     uint16
		Channels   uint16
		Rate       uint32
		ByteRate   uint32
		BlockAlign uint16
		Bits       uint16
		Data       [4]byte
		DataSize   uint32
	}{
		RIFF: [4]byte{'R', 'I', 'F', 'F'}, Size: 36 + size,
		WAVE: [4]byte{'W', 'A', 'V', 'E'}, Fmt: [4]byte{'f', 'm', 't', ' '},
		FmtSize: 16, Format: 1, Channels: 1,
		Rate: sampleRate, ByteRate: sampleRate * 2, BlockAlign: 2, Bits: 16,
		Data: [4]byte{'d', 'a', 't', 'a'}, DataSize: size,
	}
	if err := binary.Write(w, binary.LittleEndian, header); err != nil {
		return err
	}
	return binary.Write(w, binary.LittleEndian, pcm)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
