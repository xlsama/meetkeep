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
	ASRMAI      ASRProvider = "mai"
	ASRDeepSeek ASRProvider = "deepseek"
	ASRCustom   ASRProvider = "custom"
)

type ASRSettings struct {
	// Provider names the service that transcribes.
	Provider ASRProvider `json:"provider"`
	// Endpoint and Key are those of MAI-Transcribe-2.
	Endpoint string `json:"endpoint"`
	Key      string `json:"key"`
	// DeepSeek is the key of DeepSeek's speech recognition.
	DeepSeek ASRService `json:"deepseek"`
	// Service is the ID of the one of Services that transcribes when
	// Provider is ASRCustom.
	Service  string    `json:"service"`
	Services []Service `json:"services"`
	Hotwords []string  `json:"hotwords"`
	// Polish has the language model take the filler out of a transcript
	// and set misheard names right.
	Polish bool `json:"polish"`
}

type ASRService struct {
	Key string `json:"key"`
}

// Service is a service of the user's own that speaks OpenAI's API at
// BaseURL, under the name they gave it. One that asks for no key has none.
type Service struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
	Key     string `json:"key"`
	Model   string `json:"model"`
}

// Chosen returns the one of services whose ID is id, or one that has
// nothing set.
func Chosen(services []Service, id string) Service {
	for _, s := range services {
		if s.ID == id {
			return s
		}
	}
	return Service{}
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
	// Service is the ID of the one of Services that writes the notes when
	// Provider is LLMCustom.
	Service  string    `json:"service"`
	Services []Service `json:"services"`
}

type LLMService struct {
	Key   string `json:"key"`
	Model string `json:"model"`
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
		ASR:     ASRSettings{Provider: ASRMAI, Services: []Service{}, Hotwords: []string{}, Polish: true},
		LLM: LLMSettings{
			Provider:   LLMDeepSeek,
			Anthropic:  LLMService{Model: "claude-sonnet-5-5"},
			DeepSeek:   LLMService{Model: "deepseek-flash"},
			OpenRouter: LLMService{Model: "deepseek/deepseek-v4.1-flash"},
			Services:   []Service{},
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
	if s.ASR.Services == nil {
		s.ASR.Services = []Service{}
	}
	if s.LLM.Services == nil {
		s.LLM.Services = []Service{}
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
