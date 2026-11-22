package processor

import (
	"context"
	"math"
	"os"
	"sort"

	"github.com/xlsama/meetkeep/internal/store"
	"github.com/xlsama/meetkeep/modules/speech"
)

const (
	// A voice shorter than shortestVoice cannot be told from others.
	shortestVoice = 600
	longestVoice  = 20000
	// A speaker's voiceprint is that of up to voiceOfSpeaker of their
	// longest sentences.
	voiceOfSpeaker = 60000
	// Sentences shorter than recheckedBelow are checked by voice, the
	// kind that overlap alone gets wrong; at most rechecked of them.
	recheckedBelow = 4000
	rechecked      = 400
	// recheckMargin is how much more alike another speaker's voice must
	// be for a sentence to change hands; within it, the sentence is a guess.
	recheckMargin = 0.05
	// recognized is how alike a voice must be to a remembered one to take
	// its name: above it, two people are all but never mistaken.
	recognized = 0.52
)

func cosine(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / math.Sqrt(na*nb)
}

// mean returns the unit vector in the direction of the sum of vectors.
func mean(vectors [][]float32) []float32 {
	if len(vectors) == 0 {
		return nil
	}
	sum := make([]float32, len(vectors[0]))
	for _, v := range vectors {
		for i := range sum {
			sum[i] += v[i]
		}
	}
	var norm float64
	for _, v := range sum {
		norm += float64(v) * float64(v)
	}
	if norm == 0 {
		return nil
	}
	for i := range sum {
		sum[i] = float32(float64(sum[i]) / math.Sqrt(norm))
	}
	return sum
}

// recheck moves the sentences whose voice is another speaker's to that
// speaker, marks those between two voices as guesses and clears the doubt
// of the others. vectors holds the voiceprint of a sentence or nil.
func recheck(who []int, uncertain []bool, vectors [][]float32, voices map[int][]float32) {
	if len(voices) < 2 {
		return
	}
	for i, v := range vectors {
		own, ok := voices[who[i]]
		if v == nil || !ok {
			continue
		}
		mine := cosine(v, own)
		other, best := who[i], -2.0
		for speaker, voice := range voices {
			if c := cosine(v, voice); speaker != who[i] && c > best {
				other, best = speaker, c
			}
		}
		switch diff := best - mine; {
		case diff > recheckMargin:
			who[i], uncertain[i] = other, false
		case diff >= -recheckMargin:
			uncertain[i] = true
		default:
			uncertain[i] = false
		}
	}
}

// match gives each voice the remembered person it is most like, when it is
// alike enough; a person is given to the one voice most like theirs.
func match(voices map[int][]float32, people map[int][]float32) map[int]int {
	type pair struct {
		speaker, person int
		alike           float64
	}
	var pairs []pair
	for speaker, voice := range voices {
		for person, known := range people {
			if alike := cosine(voice, known); alike >= recognized {
				pairs = append(pairs, pair{speaker, person, alike})
			}
		}
	}
	sort.Slice(pairs, func(a, b int) bool { return pairs[a].alike > pairs[b].alike })
	found, taken := map[int]int{}, map[int]bool{}
	for _, p := range pairs {
		if _, has := found[p.speaker]; !has && !taken[p.person] {
			found[p.speaker], taken[p.person] = p.person, true
		}
	}
	return found
}

type voiced struct {
	// voices are the speakers' voiceprints, samples where each is heard.
	voices  map[int][]float32
	samples map[int]speech.Span
	// spans is how many stretches of the track were listened to.
	spans int
}

// listen takes the voiceprints of the sentences of a track, corrects who
// says them by voice and returns the voice of each speaker. Without the
// model it returns nothing and changes nothing.
func (p *Processor) listen(ctx context.Context, path string, sentences []store.Utterance, who []int, uncertain []bool) (voiced, error) {
	result := voiced{voices: map[int][]float32{}, samples: map[int]speech.Span{}}
	if _, err := os.Stat(p.Recognition); err != nil {
		return result, nil
	}
	usable := func(i int) bool {
		d := sentences[i].EndMs - sentences[i].StartMs
		return d >= shortestVoice && d <= longestVoice
	}
	chosen := map[int]bool{}
	bySpeaker := map[int][]int{}
	for i := range sentences {
		if usable(i) {
			bySpeaker[who[i]] = append(bySpeaker[who[i]], i)
		}
	}
	for speaker, indexes := range bySpeaker {
		sort.Slice(indexes, func(a, b int) bool {
			return sentences[indexes[a]].EndMs-sentences[indexes[a]].StartMs > sentences[indexes[b]].EndMs-sentences[indexes[b]].StartMs
		})
		var total int64
		for n, i := range indexes {
			if total >= voiceOfSpeaker {
				break
			}
			if n == 0 {
				result.samples[speaker] = speech.Span{StartMs: int(sentences[i].StartMs), EndMs: int(sentences[i].EndMs)}
			}
			chosen[i] = true
			total += sentences[i].EndMs - sentences[i].StartMs
		}
	}
	reference := map[int]bool{}
	for i := range chosen {
		reference[i] = true
	}
	checked := 0
	for i := range sentences {
		if checked < rechecked && usable(i) && !chosen[i] && (uncertain[i] || sentences[i].EndMs-sentences[i].StartMs < recheckedBelow) {
			chosen[i] = true
			checked++
		}
	}
	if len(chosen) == 0 {
		return result, nil
	}
	order := make([]int, 0, len(chosen))
	for i := range chosen {
		order = append(order, i)
	}
	sort.Ints(order)
	spans := make([]speech.Span, len(order))
	for n, i := range order {
		spans[n] = speech.Span{StartMs: int(sentences[i].StartMs), EndMs: int(sentences[i].EndMs)}
	}
	result.spans = len(spans)
	prints, err := p.Speech.Embed(ctx, path, p.Recognition, spans)
	if err != nil {
		return result, err
	}
	vectors := make([][]float32, len(sentences))
	for n, i := range order {
		if n < len(prints) && len(prints[n].Vector) > 0 {
			vectors[i] = prints[n].Vector
		}
	}
	centroids := func() map[int][]float32 {
		groups := map[int][][]float32{}
		for i := range reference {
			if vectors[i] != nil {
				groups[who[i]] = append(groups[who[i]], vectors[i])
			}
		}
		voices := map[int][]float32{}
		for speaker, group := range groups {
			if voice := mean(group); voice != nil {
				voices[speaker] = voice
			}
		}
		return voices
	}
	recheck(who, uncertain, vectors, centroids())
	result.voices = centroids()
	return result, nil
}
