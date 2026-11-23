package export

import (
	"strings"
	"testing"

	"github.com/xlsama/meetkeep/internal/store"
)

const written = `{"final":true,"headline":"周五上线","gist":"对齐上线前的事","conclusions":[{"text":"周五上线","seq":4,"atMs":65000}],
"topics":[{"title":"上线日期","points":[{"text":"测试周四收尾","seq":2,"atMs":12000,"uncertain":true}]}],
"todos":[{"text":"写回滚方案","owner":"陈昊","due":"周四","seq":5,"atMs":3723000},{"text":"通知客户","owner":"我","seq":6,"atMs":90000}],
"open":[{"text":"要不要灰度","seq":7,"atMs":120000}],
"chapters":[{"title":"定日期","text":"陈昊问能不能定下来。","seq":1,"atMs":0},{"title":"分工","text":"","seq":5,"atMs":3723000}]}`

func TestNotes(t *testing.T) {
	got := Notes(store.StoredNotes{Data: written})
	want := `## 周五上线

对齐上线前的事

- 周五上线 [01:05]

### 待办

- [ ] 通知客户（我） [01:30]
- [ ] 写回滚方案（陈昊 · 周四） [1:02:03]

### 还没定

- 要不要灰度 [02:00]

### 上线日期

- 测试周四收尾（存疑） [00:12]`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if got := Notes(store.StoredNotes{}); got != "" {
		t.Errorf("a meeting without notes reads %q", got)
	}
}

func TestNotesOfDocument(t *testing.T) {
	doc := `{"type":"doc","content":[
		{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"下周一上线"}]},
		{"type":"heading","attrs":{"level":3},"content":[{"type":"text","text":"待办"}]},
		{"type":"list","attrs":{"kind":"task","checked":true},"content":[{"type":"paragraph","content":[
			{"type":"text","text":"写回滚方案 "},{"type":"assignee","attrs":{"owner":"陈昊","due":""}},{"type":"timestamp","attrs":{"ms":3723000,"seq":5}}]},
			{"type":"list","attrs":{"kind":"bullet"},"content":[{"type":"paragraph","content":[{"type":"text","text":"含数据库"}]}]}]},
		{"type":"list","attrs":{"kind":"task","checked":false},"content":[{"type":"paragraph","content":[{"type":"text","text":"订会议室"}]}]}]}`
	kept := store.StoredNotes{Data: written, Doc: doc}
	got := Notes(kept)
	for _, want := range []string{
		"## 下周一上线\n\n### 待办\n\n- [x] 写回滚方案（陈昊） [1:02:03]\n  - 含数据库\n- [ ] 订会议室",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the document lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "周五上线") {
		t.Errorf("the notes show through the document the user made of them:\n%s", got)
	}

	tasks := Tasks(kept)
	if len(tasks) != 2 {
		t.Fatalf("got %+v, want two tasks", tasks)
	}
	if got := tasks[0]; got != (Task{Text: "写回滚方案", Owner: "陈昊", Done: true, Seq: 5, AtMs: 3723000}) {
		t.Errorf("got %+v", got)
	}
	if got := tasks[1]; got != (Task{Text: "订会议室"}) {
		t.Errorf("got %+v", got)
	}
	if tasks := Tasks(store.StoredNotes{Data: written}); len(tasks) != 2 || tasks[0].Owner != "我" || tasks[1].Due != "周四" {
		t.Errorf("got %+v, want the user's own task first", tasks)
	}
}
