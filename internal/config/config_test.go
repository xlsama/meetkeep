package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv("MEETKEEP_HOME", t.TempDir())
	s, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !s.General.OpenAtLogin || s.General.MenuBar || !s.ASR.Polish {
		t.Errorf("without a file the settings are %+v, want the app to open at login, out of the menu bar", s)
	}

	// A file written before a setting was there leaves it as it is by default.
	if err := os.WriteFile(Path(), []byte(`{"asr": {"polish": false}, "general": {"menuBar": true}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if s, err = Load(); err != nil || !s.General.OpenAtLogin || !s.General.MenuBar || s.ASR.Polish || s.ASR.Hotwords == nil || s.ASR.Provider != ASRMAI || s.ASR.Services == nil || s.LLM.Services == nil {
		t.Errorf("got %+v (%v)", s, err)
	}

	s.General.OpenAtLogin = false
	if err := Save(s); err != nil {
		t.Fatal(err)
	}
	if s, err = Load(); err != nil || s.General.OpenAtLogin || !s.General.MenuBar {
		t.Errorf("after saving, got %+v (%v)", s, err)
	}
}
