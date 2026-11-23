package main

import (
	"os"
	"syscall"
)

// The system's allocator keeps the large blocks a program frees for the next
// allocation, and counts them as the program's: after the models ran on an
// hour of audio that is a gigabyte the app no longer uses. MallocLargeCache=0
// turns that off, and is read only as a process starts: Info.plist sets it
// for an app opened from Finder or the Dock (LSEnvironment in mygo.config.ts),
// and one run directly starts itself again with it. An opened app must not
// start itself again, or macOS no longer shows its menu bar item.
func init() {
	if os.Getenv("MallocLargeCache") != "" || os.Getenv("MYGO_GENERATE") != "" {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	_ = syscall.Exec(exe, os.Args, append(os.Environ(), "MallocLargeCache=0"))
}
