//go:build windows

package service

import (
	"os"
)

// listenForDebugSignal returns a dummy channel on Windows since SIGUSR2 is unsupported.
func (*Service) listenForDebugSignal() chan os.Signal {
	kernDumpSignal := make(chan os.Signal, 1)
	return kernDumpSignal
}
