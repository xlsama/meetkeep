//go:build !darwin

package memory

func Footprint() int64 { return 0 }
