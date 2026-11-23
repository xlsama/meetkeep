package main

import (
	_ "embed"
	"fmt"
	"log"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"

	"github.com/xlsama/meetkeep/internal/config"
	"github.com/xlsama/meetkeep/internal/notes"
	"github.com/xlsama/meetkeep/internal/recorder"
	"github.com/xlsama/meetkeep/internal/store"
)

//go:embed assets/tray.png
var trayIcon []byte

//go:embed assets/tray-recording.png
var trayRecordingIcon []byte

const (
	panelWidth    = 400
	panelMinWidth = 340
	panelMaxWidth = 720
	miniWidth     = 440
	miniHeight    = 44
	miniMargin    = 16
	panelShortcut = "Alt+CmdOrCtrl+M"
	titleBar      = 52
	toastWidth    = 380
	toastHeight   = 96
	toastStays    = 10 * time.Second
	mainLingers   = 10 * time.Second
	noticeHeight  = 132
	noticeStays   = 30 * time.Second
)

type panelMode int

const (
	panelHidden panelMode = iota
	panelOpen
	panelMini
)

// windows owns the app's windows and its menu bar item.
type windows struct {
	app      *app
	quitting bool

	mu     sync.Mutex
	tray   *mygo.Tray
	main   *mygo.Window
	panel  *mygo.Window
	mini   *mygo.Window
	menu   *mygo.Window
	toast  *mygo.Window
	notice *mygo.Window
	mode   panelMode
	width  int
	active bool
	closes int
}

// start opens the main window, unless the app was started as the user
// logged in: then it waits in the background.
func (w *windows) start() error {
	w.width = panelWidth
	if err := mygo.GlobalShortcut.Register(panelShortcut, w.togglePanel); err != nil {
		log.Print(err)
	}
	go w.clock()
	go w.app.watch()
	if mygo.App.WasOpenedAtLogin() {
		mygo.App.Dock.Hide()
	} else {
		w.openMain("/")
	}
	return nil
}

// showTray puts the app's item in the menu bar, or takes it out.
func (w *windows) showTray(show bool) error {
	w.mu.Lock()
	tray, recording := w.tray, w.active
	w.mu.Unlock()
	switch {
	case show == (tray != nil):
		return nil
	case !show:
		w.mu.Lock()
		w.tray = nil
		w.mu.Unlock()
		w.closeMenu()
		tray.Destroy()
		return nil
	}
	tray, err := mygo.NewTray(mygo.TrayOptions{
		ID:              "meetkeep",
		Icon:            trayIcon,
		IconIsTemplate:  true,
		MonospacedTitle: true,
		ToolTip:         "MeetKeep",
	})
	if err != nil {
		return err
	}
	drawTray(tray, recording)
	tray.OnAppearanceChange(func() {
		w.mu.Lock()
		recording := w.active
		w.mu.Unlock()
		drawTray(tray, recording)
	})
	tray.OnClick(w.toggleMenu)
	w.mu.Lock()
	w.tray = tray
	w.mu.Unlock()
	return nil
}

// item returns the app's item in the menu bar, or nil when it has none.
func (w *windows) item() *mygo.Tray {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.tray
}

func (w *windows) showMain(route string) {
	w.mu.Lock()
	win := w.main
	w.closes++
	w.mu.Unlock()
	mygo.App.Dock.Show()
	if win != nil {
		if route != "" {
			_ = navigate.Emit(win, route)
		}
		win.Show()
		win.Focus()
		mygo.App.Focus()
		return
	}
	if route == "" {
		route = "/"
	}
	w.openMain(route)
	mygo.App.Focus()
}

func (w *windows) openMain(route string) {
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:                "MeetKeep",
		URL:                  route,
		Width:                1253,
		Height:               910,
		MinWidth:             880,
		MinHeight:            560,
		BackgroundColor:      "#ececec",
		StateKey:             "main",
		TitleBarStyle:        mygo.TitleBarHidden,
		TitleBarHeight:       titleBar,
		TrafficLightPosition: &mygo.Point{X: 18, Y: 20},
	})
	win.OnClose(func(e *mygo.CloseEvent) {
		if w.quitting {
			return
		}
		e.PreventDefault()
		win.Hide()
		mygo.App.Dock.Hide()
		w.mu.Lock()
		w.closes++
		closed := w.closes
		w.mu.Unlock()
		time.AfterFunc(mainLingers, func() { w.retireMain(win, closed) })
	})
	w.mu.Lock()
	w.main = win
	w.mu.Unlock()
}

// retireMain frees a main window that stayed closed since the close numbered
// closed: its page holds most of the memory of the app.
func (w *windows) retireMain(win *mygo.Window, closed int) {
	w.mu.Lock()
	if w.main != win || w.closes != closed {
		w.mu.Unlock()
		return
	}
	w.main = nil
	w.mu.Unlock()
	win.Destroy()
}

func (w *windows) toggleMenu() {
	w.mu.Lock()
	menu, tray := w.menu, w.tray
	w.mu.Unlock()
	if menu != nil {
		menu.Close()
		return
	}
	if tray == nil {
		return
	}
	menu = mygo.NewFlyout(mygo.FlyoutOptions{
		Tray:         tray,
		URL:          "/menu",
		Placement:    mygo.PlacementBottom,
		Gap:          6,
		Width:        320,
		Height:       392,
		FitContent:   true,
		Focusable:    true,
		Shadow:       true,
		CornerRadius: 12,
	})
	menu.OnClosed(func() {
		w.mu.Lock()
		w.menu = nil
		w.mu.Unlock()
	})
	w.mu.Lock()
	w.menu = menu
	w.mu.Unlock()
}

func (w *windows) closeMenu() {
	w.mu.Lock()
	menu := w.menu
	w.mu.Unlock()
	if menu != nil {
		menu.Close()
	}
}

func (w *windows) panelBounds() mygo.Rectangle {
	area := mygo.Screen.PrimaryDisplay().WorkArea
	return mygo.Rectangle{X: area.X + area.Width - w.width, Y: area.Y, Width: w.width, Height: area.Height}
}

func (w *windows) miniBounds() mygo.Rectangle {
	area := mygo.Screen.PrimaryDisplay().WorkArea
	return mygo.Rectangle{
		X: area.X + area.Width - miniWidth - miniMargin, Y: area.Y + miniMargin,
		Width: miniWidth, Height: miniHeight,
	}
}

// setMode shows the side panel, its collapsed strip, or neither. After a
// recording it frees them.
func (w *windows) setMode(mode panelMode) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.active {
		mode = panelHidden
	}
	w.mode = mode
	if mode == panelOpen {
		if w.panel == nil {
			w.panel = mygo.NewWindow(mygo.WindowOptions{
				Title:             "MeetKeep",
				URL:               "/panel",
				Width:             w.width,
				Height:            600,
				Frameless:         true,
				Panel:             true,
				AlwaysOnTop:       true,
				Hidden:            true,
				SkipTaskbar:       true,
				DisableResize:     true,
				DisableMinimize:   true,
				DisableMaximize:   true,
				DisableFullScreen: true,
				BackgroundColor:   "#ffffff",
			})
		}
		w.panel.SetBounds(w.panelBounds())
		w.panel.ShowInactive()
	} else if w.panel != nil && w.active {
		w.panel.Hide()
	} else if w.panel != nil {
		w.panel.Destroy()
		w.panel = nil
	}
	if mode == panelMini {
		if w.mini == nil {
			w.mini = mygo.NewWindow(mygo.WindowOptions{
				Title:             "MeetKeep",
				URL:               "/mini",
				Width:             miniWidth,
				Height:            miniHeight,
				Frameless:         true,
				Panel:             true,
				AlwaysOnTop:       true,
				Hidden:            true,
				SkipTaskbar:       true,
				Transparent:       true,
				CornerRadius:      miniHeight / 2,
				DisableResize:     true,
				DisableMinimize:   true,
				DisableMaximize:   true,
				DisableFullScreen: true,
			})
		}
		w.mini.SetBounds(w.miniBounds())
		w.mini.ShowInactive()
	} else if w.mini != nil && w.active {
		w.mini.Hide()
	} else if w.mini != nil {
		w.mini.Destroy()
		w.mini = nil
	}
}

func (w *windows) togglePanel() {
	w.mu.Lock()
	mode := w.mode
	w.mu.Unlock()
	if mode == panelHidden {
		w.setMode(panelOpen)
	} else {
		w.setMode(panelHidden)
	}
}

func (w *windows) resizePanel(width int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.width = min(max(width, panelMinWidth), panelMaxWidth)
	if w.panel != nil && w.mode == panelOpen {
		w.panel.SetBounds(w.panelBounds())
	}
}

// follow shows and hides what goes with a recording.
func (w *windows) follow(s recorder.Status) {
	w.mu.Lock()
	was := w.active
	w.active = s.Phase != recorder.Idle
	active, tray := w.active, w.tray
	w.mu.Unlock()
	switch {
	case active && !was:
		w.closeNotice()
		if tray != nil {
			drawTray(tray, true)
		}
		if w.app.current().Recording.ShowPanel {
			w.setMode(panelOpen)
		}
	case !active && was:
		if tray != nil {
			drawTray(tray, false)
			tray.SetTitle("")
		}
		w.setMode(panelHidden)
	}
}

// card shows a small window in the corner of the screen, in place of the
// one slot holds, and closes it after a while.
func (w *windows) card(slot **mygo.Window, route string, height int, stays time.Duration) {
	w.mu.Lock()
	old := *slot
	w.mu.Unlock()
	if old != nil {
		old.Close()
	}
	area := mygo.Screen.PrimaryDisplay().WorkArea
	card := mygo.NewWindow(mygo.WindowOptions{
		Title:             "MeetKeep",
		URL:               route,
		Width:             toastWidth,
		Height:            height,
		Frameless:         true,
		Panel:             true,
		AlwaysOnTop:       true,
		Hidden:            true,
		SkipTaskbar:       true,
		Transparent:       true,
		CornerRadius:      16,
		DisableResize:     true,
		DisableMinimize:   true,
		DisableMaximize:   true,
		DisableFullScreen: true,
	})
	card.SetBounds(mygo.Rectangle{
		X: area.X + area.Width - toastWidth - miniMargin, Y: area.Y + miniMargin,
		Width: toastWidth, Height: height,
	})
	card.ShowInactive()
	card.OnClosed(func() {
		w.mu.Lock()
		if *slot == card {
			*slot = nil
		}
		w.mu.Unlock()
	})
	w.mu.Lock()
	*slot = card
	w.mu.Unlock()
	time.AfterFunc(stays, func() {
		w.mu.Lock()
		current := *slot == card
		w.mu.Unlock()
		if current {
			card.Close()
		}
	})
}

func (w *windows) closeCard(slot **mygo.Window) {
	w.mu.Lock()
	card := *slot
	w.mu.Unlock()
	if card != nil {
		card.Close()
	}
}

// notesReady tells the user that a meeting's notes are written, unless
// the main window shows and has the keyboard.
func (w *windows) notesReady(meeting store.Meeting, written notes.Notes) {
	w.mu.Lock()
	main := w.main
	w.mu.Unlock()
	if main != nil && main.IsVisible() && main.IsFocused() {
		return
	}
	w.card(&w.toast, "/toast?meeting="+url.QueryEscape(meeting.ID), toastHeight, toastStays)
	if w.app.recorder.Status().Phase == recorder.Idle {
		w.app.sounds.play(cueReady)
	}
}

func (w *windows) closeToast() { w.closeCard(&w.toast) }

// announce tells the user that a call began in an app.
func (w *windows) announce(app config.AppRef) {
	route := "/detect?app=" + url.QueryEscape(app.ID) + "&name=" + url.QueryEscape(app.Name)
	w.card(&w.notice, route, noticeHeight, noticeStays)
	w.app.sounds.play(cueNotice)
}

func (w *windows) closeNotice() { w.closeCard(&w.notice) }

func (w *windows) clock() {
	shown, on := "", (*mygo.Tray)(nil)
	for range time.Tick(250 * time.Millisecond) {
		title := ""
		if s := w.app.recorder.Status(); s.Phase != recorder.Idle {
			seconds := s.ElapsedMs / 1000
			title = fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
			if seconds >= 3600 {
				title = fmt.Sprintf("%d:%02d:%02d", seconds/3600, seconds/60%60, seconds%60)
			}
		}
		if tray := w.item(); tray != nil && (title != shown || tray != on) {
			shown, on = title, tray
			tray.SetTitle(title)
			if runtime.GOOS != "darwin" {
				tray.SetToolTip(strings.TrimSpace("MeetKeep " + title))
			}
		}
	}
}
