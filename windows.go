package main

import (
	_ "embed"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/egoist/mygo"

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
	panelShortcut = "Alt+Cmd+M"
	toastWidth    = 380
	toastHeight   = 96
	toastStays    = 10 * time.Second
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
	mode   panelMode
	width  int
	active bool
}

func (w *windows) start() error {
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
	w.tray = tray
	w.width = panelWidth
	tray.OnClick(w.toggleMenu)
	if err := mygo.GlobalShortcut.Register(panelShortcut, w.togglePanel); err != nil {
		return err
	}
	go w.clock()
	w.openMain("/")
	return nil
}

func (w *windows) showMain(route string) {
	w.mu.Lock()
	win := w.main
	w.mu.Unlock()
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
		Title:           "MeetKeep",
		URL:             route,
		Width:           1120,
		Height:          740,
		MinWidth:        880,
		MinHeight:       560,
		BackgroundColor: "#ffffff",
		StateKey:        "main",
	})
	win.OnClose(func(e *mygo.CloseEvent) {
		if !w.quitting {
			e.PreventDefault()
			win.Hide()
		}
	})
	w.mu.Lock()
	w.main = win
	w.mu.Unlock()
}

func (w *windows) toggleMenu() {
	w.mu.Lock()
	menu := w.menu
	w.mu.Unlock()
	if menu != nil {
		menu.Close()
		return
	}
	menu = mygo.NewFlyout(mygo.FlyoutOptions{
		Tray:         w.tray,
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

// setMode shows the side panel, its collapsed strip, or neither.
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
	} else if w.panel != nil {
		w.panel.Hide()
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
	} else if w.mini != nil {
		w.mini.Hide()
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
	active := w.active
	w.mu.Unlock()
	switch {
	case active && !was:
		_ = w.tray.SetIcon(trayRecordingIcon, false)
		if w.app.current().Recording.ShowPanel {
			w.setMode(panelOpen)
		}
	case !active && was:
		_ = w.tray.SetIcon(trayIcon, true)
		w.tray.SetTitle("")
		w.setMode(panelHidden)
	}
}

// notesReady tells the user that a meeting's notes are written, unless
// the main window shows and has the keyboard.
func (w *windows) notesReady(meeting store.Meeting, written notes.Notes) {
	w.mu.Lock()
	main, old := w.main, w.toast
	w.mu.Unlock()
	if main != nil && main.IsVisible() && main.IsFocused() {
		return
	}
	if old != nil {
		old.Close()
	}
	area := mygo.Screen.PrimaryDisplay().WorkArea
	toast := mygo.NewWindow(mygo.WindowOptions{
		Title:             "MeetKeep",
		URL:               "/toast?meeting=" + url.QueryEscape(meeting.ID),
		Width:             toastWidth,
		Height:            toastHeight,
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
	toast.SetBounds(mygo.Rectangle{
		X: area.X + area.Width - toastWidth - miniMargin, Y: area.Y + miniMargin,
		Width: toastWidth, Height: toastHeight,
	})
	toast.ShowInactive()
	toast.OnClosed(func() {
		w.mu.Lock()
		if w.toast == toast {
			w.toast = nil
		}
		w.mu.Unlock()
	})
	w.mu.Lock()
	w.toast = toast
	w.mu.Unlock()
	time.AfterFunc(toastStays, func() {
		w.mu.Lock()
		current := w.toast == toast
		w.mu.Unlock()
		if current {
			toast.Close()
		}
	})
}

func (w *windows) closeToast() {
	w.mu.Lock()
	toast := w.toast
	w.mu.Unlock()
	if toast != nil {
		toast.Close()
	}
}

func (w *windows) clock() {
	shown := ""
	for range time.Tick(250 * time.Millisecond) {
		title := ""
		if s := w.app.recorder.Status(); s.Phase != recorder.Idle {
			seconds := s.ElapsedMs / 1000
			title = fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
			if seconds >= 3600 {
				title = fmt.Sprintf("%d:%02d:%02d", seconds/3600, seconds/60%60, seconds%60)
			}
		}
		if title != shown {
			shown = title
			w.tray.SetTitle(title)
		}
	}
}
