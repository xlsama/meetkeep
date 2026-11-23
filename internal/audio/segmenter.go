package audio

// Window is how many samples one probability of speech covers: 32 ms.
const Window = 512

const (
	windowMs    = 32
	onset       = 0.5
	release     = 0.35
	onsetRun    = 2
	preRoll     = 10
	endSilence  = 19
	softLimit   = 312
	softSilence = 8
	hardLimit   = 625
	cutSearch   = 62
	minSpeech   = 8
	trailing    = 6
)

type Kind int

const (
	Began Kind = iota
	Ended
	Dropped
)

// Event tells that an utterance began, ended with its audio, or was too
// short to keep.
type Event struct {
	Kind    Kind
	StartMs int64
	EndMs   int64
	PCM     []int16
}

type window struct {
	pcm  []int16
	prob float32
}

// Segmenter cuts a stream of audio into utterances from the probability of
// speech in each window of it.
type Segmenter struct {
	count   int64
	history []window
	run     int

	active  bool
	start   int64
	windows []window
	silence int
}

// Push takes the next window of audio and returns what it changed.
func (s *Segmenter) Push(pcm []int16, prob float32) []Event {
	w := window{pcm: pcm, prob: prob}
	defer func() { s.count++ }()

	if !s.active {
		s.history = append(s.history, w)
		if len(s.history) > preRoll {
			s.history = s.history[1:]
		}
		if prob < onset {
			s.run = 0
			return nil
		}
		if s.run++; s.run < onsetRun {
			return nil
		}
		s.active, s.run, s.silence = true, 0, 0
		s.start = s.count - int64(len(s.history)) + 1
		s.windows = append(s.windows[:0], s.history...)
		s.history = s.history[:0]
		return []Event{{Kind: Began, StartMs: s.start * windowMs}}
	}

	s.windows = append(s.windows, w)
	if prob < release {
		s.silence++
	} else if prob >= onset {
		s.silence = 0
	}
	switch {
	case s.silence >= endSilence, len(s.windows) >= softLimit && s.silence >= softSilence:
		return []Event{s.close(len(s.windows) - max(s.silence-trailing, 0))}
	case len(s.windows) >= hardLimit:
		return s.split()
	}
	return nil
}

// Turn ends the utterance in progress where another speaker starts to talk
// and goes on with what follows as the next one. An utterance too short to
// stand alone goes on as it is.
func (s *Segmenter) Turn() []Event {
	if !s.active {
		return nil
	}
	speech := 0
	for _, w := range s.windows {
		if w.prob >= onset {
			speech++
		}
	}
	if speech < 2*minSpeech {
		return nil
	}
	ended := s.cut(len(s.windows))
	s.start += int64(len(s.windows))
	s.windows, s.silence = s.windows[:0], 0
	return []Event{ended, {Kind: Began, StartMs: s.start * windowMs}}
}

// Finish ends the utterance in progress, if any.
func (s *Segmenter) Finish() []Event {
	if !s.active {
		return nil
	}
	return []Event{s.close(len(s.windows))}
}

// Active reports whether an utterance is in progress.
func (s *Segmenter) Active() bool { return s.active }

func (s *Segmenter) close(n int) Event {
	e := s.cut(n)
	s.active, s.windows = false, s.windows[:0]
	return e
}

// split ends a long utterance at the quietest moment of its last seconds
// and goes on with the rest as the next one.
func (s *Segmenter) split() []Event {
	at := len(s.windows) - cutSearch
	for i := at; i < len(s.windows); i++ {
		if s.windows[i].prob < s.windows[at].prob {
			at = i
		}
	}
	ended := s.cut(at)
	rest := append([]window(nil), s.windows[at:]...)
	s.start += int64(at)
	s.windows, s.silence = rest, 0
	return []Event{ended, {Kind: Began, StartMs: s.start * windowMs}}
}

func (s *Segmenter) cut(n int) Event {
	speech := 0
	pcm := make([]int16, 0, n*Window)
	for _, w := range s.windows[:n] {
		if w.prob >= onset {
			speech++
		}
		pcm = append(pcm, w.pcm...)
	}
	e := Event{Kind: Ended, StartMs: s.start * windowMs, EndMs: (s.start + int64(n)) * windowMs, PCM: pcm}
	if speech < minSpeech {
		e.Kind, e.PCM = Dropped, nil
	}
	return e
}
