package main

import (
	"strings"
	"time"

	"github.com/xlsama/meetkeep/internal/config"
	"github.com/xlsama/meetkeep/internal/recorder"
	"github.com/xlsama/meetkeep/modules/capture"
)

// meetingApps are the apps whose use of the microphone means a call: by
// the start of their bundle identifier on macOS, by the name of their
// executable on Windows. Browsers are among them for the meetings held in
// a tab.
var meetingApps = []string{
	"us.zoom.xos", "com.microsoft.teams", "com.tencent.meeting", "com.electron.lark", "com.bytedance.macos.feishu",
	"com.bytedance.lark", "com.alibaba.DingTalkMac", "com.cisco.webexmeetingsapp", "Cisco-Systems.Spark",
	"com.apple.FaceTime", "com.tinyspeck.slackmacgap", "com.hnc.Discord", "com.skype.skype",
	"com.google.Chrome", "com.apple.Safari", "com.microsoft.edgemac", "company.thebrowser.Browser",
	"org.mozilla.firefox", "com.brave.Browser",
	"zoom.exe", "ms-teams.exe", "teams.exe", "wemeetapp.exe", "feishu.exe", "lark.exe", "dingtalk.exe",
	"webex.exe", "slack.exe", "discord.exe", "skype.exe", "chrome.exe", "msedge.exe", "firefox.exe", "brave.exe",
}

func inMeeting(app capture.App) bool {
	for _, known := range meetingApps {
		if strings.HasPrefix(app.ID, known) {
			return true
		}
	}
	return false
}

// watch announces the calls that begin while nothing is recorded, once for
// each: an app is announced again only after it let go of the microphone.
func (a *app) watch() {
	announced := map[string]bool{}
	for range time.Tick(3 * time.Second) {
		settings := a.current()
		if !settings.Recording.DetectMeetings || a.recorder.Status().Phase != recorder.Idle {
			continue
		}
		listening, err := a.capture.Listening()
		if err != nil {
			continue
		}
		now := map[string]bool{}
		for _, app := range listening {
			now[app.ID] = true
			if announced[app.ID] || !inMeeting(app) || ignored(settings, app.ID) {
				continue
			}
			announced[app.ID] = true
			a.windows.announce(config.AppRef{ID: app.ID, Name: app.Name})
		}
		for id := range announced {
			if !now[id] {
				delete(announced, id)
			}
		}
	}
}

func ignored(s config.Settings, id string) bool {
	for _, app := range s.Recording.IgnoredApps {
		if app.ID == id {
			return true
		}
	}
	return false
}
