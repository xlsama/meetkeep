package notes

import (
	"bytes"
	"encoding/json"
	"strings"
)

// common tells the names that are words of their own, which the notes may
// use without meaning the speaker.
func common(name string) bool {
	return name == "我" || name == "对方" || len([]rune(name)) < 2
}

func digit(b byte) bool { return b >= '0' && b <= '9' }

// rename returns text with the name where old stood. "说话人 1" is left
// alone where it starts "说话人 12".
func rename(text, old, name string) string {
	if old == "" || old == name || !strings.Contains(text, old) {
		return text
	}
	var b strings.Builder
	for {
		i := strings.Index(text, old)
		if i < 0 {
			break
		}
		rest := text[i+len(old):]
		b.WriteString(text[:i])
		if digit(old[len(old)-1]) && rest != "" && digit(rest[0]) {
			b.WriteString(old)
		} else {
			b.WriteString(name)
		}
		text = rest
	}
	b.WriteString(text)
	return b.String()
}

// Renamed returns the notes as they read once a speaker called old is
// called name: who owes a todo always follows, what the notes say does
// unless old is a word of its own.
func (n Notes) Renamed(old, name string) Notes {
	said := func(text string) string {
		if common(old) {
			return text
		}
		return rename(text, old, name)
	}
	points := func(points []Point) []Point {
		renamed := make([]Point, len(points))
		for i, p := range points {
			p.Text = said(p.Text)
			renamed[i] = p
		}
		return renamed
	}
	n.Headline, n.Gist = said(n.Headline), said(n.Gist)
	n.Conclusions, n.Open = points(n.Conclusions), points(n.Open)
	topics := make([]Topic, len(n.Topics))
	for i, t := range n.Topics {
		t.Title, t.Points, t.Nodes = said(t.Title), points(t.Points), points(t.Nodes)
		topics[i] = t
	}
	todos := make([]Todo, len(n.Todos))
	for i, t := range n.Todos {
		t.Text, t.Owner = said(t.Text), rename(t.Owner, old, name)
		todos[i] = t
	}
	chapters := make([]Chapter, len(n.Chapters))
	for i, c := range n.Chapters {
		c.Title, c.Text = said(c.Title), said(c.Text)
		chapters[i] = c
	}
	n.Topics, n.Todos, n.Chapters = topics, todos, chapters
	return n
}

// RenamedDoc does to the document the user made of the notes what Renamed
// does to the notes. A document it cannot read is returned as it is.
func RenamedDoc(doc, old, name string) string {
	decoder := json.NewDecoder(strings.NewReader(doc))
	decoder.UseNumber()
	var root any
	if doc == "" || decoder.Decode(&root) != nil {
		return doc
	}
	var walk func(node any)
	walk = func(node any) {
		switch node := node.(type) {
		case []any:
			for _, child := range node {
				walk(child)
			}
		case map[string]any:
			if text, ok := node["text"].(string); ok && node["type"] == "text" && !common(old) {
				node["text"] = rename(text, old, name)
			}
			if attrs, ok := node["attrs"].(map[string]any); ok && node["type"] == "assignee" {
				if owner, ok := attrs["owner"].(string); ok {
					attrs["owner"] = rename(owner, old, name)
				}
			}
			walk(node["content"])
		}
	}
	walk(root)
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	if encoder.Encode(root) != nil {
		return doc
	}
	return strings.TrimSpace(out.String())
}

// RenamedText does the same to the words kept of that document to search.
func RenamedText(text, old, name string) string {
	if common(old) {
		return text
	}
	return rename(text, old, name)
}
