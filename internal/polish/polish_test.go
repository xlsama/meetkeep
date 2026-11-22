package polish

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fake struct {
	lines string
	terms string
	fail  bool
	// asked takes the instructions each part of the transcript is given.
	asked *[]string
}

func (f fake) CompleteJSON(_ context.Context, system, _ string) (string, error) {
	if f.fail {
		return "", errors.New("offline")
	}
	if strings.Contains(system, `{"terms"`) {
		return f.terms, nil
	}
	if f.asked != nil {
		*f.asked = append(*f.asked, system)
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
		{"一批这个按键进来以后，先要强规则去拦下", "一批案件进来以后，先要强规则去拦下", true, false},
		{"他是试进执行一次通过", "他是失信执行一次通过", true, false},
		{"我们明天上线。", "我们明天上线，然后通知所有客户。", false, false},
		{"这个结果要返回韩必达，他在模板上才能显示上面的案件信息", "要返回", false, false},
		{"这个结果要返回韩必达，他在模板上才能显示上面的案件信息", "系统需要把处理结果同步给发函平台用于展示", false, false},
		{"模型评分47通过，项目阈值小于61", "模型评分74通过，项目阈值小于61", false, false},
		{"他一进来是12000的案件", "他一进来是一万二的案件", false, false},
		{"Kafka 那边呃没问题", "kafka 那边没问题", true, false},
		{"嗯。", "好的。", false, false},
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
		lines: `{"lines":[{"n":1,"text":"这任务不都要回韩必达？"},{"n":2,"text":""},{"n":4,"text":"这个策略先走一二三。"},{"n":5,"text":"我们明天上线，并且通知所有客户。"},{"n":9,"text":"多出来的"}]}`,
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
	if len(result.Terms) != 1 || result.Terms[0].Meant != "函必达" || !result.Terms[0].Throughout {
		t.Errorf("terms are %+v", result.Terms)
	}
	if result.Refused != 1 {
		t.Errorf("%d edits were refused, want the one that added to what was said", result.Refused)
	}

	if _, err := Polish(context.Background(), fake{fail: true}, lines, nil); err == nil {
		t.Error("polishing with a model that does not answer succeeded")
	}
}

func TestWordsOfTwoLettersAreLeftToTheLines(t *testing.T) {
	lines := []Line{
		{1, "一批按键进来以后先过强规则。"},
		{2, "页面上那个按键点一下就能查按键。"},
		{3, "按键。"},
	}
	var asked []string
	model := fake{
		lines: `{"lines":[{"n":1,"text":"一批案件进来以后先过强规则。"},{"n":2,"text":"页面上那个按键点一下就能查案件。"},{"n":3,"text":"案件。"}]}`,
		terms: `{"terms":[{"heard":"按键","meant":"案件","why":"全文在谈案件"}]}`,
		asked: &asked,
	}
	result, err := Polish(context.Background(), model, lines, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Terms) != 1 || result.Terms[0].Throughout {
		t.Fatalf("terms are %+v", result.Terms)
	}
	if len(asked) != 1 || !strings.Contains(asked[0], "按键 → 案件") {
		t.Errorf("the lines were polished without being told what was misheard: %q", asked)
	}
	want := []Edit{{Seq: 1, Text: "一批案件进来以后先过强规则。"}, {Seq: 2, Text: "页面上那个按键点一下就能查案件。"}, {Seq: 3, Text: "案件。"}}
	if len(result.Edits) != 3 || result.Edits[0] != want[0] || result.Edits[1] != want[1] || result.Edits[2] != want[2] {
		t.Errorf("edits are %+v", result.Edits)
	}
}

func TestEcho(t *testing.T) {
	for text, want := range map[string]bool{
		"对对对对对。": true, "是的，是的。": true, "是吧？": true, "OK。": true, "嗯。": true, "好的好的": true,
		"不是。": false, "可以。": false, "明白。": false, "对，你来解释。": false, "是的是的是的是的是的": false, "": false,
	} {
		if got := echo(text); got != want {
			t.Errorf("%q: %v, want %v", text, got, want)
		}
	}
}

func TestEchoesAreLeftOutButAnswersKept(t *testing.T) {
	lines := []Line{
		{1, "我们先把策略排成一个策略流。"},
		{2, "对对对。"},
		{3, "这样先测试的话，不需要先去审批吗？"},
		{4, "对。"},
		{5, "是的是的。"},
	}
	result, err := Polish(context.Background(), fake{lines: `{"lines":[]}`, terms: `{"terms":[]}`}, lines, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []Edit{{Seq: 2, Filler: true}, {Seq: 5, Filler: true}}
	if len(result.Edits) != 2 || result.Edits[0] != want[0] || result.Edits[1] != want[1] {
		t.Errorf("edits are %+v, want the echoes left out and the answer kept", result.Edits)
	}
}
