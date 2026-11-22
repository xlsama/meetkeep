package memory

import (
	"os"
	"unsafe"

	"github.com/ebitengine/purego"
)

// usage is rusage_info_v0 of <sys/resource.h>.
type usage struct {
	uuid          [16]byte
	userTime      uint64
	systemTime    uint64
	idleWakeups   uint64
	interrupts    uint64
	pageins       uint64
	wired         uint64
	resident      uint64
	physFootprint uint64
	started       uint64
	exited        uint64
}

var procPidRusage func(pid int32, flavor int32, buffer unsafe.Pointer) int32

func init() {
	library, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return
	}
	defer func() {
		if recover() != nil {
			procPidRusage = nil
		}
	}()
	purego.RegisterLibFunc(&procPidRusage, library, "proc_pid_rusage")
}

// Footprint returns the memory the app holds, in bytes, as Activity
// Monitor counts it, or 0 when it cannot be read.
func Footprint() int64 {
	if procPidRusage == nil {
		return 0
	}
	var u usage
	if procPidRusage(int32(os.Getpid()), 0, unsafe.Pointer(&u)) != 0 {
		return 0
	}
	return int64(u.physFootprint)
}
