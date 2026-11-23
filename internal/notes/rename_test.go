package notes

import (
	"strings"
	"testing"
)

func TestRenamed(t *testing.T) {
	written := Empty()
	written.Gist = "说话人 1 向说话人 12 交代上线前的事"
	written.Conclusions = []Point{{Text: "协议修改版已发对方", Seq: 1}}
	written.Todos = []Todo{{Text: "说话人 1 排支付回归", Owner: "说话人 1"}, {Text: "回复协议", Owner: "对方"}}
	written.Chapters = []Chapter{{Title: "说话人 1 部署回归", Text: "要求说话人 12 周五前跑完。"}}

	named := written.Renamed("说话人 1", "林悦")
	if named.Gist != "林悦 向说话人 12 交代上线前的事" || named.Todos[0].Owner != "林悦" || named.Todos[0].Text != "林悦 排支付回归" {
		t.Errorf("got %+v", named)
	}
	if c := named.Chapters[0]; c.Title != "林悦 部署回归" || c.Text != "要求说话人 12 周五前跑完。" {
		t.Errorf("the chapter reads %+v", c)
	}
	if written.Todos[0].Owner != "说话人 1" || written.Chapters[0].Title != "说话人 1 部署回归" {
		t.Errorf("the notes renamed were changed: %+v", written)
	}

	named = written.Renamed("对方", "周然")
	if named.Conclusions[0].Text != "协议修改版已发对方" || named.Todos[1].Owner != "周然" {
		t.Errorf("a word of its own was renamed: %+v", named)
	}

	doc := `{"type":"doc","content":[{"type":"list","attrs":{"kind":"task","checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"说话人 1 排 <支付> 回归"},{"type":"assignee","attrs":{"owner":"说话人 1","due":"周五"}},{"type":"timestamp","attrs":{"ms":98000,"seq":4}}]}]}]}`
	renamed := RenamedDoc(doc, "说话人 1", "林悦")
	for _, want := range []string{`"text":"林悦 排 <支付> 回归"`, `"owner":"林悦"`, `"ms":98000`, `"checked":false`} {
		if !strings.Contains(renamed, want) {
			t.Errorf("the document lacks %s: %s", want, renamed)
		}
	}
	if got := RenamedDoc("不是 JSON", "说话人 1", "林悦"); got != "不是 JSON" {
		t.Errorf("got %q", got)
	}
}
