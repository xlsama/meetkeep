// Package config reads and writes the app's settings file.
package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
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

type ASRProvider string

const (
	ASRMAI    ASRProvider = "mai"
	ASRCustom ASRProvider = "custom"
)

type ASRSettings struct {
	// Provider names the service that transcribes.
	Provider ASRProvider `json:"provider"`
	// Endpoint and Key are those of MAI-Transcribe-2.
	Endpoint string    `json:"endpoint"`
	Key      string    `json:"key"`
	Custom   CustomASR `json:"custom"`
	Hotwords []string  `json:"hotwords"`
	// Polish has the language model take the filler out of a transcript
	// and set misheard names right.
	Polish bool `json:"polish"`
}

// CustomASR is a service that speaks OpenAI's audio transcriptions at
// BaseURL. One that asks for no key has none.
type CustomASR struct {
	BaseURL string `json:"baseUrl"`
	Key     string `json:"key"`
	Model   string `json:"model"`
}

type LLMProvider string

const (
	LLMAnthropic  LLMProvider = "anthropic"
	LLMDeepSeek   LLMProvider = "deepseek"
	LLMOpenRouter LLMProvider = "openrouter"
	LLMCustom     LLMProvider = "custom"
)

// LLMSettings keeps a key and a model for every provider, and names the one
// that writes the notes.
type LLMSettings struct {
	Provider   LLMProvider `json:"provider"`
	Anthropic  LLMService  `json:"anthropic"`
	DeepSeek   LLMService  `json:"deepseek"`
	OpenRouter LLMService  `json:"openrouter"`
	Custom     CustomLLM   `json:"custom"`
}

type LLMService struct {
	Key   string `json:"key"`
	Model string `json:"model"`
}

// CustomLLM is a service that speaks OpenAI's chat completions at BaseURL.
type CustomLLM struct {
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
		ASR:     ASRSettings{Provider: ASRMAI, Hotwords: []string{}, Polish: true},
		LLM: LLMSettings{
			Provider:   LLMDeepSeek,
			Anthropic:  LLMService{Model: "claude-sonnet-5-5"},
			DeepSeek:   LLMService{Model: "deepseek-flash"},
			OpenRouter: LLMService{Model: "deepseek/deepseek-v4.1-flash"},
		},
		Recording: RecordingSettings{
			Directory:        Contract(filepath.Join(home(), "Documents", "MeetKeep")),
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
	return filepath.Join(home(), ".config", "meetkeep")
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

func home() string {
	dir, _ := os.UserHomeDir()
	return dir
}

// Contract writes the home directory at the start of path as ~, where
// paths are written so: Windows keeps the path whole.
func Contract(path string) string {
	if home := home(); runtime.GOOS != "windows" && home != "" && strings.HasPrefix(path, home+string(filepath.Separator)) {
		return "~" + filepath.ToSlash(strings.TrimPrefix(path, home))
	}
	return path
}

// Expand turns a leading ~ into the home directory.
func Expand(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		return filepath.Join(home(), path[1:])
	}
	return path
}
