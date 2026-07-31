//go:build !windows && !plan9

package service

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/MindHunter86/addie/internal/config"
)

func (*Service) listenForDebugSignal() chan os.Signal {
	kernDumpSignal := make(chan os.Signal, 1)
	signal.Notify(kernDumpSignal, syscall.SIGUSR2)
	return kernDumpSignal
}
func (*Service) sometestfunction() {
	bypass := gCli.Generic("balancer-full-bypass").(*config.DynamicString)
	bypass.String()
	gCli.Gen

}
