package main

import (
	"testing"

	"github.com/xlsama/meetkeep/internal/config"
	"github.com/xlsama/meetkeep/modules/capture"
)

func TestInMeeting(t *testing.T) {
	for id, want := range map[string]bool{
		"us.zoom.xos":            true,
		"com.microsoft.teams2":   true,
		"com.tencent.meeting":    true,
		"com.tencent.xinWeChat":  false,
		"com.google.Chrome":      true,
		"com.apple.VoiceMemos":   false,
		"com.superlogical.rex":   false,
		"wemeetapp.exe":          true,
		"audiodg.exe":            false,
		"com.apple.Safari":       true,
		"com.apple.SafariBogus2": true,
	} {
		if got := inMeeting(capture.App{ID: id}); got != want {
			t.Errorf("%s: in a meeting %v, want %v", id, got, want)
		}
	}
	settings := config.Settings{Recording: config.RecordingSettings{IgnoredApps: []config.AppRef{{ID: "us.zoom.xos", Name: "Zoom"}}}}
	if !ignored(settings, "us.zoom.xos") || ignored(settings, "com.tencent.meeting") {
		t.Error("the ignored apps are not told from the others")
	}
}
