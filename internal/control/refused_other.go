//go:build !windows

package control

import (
	"errors"
	"syscall"
)

func refused(err error) bool { return errors.Is(err, syscall.ECONNREFUSED) }
