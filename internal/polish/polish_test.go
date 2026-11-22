package polish

import (
	"context"
	"errors"
	"testing"
)

type fake struct {
	lines string
	terms string
	fail  bool
}

func (f fake) CompleteJSON(context.Context, string, string) (string, error) {
	if f.fail {
		return "", errors.New("offline")
	}
	return f.terms, nil
}

func (f fake) CompleteQuick(context.Context, string, string) (string, error) {
	if f.fail {
		return "", errors.New("offline")
	}
	return f.lines, nil
}

func TestTaken(t *testing.T) {
	for _, c := range []struct {
		original, edited string
		kept, filler     bool
	}{
		{"呃，这个这个策略先走一二三。", "这个策略先走一二三。", true, false},
		{"那-那我再形成一个策略流以后", "那我再形成一个策略流以后", true, false},
		{"对对对对对。", "", false, true},
		{"OK。", "", false, true},
		{"然后我们再能够把这些策略再通过开发再排成一个策略流", "", false, false},
		{"我们把策略排起来", "我们把规则排起来", false, false},
		{"我们把策略排起来", "把我们策略排起来", false, false},
		{"这个结果要返回韩必达，他在模板上才能显示上面的案件信息", "要返回", false, false},
		{"Kafka 那边呃没问题", "kafka 那边没问题", true, false},
	} {
		kept, filler := taken(c.original, c.edited)
		if kept != c.kept || filler != c.filler {
			t.Errorf("%q to %q: kept %v, filler %v; want %v, %v", c.original, c.edited, kept, filler, c.kept, c.filler)
		}
	}
}

func TestAlike(t *testing.T) {
	for _, c := range []struct {
		heard, meant string
		want         bool
	}{
		{"韩必达", "函必达", true},
		{"抱税表", "报税表", true},
		{"汇款", "回款", false},
		{"借口", "接口", false},
		{"韩必达", "韩必达", false},
		{"韩必达", "顺丰", false},
		{"韩必达", "快递柜", false},
		{"达", "大", false},
	} {
		if got := alike(c.heard, c.meant); got != c.want {
			t.Errorf("%q for %q: %v, want %v", c.heard, c.meant, got, c.want)
		}
	}
}

func TestPolish(t *testing.T) {
	lines := []Line{
		{1, "呃，这任务不都要回韩必达？"},
		{2, "对对对。"},
		{3, "那没有韩必达怎么发函呢？"},
		{4, "这个这个策略先走一二三。"},
		{5, "我们明天上线。"},
		{6, "顺丰快递那边没问题，顺丰快递。"},
	}
	model := fake{
		lines: `{"lines":[{"n":1,"text":"这任务不都要回韩必达？"},{"n":2,"text":""},{"n":4,"text":"这个策略先走一二三。"},{"n":5,"text":"我们后天上线。"},{"n":9,"text":"多出来的"}]}`,
		terms: `{"terms":[{"heard":"韩必达","meant":"函必达","why":"谈的是发函"},{"heard":"顺丰快递","meant":"顺风快递","why":"猜的"},{"heard":"上线","meant":"下线","why":"两个字"},{"heard":"上线","meant":"下线到","why":"字数不同"}]}`,
	}
	result, err := Polish(context.Background(), model, lines, []string{"顺丰快递"})
	if err != nil {
		t.Fatal(err)
	}
	want := []Edit{
		{Seq: 1, Text: "这任务不都要回函必达？"},
		{Seq: 2, Filler: true},
		{Seq: 3, Text: "那没有函必达怎么发函呢？"},
		{Seq: 4, Text: "这个策略先走一二三。"},
	}
	if len(result.Edits) != len(want) {
		t.Fatalf("edits are %+v", result.Edits)
	}
	for i, w := range want {
		if result.Edits[i] != w {
			t.Errorf("edit %d is %+v, want %+v", i, result.Edits[i], w)
		}
	}
	if len(result.Terms) != 1 || result.Terms[0].Meant != "函必达" {
		t.Errorf("terms are %+v", result.Terms)
	}
	if result.Refused != 1 {
		t.Errorf("%d edits were refused, want the one that changed a word", result.Refused)
	}

	if _, err := Polish(context.Background(), fake{fail: true}, lines, nil); err == nil {
		t.Error("polishing with a model that does not answer succeeded")
	}
}
