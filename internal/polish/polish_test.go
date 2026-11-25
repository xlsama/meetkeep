package polish

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func init() { again = []time.Duration{time.Millisecond, time.Millisecond} }

type fake struct {
	lines string
	terms string
	// review answers for the lines looked at once more.
	review string
	fail   bool
	// busy is how many times more a part of the transcript is turned away.
	busy *atomic.Int32
	// asked takes the instructions each part of the transcript is given.
	asked *[]string
}

func (f fake) CompleteJSON(_ context.Context, system, user string) (string, error) {
	if f.fail {
		return "", errors.New("offline")
	}
	if !strings.HasPrefix(system, wholePrompt) || !strings.Contains(system, "\n") {
		return "", errors.New("asked without the transcript")
	}
	if strings.Contains(user, `{"terms"`) {
		if f.asked != nil && strings.Contains(user, "以往的会议") {
			*f.asked = append(*f.asked, user)
		}
		return f.terms, nil
	}
	if strings.Contains(user, "要复查的行") {
		if f.asked != nil {
			*f.asked = append(*f.asked, user)
		}
		if f.review == "" {
			return `{"lines":[]}`, nil
		}
		return f.review, nil
	}
	if f.busy != nil && f.busy.Add(-1) >= 0 {
		return "", errors.New("429")
	}
	if f.asked != nil {
		*f.asked = append(*f.asked, user)
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
	result, err := Polish(context.Background(), model, lines, Memory{Known: []string{"顺丰快递"}})
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

	if _, err := Polish(context.Background(), fake{fail: true}, lines, Memory{}); err == nil {
		t.Error("polishing with a model that does not answer succeeded")
	}

	var busy atomic.Int32
	busy.Store(2)
	model.busy = &busy
	if again, err := Polish(context.Background(), model, lines, Memory{Known: []string{"顺丰快递"}}); err != nil || again.Failed != 0 || len(again.Edits) != len(want) {
		t.Errorf("a part turned away twice is polished as %+v (%v)", again, err)
	}
	busy.Store(3)
	if _, err := Polish(context.Background(), model, lines, Memory{}); err == nil {
		t.Error("a part turned away three times was polished")
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
	result, err := Polish(context.Background(), model, lines, Memory{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Terms) != 1 || result.Terms[0].Throughout {
		t.Fatalf("terms are %+v", result.Terms)
	}
	if len(asked) != 1 || !strings.Contains(asked[0], "按键 → 案件") || !strings.Contains(asked[0], "3 按键。") {
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
	result, err := Polish(context.Background(), fake{lines: `{"lines":[]}`, terms: `{"terms":[]}`}, lines, Memory{})
	if err != nil {
		t.Fatal(err)
	}
	want := []Edit{{Seq: 2, Filler: true}, {Seq: 5, Filler: true}}
	if len(result.Edits) != 2 || result.Edits[0] != want[0] || result.Edits[1] != want[1] {
		t.Errorf("edits are %+v, want the echoes left out and the answer kept", result.Edits)
	}
}

func TestWordsSetRightBeforeAreSetRightAgain(t *testing.T) {
	lines := []Line{
		{1, "这批结果要回韩必达。"},
		{2, "飞速那条线先不动。"},
		{3, "跑皮明天晚上开始。"},
	}
	var asked []string
	model := fake{
		lines: `{"lines":[],"fixed":[]}`,
		terms: `{"terms":[{"heard":"飞速","meant":"非诉","why":"以往确认过，这里谈的是业务线"},{"heard":"结果","meant":"结裹","why":"只出现一次，以往也没见过"}]}`,
		asked: &asked,
	}
	memory := Memory{Terms: []Remembered{
		{Heard: "韩必达", Meant: "函必达", Confirmed: true},
		{Heard: "飞速", Meant: "非诉"},
		{Heard: "跑皮", Meant: "跑批"},
		{Heard: "尾案", Meant: "委案"},
	}}
	result, err := Polish(context.Background(), model, lines, memory)
	if err != nil {
		t.Fatal(err)
	}
	if len(asked) == 0 || !strings.Contains(asked[0], "飞速 → 非诉") || !strings.Contains(asked[0], "跑皮 → 跑批") || strings.Contains(asked[0], "尾案") || strings.Contains(asked[0], "韩必达 →") {
		t.Errorf("the readers were asked %q: want what was set right before and is said here, without what the user confirmed", asked)
	}
	if len(result.Terms) != 2 || result.Terms[0] != (Term{Heard: "韩必达", Meant: "函必达", Throughout: true, Remembered: true}) || result.Terms[1].Heard != "飞速" || !result.Terms[1].Remembered || result.Terms[1].Throughout {
		t.Fatalf("terms are %+v: want what the user confirmed, and what the readers found said again", result.Terms)
	}
	if len(result.Edits) != 1 || result.Edits[0] != (Edit{Seq: 1, Text: "这批结果要回函必达。"}) {
		t.Errorf("edits are %+v", result.Edits)
	}
}

func TestLinesInDoubtAreLookedAtOnceMore(t *testing.T) {
	lines := []Line{
		{1, "呃，这个尾案批次是三月一号进来的。"},
		{2, "我们明天上线。"},
		{3, "尾案以后先过强规则，然后再分。"},
		{4, "那个尾案的数据。"},
	}
	var asked []string
	model := fake{
		lines:  `{"lines":[{"n":1,"text":"这个委案批次是三月一号进来的。"},{"n":2,"text":"我们明天上线，并且通知所有客户。"}],"fixed":[{"heard":"尾案","meant":"委案"},{"heard":"上线","meant":"上线了"}]}`,
		terms:  `{"terms":[]}`,
		review: `{"lines":[{"n":3,"text":"委案以后先过强规则，然后再分。"},{"n":2,"text":"明天我们会上线并且通知全部客户和合作方。"},{"n":9,"text":"多出来的"}]}`,
		asked:  &asked,
	}
	result, err := Polish(context.Background(), model, lines, Memory{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Terms) != 1 || result.Terms[0] != (Term{Heard: "尾案", Meant: "委案", Local: true}) {
		t.Fatalf("terms are %+v, want the word the lines told was misheard", result.Terms)
	}
	review := asked[len(asked)-1]
	if result.Reviewed != 3 || !strings.Contains(review, "2 我们明天上线。\n—— 上一轮改成「我们明天上线，并且通知所有客户。」") || !strings.Contains(review, "3 尾案以后先过强规则，然后再分。\n—— 别处把「尾案」改成了「委案」") || strings.Contains(review, "1 ") {
		t.Errorf("%d lines were looked at once more, asked as %q", result.Reviewed, review)
	}
	want := []Edit{{Seq: 1, Text: "这个委案批次是三月一号进来的。"}, {Seq: 3, Text: "委案以后先过强规则，然后再分。"}}
	if result.Revised != 1 || result.Refused != 1 || len(result.Edits) != 2 || result.Edits[0] != want[0] || result.Edits[1] != want[1] {
		t.Errorf("revised %d, refused %d, edits %+v", result.Revised, result.Refused, result.Edits)
	}
}

func TestRecall(t *testing.T) {
	memory := Recall([]Learned{
		{Heard: "含币达", Meant: "韩必达", Meetings: 1},
		{Heard: "韩必达", Meant: "函必达", Meetings: 5},
		{Heard: "韩必达", Meant: "汉必达", Meetings: 1},
		{Heard: "跑皮", Meant: "跑批", Meetings: 2},
		{Heard: "按键", Meant: "案件", Meetings: 3, Dropped: true},
		{Heard: "清除", Meant: "清楚", Meetings: 1},
		{Heard: "飞速", Meant: "非诉", Confirmed: true},
		{Heard: "顺丰", Meant: "顺风", Meetings: 4},
	}, []string{"顺丰"})
	want := []Remembered{
		{Heard: "飞速", Meant: "非诉", Confirmed: true},
		{Heard: "韩必达", Meant: "函必达"},
		{Heard: "跑皮", Meant: "跑批"},
		{Heard: "含币达", Meant: "函必达"},
		{Heard: "清除", Meant: "清楚"},
	}
	if len(memory.Terms) != len(want) {
		t.Fatalf("recalled %+v", memory.Terms)
	}
	for i, w := range want {
		if memory.Terms[i] != w {
			t.Errorf("term %d is %+v, want %+v", i, memory.Terms[i], w)
		}
	}
	if got := strings.Join(memory.Known, " "); got != "顺丰 非诉 函必达 跑批" {
		t.Errorf("the words spelled right are %q", got)
	}
	if len(memory.Dropped) != 1 || memory.Dropped[0].Heard != "按键" {
		t.Errorf("dropped %+v", memory.Dropped)
	}

	lines := []Line{{1, "这个按键点一下，按键就查出来了。"}, {2, "函必达那边没问题，函必达。"}}
	model := fake{lines: `{"lines":[],"fixed":[{"heard":"按键","meant":"案件"}]}`, terms: `{"terms":[{"heard":"按键","meant":"案件","why":"x"},{"heard":"函必达","meant":"韩必达","why":"x"}]}`}
	if result, err := Polish(context.Background(), model, lines, memory); err != nil || len(result.Terms) != 0 || len(result.Edits) != 0 {
		t.Errorf("what the user dropped and what is spelled right were set to something else: %+v (%v)", result, err)
	}
}
