package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"runtime"

	"github.com/egoist/mygo"
)

// light returns a template icon, black on nothing, with its ink white.
func light(icon []byte) []byte {
	source, err := png.Decode(bytes.NewReader(icon))
	if err != nil {
		return icon
	}
	bounds := source.Bounds()
	drawn := image.NewNRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, alpha := source.At(x, y).RGBA()
			drawn.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: uint8(alpha >> 8)})
		}
	}
	var out bytes.Buffer
	if png.Encode(&out, drawn) != nil {
		return icon
	}
	return out.Bytes()
}

// drawTray shows the icon of the app at rest or recording. macOS tints
// the icon at rest for its menu bar; elsewhere its ink is made light on a
// dark taskbar.
func drawTray(tray *mygo.Tray, recording bool) {
	switch {
	case recording:
		_ = tray.SetIcon(trayRecordingIcon, false)
	case runtime.GOOS == "darwin":
		_ = tray.SetIcon(trayIcon, true)
	case tray.Dark():
		_ = tray.SetIcon(light(trayIcon), false)
	default:
		_ = tray.SetIcon(trayIcon, false)
	}
}
