package asr

import (
	"reflect"
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
	if got := marks(text, words); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := marks("好的。", []word{{Text: "这句话里没有的一个很长很长的词"}, {Text: "的", Offset: 40}}); !reflect.DeepEqual(got, []Mark{{1, 40}}) {
		t.Errorf("got %v", got)
	}
}
