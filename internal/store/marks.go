package store

import (
	"encoding/json"
	"unicode/utf16"
)

// Mark is a place in an utterance's text, counted in UTF-16 code units as
// a web view counts them, and when what stands there is said.
type Mark struct {
	At int   `json:"at"`
	Ms int64 `json:"ms"`
}

func encodeMarks(marks []Mark) string {
	if len(marks) == 0 {
		return ""
	}
	data, _ := json.Marshal(marks)
	return string(data)
}

func decodeMarks(data string) []Mark {
	if data == "" {
		return nil
	}
	var marks []Mark
	_ = json.Unmarshal([]byte(data), &marks)
	return marks
}

// compared is how many pairs of letters Moved compares at most. Texts
// that differ over more keep only the marks of what they share at their
// ends.
const compared = 1 << 22

// Moved returns marks, which are places in text, at their places in
// polished: those of the letters polishing kept, the others being left
// out.
func Moved(marks []Mark, text, polished string) []Mark {
	if len(marks) == 0 {
		return nil
	}
	a, b := utf16.Encode([]rune(text)), utf16.Encode([]rune(polished))
	to := make([]int, len(a))
	for i := range to {
		to[i] = -1
	}
	head := 0
	for head < len(a) && head < len(b) && a[head] == b[head] {
		to[head] = head
		head++
	}
	tail := 0
	for tail < len(a)-head && tail < len(b)-head && a[len(a)-1-tail] == b[len(b)-1-tail] {
		tail++
		to[len(a)-tail] = len(b) - tail
	}
	x, y := a[head:len(a)-tail], b[head:len(b)-tail]
	if len(x) > 0 && len(y) > 0 && len(x)*len(y) <= compared {
		width := len(y) + 1
		shared := make([]uint16, (len(x)+1)*width)
		for i := len(x) - 1; i >= 0; i-- {
			for j := len(y) - 1; j >= 0; j-- {
				switch {
				case x[i] == y[j]:
					shared[i*width+j] = shared[(i+1)*width+j+1] + 1
				case shared[(i+1)*width+j] >= shared[i*width+j+1]:
					shared[i*width+j] = shared[(i+1)*width+j]
				default:
					shared[i*width+j] = shared[i*width+j+1]
				}
			}
		}
		for i, j := 0, 0; i < len(x) && j < len(y); {
			switch {
			case x[i] == y[j]:
				to[head+i] = head + j
				i++
				j++
			case shared[(i+1)*width+j] >= shared[i*width+j+1]:
				i++
			default:
				j++
			}
		}
	}
	moved := make([]Mark, 0, len(marks))
	for _, m := range marks {
		if m.At >= 0 && m.At < len(to) && to[m.At] >= 0 {
			moved = append(moved, Mark{At: to[m.At], Ms: m.Ms})
		}
	}
	return moved
}
