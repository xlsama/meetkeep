package main

import (
	"flag"
	"strconv"
	"strings"
	"time"
)

// span is a stretch of time: from, and up to but not at to. A zero end
// leaves it open.
type span struct{ from, to time.Time }

func (s span) holds(ms int64) bool {
	t := time.UnixMilli(ms)
	return (s.from.IsZero() || !t.Before(s.from)) && (s.to.IsZero() || t.Before(s.to))
}

// moment reads a time as the flags write it and tells whether it names a
// whole day.
func moment(text string, now time.Time) (t time.Time, whole bool, err error) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch text {
	case "now":
		return now, false, nil
	case "today":
		return today, true, nil
	case "yesterday":
		return today.AddDate(0, 0, -1), true, nil
	}
	if n, err := strconv.Atoi(strings.TrimLeft(text[:max(0, len(text)-1)], "-")); err == nil && strings.HasPrefix(text, "-") {
		switch text[len(text)-1] {
		case 'd':
			return today.AddDate(0, 0, -n), true, nil
		case 'h':
			return now.Add(-time.Duration(n) * time.Hour), false, nil
		case 'm':
			return now.Add(-time.Duration(n) * time.Minute), false, nil
		}
	}
	if t, err := time.ParseInLocation("2006-01-02", text, now.Location()); err == nil {
		return t, true, nil
	}
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02T15:04:05", "2006-01-02 15:04"} {
		if t, err := time.ParseInLocation(layout, text, now.Location()); err == nil {
			return t, false, nil
		}
	}
	if t, err := time.Parse(time.RFC3339, text); err == nil {
		return t, false, nil
	}
	if ms, err := strconv.ParseInt(text, 10, 64); err == nil && len(text) >= 12 {
		return time.UnixMilli(ms), false, nil
	}
	return time.Time{}, false, usage("cannot read the time %q: write today, yesterday, 2026-11-20, 2026-11-20T15:00, -7d, -3h or -30m", text)
}

// between holds the flags that pick the meetings of a stretch of time.
type between struct{ date, from, to string }

func (b *between) flags(fs *flag.FlagSet) {
	fs.StringVar(&b.date, "date", "", "only the meetings of this `day`")
	fs.StringVar(&b.from, "from", "", "only the meetings from this `time` on")
	fs.StringVar(&b.to, "to", "", "only the meetings up to this `time`; a day includes that day")
}

func (b *between) span(now time.Time) (span, error) {
	var s span
	if b.date != "" {
		if b.from != "" || b.to != "" {
			return s, usage("--date goes without --from and --to")
		}
		t, whole, err := moment(b.date, now)
		if err != nil {
			return s, err
		}
		if !whole {
			return s, usage("--date takes a day, such as 2026-11-20 or yesterday")
		}
		return span{t, t.AddDate(0, 0, 1)}, nil
	}
	if b.from != "" {
		t, _, err := moment(b.from, now)
		if err != nil {
			return s, err
		}
		s.from = t
	}
	if b.to != "" {
		t, whole, err := moment(b.to, now)
		if err != nil {
			return s, err
		}
		if whole {
			t = t.AddDate(0, 0, 1)
		}
		s.to = t
	}
	return s, nil
}

// offset reads a moment of a meeting, in milliseconds from its start:
// 12:30, 1:02:03, or seconds.
func offset(text string) (int64, error) {
	var seconds int64
	parts := strings.Split(text, ":")
	if len(parts) > 3 {
		return 0, usage("cannot read the moment %q: write 12:30, 1:02:03 or seconds", text)
	}
	for _, part := range parts {
		n, err := strconv.ParseInt(part, 10, 64)
		if err != nil || n < 0 {
			return 0, usage("cannot read the moment %q: write 12:30, 1:02:03 or seconds", text)
		}
		seconds = seconds*60 + n
	}
	return seconds * 1000, nil
}
