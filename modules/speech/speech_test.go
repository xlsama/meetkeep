package speech_test

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/xlsama/meetkeep/modules/speech"
)

func engine(t *testing.T) speech.Engine {
	t.Helper()
	e, err := speech.NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func fixtures(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("MEETKEEP_SPEECH_FIXTURES")
	if dir == "" {
		t.Skip("MEETKEEP_SPEECH_FIXTURES is not set: no models to run")
	}
	return dir
}

func TestRecordWritesOggOpus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tone.opus")
	recording, err := engine(t).Record(path)
	if err != nil {
		t.Fatal(err)
	}
	const seconds = 3
	frame := make([]byte, 9600)
	for i := range seconds * 10 {
		for j := range 4800 {
			v := 8000 * math.Sin(2*math.Pi*440*float64(i*4800+j)/48000)
			binary.LittleEndian.PutUint16(frame[2*j:], uint16(int16(v)))
		}
		if err := recording.Write(frame); err != nil {
			t.Fatal(err)
		}
		if i == 14 {
			if info, err := os.Stat(path); err != nil || info.Size() < 2000 {
				t.Errorf("after 1.5 s the file holds %v bytes: a crash would lose the recording", info.Size())
			}
		}
	}
	if err := recording.Close(); err != nil {
		t.Fatal(err)
	}
	if err := recording.Write(frame); err == nil {
		t.Error("a closed recording took more audio")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("OggS")) || !bytes.Contains(data[:64], []byte("OpusHead")) {
		t.Fatalf("the file does not start as Ogg Opus: %q", data[:min(len(data), 48)])
	}
	if kbps := float64(len(data)) * 8 / seconds / 1000; kbps < 16 || kbps > 48 {
		t.Errorf("the file runs at %.0f kbps, want about 32", kbps)
	}
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Log("ffprobe is not installed: the file's duration is not checked")
		return
	}
	out, err := exec.Command(ffprobe, "-v", "error", "-show_entries", "stream=codec_name:format=duration",
		"-of", "csv=p=0", path).Output()
	if err != nil {
		t.Fatalf("ffprobe: %v", err)
	}
	lines := strings.Fields(string(out))
	if len(lines) != 2 || lines[0] != "opus" {
		t.Fatalf("ffprobe read %q", out)
	}
	if duration, _ := strconv.ParseFloat(lines[1], 64); math.Abs(duration-seconds) > 0.05 {
		t.Errorf("the file lasts %v s, want %d", duration, seconds)
	}
}

func duration(t *testing.T, path string) float64 {
	t.Helper()
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Skip("ffprobe is not installed")
	}
	out, err := exec.Command(ffprobe, "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", path).Output()
	if err != nil {
		t.Fatalf("ffprobe: %v", err)
	}
	seconds, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		t.Fatalf("ffprobe read %q", out)
	}
	return seconds
}

func TestImportAndMix(t *testing.T) {
	e := engine(t)
	dir := t.TempDir()
	tone := filepath.Join(dir, "tone.opus")
	ms, err := e.Import(t.Context(), filepath.Join("..", "capture", "testdata", "tone.wav"), tone)
	if err != nil {
		t.Fatal(err)
	}
	if ms < 2900 || ms > 3100 {
		t.Errorf("a 3 s file was imported as %d ms", ms)
	}
	if got := duration(t, tone); math.Abs(got-3) > 0.1 {
		t.Errorf("the imported file lasts %v s, want 3", got)
	}

	long := filepath.Join(dir, "long.opus")
	recording, err := e.Record(long)
	if err != nil {
		t.Fatal(err)
	}
	for range 50 {
		if err := recording.Write(make([]byte, 9600)); err != nil {
			t.Fatal(err)
		}
	}
	recording.Close()

	mixed := filepath.Join(dir, "mixed.opus")
	if err := e.Mix(t.Context(), []string{tone, long}, mixed); err != nil {
		t.Fatal(err)
	}
	if got := duration(t, mixed); math.Abs(got-5) > 0.1 {
		t.Errorf("the mix lasts %v s, want the longer input's 5", got)
	}
	if _, err := e.Import(t.Context(), "speech_test.go", filepath.Join(dir, "no.opus")); err == nil {
		t.Error("a Go file was imported as audio")
	}
}

func TestSeparateReadsOggOpus(t *testing.T) {
	dir := fixtures(t)
	e := engine(t)
	path := filepath.Join(t.TempDir(), "meeting.opus")
	if _, err := e.Import(t.Context(), filepath.Join(dir, "0-four-speakers-zh.wav"), path); err != nil {
		t.Fatal(err)
	}
	segments, err := e.Separate(t.Context(), path, speech.Models{
		Segmentation: filepath.Join(dir, "segmentation.onnx"),
		Embedding:    filepath.Join(dir, "embedding.onnx"),
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	speakers := map[int]bool{}
	for _, s := range segments {
		speakers[s.Speaker] = true
	}
	if len(speakers) != 4 {
		t.Errorf("found %d speakers in the Ogg Opus copy, want 4", len(speakers))
	}
}

func TestDetectorStaysQuietOnSilence(t *testing.T) {
	detector, err := engine(t).Detector()
	if err != nil {
		t.Fatal(err)
	}
	defer detector.Close()
	var probabilities []float32
	for range 10 {
		out, err := detector.Feed(make([]byte, 3200))
		if err != nil {
			t.Fatal(err)
		}
		probabilities = append(probabilities, out...)
	}
	if want := 16000 / 512; len(probabilities) != want {
		t.Fatalf("a second gave %d windows, want %d", len(probabilities), want)
	}
	for i, p := range probabilities {
		if p > 0.2 {
			t.Errorf("window %d of silence is speech with probability %v", i, p)
		}
	}
}

func TestDetectorHearsSpeech(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(fixtures(t), "0-four-speakers-zh.wav"))
	if err != nil {
		t.Fatal(err)
	}
	detector, err := engine(t).Detector()
	if err != nil {
		t.Fatal(err)
	}
	defer detector.Close()
	pcm := data[44:min(len(data), 44+30*32000)]
	speech, windows := 0, 0
	for at := 0; at+3200 <= len(pcm); at += 3200 {
		out, err := detector.Feed(pcm[at : at+3200])
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range out {
			windows++
			if p >= 0.5 {
				speech++
			}
		}
	}
	t.Logf("%d of %d windows are speech", speech, windows)
	if ratio := float64(speech) / float64(windows); ratio < 0.3 || ratio > 0.98 {
		t.Errorf("%.0f%% of a meeting's first 30 s is speech", 100*ratio)
	}
}

func TestSeparateFourSpeakers(t *testing.T) {
	dir := fixtures(t)
	segments, err := engine(t).Separate(t.Context(), filepath.Join(dir, "0-four-speakers-zh.wav"), speech.Models{
		Segmentation: filepath.Join(dir, "segmentation.onnx"),
		Embedding:    filepath.Join(dir, "embedding.onnx"),
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	speakers := map[int]int{}
	for i, s := range segments {
		if s.EndMs <= s.StartMs {
			t.Errorf("segment %d is empty: %+v", i, s)
		}
		if i > 0 && s.StartMs < segments[i-1].StartMs {
			t.Errorf("segment %d starts before the one before it", i)
		}
		speakers[s.Speaker] += s.EndMs - s.StartMs
	}
	t.Logf("%d segments, speaking time by speaker: %v", len(segments), speakers)
	if len(speakers) != 4 {
		t.Errorf("found %d speakers, want 4", len(speakers))
	}
}
