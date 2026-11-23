package asr

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/xlsama/meetkeep/internal/audio"
	"github.com/xlsama/meetkeep/internal/config"
	"github.com/xlsama/meetkeep/modules/speech"
)

// service answers a transcription with what answer makes of the request.
func service(t *testing.T, answer func(r *http.Request, windows int) (int, string)) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/transcriptions" {
			t.Errorf("asked %s", r.URL.Path)
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			return
		}
		wav, _ := io.ReadAll(file)
		status, body := answer(r, ((len(wav)-44)/2-tail)/audio.Window)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestOpenAITranscribe(t *testing.T) {
	server := service(t, func(r *http.Request, windows int) (int, string) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Errorf("authorized with %q", got)
		}
		if r.FormValue("model") != "whisper" || r.FormValue("prompt") != "函必达、委案" || windows != 4 {
			t.Errorf("asked for %q with %q, %d windows", r.FormValue("model"), r.FormValue("prompt"), windows)
		}
		return http.StatusOK, `{"text": " 回函必达  之前先\n分案|"}`
	})
	o, err := NewOpenAI(server.URL+"/v1/", "sk-test", "whisper", []string{"函必达", "委案"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := o.Transcribe(context.Background(), make([]int16, 4*audio.Window)); err != nil || got != "回函必达 之前先 分案" {
		t.Errorf("got %q (%v)", got, err)
	}

	if _, err := NewOpenAI("", "", "whisper", nil, nil); err != ErrNotConfigured {
		t.Errorf("a service without an address is made: %v", err)
	}
	refusing := service(t, func(*http.Request, int) (int, string) { return http.StatusUnauthorized, `{"error": "Invalid API key"}` })
	o, _ = NewOpenAI(refusing.URL+"/v1", "", "whisper", nil, nil)
	if _, err := o.Transcribe(context.Background(), make([]int16, audio.Window)); err == nil || !strings.Contains(err.Error(), "Invalid API key") {
		t.Errorf("a refusal reads %v", err)
	}
}

type fakeFiles struct {
	heard   speech.Hearing
	windows int
}

func (f fakeFiles) Hear(context.Context, string) (speech.Hearing, error) { return f.heard, nil }
func (f fakeFiles) Listen(string) (speech.Audio, error) {
	return &fakeAudio{left: f.windows * audio.Window * 2}, nil
}

type fakeAudio struct{ left int }

func (a *fakeAudio) Close() error { return nil }
func (a *fakeAudio) Read() ([]byte, error) {
	n := min(a.left, 30000)
	a.left -= n
	return make([]byte, n), nil
}

func TestOpenAICutsAFileWhereItIsQuietAndWhereTheSpeakerChanges(t *testing.T) {
	var probabilities []float32
	for _, run := range []struct {
		p float32
		n int
	}{{0.02, 20}, {0.9, 120}, {0.02, 40}, {0.9, 30}, {0.02, 40}} {
		for range run.n {
			probabilities = append(probabilities, run.p)
		}
	}
	files := fakeFiles{heard: speech.Hearing{Speech: probabilities, Turns: []int{80}}, windows: len(probabilities)}
	server := service(t, func(_ *http.Request, windows int) (int, string) {
		if windows == 44 {
			return http.StatusOK, `{"text": ""}`
		}
		return http.StatusOK, fmt.Sprintf(`{"text": "这一段有%d个窗。"}`, windows)
	})
	o, _ := NewOpenAI(server.URL+"/v1", "", "whisper", nil, files)
	got, err := o.TranscribeFile(context.Background(), "meeting.opus")
	if err != nil {
		t.Fatal(err)
	}
	want := []Phrase{
		{StartMs: 12 * 32, EndMs: 80 * 32, Text: "这一段有68个窗。"},
		{StartMs: 80 * 32, EndMs: 146 * 32, Text: "这一段有66个窗。"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}

	failing := service(t, func(*http.Request, int) (int, string) { return http.StatusBadGateway, "down" })
	o, _ = NewOpenAI(failing.URL+"/v1", "", "whisper", nil, files)
	if _, err := o.TranscribeFile(context.Background(), "meeting.opus"); err == nil || !strings.Contains(err.Error(), "502") {
		t.Errorf("a service that is down reads %v", err)
	}
}

func TestDeepSeekSendsAFileAndNothingElse(t *testing.T) {
	server := service(t, func(r *http.Request, windows int) (int, string) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-ds" || len(r.MultipartForm.Value) != 0 {
			t.Errorf("authorized with %q, sent the fields %v", got, r.MultipartForm.Value)
		}
		return http.StatusOK, fmt.Sprintf(`{"request_id": "r", "text": "这一段有%d个窗。", "duration_ms": 2000}`, windows)
	})
	got, err := From(config.ASRSettings{Provider: config.ASRDeepSeek, DeepSeek: config.ASRService{Key: "sk-ds"}, Hotwords: []string{"函必达"}}, nil)
	d, ok := got.(*DeepSeek)
	if err != nil || !ok || d.URL != "https://api.deepseek.com/v1/tools/asr" {
		t.Fatalf("got %+v (%v)", got, err)
	}
	d.URL = server.URL + "/v1/audio/transcriptions"
	if text, err := d.Transcribe(context.Background(), make([]int16, 4*audio.Window)); err != nil || text != "这一段有4个窗。" {
		t.Errorf("got %q (%v)", text, err)
	}
	if _, err := From(config.ASRSettings{Provider: config.ASRDeepSeek}, nil); err != ErrNotConfigured {
		t.Errorf("DeepSeek without a key transcribes: %v", err)
	}
}

func TestFromTakesTheServiceTheSettingsChoose(t *testing.T) {
	settings := config.ASRSettings{
		Provider: config.ASRCustom,
		Service:  "b",
		Services: []config.Service{{ID: "a", BaseURL: "http://a/v1", Model: "one"}, {ID: "b", BaseURL: "http://b/v1/", Model: "two"}},
	}
	got, err := From(settings, nil)
	if o, ok := got.(*OpenAI); err != nil || !ok || o.BaseURL != "http://b/v1" || o.Model != "two" {
		t.Errorf("got %+v (%v)", got, err)
	}
	settings.Service = "gone"
	if _, err := From(settings, nil); err != ErrNotConfigured {
		t.Errorf("a service that was removed transcribes: %v", err)
	}
}

func TestSpread(t *testing.T) {
	first := "明天之前必须拿出第一版标书的初稿给我检查。"
	second := "第二件事是交付工程师的培训认证怎么做？"
	got := spread(Phrase{StartMs: 1000, EndMs: 9000, Text: first + second + "好。"})
	cut := 1000 + 8000*int64(len([]rune(first)))/int64(len([]rune(first+second+"好。")))
	want := []Phrase{{StartMs: 1000, EndMs: cut, Text: first}, {StartMs: cut, EndMs: 9000, Text: second + "好。"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if got := spread(Phrase{StartMs: 0, EndMs: 500}); got != nil {
		t.Errorf("silence is said: %+v", got)
	}
}
