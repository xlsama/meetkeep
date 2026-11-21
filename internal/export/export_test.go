package export

import (
	"strings"
	"testing"

	"github.com/xlsama/meetkeep/internal/store"
)

func TestRender(t *testing.T) {
	m := store.Meeting{Title: "上线前对齐会", DurationMs: 24 * 60000}
	speakers := []store.Speaker{{ID: 1, Name: "我", Self: true}, {ID: 2, Name: "陈昊"}}
	utterances := []store.Utterance{
		{Seq: 1, Track: store.System, StartMs: 12000, EndMs: 15500, Text: "今天能不能把上线日期定下来？", Speaker: 2},
		{Seq: 2, Track: store.System, StartMs: 16000, EndMs: 18000, Text: "先别聊范围。", Speaker: 2},
		{Seq: 3, Track: store.Mic, StartMs: 3723004, EndMs: 3725000, Text: "我来排。", Speaker: 1},
		{Seq: 4, Track: store.System, StartMs: 3726000, EndMs: 3727000, Text: "好。"},
	}

	md := Render(m, utterances, speakers, Markdown)
	for _, want := range []string{
		"# 上线前对齐会\n", "· 24 分钟",
		"**陈昊** 00:12\n\n今天能不能把上线日期定下来？\n先别聊范围。\n",
		"**我** 1:02:03\n\n我来排。\n", "**对方** 1:02:06",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("the Markdown lacks %q:\n%s", want, md)
		}
	}
	if n := strings.Count(md, "**陈昊**"); n != 1 {
		t.Errorf("陈昊 heads %d groups, want consecutive lines under one", n)
	}

	srt := Render(m, utterances, speakers, SRT)
	if !strings.HasPrefix(srt, "1\n00:00:12,000 --> 00:00:15,500\n陈昊：今天能不能把上线日期定下来？\n\n2\n") {
		t.Errorf("the SRT starts:\n%s", srt)
	}
	if !strings.Contains(srt, "01:02:03,004 --> 01:02:05,000") {
		t.Errorf("the SRT lacks the hour-long timestamp:\n%s", srt)
	}

	text := Render(m, utterances, speakers, Text)
	if !strings.Contains(text, "[00:12] 陈昊：今天能不能把上线日期定下来？\n") {
		t.Errorf("the text is:\n%s", text)
	}
	if Markdown.Extension() != ".md" || Text.Extension() != ".txt" || SRT.Extension() != ".srt" {
		t.Error("extensions do not match the formats")
	}
}
