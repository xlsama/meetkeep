// Package polish makes a transcript easier to read without changing what it
// says: it takes out filler and sets right the names speech recognition
// misheard.
package polish

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode"
)

// Line is an utterance to polish.
type Line struct {
	Seq  int
	Text string
}

// Edit is what polishing made of a line: another text, or nothing when the
// line is only filler.
type Edit struct {
	Seq    int
	Text   string
	Filler bool
}

// Term is a name that was misheard throughout and what it is.
type Term struct {
	Heard string `json:"heard"`
	Meant string `json:"meant"`
	Why   string `json:"why"`
}

type Result struct {
	// Edits are the lines that changed.
	Edits []Edit
	Terms []Term
	// Refused is how many of the model's edits were not taken because
	// they did more than take words out.
	Refused int
	// Failed is how many parts of the transcript the model did not answer
	// for, asked twice, and Failure why the last of them.
	Failed  int
	Failure string
}

// Completer asks a language model for a JSON object: with CompleteQuick,
// one that takes no reasoning to give.
type Completer interface {
	CompleteJSON(ctx context.Context, system, user string) (string, error)
	CompleteQuick(ctx context.Context, system, user string) (string, error)
}

const (
	// part is how many lines the model is given at once, parallel how many
	// such parts are asked for at a time.
	part     = 60
	parallel = 8
	// A line longer than short is never taken for filler, and keeps at
	// least keepPercent of its letters.
	short       = 12
	keepPercent = 40
	// The names are looked for in at most this much of the transcript.
	readable = 60000
	// At most terms names are set right, each of shortest to longest
	// letters: of two letters, too many are words of their own.
	terms    = 12
	shortest = 3
	longest  = 8
)

const fillerPrompt = `你在整理会议的逐句转写，让它更好读。每行是「编号 原话」。你只能删字，不能做别的：
- 删掉语气词和口头填充：呃、嗯、啊、哎、那个、就是说，以及只起填充作用的「然后」「就是」「这个」；
- 删掉口吃式的重复和说到一半重来的碎片，例如「这个这个」留一个，「那-那我」留「那我」，「它它有它要有个顺序」留「它要有个顺序」；
- 一整行只是附和或应声、不带任何信息（如「对对对」「嗯」「是的是的」「是吧？」「OK」），把它的 text 写成空字符串，表示整行去掉。但如果它是在回答问题或表示决定，要保留；
- 不许换词、补词、调整语序、改写句式，不许合并或拆分行，不许纠正你认为的错别字；标点可以随删除调整；
- 拿不准就不动。
只输出有改动的行，没改动的行不要输出。输出 JSON：{"lines":[{"n":编号,"text":"删改后的话"}]}`

const termsPrompt = `下面是一场会议的逐句转写。语音识别有时会把专有名词（产品、系统、公司、术语的名字）听成读音相同的别的字。请找出这类错误，必须同时满足：
1. 这个词在文中出现至少两次；
2. 上下文能明确看出它本来是哪几个字，例如全文都在谈报税，那么「抱税表」应是「报税表」；
3. 改正后与原词读音相同或几乎相同，字数相同；
4. 它在文中每一处都该这样改。只要有一处它就是字面的意思，就不要列。
普通词语、拿不准的人名、你只是觉得更常见的写法，都不要列。宁可漏掉，不要猜。
输出 JSON：{"terms":[{"heard":"听成的词","meant":"应该是的词","why":"一句话依据"}]}，没有就输出 {"terms":[]}`

// Polish returns what to change in lines. known are names the user gave as
// spelled right. It fails only when the model answered for no part of the
// transcript.
func Polish(ctx context.Context, model Completer, lines []Line, known []string) (Result, error) {
	var result Result
	if len(lines) == 0 {
		return result, nil
	}
	var wg sync.WaitGroup
	var found []Term
	wg.Add(1)
	go func() {
		defer wg.Done()
		found = names(ctx, model, lines, known)
	}()

	edited := map[int]string{}
	var mu sync.Mutex
	var failure error
	parts := 0
	slots := make(chan struct{}, parallel)
	for start := 0; start < len(lines); start += part {
		chunk := lines[start:min(start+part, len(lines))]
		parts++
		wg.Add(1)
		slots <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-slots }()
			edits, err := fillers(ctx, model, chunk)
			if err != nil && ctx.Err() == nil {
				edits, err = fillers(ctx, model, chunk)
			}
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				result.Failed++
				failure = err
				return
			}
			for seq, text := range edits {
				edited[seq] = text
			}
		}()
	}
	wg.Wait()
	if failure != nil {
		result.Failure = failure.Error()
	}
	if result.Failed == parts {
		return result, fmt.Errorf("整理文字记录失败：%w", failure)
	}

	result.Terms = found
	var pairs []string
	for _, term := range found {
		pairs = append(pairs, term.Heard, term.Meant)
	}
	replacer := strings.NewReplacer(pairs...)
	for _, line := range lines {
		text := line.Text
		if next, ok := edited[line.Seq]; ok {
			switch kept, filler := taken(line.Text, next); {
			case filler:
				result.Edits = append(result.Edits, Edit{Seq: line.Seq, Filler: true})
				continue
			case kept:
				text = next
			default:
				result.Refused++
			}
		}
		if text = replacer.Replace(text); text != line.Text {
			result.Edits = append(result.Edits, Edit{Seq: line.Seq, Text: text})
		}
	}
	return result, nil
}

// letters returns the letters and digits of s, in lower case.
func letters(s string) []rune {
	var kept []rune
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			kept = append(kept, unicode.ToLower(r))
		}
	}
	return kept
}

// taken reports whether edited may stand for original: it only leaves
// letters of it out, in their order, and not most of a long line. filler
// tells that it leaves all of a short line out.
func taken(original, edited string) (kept, filler bool) {
	from, to := letters(original), letters(edited)
	if len(to) == 0 {
		return false, len(from) <= short
	}
	if len(from) > short && len(to)*100 < len(from)*keepPercent {
		return false, false
	}
	i := 0
	for _, r := range from {
		if i < len(to) && to[i] == r {
			i++
		}
	}
	return i == len(to), false
}

func fillers(ctx context.Context, model Completer, lines []Line) (map[int]string, error) {
	var user strings.Builder
	for _, line := range lines {
		fmt.Fprintf(&user, "%d %s\n", line.Seq, line.Text)
	}
	answer, err := model.CompleteQuick(ctx, fillerPrompt, user.String())
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Lines []struct {
			N    int    `json:"n"`
			Text string `json:"text"`
		} `json:"lines"`
	}
	if err := json.Unmarshal([]byte(answer), &parsed); err != nil {
		return nil, errors.New("模型返回的不是约定的格式")
	}
	given := map[int]bool{}
	for _, line := range lines {
		given[line.Seq] = true
	}
	edits := map[int]string{}
	for _, line := range parsed.Lines {
		if given[line.N] {
			edits[line.N] = strings.TrimSpace(line.Text)
		}
	}
	return edits, nil
}

// names returns the names misheard throughout lines. What the model is
// unsure of, and what fails, yields none.
func names(ctx context.Context, model Completer, lines []Line, known []string) []Term {
	var text strings.Builder
	for _, line := range lines {
		text.WriteString(line.Text)
		text.WriteByte('\n')
	}
	whole := text.String()
	shown := []rune(whole)
	if len(shown) > readable {
		shown = shown[:readable]
	}
	system := termsPrompt
	if len(known) > 0 {
		system += "\n已知写法正确的词：" + strings.Join(known, "、")
	}
	answer, err := model.CompleteJSON(ctx, system, string(shown))
	if err != nil {
		return nil
	}
	var parsed struct {
		Terms []Term `json:"terms"`
	}
	if json.Unmarshal([]byte(answer), &parsed) != nil {
		return nil
	}
	right := map[string]bool{}
	for _, word := range known {
		right[word] = true
	}
	var found []Term
	seen := map[string]bool{}
	for _, term := range parsed.Terms {
		term.Heard, term.Meant = strings.TrimSpace(term.Heard), strings.TrimSpace(term.Meant)
		if len(found) == terms || seen[term.Heard] || right[term.Heard] || !alike(term.Heard, term.Meant) {
			continue
		}
		if strings.Count(whole, term.Heard) < 2 {
			continue
		}
		seen[term.Heard] = true
		found = append(found, term)
	}
	return found
}

// alike reports whether meant could be heard as heard: a name of as many
// letters, at least half of them the same.
func alike(heard, meant string) bool {
	a, b := []rune(heard), []rune(meant)
	if heard == meant || len(a) != len(b) || len(a) < shortest || len(a) > longest {
		return false
	}
	differing := 0
	for i := range a {
		if a[i] != b[i] {
			differing++
		}
	}
	return differing*2 <= len(a)
}
