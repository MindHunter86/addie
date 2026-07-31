//go:build !windows && !plan9

package service

import (
	"os"
	"os/signal"
	"syscall"
)

func (*Service) listenForDebugSignal() chan os.Signal {
	kernDumpSignal := make(chan os.Signal, 1)
	signal.Notify(kernDumpSignal, syscall.SIGUSR2)
	return kernDumpSignal
}
