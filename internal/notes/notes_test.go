package notes

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type scripted struct {
	answers []string
	asked   []string
}

func (s *scripted) CompleteJSON(_ context.Context, system, user string) (string, error) {
	s.asked = append(s.asked, user)
	answer := s.answers[0]
	s.answers = s.answers[1:]
	return answer, nil
}

func lines(n int) []Line {
	out := make([]Line, n)
	for i := range out {
		out[i] = Line{Seq: i + 1, StartMs: int64(i) * 10000, EndMs: int64(i)*10000 + 8000, Speaker: "对方", Text: "发言"}
	}
	return out
}

func TestUpdateAppendsAndRewritesOnlyTheLastTopics(t *testing.T) {
	ctx := context.Background()
	model := &scripted{answers: []string{
		`{"topics":[
			{"id":"","title":"上线日期","points":[{"text":"定在 3 月 14 号","seq":2},{"text":"没有出处的话","seq":99}]},
			{"id":"","title":"价格页","points":[{"text":"对方认领，周四给初稿","seq":3}],
			 "nodes":[{"text":"对方认领","seq":3},{"text":"周四给初稿","seq":3},{"text":"没有出处","seq":77}]},
			{"id":"","title":"空话题","points":[]}],
		  "todos":[{"text":"写价格页初稿","owner":"对方","due":"周四","seq":3},{"text":"没有出处的待办","owner":"我","due":"","seq":0}]}`,
		"好的，下面是结果：\n```json\n" + `{"topics":[
			{"id":"t1","title":"上线日期","points":[{"text":"定在 3 月 14 号","seq":2},{"text":"赶不上就推到 21 号","seq":5}]},
			{"id":"t2","title":"价格页","points":[{"text":"对方认领，周四给初稿","seq":3}]},
			{"id":"","title":"首页文案","points":[{"text":"对方说弄完可以接","seq":6,"uncertain":true}]}],
		  "todos":[{"text":"写价格页初稿","owner":"对方","due":"周四","seq":3}]}` + "\n```",
		`{"topics":[
			{"id":"t1","title":"被改写的旧话题","points":[{"text":"不该生效","seq":1}]},
			{"id":"t3","title":"首页文案","points":[{"text":"还没人认领","seq":6}]},
			{"id":"","title":"支付回归","points":[{"text":"我来排，周五前跑完","seq":8}]}],
		  "todos":[{"text":"排支付回归","owner":"我","due":"周五","seq":8}]}`,
	}}

	notes, err := Update(ctx, model, Empty(), lines(4), []string{"MeetKeep"})
	if err != nil {
		t.Fatal(err)
	}
	if len(notes.Topics) != 2 || notes.Topics[0].ID != "t1" || len(notes.Topics[0].Points) != 1 || notes.Topics[1].AtMs != 20000 {
		t.Fatalf("first update gave %+v", notes.Topics)
	}
	if nodes := notes.Topics[1].Nodes; len(nodes) != 2 || nodes[1].Text != "周四给初稿" || len(notes.Topics[0].Nodes) != 0 {
		t.Errorf("the whiteboard's nodes are %+v", nodes)
	}
	if len(notes.Todos) != 1 || notes.Todos[0].AtMs != 20000 || notes.CoveredMs != 38000 {
		t.Errorf("todos %+v, covered %d", notes.Todos, notes.CoveredMs)
	}

	notes, err = Update(ctx, model, notes, lines(6), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes.Topics) != 3 || len(notes.Topics[0].Points) != 2 || !notes.Topics[2].Points[0].Uncertain {
		t.Fatalf("second update gave %+v", notes.Topics)
	}
	if notes.Topics[0].Updating || !notes.Topics[1].Updating || !notes.Topics[2].Updating {
		t.Errorf("the last two topics should be the ones still updating: %+v", notes.Topics)
	}
	if len(notes.Todos) != 1 {
		t.Errorf("a repeated todo was added: %+v", notes.Todos)
	}

	notes, err = Update(ctx, model, notes, lines(8), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes.Topics) != 4 || notes.Topics[0].Title != "上线日期" || len(notes.Topics[0].Points) != 2 {
		t.Fatalf("a frozen topic was rewritten: %+v", notes.Topics)
	}
	if notes.Topics[2].Points[0].Text != "还没人认领" || notes.Topics[3].ID != "t4" || len(notes.Todos) != 2 {
		t.Errorf("third update gave %+v, todos %+v", notes.Topics, notes.Todos)
	}

	var asked struct {
		Frozen []string
		Open   []struct{ ID string }
		Lines  []Line
	}
	if err := json.Unmarshal([]byte(model.asked[2]), &asked); err != nil {
		t.Fatal(err)
	}
	if len(asked.Frozen) != 1 || len(asked.Open) != 2 || asked.Open[0].ID != "t2" {
		t.Errorf("the third request froze %v and left open %v", asked.Frozen, asked.Open)
	}
	if len(asked.Lines) != 6 || asked.Lines[0].Seq != 3 {
		t.Errorf("the third request carried %d lines from seq %d, want those since the open topics began", len(asked.Lines), asked.Lines[0].Seq)
	}
}

func TestFinalize(t *testing.T) {
	model := &scripted{answers: []string{`{"headline":"3 月 14 号上线","topics":[{"title":"上线日期","points":[{"text":"定在 3 月 14 号","seq":2}]}],
		"todos":[{"text":"排支付回归","owner":"我","due":"周五","seq":4}],"open":[{"text":"首页文案谁来写","seq":3},{"text":"编的","seq":40}]}`}}
	notes, err := Finalize(context.Background(), model, "上线前对齐会", lines(4), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !notes.Final || notes.Headline != "3 月 14 号上线" || len(notes.Topics) != 1 || len(notes.Todos) != 1 || len(notes.Open) != 1 {
		t.Fatalf("got %+v", notes)
	}
	if notes.Topics[0].Updating || notes.Open[0].AtMs != 20000 {
		t.Errorf("got %+v", notes)
	}
	if text := notes.Text(); !strings.Contains(text, "首页文案谁来写") || !strings.Contains(text, "排支付回归") {
		t.Errorf("the searchable text is %q", text)
	}
	if !strings.Contains(model.asked[0], "上线前对齐会") {
		t.Error("the title was not sent")
	}
}

func TestBadAnswersKeepTheNotes(t *testing.T) {
	current := Empty()
	current.Topics = []Topic{{ID: "t1", Title: "上线日期", Points: []Point{{Text: "定了", Seq: 1}}}}
	for _, bad := range []string{"抱歉，我做不到", `{"topics": [`} {
		notes, err := Update(context.Background(), &scripted{answers: []string{bad}}, current, lines(2), nil)
		if err == nil {
			t.Errorf("%q was taken as an answer", bad)
		}
		if len(notes.Topics) != 1 || notes.Topics[0].Title != "上线日期" {
			t.Errorf("the notes were changed to %+v", notes)
		}
	}
}
