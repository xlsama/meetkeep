//go:build e2e

// Command smoke runs a build of the app that has the automation plugin,
// walks its windows through the plugin's socket and keeps pictures of
// them: how the app looks and works on a platform nobody sits at.
//
//	smoke <the app's executable> <a directory for the pictures> [an audio file to import]
package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type window struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Kind    string `json:"kind"`
	URL     string `json:"url"`
	Visible bool   `json:"visible"`
	Bounds  struct{ X, Y, Width, Height int }
}

type driver struct {
	conn   net.Conn
	reader *bufio.Reader
	out    string
	shots  int
	failed int
}

func (d *driver) call(method string, params any) (json.RawMessage, error) {
	line, err := json.Marshal(map[string]any{"id": 1, "method": method, "params": params})
	if err != nil {
		return nil, err
	}
	_ = d.conn.SetDeadline(time.Now().Add(40 * time.Second))
	if _, err := d.conn.Write(append(line, '\n')); err != nil {
		return nil, err
	}
	answer, err := d.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	var res struct {
		Result json.RawMessage
		Error  *struct{ Code, Message string }
	}
	if err := json.Unmarshal(answer, &res); err != nil {
		return nil, err
	}
	if res.Error != nil {
		return nil, fmt.Errorf("%s: %s", res.Error.Code, res.Error.Message)
	}
	return res.Result, nil
}

func (d *driver) windows() []window {
	raw, err := d.call("windows", nil)
	if err != nil {
		d.fail("windows", err)
		return nil
	}
	var list struct{ Windows []window }
	_ = json.Unmarshal(raw, &list)
	return list.Windows
}

// find waits for a window whose URL has part in it.
func (d *driver) find(part string, wait time.Duration) (window, bool) {
	for until := time.Now().Add(wait); ; time.Sleep(300 * time.Millisecond) {
		for _, w := range d.windows() {
			if strings.Contains(w.URL, part) {
				return w, true
			}
		}
		if time.Now().After(until) {
			return window{}, false
		}
	}
}

// eval returns what a script makes in the page of a window, as JSON.
func (d *driver) eval(w window, script string) (string, error) {
	raw, err := d.call("eval", map[string]any{"window": w.ID, "script": script})
	if err != nil {
		return "", err
	}
	var res struct{ Value json.RawMessage }
	_ = json.Unmarshal(raw, &res)
	return string(res.Value), nil
}

// until waits for a script to make something true in a window.
func (d *driver) until(w window, script string, wait time.Duration) bool {
	for until := time.Now().Add(wait); ; time.Sleep(300 * time.Millisecond) {
		if value, err := d.eval(w, script); err == nil && value != "false" && value != "null" && value != "" {
			return true
		}
		if time.Now().After(until) {
			return false
		}
	}
}

func (d *driver) click(w window, testID string) bool {
	found := d.until(w, fmt.Sprintf(`!!document.querySelector('[data-testid="%s"]')`, testID), 8*time.Second)
	if !found {
		d.fail("click "+testID, fmt.Errorf("nothing has this test id in %s", w.URL))
		return false
	}
	_, err := d.eval(w, fmt.Sprintf(`document.querySelector('[data-testid="%s"]').click()`, testID))
	if err != nil {
		d.fail("click "+testID, err)
	}
	return err == nil
}

func (d *driver) visit(w window, route string) {
	_, err := d.eval(w, fmt.Sprintf(`history.pushState({}, "", %q); dispatchEvent(new PopStateEvent("popstate"))`, route))
	if err != nil {
		d.fail("visit "+route, err)
	}
	time.Sleep(900 * time.Millisecond)
}

func (d *driver) picture(name string) string {
	d.shots++
	return filepath.Join(d.out, fmt.Sprintf("%02d-%s.png", d.shots, name))
}

// capture keeps the picture of a window as the app draws it.
func (d *driver) capture(w window, name string) {
	_, _ = d.call("idle", map[string]any{"window": w.ID, "timeout": 5000})
	raw, err := d.call("capture", map[string]any{"window": w.ID})
	if err != nil {
		d.fail("capture "+name, err)
		return
	}
	var res struct{ PNG string }
	_ = json.Unmarshal(raw, &res)
	data, err := base64.StdEncoding.DecodeString(res.PNG)
	if err != nil || len(data) == 0 {
		d.fail("capture "+name, fmt.Errorf("no picture (%v)", err))
		return
	}
	_ = os.WriteFile(d.picture(name), data, 0o644)
	d.pass("capture "+name, fmt.Sprintf("%d bytes", len(data)))
}

// desktop keeps the picture of the whole screen, with the windows as the
// system frames them and the taskbar.
func (d *driver) desktop(name string) {
	if runtime.GOOS != "windows" {
		return
	}
	script := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms,System.Drawing
$screen = [System.Windows.Forms.SystemInformation]::VirtualScreen
$picture = New-Object System.Drawing.Bitmap $screen.Width, $screen.Height
[System.Drawing.Graphics]::FromImage($picture).CopyFromScreen($screen.Left, $screen.Top, 0, 0, $picture.Size)
$picture.Save('%s')`, d.picture("desktop-"+name))
	if out, err := exec.Command("powershell", "-NoProfile", "-Command", script).CombinedOutput(); err != nil {
		d.fail("desktop "+name, fmt.Errorf("%v: %s", err, out))
	}
}

func (d *driver) pass(step, detail string) { fmt.Printf("ok    %s: %s\n", step, detail) }

func (d *driver) fail(step string, err error) {
	d.failed++
	fmt.Printf("FAIL  %s: %v\n", step, err)
}

// tell prints what a script makes in a window.
func (d *driver) tell(w window, step, script string) string {
	value, err := d.eval(w, script)
	if err != nil {
		d.fail(step, err)
		return ""
	}
	d.pass(step, value)
	return value
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: smoke <executable> <directory> [audio file]")
		os.Exit(2)
	}
	program, out := os.Args[1], os.Args[2]
	out, _ = filepath.Abs(out)
	_ = os.MkdirAll(out, 0o755)
	home := filepath.Join(out, "home")
	_ = os.MkdirAll(home, 0o700)
	settings := map[string]any{
		"general":   map[string]any{"openAtLogin": false, "menuBar": true},
		"recording": map[string]any{"directory": filepath.Join(out, "recordings"), "showPanel": true, "detectMeetings": false},
	}
	data, _ := json.Marshal(settings)
	_ = os.WriteFile(filepath.Join(home, "settings.json"), data, 0o600)

	socket := filepath.Join(os.TempDir(), "mk-smoke.sock")
	_ = os.Remove(socket)
	logs, _ := os.Create(filepath.Join(out, "app.log"))
	app := exec.Command(program)
	app.Env = append(os.Environ(), "MEETKEEP_HOME="+home, "MYGO_AUTOMATION="+socket)
	app.Stdout, app.Stderr = logs, logs
	if err := app.Start(); err != nil {
		fmt.Println("FAIL  start:", err)
		os.Exit(1)
	}
	exited := make(chan error, 1)
	go func() { exited <- app.Wait() }()
	defer func() { _ = app.Process.Kill() }()

	d := &driver{out: out}
	for began := time.Now(); d.conn == nil; time.Sleep(300 * time.Millisecond) {
		conn, err := net.Dial("unix", socket)
		if err == nil {
			d.conn, d.reader = conn, bufio.NewReaderSize(conn, 1<<20)
			break
		}
		select {
		case err := <-exited:
			fmt.Println("FAIL  start: the app ended:", err)
			os.Exit(1)
		default:
		}
		if time.Since(began) > 90*time.Second {
			fmt.Println("FAIL  start: no socket after 90 s:", err)
			os.Exit(1)
		}
	}
	d.pass("start", "the app listens")

	main, ok := d.find("localhost", 60*time.Second)
	if !ok {
		d.fail("main window", fmt.Errorf("none of %+v", d.windows()))
		os.Exit(1)
	}
	if !d.until(main, `!!document.querySelector('[data-testid="records"]')`, 30*time.Second) {
		d.fail("records", fmt.Errorf("the page did not draw"))
	}
	d.pass("main window", fmt.Sprintf("%+v", main))
	d.tell(main, "page", `({agent: navigator.userAgent, platform: document.documentElement.dataset.platform, size: [innerWidth, innerHeight], ratio: devicePixelRatio})`)
	d.tell(main, "title bar", `(() => { const s = getComputedStyle(document.documentElement); const h = getComputedStyle(document.querySelector("header")); return {height: s.getPropertyValue("--mygo-titlebar-height"), left: s.getPropertyValue("--mygo-titlebar-inset-left"), right: s.getPropertyValue("--mygo-titlebar-inset-right"), headerLeft: h.paddingLeft, headerRight: h.paddingRight}; })()`)
	d.tell(main, "fonts", `({body: getComputedStyle(document.body).fontFamily, segoe: document.fonts.check('14px "Segoe UI"'), yahei: document.fonts.check('14px "Microsoft YaHei UI"'), cascadia: document.fonts.check('14px "Cascadia Mono"'), consolas: document.fonts.check('14px Consolas')})`)
	d.capture(main, "records")
	d.desktop("records")

	for _, section := range []string{"general", "transcription", "notes", "recording", "storage", "agent"} {
		d.visit(main, "/settings/"+section)
		d.capture(main, "settings-"+section)
	}
	d.desktop("settings")

	// The command line: installed from the settings, then run as a user would.
	command := ""
	if d.click(main, "settings-cli-install") && d.until(main, `!!document.querySelector('[data-testid="settings-cli-path"]')`, 8*time.Second) {
		value := d.tell(main, "command line", `document.querySelector('[data-testid="settings-cli-path"]').textContent`)
		_ = json.Unmarshal([]byte(value), &command)
		d.capture(main, "settings-agent-installed")
	} else {
		d.tell(main, "command line", `document.querySelector('[data-testid="settings-error"]')?.textContent ?? "no error shown"`)
	}
	run := func(step string, args ...string) string {
		if command == "" {
			return ""
		}
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", append([]string{"/c", "meetkeep"}, args...)...)
		} else {
			cmd = exec.Command(command, args...)
		}
		cmd.Env = append(os.Environ(), "MEETKEEP_HOME="+home)
		text, err := cmd.CombinedOutput()
		if err != nil {
			d.fail(step, fmt.Errorf("%v: %s", err, text))
			return ""
		}
		d.pass(step, strings.TrimSpace(string(text)))
		return string(text)
	}
	if command != "" {
		script, _ := os.ReadFile(command)
		d.pass("launcher", fmt.Sprintf("%s holds %q", command, script))
	}
	run("meetkeep status", "status")

	// A recording, with the windows that go with one.
	d.visit(main, "/")
	d.tell(main, "sources", `document.querySelector('[data-testid="capture-sources"]')?.textContent ?? null`)
	if d.click(main, "start-recording") {
		time.Sleep(time.Second)
		if value, _ := d.eval(main, `!!document.querySelector('[data-testid="new-sheet"]')`); value == "true" {
			d.capture(main, "new-sheet")
			d.click(main, "sheet-submit")
		}
		if panel, ok := d.find("/panel", 20*time.Second); ok {
			d.pass("panel", fmt.Sprintf("%+v", panel))
			time.Sleep(6 * time.Second)
			d.tell(panel, "elapsed", `document.querySelector('[data-testid="panel-elapsed"]')?.textContent ?? null`)
			d.tell(panel, "recording error", `document.querySelector('[data-testid="rec-error"]')?.textContent ?? null`)
			d.capture(panel, "panel")
			d.capture(main, "recording")
			d.desktop("recording")
			if d.click(panel, "panel-collapse") {
				if mini, ok := d.find("/mini", 10*time.Second); ok {
					time.Sleep(time.Second)
					d.capture(mini, "mini")
					d.desktop("mini")
					d.click(mini, "mini-stop")
				} else {
					d.fail("mini", fmt.Errorf("no window: %+v", d.windows()))
				}
			}
		} else {
			d.tell(main, "start error", `document.querySelector('[data-testid="sheet-error"]')?.textContent ?? document.body.innerText.slice(0, 400)`)
			d.fail("panel", fmt.Errorf("no window: %+v", d.windows()))
			d.capture(main, "no-panel")
		}
		time.Sleep(8 * time.Second)
		d.tell(main, "after the recording", `({path: location.pathname, text: document.querySelector("main")?.innerText.slice(0, 300)})`)
		d.capture(main, "after-recording")
		d.desktop("after-recording")
	}
	recorded := run("meetkeep list", "list", "--json")

	// A file handed over by the command line, and played back.
	if len(os.Args) > 3 {
		file, _ := filepath.Abs(os.Args[3])
		run("meetkeep import", "import", file)
		time.Sleep(6 * time.Second)
	}
	listed := run("meetkeep list", "list", "--json")
	var listing struct {
		Meetings []struct {
			ID string `json:"id"`
		} `json:"meetings"`
	}
	for _, text := range []string{listed, recorded} {
		if start := strings.Index(text, "{"); start >= 0 && len(listing.Meetings) == 0 {
			_ = json.Unmarshal([]byte(text[start:]), &listing)
		}
	}
	for _, meeting := range listing.Meetings {
		d.visit(main, "/meetings/"+meeting.ID)
		time.Sleep(2 * time.Second)
		d.tell(main, "playback of "+meeting.ID, fmt.Sprintf(`new Promise((done) => { const a = new Audio("recording://localhost/%s/audio"); a.preload = "metadata"; a.onloadedmetadata = () => done("loaded, " + a.duration + " s"); a.onerror = () => done("error " + (a.error && a.error.code)); setTimeout(() => done("nothing after 10 s"), 10000); })`, meeting.ID))
		d.capture(main, "meeting")
	}
	d.desktop("meeting")

	fmt.Printf("%d failed\n", d.failed)
	if d.failed > 0 {
		os.Exit(1)
	}
}
