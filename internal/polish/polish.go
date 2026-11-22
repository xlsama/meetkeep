// Package polish makes a transcript easier to read without changing what it
// says: it takes out filler and sets right what speech recognition
// misheard.
package polish

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
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

// Term is a word that was misheard in the meeting and what it is.
type Term struct {
	Heard string `json:"heard"`
	Meant string `json:"meant"`
	Why   string `json:"why"`
	// Throughout tells that it is a name unlike any other word, replaced
	// wherever it stands. The others are replaced where a line means them.
	Throughout bool `json:"throughout"`
}

type Result struct {
	// Edits are the lines that changed.
	Edits []Edit
	Terms []Term
	// Refused is how many of the model's edits were not taken because
	// they rewrote a line more than lightly.
	Refused int
	// Failed is how many parts of the transcript the model did not answer
	// for, asked twice, and Failure why the last of them.
	Failed  int
	Failure string
	// Unread tells why the model did not say what was misheard, asked twice.
	Unread string
}

// Completer asks a language model for a JSON object.
type Completer interface {
	CompleteJSON(ctx context.Context, system, user string) (string, error)
}

const (
	// part is how many lines the model is given at once, parallel how many
	// such parts are asked for at a time.
	part     = 60
	parallel = 16
	// A line longer than short is never taken for filler, and keeps at
	// least keepPercent of its letters. An edit may bring in a letter for
	// every perNew of the line's, two at least, and never more than a
	// third of them.
	short       = 12
	keepPercent = 40
	perNew      = 10
	// A line of up to echoed letters, all of them echoes, only says yes.
	echoed = 8
	echoes = "嗯啊哦噢呃哎唉诶欸哈对是的吧好行嘛呀哇ok"
	// The names are looked for in at most this much of the transcript, by
	// readers at once of which the first heard are believed.
	readable = 60000
	readers  = 3
	heard    = 2
	// At most terms words are set right, each of up to longest letters.
	// Those of fewer than distinct are too often words of their own to be
	// replaced throughout.
	terms    = 20
	distinct = 3
	longest  = 8
)

const fillerPrompt = `你在整理会议的逐句转写，让没到场的人也能读懂。每行是「编号 原话」，是语音识别的结果：口语很重，也有听错的字。对每一行做最轻的修订：
- 删掉语气词和口头填充：呃、嗯、啊、哎、那个、就是说，以及只起填充作用的「然后」「就是」「这个」；
- 删掉口吃式的重复和说到一半重来的碎片，例如「这个这个」留一个，「那-那我」留「那我」；
- 改正明显听错的字词：从这一批的上下文能确定它本来是哪个词的同音、近音错字，例如谈报销时的「保销」应是「报销」。同一个名字前后写法保持一致；
- 实在读不通的地方可以调顺，只动必要的几个字；
- 一整行只是附和或应声、不带任何信息（如「对对对」「嗯」「是的是的」「是吧？」「OK」），把它的 text 写成空字符串，表示整行去掉。但如果它是在回答问题或表示决定，要保留；
- 不许补充原话没有的内容，不许概括、合并或拆分行，不许改变说话人的意思和语气；数字、金额、日期、人名照原样；
- 拿不准就不动。
只输出有改动的行，没改动的行不要输出。输出 JSON：{"lines":[{"n":编号,"text":"修订后的话"}]}`

const termsPrompt = `下面是一场会议的逐句转写。语音识别会把词听成读音相同或相近的别的字，尤其是专有名词（产品、系统、公司的名字）和这个行业的术语。请通读全文，列出这类听错的词，必须同时满足：
1. 这个写法在文中出现至少两次；
2. 上下文能明确看出它本来是哪个词，例如全文都在谈报税，那么「抱税表」应是「报税表」；
3. 改正后与原词读音相同或几乎相同，字数相同。
只是你觉得更常见的写法、拿不准的人名，都不要列。宁可漏掉，不要猜。
输出 JSON：{"terms":[{"heard":"听成的词","meant":"应该是的词","why":"一句话依据"}]}，没有就输出 {"terms":[]}`

// Polish returns what to change in lines. known are names the user gave as
// spelled right. It fails only when the model answered for no part of the
// transcript.
func Polish(ctx context.Context, model Completer, lines []Line, known []string) (Result, error) {
	var result Result
	if len(lines) == 0 {
		return result, nil
	}
	found, err := misheard(ctx, model, lines, known)
	if err != nil {
		result.Unread = err.Error()
	}
	result.Terms = found
	var pairs, every []string
	var hint strings.Builder
	for _, term := range found {
		if term.Throughout {
			pairs = append(pairs, term.Heard, term.Meant)
		}
		every = append(every, term.Heard, term.Meant)
		fmt.Fprintf(&hint, "\n%s → %s", term.Heard, term.Meant)
	}
	system := fillerPrompt
	if hint.Len() > 0 {
		system += "\n这场会议里已经确认的听错写法（听成 → 应为）。遇到时，那一句说的确实是后者才改：" + hint.String()
	}

	var wg sync.WaitGroup
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
			edits, err := fillers(ctx, model, system, chunk)
			if err != nil && ctx.Err() == nil {
				edits, err = fillers(ctx, model, system, chunk)
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

	replacer, confirmed := strings.NewReplacer(pairs...), strings.NewReplacer(every...)
	asked := false
	for _, line := range lines {
		text := line.Text
		if echo(text) && !asked {
			result.Edits = append(result.Edits, Edit{Seq: line.Seq, Filler: true})
			continue
		}
		asked = strings.HasSuffix(strings.TrimSpace(text), "？") || strings.HasSuffix(strings.TrimSpace(text), "?")
		if next, ok := edited[line.Seq]; ok {
			kept, filler := taken(line.Text, next)
			if !kept && !filler {
				// What was confirmed as misheard does not count as rewriting.
				kept, _ = taken(confirmed.Replace(line.Text), next)
			}
			switch {
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

// echo reports whether text only says yes: a few words of assent and
// nothing else. A line that follows a question may be its answer, which the
// caller keeps.
func echo(text string) bool {
	said := letters(text)
	if len(said) == 0 || len(said) > echoed {
		return false
	}
	for _, r := range said {
		if !strings.ContainsRune(echoes, r) {
			return false
		}
	}
	return true
}

// misheard returns the words misheard in lines as the first readers to
// answer name them, those more of them name first. It fails when none
// answers.
func misheard(ctx context.Context, model Completer, lines []Line, known []string) ([]Term, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	type reading struct {
		terms []Term
		err   error
	}
	read := make(chan reading, readers)
	for range readers {
		go func() {
			terms, err := names(ctx, model, lines, known)
			read <- reading{terms, err}
		}()
	}
	votes := map[Term]int{}
	var order []Term
	answered := 0
	var failure error
	for range readers {
		r := <-read
		if r.err != nil {
			failure = r.err
			continue
		}
		for _, term := range r.terms {
			key := Term{Heard: term.Heard, Meant: term.Meant}
			if votes[key] == 0 {
				order = append(order, term)
			}
			votes[key]++
		}
		if answered++; answered == heard {
			break
		}
	}
	if answered == 0 {
		return nil, failure
	}
	sort.SliceStable(order, func(a, b int) bool {
		return votes[Term{Heard: order[a].Heard, Meant: order[a].Meant}] > votes[Term{Heard: order[b].Heard, Meant: order[b].Meant}]
	})
	var found []Term
	taken := map[string]bool{}
	for _, term := range order {
		if len(found) < terms && !taken[term.Heard] {
			taken[term.Heard] = true
			found = append(found, term)
		}
	}
	return found, nil
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

// taken reports whether edited may stand for original: it is the line
// with letters left out and only a few brought in, not most of a long line
// gone, and no figure that the line does not have. filler tells that it
// leaves all of a short line out.
func taken(original, edited string) (kept, filler bool) {
	from, to := letters(original), letters(edited)
	if len(to) == 0 {
		return false, len(from) <= short
	}
	if len(from) > short && len(to)*100 < len(from)*keepPercent {
		return false, false
	}
	if !within(digits(to), digits(from)) {
		return false, false
	}
	allowed := min(max(2, len(from)/perNew), len(from)/3)
	return len(to)-common(from, to) <= allowed, false
}

func digits(letters []rune) []rune {
	var kept []rune
	for _, r := range letters {
		if unicode.IsDigit(r) {
			kept = append(kept, r)
		}
	}
	return kept
}

// within reports whether part is whole with some of it left out.
func within(part, whole []rune) bool {
	i := 0
	for _, r := range whole {
		if i < len(part) && part[i] == r {
			i++
		}
	}
	return i == len(part)
}

// common returns how many letters a and b share, in their order.
func common(a, b []rune) int {
	previous, current := make([]int, len(b)+1), make([]int, len(b)+1)
	for i := range a {
		for j := range b {
			if a[i] == b[j] {
				current[j+1] = previous[j] + 1
			} else {
				current[j+1] = max(previous[j+1], current[j])
			}
		}
		previous, current = current, previous
	}
	return previous[len(b)]
}

func fillers(ctx context.Context, model Completer, system string, lines []Line) (map[int]string, error) {
	var user strings.Builder
	for _, line := range lines {
		fmt.Fprintf(&user, "%d %s\n", line.Seq, line.Text)
	}
	answer, err := model.CompleteJSON(ctx, system, user.String())
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

// names returns the words misheard in lines, as the model reads them, and
// tells when it could not be asked.
func names(ctx context.Context, model Completer, lines []Line, known []string) ([]Term, error) {
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
		return nil, err
	}
	var parsed struct {
		Terms []Term `json:"terms"`
	}
	if json.Unmarshal([]byte(answer), &parsed) != nil {
		return nil, errors.New("模型返回的不是约定的格式")
	}
	right := map[string]bool{}
	for _, word := range known {
		right[word] = true
	}
	var found []Term
	seen := map[string]bool{}
	for _, term := range parsed.Terms {
		term.Heard, term.Meant = strings.TrimSpace(term.Heard), strings.TrimSpace(term.Meant)
		was, is := []rune(term.Heard), []rune(term.Meant)
		if seen[term.Heard] || right[term.Heard] || term.Heard == term.Meant {
			continue
		}
		if len(was) != len(is) || len(was) < 2 || len(was) > longest || strings.Count(whole, term.Heard) < 2 {
			continue
		}
		seen[term.Heard] = true
		term.Throughout = alike(term.Heard, term.Meant)
		found = append(found, term)
	}
	return found, nil
}

// alike reports whether heard is a name unlike any other word that could
// only be meant: long enough, and with at least half of its letters the
// same.
func alike(heard, meant string) bool {
	a, b := []rune(heard), []rune(meant)
	if heard == meant || len(a) != len(b) || len(a) < distinct || len(a) > longest {
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
