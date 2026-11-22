package store

import (
	"reflect"
	"testing"
)

func TestMoved(t *testing.T) {
	every := func(text string) []Mark {
		var marks []Mark
		at := 0
		for _, r := range text {
			marks = append(marks, Mark{At: at, Ms: int64(at) * 100})
			at++
			if r > 0xffff {
				at++
			}
		}
		return marks
	}
	for _, c := range []struct {
		text, polished string
		want           []Mark
	}{
		{"嗯结果要把写回看板", "结果要写回看板。", []Mark{{0, 100}, {1, 200}, {2, 300}, {3, 500}, {4, 600}, {5, 700}, {6, 800}}},
		{"上线Mutekip版本", "上线 MeetKeep 版本", []Mark{{0, 0}, {1, 100}, {3, 200}, {6, 400}, {8, 500}, {10, 800}, {12, 900}, {13, 1000}}},
		{"好的好的", "好的", []Mark{{0, 0}, {1, 100}}},
		{"😀好", "好", []Mark{{0, 200}}},
		{"完全不同", "另一句话", []Mark{}},
		{"一样的话", "一样的话", every("一样的话")},
	} {
		if got := Moved(every(c.text), c.text, c.polished); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q to %q: got %v, want %v", c.text, c.polished, got, c.want)
		}
	}
	if got := Moved(nil, "没有记下时间", "没记下时间"); got != nil {
		t.Errorf("got %v", got)
	}
}
