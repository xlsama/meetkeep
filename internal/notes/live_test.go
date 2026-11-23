package notes

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xlsama/meetkeep/internal/config"
	"github.com/xlsama/meetkeep/internal/llm"
)

// meeting is the demo meeting of the design: a launch date gets set, three
// things get owners and one is left open.
var meeting = []Line{
	{Seq: 1, StartMs: 12000, EndMs: 16000, Speaker: "陈昊", Text: "先别聊范围——今天能不能把上线日期定下来？"},
	{Seq: 2, StartMs: 41000, EndMs: 49000, Speaker: "林悦", Text: "3 月 14 号可以，前提是新的价格页这周能好。不然就得推到 21 号。"},
	{Seq: 3, StartMs: 80000, EndMs: 84000, Speaker: "周然", Text: "价格页我来，周四给初稿。"},
	{Seq: 4, StartMs: 98000, EndMs: 103000, Speaker: "我", Text: "那就剩首页文案了，现在没人认领。"},
	{Seq: 5, StartMs: 112000, EndMs: 115000, Speaker: "林悦", Text: "价格页弄完我可以接。"},
	{Seq: 6, StartMs: 151000, EndMs: 157000, Speaker: "我", Text: "客服那边要提前一周知道，培训材料得跟着改。"},
	{Seq: 7, StartMs: 167000, EndMs: 172000, Speaker: "陈昊", Text: "那最晚 7 号要把改动清单给到客服。"},
	{Seq: 8, StartMs: 195000, EndMs: 201000, Speaker: "林悦", Text: "支付那块的回归谁跑？上次漏了优惠券叠加。"},
	{Seq: 9, StartMs: 208000, EndMs: 211000, Speaker: "我", Text: "我来排，周五前跑完。"},
	{Seq: 10, StartMs: 232000, EndMs: 237000, Speaker: "陈昊", Text: "发版窗口放在晚上十点以后，避开高峰。"},
	{Seq: 11, StartMs: 238000, EndMs: 243000, Speaker: "陈昊", Text: "行，就 3 月 14 号。我去跟客服说一声。"},
}

// TestWithTheConfiguredModel writes notes with the model of the settings.
// It runs when MEETKEEP_LLM_TEST is set.
func TestWithTheConfiguredModel(t *testing.T) {
	if os.Getenv("MEETKEEP_LLM_TEST") == "" {
		t.Skip("MEETKEEP_LLM_TEST is not set")
	}
	settings, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	model, err := llm.New(settings.LLM.BaseURL, settings.LLM.Key, settings.LLM.Model)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	show := func(what string, n Notes, began time.Time) {
		var b strings.Builder
		fmt.Fprintf(&b, "%s in %v, to %d s: %s | %s\n", what, time.Since(began).Round(time.Millisecond), n.CoveredMs/1000, n.Title, n.Headline)
		for _, topic := range n.Topics {
			fmt.Fprintf(&b, "  %s %s (updating %v)\n", topic.ID, topic.Title, topic.Updating)
			for _, p := range topic.Points {
				fmt.Fprintf(&b, "    - %s [%d, uncertain %v]\n", p.Text, p.Seq, p.Uncertain)
			}
			for _, p := range topic.Nodes {
				fmt.Fprintf(&b, "    ○ %s [%d, uncertain %v]\n", p.Text, p.Seq, p.Uncertain)
			}
		}
		for _, todo := range n.Todos {
			fmt.Fprintf(&b, "  todo: %s | %s | %s [%d]\n", todo.Text, todo.Owner, todo.Due, todo.Seq)
		}
		for _, p := range n.Open {
			fmt.Fprintf(&b, "  open: %s [%d]\n", p.Text, p.Seq)
		}
		t.Log(b.String())
	}

	current := Empty()
	for _, upTo := range []int{5, 9, 11} {
		began := time.Now()
		current, err = Update(ctx, model, current, meeting[:upTo], nil)
		if err != nil {
			t.Fatal(err)
		}
		show("live notes", current, began)
	}
	if len(current.Topics) < 2 || len(current.Todos) < 2 {
		t.Errorf("the live notes hold %d topics and %d todos", len(current.Topics), len(current.Todos))
	}

	began := time.Now()
	final, err := Finalize(ctx, model, "3月14日 10:04 的会议", meeting, nil)
	if err != nil {
		t.Fatal(err)
	}
	show("final notes", final, began)
	if final.Title == "" || strings.Contains(final.Title, "的会议") || final.Headline == "" || len(final.Topics) < 2 || len(final.Todos) < 3 {
		t.Errorf("the final notes hold a title %q, a headline %q, %d topics and %d todos", final.Title, final.Headline, len(final.Topics), len(final.Todos))
	}
}
