package export

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/xlsama/meetkeep/internal/notes"
	"github.com/xlsama/meetkeep/internal/store"
)

// node is a node of the document the notes are read and edited as.
type node struct {
	Type    string         `json:"type"`
	Attrs   map[string]any `json:"attrs"`
	Content []node         `json:"content"`
	Text    string         `json:"text"`
}

func (n node) str(key string) string {
	s, _ := n.Attrs[key].(string)
	return strings.TrimSpace(s)
}

func (n node) num(key string) int64 {
	f, _ := n.Attrs[key].(float64)
	return int64(f)
}

func words(value string) []node {
	if value = strings.TrimSpace(value); value == "" {
		return nil
	}
	return []node{{Type: "text", Text: value}}
}

func titled(level int, value string) node {
	return node{Type: "heading", Attrs: map[string]any{"level": float64(level)}, Content: words(value)}
}

func item(kind string, seq int, atMs int64, content ...node) node {
	content = append(content, node{Type: "timestamp", Attrs: map[string]any{"ms": float64(atMs), "seq": float64(seq)}})
	return node{Type: "list", Attrs: map[string]any{"kind": kind}, Content: []node{{Type: "paragraph", Content: content}}}
}

// document lays notes out as the app does before the user edits them.
func document(n notes.Notes) node {
	var blocks []node
	points := func(title string, points []notes.Point) {
		if title != "" && len(points) > 0 {
			blocks = append(blocks, titled(3, title))
		}
		for _, p := range points {
			content := words(p.Text)
			if p.Uncertain {
				content = append(content, node{Type: "doubt"})
			}
			blocks = append(blocks, item("bullet", p.Seq, p.AtMs, content...))
		}
	}
	todos := func() {
		if len(n.Todos) > 0 {
			blocks = append(blocks, titled(3, "待办"))
		}
		for _, own := range []bool{true, false} {
			for _, t := range n.Todos {
				if (strings.TrimSpace(t.Owner) == "我") != own {
					continue
				}
				content := words(t.Text)
				if owner, due := strings.TrimSpace(t.Owner), strings.TrimSpace(t.Due); owner != "" || due != "" {
					content = append(content, node{Type: "assignee", Attrs: map[string]any{"owner": owner, "due": due}})
				}
				blocks = append(blocks, item("task", t.Seq, t.AtMs, content...))
			}
		}
	}
	topics := func() {
		for _, t := range n.Topics {
			blocks = append(blocks, titled(2, t.Title))
			points("", t.Points)
		}
	}
	if !n.Final {
		topics()
		todos()
		return node{Type: "doc", Content: blocks}
	}
	if strings.TrimSpace(n.Headline) != "" {
		blocks = append(blocks, titled(1, n.Headline))
	}
	if gist := words(n.Gist); gist != nil {
		blocks = append(blocks, node{Type: "paragraph", Content: gist})
	}
	points("", n.Conclusions)
	todos()
	points("还没定", n.Open)
	topics()
	return node{Type: "doc", Content: blocks}
}

var spaces = regexp.MustCompile(`[ \x{2060}]+`)

// inline writes what a block says. A moment of the meeting reads [12:03],
// unless bare leaves it and who is to do a task out.
func inline(nodes []node, bare bool) string {
	var b strings.Builder
	for _, n := range nodes {
		switch n.Type {
		case "text":
			b.WriteString(n.Text)
		case "hardBreak":
			b.WriteByte('\n')
		case "doubt":
			b.WriteString("（存疑）")
		case "assignee":
			var label []string
			for _, key := range []string{"owner", "due"} {
				if part := n.str(key); part != "" {
					label = append(label, part)
				}
			}
			if len(label) > 0 && !bare {
				b.WriteString("（" + strings.Join(label, " · ") + "）")
			}
		case "timestamp":
			if !bare {
				b.WriteString(" [" + Clock(n.num("ms")) + "]")
			}
		}
	}
	text := spaces.ReplaceAllString(b.String(), " ")
	return strings.TrimSpace(strings.ReplaceAll(text, " （", "（"))
}

func block(n node) string {
	switch n.Type {
	case "heading":
		if n.num("level") == 1 {
			return "## " + inline(n.Content, false)
		}
		return "### " + inline(n.Content, false)
	case "list":
		mark := "-"
		switch checked, _ := n.Attrs["checked"].(bool); {
		case n.str("kind") == "ordered":
			mark = "1."
		case n.str("kind") == "task" && checked:
			mark = "- [x]"
		case n.str("kind") == "task":
			mark = "- [ ]"
		}
		var lines []string
		for i, child := range n.Content {
			text := block(child)
			if i == 0 {
				lines = append(lines, mark+" "+text)
			} else if strings.TrimSpace(text) != "" {
				lines = append(lines, "  "+strings.ReplaceAll(text, "\n", "\n  "))
			}
		}
		if lines == nil {
			return mark
		}
		return strings.Join(lines, "\n")
	}
	for _, child := range n.Content {
		if child.Type != "text" && len(child.Content) > 0 {
			parts := make([]string, len(n.Content))
			for i, child := range n.Content {
				parts[i] = block(child)
			}
			return strings.Join(parts, "\n")
		}
	}
	return inline(n.Content, false)
}

// read returns a meeting's notes as their document: the one the user made
// of them when there is one.
func read(kept store.StoredNotes) node {
	var doc node
	if kept.Doc == "" || json.Unmarshal([]byte(kept.Doc), &doc) != nil {
		var written notes.Notes
		_ = json.Unmarshal([]byte(kept.Data), &written)
		doc = document(written)
	}
	return doc
}

// Notes writes a meeting's notes as Markdown: what it settled, what is to
// do, what is left open and what it went through, each with the moment of
// the meeting it rests on. It returns "" when the meeting has no notes.
func Notes(kept store.StoredNotes) string {
	doc := read(kept)
	var b strings.Builder
	previous := ""
	for _, n := range doc.Content {
		text := block(n)
		if strings.TrimSpace(text) == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
			if n.Type != "list" || previous != "list" {
				b.WriteByte('\n')
			}
		}
		b.WriteString(text)
		previous = n.Type
	}
	return b.String()
}

// Task is something a meeting left to do.
type Task struct {
	Text  string `json:"text"`
	Owner string `json:"owner"`
	Due   string `json:"due"`
	// Done tells that the user checked it off.
	Done bool `json:"done"`
	// Seq and AtMs are the utterance it rests on and when it is said, 0
	// for a task the user wrote.
	Seq  int   `json:"seq"`
	AtMs int64 `json:"atMs"`
}

// Tasks returns what a meeting's notes leave to do, in their order.
func Tasks(kept store.StoredNotes) []Task {
	doc := read(kept)
	tasks := []Task{}
	var walk func(nodes []node)
	walk = func(nodes []node) {
		for _, n := range nodes {
			if n.Type == "list" && n.str("kind") == "task" && len(n.Content) > 0 {
				task := Task{Text: inline(n.Content[0].Content, true)}
				task.Done, _ = n.Attrs["checked"].(bool)
				for _, part := range n.Content[0].Content {
					switch part.Type {
					case "assignee":
						task.Owner, task.Due = part.str("owner"), part.str("due")
					case "timestamp":
						task.Seq, task.AtMs = int(part.num("seq")), part.num("ms")
					}
				}
				if task.Text != "" {
					tasks = append(tasks, task)
				}
				walk(n.Content[1:])
				continue
			}
			walk(n.Content)
		}
	}
	walk(doc.Content)
	return tasks
}
