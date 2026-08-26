//go:build windows

package api

import (
	"errors"

	"golang.org/x/sys/windows"
)

// isAddrInUse reports whether a bind failed because something else already
// holds the port.
//
// Winsock reports that as [windows.WSAEADDRINUSE], which is not the value
// [syscall.EADDRINUSE] carries here: on Windows that constant is a
// synthesised APPLICATION_ERROR with nothing mapping the Winsock errno back
// to it, so comparing against it is always false and the walk would never
// happen.
func isAddrInUse(err error) bool {
	return errors.Is(err, windows.WSAEADDRINUSE)
}
