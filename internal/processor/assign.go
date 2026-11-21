package processor

import (
	"github.com/xlsama/meetkeep/internal/asr"
	"github.com/xlsama/meetkeep/modules/speech"
)

// certain is the share of a sentence one speaker must hold for the
// sentence to be theirs without doubt.
const certain = 0.6

// assign tells who says each phrase from who speaks when: the speaker who
// talks the most while it is said, or the one who talks nearest to it.
// uncertain marks the phrases shared with another speaker and those said
// while nobody was found speaking.
func assign(phrases []asr.Phrase, segments []speech.Segment) (speakers []int, uncertain []bool) {
	speakers = make([]int, len(phrases))
	uncertain = make([]bool, len(phrases))
	for i, p := range phrases {
		overlap := map[int]int64{}
		var total int64
		nearest, distance := 0, int64(-1)
		for _, s := range segments {
			start, end := int64(s.StartMs), int64(s.EndMs)
			if shared := min(end, p.EndMs) - max(start, p.StartMs); shared > 0 {
				overlap[s.Speaker] += shared
				total += shared
				continue
			}
			gap := max(start-p.EndMs, p.StartMs-end)
			if distance < 0 || gap < distance {
				nearest, distance = s.Speaker, gap
			}
		}
		if total == 0 {
			speakers[i], uncertain[i] = nearest, true
			continue
		}
		best, most := 0, int64(-1)
		for speaker, shared := range overlap {
			if shared > most || shared == most && speaker < best {
				best, most = speaker, shared
			}
		}
		speakers[i] = best
		uncertain[i] = float64(most) < certain*float64(total)
	}
	return speakers, uncertain
}
