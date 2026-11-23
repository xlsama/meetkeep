package asr

import (
	"reflect"
	"strings"
	"testing"
)

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
