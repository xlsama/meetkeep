package control

import (
	"errors"
	"syscall"
)

// wsaeconnrefused is what Windows answers a socket nothing listens on with.
const wsaeconnrefused = syscall.Errno(10061)

func refused(err error) bool { return errors.Is(err, wsaeconnrefused) }
