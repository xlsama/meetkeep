package asr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/xlsama/meetkeep/internal/audio"
)

func TestMAIHearsARecordingInPartsAtOnce(t *testing.T) {
	var mu sync.Mutex
	asked := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, _, err := r.FormFile("audio")
		if err != nil || r.Header.Get("Ocp-Apim-Subscription-Key") != "key" {
			t.Errorf("asked with %v, key %q", err, r.Header.Get("Ocp-Apim-Subscription-Key"))
			return
		}
		clip, _ := io.ReadAll(file)
		var def definition
		if json.Unmarshal([]byte(r.FormValue("definition")), &def) != nil || def.EnhancedMode.ModelOptions == nil {
			t.Errorf("asked for %q", r.FormValue("definition"))
		}
		mu.Lock()
		asked[string(clip)]++
		again := asked[string(clip)]
		mu.Unlock()
		if strings.HasPrefix(string(clip), "200000-") && again == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = fmt.Fprintf(w, `{"phrases":[{"offsetMilliseconds":1000,"durationMilliseconds":900,"text":"从%s说起。","words":[{"text":"从","offsetMilliseconds":1200,"durationMilliseconds":100}]}]}`, clip)
	}))
	defer server.Close()
	m, err := NewMAI(server.URL, "key", nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := m.TranscribeSound(context.Background(), fakeSound{samples: 10 * 60 * audio.SampleRate})
	if err != nil {
		t.Fatal(err)
	}
	want := []Phrase{
		{StartMs: 1200, EndMs: 1300, Text: "从0-200000说起。", Marks: []Mark{{At: 0, Ms: 1200}}},
		{StartMs: 201200, EndMs: 201300, Text: "从200000-400000说起。", Marks: []Mark{{At: 0, Ms: 201200}}},
		{StartMs: 401200, EndMs: 401300, Text: "从400000-600000说起。", Marks: []Mark{{At: 0, Ms: 401200}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if asked["200000-400000"] != 2 {
		t.Errorf("a part the service refused was asked for %d times, want 2", asked["200000-400000"])
	}

	if got, err := m.TranscribeSound(context.Background(), fakeSound{samples: 90 * audio.SampleRate}); err != nil || len(got) != 1 || got[0].Text != "从0-90000说起。" {
		t.Errorf("a short recording is heard as %+v (%v)", got, err)
	}
	server.Close()
	if _, err := m.TranscribeSound(context.Background(), fakeSound{samples: 10 * 60 * audio.SampleRate}); err == nil {
		t.Error("a service that is gone transcribed")
	}
}

func TestMarks(t *testing.T) {
	text := "结果，2026年上线 MeetKeep 1.5 😀好。"
	words := []word{
		{Text: "结", Offset: 100}, {Text: "果", Offset: 300}, {Text: "，2026", Offset: 500}, {Text: "年", Offset: 900},
		{Text: "尚", Offset: 1000}, {Text: "线", Offset: 1200}, {Text: " MeetKeep", Offset: 1400}, {Text: "1.5", Offset: 2000},
		{Text: "😀", Offset: 2400}, {Text: "好", Offset: 2600}, {Text: "", Offset: 2700}, {Text: "。", Offset: 2800},
	}
	want := []Mark{
		{0, 100}, {1, 300}, {2, 500}, {7, 900}, {9, 1200}, {11, 1400}, {20, 2000}, {24, 2400}, {26, 2600}, {27, 2800},
	}
	if got, _ := marks(text, words); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	got, ends := marks("好的。", []word{{Text: "这句话里没有的一个很长很长的词"}, {Text: "的", Offset: 40, Duration: 160}})
	if !reflect.DeepEqual(got, []Mark{{1, 40}}) || !reflect.DeepEqual(ends, []int64{200}) {
		t.Errorf("got %v, ending %v", got, ends)
	}
}

// spoken makes the phrase of a text said a letter every 100 ms from start.
func spoken(text string, start int64) (Phrase, []int64) {
	var words []word
	for i, r := range []rune(text) {
		words = append(words, word{Text: string(r), Offset: start + int64(i)*100, Duration: 80})
	}
	found, ends := marks(text, words)
	return Phrase{StartMs: start, EndMs: ends[len(ends)-1], Text: text, Marks: found}, ends
}

func TestSentences(t *testing.T) {
	first := "明天之前必须拿出第一版标书的初稿给我检查，不能再拖到最后一天。"
	second := "好。第二件事是交付工程师的培训认证，我们分成五个级别来做？"
	third := "对吧。"
	phrase, ends := spoken(first+second+third, 5000)
	got := sentences(phrase, ends)
	if len(got) != 2 || got[0].Text != first || got[1].Text != second+third {
		t.Fatalf("got %+v", got)
	}
	if got[0].StartMs != 5000 || got[0].EndMs != 5000+int64(len([]rune(first))-1)*100+80 {
		t.Errorf("the first sentence is said from %d to %d", got[0].StartMs, got[0].EndMs)
	}
	if got[1].StartMs != 5000+int64(len([]rune(first)))*100 || got[1].EndMs != phrase.EndMs {
		t.Errorf("the second sentence is said from %d to %d", got[1].StartMs, got[1].EndMs)
	}
	if m := got[1].Marks; len(m) != len([]rune(second+third)) || m[0] != (Mark{0, got[1].StartMs}) || m[1].At != 1 {
		t.Errorf("the second sentence is marked %v", m)
	}

	long := strings.Repeat("我们要把这件事情做好，", 20) + "就这样。"
	phrase, ends = spoken(long, 0)
	got = sentences(phrase, ends)
	joined := ""
	for _, s := range got {
		if n := len([]rune(s.Text)); n > most || n < least {
			t.Errorf("a sentence holds %d letters: %q", n, s.Text)
		}
		joined += s.Text
	}
	if len(got) < 2 || joined != long {
		t.Errorf("a sentence too long was cut into %d", len(got))
	}

	english := "We ship on March 14. The price page is ready by v1.5 this week."
	phrase, ends = spoken(english, 0)
	if got := sentences(phrase, ends); len(got) != 2 || got[1].Text != "The price page is ready by v1.5 this week." || got[1].Marks[0].At != 0 {
		t.Errorf("got %+v", got)
	}

	if got := sentences(Phrase{Text: "没有时间的一句。还有一句没有时间的话在这里。"}, nil); len(got) != 1 {
		t.Errorf("a phrase without marks was cut: %+v", got)
	}
}
