// Package memory tells how much memory the app holds.
package memory

import "time"

// Watch follows the memory the app holds until stop is called, which
// returns the most it held, in bytes, or 0 where it cannot be measured.
func Watch() (stop func() int64) {
	peak := Footprint()
	done, ended := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(ended)
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				peak = max(peak, Footprint())
			}
		}
	}()
	return func() int64 {
		close(done)
		<-ended
		return max(peak, Footprint())
	}
}
