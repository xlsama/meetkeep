// Package asr turns speech into text with a cloud service.
package asr

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/xlsama/meetkeep/internal/audio"
	"github.com/xlsama/meetkeep/internal/config"
	"github.com/xlsama/meetkeep/modules/speech"
)

// Transcriber turns speech into text.
type Transcriber interface {
	// Transcribe returns what is said in 16 kHz mono speech.
	Transcribe(ctx context.Context, pcm []int16) (string, error)
	// TranscribeSound returns the sentences of a sound with when they are
	// said.
	TranscribeSound(ctx context.Context, sound speech.Sound) ([]Phrase, error)
}

// Phrase is a sentence and when it is said.
type Phrase struct {
	StartMs int64
	EndMs   int64
	Text    string
	// Marks tell when places in Text are said, in order.
	Marks []Mark
}

// Mark is a place in a text, counted in UTF-16 code units as a web view
// counts them, and when what stands there is said.
type Mark struct {
	At int
	Ms int64
}

// ErrNotConfigured means what the service needs to be reached is missing.
var ErrNotConfigured = errors.New("转写服务还没有配置")

// From returns what transcribes with the settings. hearing is the file of
// pyannote segmentation 3.0, which tells where the speech is in a sound that
// goes to a service that times nothing in pieces.
func From(s config.ASRSettings, hearing string) (Transcriber, error) {
	switch s.Provider {
	case config.ASRDeepSeek:
		return NewDeepSeek(s.DeepSeek.Key, hearing)
	case config.ASRCustom:
		chosen := config.Chosen(s.Services, s.Service)
		return NewOpenAI(chosen.BaseURL, chosen.Key, chosen.Model, s.Hotwords, hearing)
	}
	return NewMAI(s.Endpoint, s.Key, s.Hotwords)
}

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
		Client:   client(),
	}, nil
}

// client returns what sends sound to a service, each request on a
// connection of its own. Requests that share one, as HTTP/2 has them do,
// share what that connection carries, and a recording sent in parts then
// takes as long as one sent whole.
func client() *http.Client {
	return &http.Client{Transport: &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		TLSNextProto:        map[string]func(string, *tls.Conn) http.RoundTripper{},
		MaxIdleConnsPerHost: atOnce,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}}
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
		Words    []word `json:"words"`
	} `json:"phrases"`
}

// word is what the service times: a letter of Chinese, or a word or a
// number of other writing.
type word struct {
	Text     string `json:"text"`
	Offset   int64  `json:"offsetMilliseconds"`
	Duration int64  `json:"durationMilliseconds"`
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

const (
	// The service answers in a time that grows with the sound it is sent,
	// so a recording goes to it in parts of about partMs, atOnce of them at
	// a time: an hour is heard in the time a part takes.
	partMs = 4 * 60 * 1000
	atOnce = 32
	// A part ends at the quietest pause of pauseMs within swingMs of where
	// it would end, so that no word is cut.
	swingMs = 20 * 1000
	pauseMs = 400
)

// parts returns where a sound that lasts total is cut into parts, from its
// start to its end.
func parts(sound speech.Sound, total int) ([]int, error) {
	count := max(1, (total+partMs/2)/partMs)
	cuts := make([]int, 0, count+1)
	cuts = append(cuts, 0)
	for i := 1; i < count; i++ {
		from := max(total*i/count-swingMs, 0)
		pcm, err := sound.Read(from, total*i/count+swingMs)
		if err != nil {
			return nil, err
		}
		cuts = append(cuts, from+audio.Quietest(audio.Samples(pcm), pauseMs)+pauseMs/2)
	}
	return append(cuts, total), nil
}

func (m *MAI) TranscribeSound(ctx context.Context, sound speech.Sound) ([]Phrase, error) {
	total, err := sound.Duration()
	if err != nil || total == 0 {
		return nil, err
	}
	cuts, err := parts(sound, total)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	said := make([][]Phrase, len(cuts)-1)
	slots := make(chan struct{}, atOnce)
	var wait sync.WaitGroup
	for i := range said {
		wait.Go(func() {
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
			case <-ctx.Done():
				return
			}
			phrases, err := m.part(ctx, sound, cuts[i], cuts[i+1])
			if err != nil {
				cancel(err)
				return
			}
			said[i] = phrases
		})
	}
	wait.Wait()
	if ctx.Err() != nil {
		return nil, context.Cause(ctx)
	}
	return slices.Concat(said...), nil
}

// part returns the sentences of a stretch of a sound, said when they are in
// the sound. A request that fails is made once more.
func (m *MAI) part(ctx context.Context, sound speech.Sound, startMs, endMs int) ([]Phrase, error) {
	clip, err := sound.Clip(startMs, endMs)
	if err != nil {
		return nil, err
	}
	result, err := m.send(ctx, "audio.opus", clip, true)
	if err != nil && ctx.Err() == nil {
		time.Sleep(time.Second)
		result, err = m.send(ctx, "audio.opus", clip, true)
	}
	if err != nil {
		return nil, err
	}
	phrases := make([]Phrase, 0, len(result.Phrases))
	for _, p := range result.Phrases {
		text := strings.Join(strings.Fields(p.Text), " ")
		if text == "" {
			continue
		}
		for i := range p.Words {
			p.Words[i].Offset += int64(startMs)
		}
		p.Offset += int64(startMs)
		found, ends := marks(text, p.Words)
		phrase := Phrase{StartMs: p.Offset, EndMs: p.Offset + p.Duration, Text: text, Marks: found}
		if n := len(p.Words); n > 0 {
			phrase.StartMs = p.Words[0].Offset
			phrase.EndMs = p.Words[n-1].Offset + p.Words[n-1].Duration
		}
		phrases = append(phrases, sentences(phrase, ends)...)
	}
	return phrases, nil
}

// reach is how many letters past the last word found the next one may
// stand: a word the text spells otherwise is passed over.
const reach = 12

// marks returns where in text each of words starts and when it is said,
// and when each of them ends.
func marks(text string, words []word) (found []Mark, ends []int64) {
	found = make([]Mark, 0, len(words))
	at, units := 0, 0
	for _, w := range words {
		said := strings.Join(strings.Fields(w.Text), " ")
		if said == "" {
			continue
		}
		skip := strings.Index(text[at:], said)
		if skip < 0 || utf8.RuneCountInString(text[at:at+skip]) > reach {
			continue
		}
		units += len(utf16.Encode([]rune(text[at : at+skip])))
		found = append(found, Mark{At: units, Ms: w.Offset})
		ends = append(ends, w.Offset+w.Duration)
		units += len(utf16.Encode([]rune(said)))
		at += skip + len(said)
	}
	return found, ends
}

const (
	// least is how many letters a sentence holds at least: a shorter one
	// joins the next.
	least = 16
	// most is how many letters a sentence holds before it is cut at a comma.
	most = 120
)

func among(unit uint16, set string) bool {
	return strings.ContainsRune(set, rune(unit))
}

// stops returns where the sentences of a text end, the last at its end.
func stops(units []uint16) []int {
	var found []int
	for i := 0; i < len(units); i++ {
		full := among(units[i], "。！？!?…；;")
		if units[i] == '.' && i > 0 && among(units[i-1], "abcdefghijklmnopqrstuvwxyz0123456789)\"") {
			full = i+1 == len(units) || units[i+1] == ' '
		}
		if !full {
			continue
		}
		for i+1 < len(units) && among(units[i+1], "。！？!?…；;”’」』）)\"") {
			i++
		}
		found = append(found, i+1)
	}
	if n := len(found); n == 0 || found[n-1] != len(units) {
		found = append(found, len(units))
	}
	return found
}

// halved returns where a sentence too long is cut, at the commas nearest
// to the middle of its parts, with its own end.
func halved(units []uint16, from, to int) []int {
	if to-from <= most {
		return []int{to}
	}
	middle, cut := (from+to)/2, -1
	for i := from; i < to-1; i++ {
		if among(units[i], "，、：,:") && (cut < 0 || max(i+1-middle, middle-i-1) < max(cut-middle, middle-cut)) {
			cut = i + 1
		}
	}
	if cut < 0 {
		return []int{to}
	}
	return append(halved(units, from, cut), halved(units, cut, to)...)
}

// cuts returns where a text is cut into sentences, the last at its end. A
// sentence shorter than least, or one timed tells nothing of, joins the next.
func cuts(units []uint16, timed func(from, to int) bool) []int {
	var parts []int
	from := 0
	for _, stop := range stops(units) {
		parts = append(parts, halved(units, from, stop)...)
		from = stop
	}
	var found []int
	from = 0
	for _, part := range parts[:len(parts)-1] {
		if part-from >= least && timed(from, part) {
			found = append(found, part)
			from = part
		}
	}
	if len(found) > 0 && (len(units)-from < least || !timed(from, len(units))) {
		found = found[:len(found)-1]
	}
	return append(found, len(units))
}

// sentences cuts a phrase into its sentences, each with when it is said:
// the service gives what is said without a pause as one phrase, however
// long. ends tell when what stands at each of the phrase's marks ends.
func sentences(p Phrase, ends []int64) []Phrase {
	if len(p.Marks) == 0 {
		return []Phrase{p}
	}
	units := utf16.Encode([]rune(p.Text))
	marked := func(from, to int) (first, last int) {
		first, last = -1, -1
		for i, m := range p.Marks {
			if m.At >= from && m.At < to {
				if first < 0 {
					first = i
				}
				last = i
			}
		}
		return first, last
	}
	cuts := cuts(units, func(from, to int) bool {
		first, _ := marked(from, to)
		return first >= 0
	})

	cut := make([]Phrase, 0, len(cuts))
	from := 0
	for _, to := range cuts {
		first, last := marked(from, to)
		for from < to && units[from] == ' ' {
			from++
		}
		sentence := Phrase{
			StartMs: p.Marks[first].Ms, EndMs: max(ends[last], p.Marks[first].Ms),
			Text: strings.TrimSpace(string(utf16.Decode(units[from:to]))),
		}
		for _, m := range p.Marks[first : last+1] {
			sentence.Marks = append(sentence.Marks, Mark{At: max(m.At-from, 0), Ms: m.Ms})
		}
		cut = append(cut, sentence)
		from = to
	}
	cut[0].StartMs, cut[len(cut)-1].EndMs = p.StartMs, p.EndMs
	return cut
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
