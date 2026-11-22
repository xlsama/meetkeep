package memory

import (
	"runtime"
	"testing"
)

func TestWatchSeesWhatIsHeld(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("not measured here")
	}
	before := Footprint()
	if before < 1<<20 {
		t.Fatalf("the footprint reads %d bytes", before)
	}
	stop := Watch()
	held := make([]byte, 256<<20)
	for i := range held {
		held[i] = 1
	}
	peak := stop()
	runtime.KeepAlive(held)
	if peak-before < 200<<20 {
		t.Errorf("the peak rose by %d MB while 256 MB were held", (peak-before)>>20)
	}
}
