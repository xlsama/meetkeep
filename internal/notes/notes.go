// Package notes writes the notes of a meeting with a language model: an
// outline that grows while the meeting goes on, and the final notes once
// it has ended.
package notes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Notes are the notes of a meeting.
type Notes struct {
	// Final tells the notes written after the meeting from those that
	// grow during it.
	Final bool `json:"final"`
	// Title names the meeting, in final notes, Headline says in a sentence
	// what it settled, Gist what it was about, and Conclusions what it
	// agreed on.
	Title       string  `json:"title"`
	Headline    string  `json:"headline"`
	Gist        string  `json:"gist"`
	Conclusions []Point `json:"conclusions"`
	// CoveredMs is how far into the meeting the notes reach.
	CoveredMs int64   `json:"coveredMs"`
	Topics    []Topic `json:"topics"`
	Todos     []Todo  `json:"todos"`
	// Open are the matters left undecided, in final notes.
	Open []Point `json:"open"`
}

type Topic struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	AtMs  int64  `json:"atMs"`
	// Updating marks the topics that the next update may still rewrite.
	Updating bool    `json:"updating"`
	Points   []Point `json:"points"`
	// Nodes are what the topic branches into on the whiteboard: its
	// points, cut into short phrases.
	Nodes []Point `json:"nodes"`
}

// Point is a line of the notes with the utterance it rests on.
type Point struct {
	Text string `json:"text"`
	Seq  int    `json:"seq"`
	AtMs int64  `json:"atMs"`
	// Uncertain marks what was said without being settled.
	Uncertain bool `json:"uncertain"`
}

type Todo struct {
	Text  string `json:"text"`
	Owner string `json:"owner"`
	Due   string `json:"due"`
	Seq   int    `json:"seq"`
	AtMs  int64  `json:"atMs"`
}

// Line is an utterance as the model reads it.
type Line struct {
	Seq     int    `json:"seq"`
	StartMs int64  `json:"-"`
	EndMs   int64  `json:"-"`
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
}

// Completer asks a language model for a JSON object.
type Completer interface {
	CompleteJSON(ctx context.Context, system, user string) (string, error)
}

// open is how many of the last topics an update may rewrite: the one
// being talked about. What is above it stays as it was read.
const open = 1

// Empty returns notes with nothing in them.
func Empty() Notes {
	return Notes{Conclusions: []Point{}, Topics: []Topic{}, Todos: []Todo{}, Open: []Point{}}
}

// Text returns the words of the notes, for searching.
func (n Notes) Text() string {
	var b strings.Builder
	add := func(parts ...string) {
		for _, p := range parts {
			if p != "" {
				b.WriteString(p)
				b.WriteByte('\n')
			}
		}
	}
	add(n.Headline, n.Gist)
	for _, p := range n.Conclusions {
		add(p.Text)
	}
	for _, t := range n.Topics {
		add(t.Title)
		for _, p := range t.Points {
			add(p.Text)
		}
	}
	for _, t := range n.Todos {
		add(t.Text, t.Owner)
	}
	for _, p := range n.Open {
		add(p.Text)
	}
	return b.String()
}

type answer struct {
	Title       string        `json:"title"`
	Headline    string        `json:"headline"`
	Gist        string        `json:"gist"`
	Conclusions []answerPoint `json:"conclusions"`
	Topics      []struct {
		ID     string        `json:"id"`
		Title  string        `json:"title"`
		Points []answerPoint `json:"points"`
		Nodes  []answerPoint `json:"nodes"`
	} `json:"topics"`
	Todos []struct {
		Text  string `json:"text"`
		Owner string `json:"owner"`
		Due   string `json:"due"`
		Seq   int    `json:"seq"`
	} `json:"todos"`
	Open []answerPoint `json:"open"`
}

type answerPoint struct {
	Text      string `json:"text"`
	Seq       int    `json:"seq"`
	Uncertain bool   `json:"uncertain"`
}

func parse(text string) (answer, error) {
	start, end := strings.Index(text, "{"), strings.LastIndex(text, "}")
	if start < 0 || end < start {
		return answer{}, errors.New("模型没有返回 JSON")
	}
	var a answer
	if err := json.Unmarshal([]byte(text[start:end+1]), &a); err != nil {
		return answer{}, fmt.Errorf("模型返回的 JSON 读不了：%w", err)
	}
	return a, nil
}

// cited keeps the points that rest on an utterance that was said.
func cited(points []answerPoint, lines map[int]Line) []Point {
	kept := []Point{}
	for _, p := range points {
		line, ok := lines[p.Seq]
		if text := strings.TrimSpace(p.Text); ok && text != "" {
			kept = append(kept, Point{Text: text, Seq: p.Seq, AtMs: line.StartMs, Uncertain: p.Uncertain})
		}
	}
	return kept
}

func index(lines []Line) map[int]Line {
	bySeq := make(map[int]Line, len(lines))
	for _, l := range lines {
		bySeq[l.Seq] = l
	}
	return bySeq
}

func (a answer) todos(lines map[int]Line, known []Todo) []Todo {
	todos := known
	for _, t := range a.Todos {
		line, ok := lines[t.Seq]
		text := strings.TrimSpace(t.Text)
		if !ok || text == "" {
			continue
		}
		repeated := false
		for _, k := range known {
			repeated = repeated || k.Text == text
		}
		if !repeated {
			todos = append(todos, Todo{
				Text: text, Owner: strings.TrimSpace(t.Owner), Due: strings.TrimSpace(t.Due), Seq: t.Seq, AtMs: line.StartMs,
			})
		}
	}
	return todos
}

const liveRules = `你在为一场正在进行的会议做实时纪要。纪要是按话题排列的大纲，只往后追加：早先的话题已经定下，不能改；你只能改写「可更新的话题」，也就是正在谈的那一个，或者在它后面追加新话题。

输入是一个 JSON 对象：
- frozen：已经定下的话题标题，只供了解前文
- open：可更新的话题，带它们现有的要点
- lines：这些话题开始以来的发言，每句有 seq 编号、说话人和内容
- todos：已经记下的待办
- glossary：专有名词的正确写法

输出一个 JSON 对象，不要输出别的内容：
{"topics":[{"id":"沿用 open 里的 id，新话题留空字符串","title":"话题名","points":[{"text":"要点","seq":12,"uncertain":false}],"nodes":[{"text":"短语","seq":12,"uncertain":false}]}],"todos":[{"text":"要做的事","owner":"谁来做","due":"什么时候","seq":12}]}

规则：
- topics 里先按原顺序写回 open 里的话题，再写新话题。新发言还在谈 open 里的话题就并进去；换了话题就开新话题，回头谈起 frozen 里的旧话题也开新话题。没有新内容的话题原样写回。
- 话题名不超过 10 个字。要点写结论、决定、数字和分工，不写讨论过程；每条不超过 30 个字，每个话题最多 4 条。
- nodes 是这个话题画在白板上的分支：把要点拆成更短的短语，一个节点只说一件事，每个不超过 12 个字，每个话题最多 4 个；没说定的在末尾加「？」并标 "uncertain": true。
- 每条要点、节点和待办都要带 seq，指向它依据的那句发言。没有依据的内容不要写。
- 说话人是「我」「对方」这样的称呼时，owner 照写这个称呼，不要猜人名。
- 只是有人提议或表示可以、但没有说定由谁做或做不做的事，标 "uncertain": true，并且不写进 todos。带前提条件的决定（"某事做完就按这个来"）是说定了的，不标。
- todos 只写这次新出现的待办，已经在输入的 todos 里的不要重复；没提到期限时 due 留空字符串。
- 用发言所用的语言写。glossary 里的词按它的写法写。
- 发言里没有值得记的内容时，原样写回 open，todos 给空数组。`

const finalRules = `你在为一场已经结束的会议写定稿纪要。

输入是一个 JSON 对象：
- title：会议现在的标题，可能只是录制时间或文件名
- lines：全部发言，每句有 seq 编号、说话人和内容
- glossary：专有名词的正确写法

输出一个 JSON 对象，不要输出别的内容：
{"title":"这场会的名字","headline":"一句话说清这场会定了什么","gist":"一句话交代这场会在谈什么","conclusions":[{"text":"定下来的一件事","seq":12}],"todos":[{"text":"要做的事","owner":"谁来做","due":"什么时候","seq":12}],"open":[{"text":"还没定的事，说清卡在哪","seq":12}],"topics":[{"title":"话题名","points":[{"text":"要点","seq":12}],"nodes":[{"text":"短语","seq":12,"uncertain":false}]}]}

规则：
- title 不超过 14 个字，像日历上的会议名，说它是什么会、围绕什么事，比如"交付周会""定价方案评审"；不写结论，不带日期。输入的 title 已经是这样的名字时照用。
- headline 不超过 30 个字，写结论，不写"本次会议讨论了"。
- gist 不超过 40 个字，交代是谁和谁、围绕什么事在谈，让没到场的人知道背景；不重复 headline 里的结论。
- conclusions 是这场会明确达成的共识或决定，最多 4 条，每条不超过 30 个字，按重要程度排。只写说定了的，没有就给空数组；不要为了凑数把讨论过程或还没定的事写进来。
- topics 按会议里出现的顺序排。话题名不超过 10 个字。要点写结论、决定、数字和分工，不写讨论过程；每条不超过 30 个字，每个话题最多 5 条。
- nodes 是这个话题画在白板上的分支：把要点拆成更短的短语，一个节点只说一件事，每个不超过 12 个字，每个话题最多 5 个；没说定的在末尾加「？」并标 "uncertain": true。
- 每条结论、要点、节点、待办和未定事项都要带 seq，指向它依据的那句发言。没有依据的内容不要写。
- todos 只写明确有人认领或被指派的事；随口一提、没人应下的不算。owner 写发言里的说话人称呼，不要猜人名；没提到期限时 due 留空字符串。
- 提出来但没有说定的事写进 open，不写进 todos。有人只是表示"可以接""到时候看"而没有明确应下的，也算没说定。
- 用发言所用的语言写。glossary 里的词按它的写法写。
- 发言太少、没有可记的内容时，各项给空数组，title、headline 和 gist 给空字符串。`

// Update brings the notes of a meeting in progress up to date with what
// has been said: lines are all its utterances so far, in order.
func Update(ctx context.Context, model Completer, current Notes, lines []Line, glossary []string) (Notes, error) {
	frozen := max(len(current.Topics)-open, 0)
	from := int64(0)
	if frozen < len(current.Topics) {
		from = current.Topics[frozen].AtMs
	} else if frozen > 0 {
		from = current.CoveredMs
	}
	type openTopic struct {
		ID     string   `json:"id"`
		Title  string   `json:"title"`
		Points []string `json:"points"`
	}
	input := struct {
		Frozen   []string    `json:"frozen"`
		Open     []openTopic `json:"open"`
		Lines    []Line      `json:"lines"`
		Todos    []string    `json:"todos"`
		Glossary []string    `json:"glossary"`
	}{Frozen: []string{}, Open: []openTopic{}, Lines: []Line{}, Todos: []string{}, Glossary: glossary}
	for _, t := range current.Topics[:frozen] {
		input.Frozen = append(input.Frozen, t.Title)
	}
	for _, t := range current.Topics[frozen:] {
		topic := openTopic{ID: t.ID, Title: t.Title, Points: []string{}}
		for _, p := range t.Points {
			topic.Points = append(topic.Points, p.Text)
		}
		input.Open = append(input.Open, topic)
	}
	for _, l := range lines {
		if l.StartMs >= from {
			input.Lines = append(input.Lines, l)
		}
	}
	for _, t := range current.Todos {
		input.Todos = append(input.Todos, t.Text)
	}
	if input.Glossary == nil {
		input.Glossary = []string{}
	}
	user, err := json.Marshal(input)
	if err != nil {
		return current, err
	}
	text, err := model.CompleteJSON(ctx, liveRules, string(user))
	if err != nil {
		return current, err
	}
	a, err := parse(text)
	if err != nil {
		return current, err
	}

	bySeq := index(lines)
	next := current
	next.Topics = append([]Topic{}, current.Topics[:frozen]...)
	rewritten := map[string]Topic{}
	var added []Topic
	for _, t := range a.Topics {
		points := cited(t.Points, bySeq)
		title := strings.TrimSpace(t.Title)
		if title == "" || len(points) == 0 {
			continue
		}
		topic := Topic{ID: t.ID, Title: title, AtMs: points[0].AtMs, Points: points, Nodes: cited(t.Nodes, bySeq)}
		if t.ID == "" {
			added = append(added, topic)
		} else {
			rewritten[t.ID] = topic
		}
	}
	for _, t := range current.Topics[frozen:] {
		if again, ok := rewritten[t.ID]; ok {
			again.AtMs = min(again.AtMs, t.AtMs)
			t = again
		}
		next.Topics = append(next.Topics, t)
	}
	for _, t := range added {
		t.ID = fmt.Sprintf("t%d", len(next.Topics)+1)
		next.Topics = append(next.Topics, t)
	}
	for i := range next.Topics {
		next.Topics[i].Updating = i >= len(next.Topics)-open
	}
	next.Todos = a.todos(bySeq, current.Todos)
	if len(lines) > 0 {
		next.CoveredMs = lines[len(lines)-1].EndMs
	}
	return next, nil
}

// Finalize writes the notes of a meeting that has ended.
func Finalize(ctx context.Context, model Completer, title string, lines []Line, glossary []string) (Notes, error) {
	if glossary == nil {
		glossary = []string{}
	}
	user, err := json.Marshal(struct {
		Title    string   `json:"title"`
		Lines    []Line   `json:"lines"`
		Glossary []string `json:"glossary"`
	}{title, lines, glossary})
	if err != nil {
		return Notes{}, err
	}
	text, err := model.CompleteJSON(ctx, finalRules, string(user))
	if err != nil {
		return Notes{}, err
	}
	a, err := parse(text)
	if err != nil {
		return Notes{}, err
	}
	bySeq := index(lines)
	notes := Empty()
	notes.Final = true
	notes.Title = strings.TrimSpace(a.Title)
	notes.Headline = strings.TrimSpace(a.Headline)
	notes.Gist = strings.TrimSpace(a.Gist)
	notes.Conclusions = cited(a.Conclusions, bySeq)
	for _, t := range a.Topics {
		points := cited(t.Points, bySeq)
		title := strings.TrimSpace(t.Title)
		if title == "" || len(points) == 0 {
			continue
		}
		notes.Topics = append(notes.Topics, Topic{
			ID: fmt.Sprintf("t%d", len(notes.Topics)+1), Title: title, AtMs: points[0].AtMs, Points: points,
			Nodes: cited(t.Nodes, bySeq),
		})
	}
	notes.Todos = a.todos(bySeq, []Todo{})
	notes.Open = cited(a.Open, bySeq)
	if len(lines) > 0 {
		notes.CoveredMs = lines[len(lines)-1].EndMs
	}
	return notes, nil
}
