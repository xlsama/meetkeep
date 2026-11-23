package asr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	"github.com/xlsama/meetkeep/internal/audio"
	"github.com/xlsama/meetkeep/modules/speech"
)

// OpenAI is a service that speaks OpenAI's audio transcriptions. It is asked
// for text alone, which every such service gives: a file is cut here into
// utterances, at the pauses and where another speaker starts to talk, and
// each is said where it was cut.
type OpenAI struct {
	BaseURL  string
	Key      string
	Model    string
	Hotwords []string
	Files    Files
	Client   *http.Client
}

// Files reads audio files and tells where the speech in them is.
type Files interface {
	Hear(ctx context.Context, path string) (speech.Hearing, error)
	Listen(path string) (speech.Audio, error)
}

// Recordings reads files with the engine, which hears them with Model:
// pyannote segmentation 3.0.
type Recordings struct {
	Engine speech.Engine
	Model  string
}

func (r Recordings) Hear(ctx context.Context, path string) (speech.Hearing, error) {
	return r.Engine.Hear(ctx, path, r.Model)
}

func (r Recordings) Listen(path string) (speech.Audio, error) { return r.Engine.Listen(path) }

const (
	// parallel is how many utterances of a file are with a service of the
	// user's own at once, and busy how many are with DeepSeek, which takes
	// 50 of an account.
	parallel = 8
	busy     = 16
	// tail is how much silence follows an utterance sent to the service. One
	// that hears speech as a stream waits for more where an utterance was
	// cut, and leaves its last words out: the silence tells it the end.
	tail = audio.SampleRate * 3 / 10
)

func NewOpenAI(baseURL, key, model string, hotwords []string, files Files) (*OpenAI, error) {
	if baseURL == "" || model == "" {
		return nil, ErrNotConfigured
	}
	return &OpenAI{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		Key:      key,
		Model:    model,
		Hotwords: hotwords,
		Files:    files,
		Client:   &http.Client{},
	}, nil
}

func (o *OpenAI) Transcribe(ctx context.Context, pcm []int16) (string, error) {
	fields := map[string]string{"model": o.Model, "response_format": "json"}
	if len(o.Hotwords) > 0 {
		fields["prompt"] = strings.Join(o.Hotwords, "、")
	}
	text, err := ask(ctx, o.Client, o.BaseURL+"/audio/transcriptions", o.Key, fields, pcm)
	// Confucius ends what was cut off in the middle of a sentence with a bar.
	return strings.TrimSpace(strings.TrimRight(text, "|")), err
}

func (o *OpenAI) TranscribeFile(ctx context.Context, path string) ([]Phrase, error) {
	return pieces(ctx, o.Files, path, parallel, o.Transcribe)
}

// DeepSeek is DeepSeek's speech recognition. It gives text alone, of two
// minutes at most, and takes no words to listen for: a file goes to it in
// the pieces OpenAI's services get.
type DeepSeek struct {
	URL    string
	Key    string
	Files  Files
	Client *http.Client
}

func NewDeepSeek(key string, files Files) (*DeepSeek, error) {
	if key == "" {
		return nil, ErrNotConfigured
	}
	return &DeepSeek{URL: "https://api.deepseek.com/v1/tools/asr", Key: key, Files: files, Client: &http.Client{}}, nil
}

func (d *DeepSeek) Transcribe(ctx context.Context, pcm []int16) (string, error) {
	return ask(ctx, d.Client, d.URL, d.Key, nil, pcm)
}

func (d *DeepSeek) TranscribeFile(ctx context.Context, path string) ([]Phrase, error) {
	return pieces(ctx, d.Files, path, busy, d.Transcribe)
}

// ask sends an utterance to url as a WAV file, with the fields, and returns
// the text of the answer.
func ask(ctx context.Context, client *http.Client, url, key string, fields map[string]string, pcm []int16) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", err
	}
	if _, err := file.Write(audio.EncodeWAV(append(slices.Clip(pcm), make([]int16, tail)...))); err != nil {
		return "", err
	}
	for name, value := range fields {
		if err := form.WriteField(name, value); err != nil {
			return "", err
		}
	}
	if err := form.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	answer, err := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("转写服务返回 %s：%s", res.Status, snippet(answer))
	}
	var parsed struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(answer, &parsed); err != nil {
		return "", fmt.Errorf("转写服务的返回不是 JSON：%s", snippet(answer))
	}
	return strings.Join(strings.Fields(parsed.Text), " "), nil
}

// pieces cuts the file at path into utterances, at the pauses and where
// another speaker starts to talk, has transcribe hear each, several at
// once, and returns their sentences, said where the utterances were cut.
func pieces(ctx context.Context, files Files, path string, several int, transcribe func(context.Context, []int16) (string, error)) ([]Phrase, error) {
	heard, err := files.Hear(ctx, path)
	if err != nil {
		return nil, err
	}
	sound, err := files.Listen(path)
	if err != nil {
		return nil, err
	}
	defer sound.Close()

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	var (
		segmenter audio.Segmenter
		pending   []int16
		window    int
		turns     = heard.Turns
		said      []*[]Phrase
		wait      sync.WaitGroup
		slots     = make(chan struct{}, several)
	)
	send := func(events []audio.Event) {
		for _, e := range events {
			if e.Kind != audio.Ended {
				continue
			}
			select {
			case slots <- struct{}{}:
			case <-ctx.Done():
				return
			}
			into := new([]Phrase)
			said = append(said, into)
			wait.Add(1)
			go func() {
				defer wait.Done()
				defer func() { <-slots }()
				text, err := transcribe(ctx, e.PCM)
				if err != nil && ctx.Err() == nil {
					time.Sleep(time.Second)
					text, err = transcribe(ctx, e.PCM)
				}
				if err != nil {
					cancel(err)
					return
				}
				*into = spread(Phrase{StartMs: e.StartMs, EndMs: e.EndMs, Text: text})
			}()
		}
	}
	for ctx.Err() == nil {
		pcm, err := sound.Read()
		if err != nil {
			cancel(err)
			break
		}
		if len(pcm) == 0 {
			send(segmenter.Finish())
			break
		}
		pending = append(pending, audio.Samples(pcm)...)
		for ; len(pending) >= audio.Window && window < len(heard.Speech); window++ {
			if len(turns) > 0 && turns[0] <= window {
				turns = turns[1:]
				send(segmenter.Turn())
			}
			send(segmenter.Push(slices.Clone(pending[:audio.Window]), heard.Speech[window]))
			pending = pending[audio.Window:]
		}
	}
	wait.Wait()
	if ctx.Err() != nil {
		return nil, context.Cause(ctx)
	}
	var phrases []Phrase
	for _, sentences := range said {
		phrases = append(phrases, *sentences...)
	}
	return phrases, nil
}

// spread cuts an utterance into its sentences, each said for its share of
// the utterance's time by its share of the letters.
func spread(p Phrase) []Phrase {
	if p.Text == "" {
		return nil
	}
	units := utf16.Encode([]rune(p.Text))
	span, all := p.EndMs-p.StartMs, int64(len(units))
	var cut []Phrase
	from := 0
	for _, to := range cuts(units, func(int, int) bool { return true }) {
		cut = append(cut, Phrase{
			StartMs: p.StartMs + span*int64(from)/all,
			EndMs:   p.StartMs + span*int64(to)/all,
			Text:    strings.TrimSpace(string(utf16.Decode(units[from:to]))),
		})
		from = to
	}
	return cut
}
