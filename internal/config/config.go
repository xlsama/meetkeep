// Package config reads and writes the app's settings file.
package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Settings struct {
	General   GeneralSettings   `json:"general"`
	ASR       ASRSettings       `json:"asr"`
	LLM       LLMSettings       `json:"llm"`
	Recording RecordingSettings `json:"recording"`
	Sources   SourceSettings    `json:"sources"`
}

type GeneralSettings struct {
	// OpenAtLogin has the app start when the user logs in.
	OpenAtLogin bool `json:"openAtLogin"`
	// MenuBar keeps an item of the app in the menu bar.
	MenuBar bool `json:"menuBar"`
}

type ASRSettings struct {
	Endpoint string   `json:"endpoint"`
	Key      string   `json:"key"`
	Hotwords []string `json:"hotwords"`
	// Polish has the language model take the filler out of a transcript
	// and set misheard names right.
	Polish bool `json:"polish"`
}

type LLMSettings struct {
	BaseURL string `json:"baseUrl"`
	Key     string `json:"key"`
	Model   string `json:"model"`
}

type EchoCancellation string

const (
	EchoAuto EchoCancellation = "auto"
	EchoOn   EchoCancellation = "on"
	EchoOff  EchoCancellation = "off"
)

type RecordingSettings struct {
	Directory        string           `json:"directory"`
	DetectMeetings   bool             `json:"detectMeetings"`
	ShowPanel        bool             `json:"showPanel"`
	EchoCancellation EchoCancellation `json:"echoCancellation"`
	// Sounds has the app sound what the recorder does and what it tells of.
	Sounds bool `json:"sounds"`
	// IgnoredApps are the apps whose calls are not announced.
	IgnoredApps []AppRef `json:"ignoredApps"`
}

// AppRef names an app whose sound is recorded alone.
type AppRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SourceSettings struct {
	Microphone bool     `json:"microphone"`
	System     bool     `json:"system"`
	Apps       []AppRef `json:"apps"`
}

func defaults() Settings {
	return Settings{
		General: GeneralSettings{OpenAtLogin: true},
		ASR:     ASRSettings{Hotwords: []string{}, Polish: true},
		LLM:     LLMSettings{BaseURL: "https://api.deepseek.com", Model: "deepseek-flash"},
		Recording: RecordingSettings{
			Directory:        "~/Documents/MeetKeep",
			DetectMeetings:   true,
			ShowPanel:        true,
			EchoCancellation: EchoAuto,
			Sounds:           true,
			IgnoredApps:      []AppRef{},
		},
		Sources: SourceSettings{Microphone: true, System: true, Apps: []AppRef{}},
	}
}

// Dir is where the settings file and the database live: ~/.config/meetkeep,
// or what MEETKEEP_HOME names, which keeps a test run apart.
func Dir() string {
	if dir := os.Getenv("MEETKEEP_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "meetkeep")
}

func Path() string { return filepath.Join(Dir(), "settings.json") }

func Load() (Settings, error) {
	s := defaults()
	data, err := os.ReadFile(Path())
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return defaults(), err
	}
	if s.ASR.Hotwords == nil {
		s.ASR.Hotwords = []string{}
	}
	if s.Sources.Apps == nil {
		s.Sources.Apps = []AppRef{}
	}
	if s.Recording.IgnoredApps == nil {
		s.Recording.IgnoredApps = []AppRef{}
	}
	return s, nil
}

func Save(s Settings) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := Path() + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, Path())
}

// Contract writes the home directory at the start of path as ~.
func Contract(path string) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(path, home+string(filepath.Separator)) {
		return "~" + filepath.ToSlash(strings.TrimPrefix(path, home))
	}
	return path
}

// Expand turns a leading ~ into the home directory.
func Expand(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}
