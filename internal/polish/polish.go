// Package polish makes a transcript easier to read without changing what it
// says: it takes out filler and sets right what speech recognition
// misheard.
package polish

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
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
	// Remembered tells that earlier meetings or the user set it right
	// before, and Local that the lines told of it, not the whole.
	Remembered bool `json:"remembered,omitempty"`
	Local      bool `json:"local,omitempty"`
}

// Remembered is a word set right before this meeting.
type Remembered struct {
	Heard string
	Meant string
	// Confirmed tells that the user said so: it is taken without asking
	// the model whether this meeting means it.
	Confirmed bool
}

// Memory is what is known of the words of a meeting before it is read.
type Memory struct {
	// Known are words spelled right: no reading of the meeting sets them
	// to anything else.
	Known []string
	// Terms are words misheard in earlier meetings and what they were, the
	// surest first.
	Terms []Remembered
	// Dropped are what the user said is not misheard: no reading of the
	// meeting sets one to the other.
	Dropped []Remembered
}

// Learned is a word that meetings set right, as it is kept between them.
type Learned struct {
	Heard string
	Meant string
	// Meetings is how many meetings set it right this way.
	Meetings int
	// Confirmed and Dropped tell what the user said of it.
	Confirmed bool
	Dropped   bool
}

// sure is how many meetings must have set a word right the same way for
// what it was set to to count as spelled right.
const sure = 2

// Recall makes of what was learned, and of the words the user gave as
// spelled right, what a meeting is read with. Of the ways a word was set
// right, the user's counts, then the one of the most meetings; a word set
// right to one that was itself set right is set to the last of them.
func Recall(learned []Learned, known []string) Memory {
	memory := Memory{Known: slices.Clone(known)}
	sorted := slices.Clone(learned)
	sort.SliceStable(sorted, func(a, b int) bool {
		if sorted[a].Confirmed != sorted[b].Confirmed {
			return sorted[a].Confirmed
		}
		if sorted[a].Meetings != sorted[b].Meetings {
			return sorted[a].Meetings > sorted[b].Meetings
		}
		return sorted[a].Heard < sorted[b].Heard
	})
	best := map[string]Learned{}
	for _, term := range sorted {
		if term.Dropped {
			memory.Dropped = append(memory.Dropped, Remembered{Heard: term.Heard, Meant: term.Meant})
		} else if _, ok := best[term.Heard]; !ok && term.Heard != term.Meant {
			best[term.Heard] = term
		}
	}
	right := map[string]bool{}
	for _, word := range known {
		right[word] = true
	}
	for _, term := range sorted {
		if term.Dropped || best[term.Heard] != term || right[term.Heard] {
			continue
		}
		meant := term.Meant
		for range 3 {
			next, ok := best[meant]
			if !ok || next.Meant == term.Heard || len([]rune(next.Meant)) != len([]rune(term.Heard)) {
				break
			}
			meant = next.Meant
		}
		memory.Terms = append(memory.Terms, Remembered{Heard: term.Heard, Meant: meant, Confirmed: term.Confirmed})
		if (term.Confirmed || term.Meetings >= sure) && !right[meant] {
			right[meant] = true
			memory.Known = append(memory.Known, meant)
		}
	}
	return memory
}

type Result struct {
	// Edits are the lines that changed.
	Edits []Edit
	Terms []Term
	// Refused is how many of the model's edits were not taken because
	// they rewrote a line more than lightly.
	Refused int
	// Failed is how many parts of the transcript the model did not answer
	// for, asked three times, and Failure why the last of them.
	Failed  int
	Failure string
	// Unread tells why the model did not say what was misheard, asked twice.
	Unread string
	// Read is how long finding what was misheard took: the lines wait for
	// it to be polished.
	Read time.Duration
	// Reviewed is how many lines were looked at once more, Revised how
	// many of them that changed, and Review how long it took.
	Reviewed int
	Revised  int
	Review   time.Duration
	// Guesses are the words the model took for misheard and left as they
	// were, not sure enough of them: the user is asked.
	Guesses []Guess
}

// Guess is a word of a line that may be misheard, and what it may be.
type Guess struct {
	Seq   int    `json:"seq"`
	Heard string `json:"heard"`
	Meant string `json:"meant"`
}

// Completer asks a language model for a JSON object.
type Completer interface {
	CompleteJSON(ctx context.Context, system, user string) (string, error)
}

const (
	// part is how many lines the model is given at once, parallel how many
	// such parts are asked for at a time: more, and a service turns away
	// those past what it takes from an account at once.
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
	// The misheard words are named by readers at once, of which the first
	// heard are believed.
	readers = 3
	heard   = 2
	// At most terms words are set right, each of up to longest letters.
	// Those of fewer than distinct are too often words of their own to be
	// replaced throughout.
	terms    = 20
	distinct = 3
	longest  = 8
	// At most guesses words are left for the user to say.
	guesses = 30
)

// again is how long a part the model did not answer for waits before it is
// asked for once more, each time it fails: a service that is busy turns
// requests away for a while.
var again = []time.Duration{2 * time.Second, 8 * time.Second}

// The whole transcript leads every request, as what the service is to know
// before it is asked anything: a service that keeps what requests start
// with reads it once for all of them, and each part is polished by one who
// has read the meeting.
const wholePrompt = `下面是一场会议的逐句转写，每行是「编号 原话」。它是语音识别的结果：口语很重，也有听错的字。先通读全文，弄清这场会在谈什么、有哪些人和系统、用了哪些术语，再按用户的要求处理。

`

const fillerPrompt = `你在整理上面这场会议的逐句转写，让没到场的人也能读懂。全文供你理解上下文、认出术语和人名；这一次只修订下面列出的这些行。对每一行做最轻的修订：
- 删掉语气词和口头填充：呃、嗯、啊、哎、那个、就是说，以及只起填充作用的「然后」「就是」「这个」；
- 删掉口吃式的重复和说到一半重来的碎片，例如「这个这个」留一个，「那-那我」留「那我」；
- 改正听错的字词：从全文能确定它本来是哪个词的同音、近音错字，例如谈报销时的「保销」应是「报销」。同一个名字、同一个术语，全文怎么写这里就怎么写；
- 实在读不通的地方可以调顺，只动必要的几个字；
- 一整行只是附和或应声、不带任何信息（如「对对对」「嗯」「是的是的」「是吧？」「OK」），把它的 text 写成空字符串，表示整行去掉。但如果它是在回答问题或表示决定，要保留；
- 不许补充原话没有的内容，不许概括、合并或拆分行，不许改变说话人的意思和语气；数字、金额、日期、人名照原样；
- 拿不准就不动。
只输出有改动的行，没改动的行不要输出。fixed 列出你在这些行里改正的听错词（听成的写法和改成的写法字数相同；语气词和重复不算），没有就给空数组。能从全文确定的照常直接改，不要因为有 unsure 就少改。unsure 只列你确定不了、所以没改的词：这个写法放在这句话里讲不通或很别扭，而有一个读音相近的词放进去就通了。n 是它所在的行，heard 是那一行里的写法（至少两个字，必要时带上相邻的字），meant 是你猜的写法，两者字数相同。只是口语不规范、换个说法也行的不算；没有就给空数组。
输出 JSON：{"lines":[{"n":编号,"text":"修订后的话"}],"fixed":[{"heard":"听成的词","meant":"改成的词"}],"unsure":[{"n":编号,"heard":"文中的写法","meant":"猜的写法"}]}`

const reviewPrompt = `你在整理上面这场会议的逐句转写。下面这些行已经修订过一轮，是挑出来复查的：每行先是「编号 现在的写法」，下一行「——」后面是要你复查的事。规则和第一轮一样：只删语气词和重复、改正听错的字词，不补充、不改写、不改变意思，数字和人名照原样；拿不准就不动。
只输出复查后要改的行，确认不用改的行不要输出。
输出 JSON：{"lines":[{"n":编号,"text":"最终的写法"}]}`

const termsPrompt = `通读上面这场会议的转写。语音识别会把词听成读音相同或相近的别的字，尤其是专有名词（产品、系统、公司的名字）和这个行业的术语。列出这类听错的词，必须同时满足：
1. 这个写法在文中出现至少两次；
2. 上下文能明确看出它本来是哪个词，例如全文都在谈报税，那么「抱税表」应是「报税表」；
3. 改正后与原词读音相同或几乎相同，字数相同。
只是你觉得更常见的写法、拿不准的人名，都不要列。宁可漏掉，不要猜。
输出 JSON：{"terms":[{"heard":"听成的词","meant":"应该是的词","why":"一句话依据"}]}，没有就输出 {"terms":[]}`

// whole returns the transcript as every request reads it.
func whole(lines []Line) string {
	var b strings.Builder
	b.WriteString(wholePrompt)
	for _, line := range lines {
		fmt.Fprintf(&b, "%d %s\n", line.Seq, line.Text)
	}
	return b.String()
}

// Polish returns what to change in lines, with what memory holds of their
// words. It fails only when the model answered for no part of the
// transcript.
func Polish(ctx context.Context, model Completer, lines []Line, memory Memory) (Result, error) {
	var result Result
	if len(lines) == 0 {
		return result, nil
	}
	read := whole(lines)
	started := time.Now()
	found, err := misheard(ctx, model, read, lines, memory)
	if err != nil {
		result.Unread = err.Error()
	}
	result.Read = time.Since(started)

	asked := fillerPrompt
	if hint := hinted(found); hint != "" {
		asked += "\n这场会议里已经确认的听错写法（听成 → 应为）。遇到时，那一句说的确实是后者才改：" + hint
	}
	var parts [][]Line
	for start := 0; start < len(lines); start += part {
		parts = append(parts, lines[start:min(start+part, len(lines))])
	}
	edited := map[int]string{}
	fixed := map[Term]int{}
	var unsure []Guess
	var mu sync.Mutex
	var failure error
	several(ctx, len(parts), func(i int) {
		var user strings.Builder
		user.WriteString(asked)
		user.WriteString("\n\n要修订的行：\n")
		for _, line := range parts[i] {
			fmt.Fprintf(&user, "%d %s\n", line.Seq, line.Text)
		}
		edits, told, wondered, err := revised(ctx, model, read, user.String(), parts[i])
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
		for _, term := range told {
			fixed[term]++
		}
		unsure = append(unsure, wondered...)
	})
	if failure != nil {
		result.Failure = failure.Error()
	}
	if result.Failed == len(parts) {
		return result, fmt.Errorf("整理文字失败：%w", failure)
	}
	found = append(found, local(fixed, found, lines, memory)...)
	result.Terms = found

	var pairs, every []string
	for _, term := range found {
		if term.Throughout {
			pairs = append(pairs, term.Heard, term.Meant)
		}
		every = append(every, term.Heard, term.Meant)
	}
	replacer, confirmed := strings.NewReplacer(pairs...), strings.NewReplacer(every...)

	// settle tells what an edit makes of a line: its text, whether it
	// leaves the line out, and whether it was refused.
	settle := func(line Line, next string) (text string, filler, refused bool) {
		kept, filler := taken(line.Text, next)
		if !kept && !filler {
			// What was confirmed as misheard does not count as rewriting.
			kept, _ = taken(confirmed.Replace(line.Text), next)
		}
		switch {
		case filler:
			return "", true, false
		case kept:
			return next, false, false
		}
		return line.Text, false, true
	}

	// A guess at a word this meeting or earlier ones spell is worth another
	// look: the reader is told that the word is one of them.
	named := map[string]bool{}
	for _, term := range found {
		named[term.Meant] = true
	}
	for _, word := range memory.Known {
		named[word] = true
	}
	wondered := map[int][]Guess{}
	for _, guess := range unsure {
		if named[guess.Meant] && guess.Heard != "" && guess.Heard != guess.Meant {
			wondered[guess.Seq] = append(wondered[guess.Seq], guess)
		}
	}

	started = time.Now()
	doubts := doubted(lines, edited, found, wondered, settle)
	result.Reviewed = len(doubts)
	if len(doubts) > 0 {
		groups := slices.Collect(slices.Chunk(doubts, part))
		several(ctx, len(groups), func(i int) {
			var user strings.Builder
			user.WriteString(reviewPrompt)
			user.WriteString("\n\n要复查的行：\n")
			given := make([]Line, len(groups[i]))
			for k, d := range groups[i] {
				fmt.Fprintf(&user, "%d %s\n—— %s\n", d.line.Seq, d.now, d.why)
				given[k] = d.line
			}
			edits, _, _, err := revised(ctx, model, read, user.String(), given)
			if err != nil {
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, d := range groups[i] {
				next, ok := edits[d.line.Seq]
				if !ok || next == d.now {
					continue
				}
				if _, _, refused := settle(d.line, next); !refused {
					edited[d.line.Seq] = next
					result.Revised++
				}
			}
		})
	}
	result.Review = time.Since(started)

	reads := map[int]string{}
	answered := false
	for _, line := range lines {
		text := line.Text
		if echo(text) && !answered {
			result.Edits = append(result.Edits, Edit{Seq: line.Seq, Filler: true})
			continue
		}
		answered = strings.HasSuffix(strings.TrimSpace(text), "？") || strings.HasSuffix(strings.TrimSpace(text), "?")
		if next, ok := edited[line.Seq]; ok {
			settled, filler, refused := settle(line, next)
			if filler {
				result.Edits = append(result.Edits, Edit{Seq: line.Seq, Filler: true})
				continue
			}
			if refused {
				result.Refused++
			}
			text = settled
		}
		if text = replacer.Replace(text); text != line.Text {
			result.Edits = append(result.Edits, Edit{Seq: line.Seq, Text: text})
		}
		reads[line.Seq] = text
	}
	result.Guesses = guessed(unsure, reads, found, memory)
	return result, nil
}

// guessed returns, of the words the model was unsure of, those that still
// stand in their lines as reads has them and that nothing settled: the
// ones more lines spell so first, then in the order of the lines.
func guessed(unsure []Guess, reads map[int]string, found []Term, memory Memory) []Guess {
	settled := map[string]bool{}
	for _, term := range found {
		settled[term.Heard] = true
	}
	for _, word := range memory.Known {
		settled[word] = true
	}
	dropped := map[Remembered]bool{}
	for _, term := range memory.Dropped {
		dropped[Remembered{Heard: term.Heard, Meant: term.Meant}] = true
	}
	type place struct {
		seq   int
		heard string
	}
	seen := map[place]bool{}
	often := map[Remembered]int{}
	var kept []Guess
	for _, guess := range unsure {
		was, is := []rune(guess.Heard), []rune(guess.Meant)
		at := place{guess.Seq, guess.Heard}
		if seen[at] || settled[guess.Heard] || dropped[Remembered{Heard: guess.Heard, Meant: guess.Meant}] ||
			guess.Heard == guess.Meant || len(was) != len(is) || len(was) < 2 || len(was) > longest ||
			!strings.Contains(reads[guess.Seq], guess.Heard) {
			continue
		}
		seen[at] = true
		often[Remembered{Heard: guess.Heard, Meant: guess.Meant}]++
		kept = append(kept, guess)
	}
	sort.SliceStable(kept, func(a, b int) bool {
		most, fewer := often[Remembered{Heard: kept[a].Heard, Meant: kept[a].Meant}], often[Remembered{Heard: kept[b].Heard, Meant: kept[b].Meant}]
		if most != fewer {
			return most > fewer
		}
		return kept[a].Seq < kept[b].Seq
	})
	kept = kept[:min(len(kept), guesses)]
	sort.SliceStable(kept, func(a, b int) bool { return kept[a].Seq < kept[b].Seq })
	return kept
}

// several does count jobs, parallel of them at a time.
func several(ctx context.Context, count int, job func(i int)) {
	var wait sync.WaitGroup
	slots := make(chan struct{}, parallel)
	for i := range count {
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
			wait.Wait()
			return
		}
		wait.Go(func() {
			defer func() { <-slots }()
			job(i)
		})
	}
	wait.Wait()
}

func hinted(found []Term) string {
	var hint strings.Builder
	for _, term := range found {
		fmt.Fprintf(&hint, "\n%s → %s", term.Heard, term.Meant)
	}
	return hint.String()
}

// doubt is a line to look at once more: how it reads now, and what about it.
type doubt struct {
	line Line
	now  string
	why  string
}

// doubted returns the lines the first reading left in doubt: those whose
// edit was refused, those that still spell a word as other lines told it
// was misheard, which their own reading was not told, and those with a
// word it wondered about that may be one the meeting names.
func doubted(lines []Line, edited map[int]string, found []Term, wondered map[int][]Guess, settle func(Line, string) (string, bool, bool)) []doubt {
	var doubts []doubt
	for _, line := range lines {
		now := line.Text
		var why []string
		if next, ok := edited[line.Seq]; ok {
			text, filler, refused := settle(line, next)
			if filler {
				continue
			}
			if now = text; refused {
				why = append(why, fmt.Sprintf("上一轮改成「%s」，增删太多被退回。只做最轻的修订", next))
			}
		}
		for _, term := range found {
			if term.Local && strings.Contains(now, term.Heard) {
				why = append(why, fmt.Sprintf("别处把「%s」改成了「%s」，这一句如果也是这个意思就改，确实是原词就不动", term.Heard, term.Meant))
			}
		}
		for _, guess := range wondered[line.Seq] {
			if strings.Contains(now, guess.Heard) {
				why = append(why, fmt.Sprintf("上一轮怀疑这一句的「%s」是这场会里的「%s」，没有改。全文看确实是它就改，不是就不动", guess.Heard, guess.Meant))
			}
		}
		if len(why) > 0 {
			doubts = append(doubts, doubt{line, now, strings.Join(why, "；")})
		}
	}
	return doubts
}

// local returns the words the lines told were misheard that the whole did
// not: those more than one part of the transcript set right the same way,
// or that still stand as they were heard somewhere.
func local(fixed map[Term]int, found []Term, lines []Line, memory Memory) []Term {
	taken := map[string]bool{}
	for _, term := range found {
		taken[term.Heard] = true
	}
	for _, word := range memory.Known {
		taken[word] = true
	}
	dropped := map[Remembered]bool{}
	for _, term := range memory.Dropped {
		dropped[Remembered{Heard: term.Heard, Meant: term.Meant}] = true
	}
	var text strings.Builder
	for _, line := range lines {
		text.WriteString(line.Text)
		text.WriteByte('\n')
	}
	said := text.String()
	most := map[string]Term{}
	for term, count := range fixed {
		was, is := []rune(term.Heard), []rune(term.Meant)
		if taken[term.Heard] || dropped[Remembered{Heard: term.Heard, Meant: term.Meant}] || term.Heard == term.Meant || len(was) != len(is) || len(was) < 2 || len(was) > longest {
			continue
		}
		if count < 2 && strings.Count(said, term.Heard) < 2 {
			continue
		}
		if best, ok := most[term.Heard]; !ok || count > fixed[best] || count == fixed[best] && term.Meant < best.Meant {
			most[term.Heard] = term
		}
	}
	told := make([]Term, 0, len(most))
	for _, term := range most {
		term.Local = true
		told = append(told, term)
	}
	sort.Slice(told, func(a, b int) bool { return told[a].Heard < told[b].Heard })
	return told[:min(len(told), terms)]
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

// misheard returns the words misheard in lines: those the user set right
// before, and those the first readers to answer name, the ones more of
// them name first. It fails when no reader answers.
func misheard(ctx context.Context, model Completer, read string, lines []Line, memory Memory) ([]Term, error) {
	var text strings.Builder
	for _, line := range lines {
		text.WriteString(line.Text)
		text.WriteByte('\n')
	}
	said := text.String()
	var found []Term
	taken := map[string]bool{}
	before := map[string]string{}
	for _, term := range memory.Terms {
		if !strings.Contains(said, term.Heard) || taken[term.Heard] {
			continue
		}
		if term.Confirmed {
			taken[term.Heard] = true
			found = append(found, Term{Heard: term.Heard, Meant: term.Meant, Throughout: alike(term.Heard, term.Meant), Remembered: true})
		} else if _, ok := before[term.Heard]; !ok {
			before[term.Heard] = term.Meant
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	type reading struct {
		terms []Term
		err   error
	}
	readings := make(chan reading, readers)
	for range readers {
		go func() {
			terms, err := names(ctx, model, read, said, memory, before)
			readings <- reading{terms, err}
		}()
	}
	votes := map[Term]int{}
	var order []Term
	answered := 0
	var failure error
	for range readers {
		r := <-readings
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
		return found, failure
	}
	sort.SliceStable(order, func(a, b int) bool {
		return votes[Term{Heard: order[a].Heard, Meant: order[a].Meant}] > votes[Term{Heard: order[b].Heard, Meant: order[b].Meant}]
	})
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

// revised asks the model, which has read the whole transcript, for what
// user says of lines, and returns the lines it changed, the words it tells
// it set right and those of the lines it is unsure of. A request that
// fails is made again after a wait.
func revised(ctx context.Context, model Completer, read, user string, lines []Line) (map[int]string, []Term, []Guess, error) {
	answer, err := model.CompleteJSON(ctx, read, user)
	for _, wait := range again {
		if err == nil || ctx.Err() != nil {
			break
		}
		select {
		case <-time.After(wait):
		case <-ctx.Done():
		}
		answer, err = model.CompleteJSON(ctx, read, user)
	}
	if err != nil {
		return nil, nil, nil, err
	}
	var parsed struct {
		Lines []struct {
			N    int    `json:"n"`
			Text string `json:"text"`
		} `json:"lines"`
		Fixed  []Term `json:"fixed"`
		Unsure []struct {
			N     int    `json:"n"`
			Heard string `json:"heard"`
			Meant string `json:"meant"`
		} `json:"unsure"`
	}
	if err := json.Unmarshal([]byte(answer), &parsed); err != nil {
		return nil, nil, nil, errors.New("模型返回的不是约定的格式")
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
	told := make([]Term, 0, len(parsed.Fixed))
	for _, term := range parsed.Fixed {
		told = append(told, Term{Heard: strings.TrimSpace(term.Heard), Meant: strings.TrimSpace(term.Meant)})
	}
	var unsure []Guess
	for _, guess := range parsed.Unsure {
		if given[guess.N] {
			unsure = append(unsure, Guess{Seq: guess.N, Heard: strings.TrimSpace(guess.Heard), Meant: strings.TrimSpace(guess.Meant)})
		}
	}
	return edits, told, unsure, nil
}

// names returns the words misheard in what was said, as the model that
// read it tells them, and tells when it could not be asked. before are
// words earlier meetings set right, which count when said once.
func names(ctx context.Context, model Completer, read, said string, memory Memory, before map[string]string) ([]Term, error) {
	asked := termsPrompt
	if len(memory.Known) > 0 {
		asked += "\n已知写法正确的词：" + strings.Join(memory.Known, "、")
	}
	if len(before) > 0 {
		heard := slices.Sorted(func(yield func(string) bool) {
			for word := range before {
				if !yield(word) {
					return
				}
			}
		})
		asked += "\n以往的会议里确认过的听错写法（听成 → 应为）。这场会里说的是同一个东西时照样列出，只出现一次也算；说的不是它就不要列："
		for _, word := range heard {
			asked += "\n" + word + " → " + before[word]
		}
	}
	answer, err := model.CompleteJSON(ctx, read, asked)
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
	for _, word := range memory.Known {
		right[word] = true
	}
	dropped := map[Remembered]bool{}
	for _, term := range memory.Dropped {
		dropped[Remembered{Heard: term.Heard, Meant: term.Meant}] = true
	}
	var found []Term
	seen := map[string]bool{}
	for _, term := range parsed.Terms {
		term.Heard, term.Meant = strings.TrimSpace(term.Heard), strings.TrimSpace(term.Meant)
		was, is := []rune(term.Heard), []rune(term.Meant)
		if seen[term.Heard] || right[term.Heard] || term.Heard == term.Meant || dropped[Remembered{Heard: term.Heard, Meant: term.Meant}] {
			continue
		}
		term.Remembered = before[term.Heard] == term.Meant
		least := 2
		if term.Remembered {
			least = 1
		}
		if len(was) != len(is) || len(was) < 2 || len(was) > longest || strings.Count(said, term.Heard) < least {
			continue
		}
		seen[term.Heard] = true
		term.Throughout = alike(term.Heard, term.Meant)
		term.Local = false
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
