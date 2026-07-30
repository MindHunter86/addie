//go:build !windows && !plan9

package utils

import (
	"io"

	"github.com/MindHunter86/addie/internal/utils/syslog"
	"github.com/urfave/cli/v2"
)

func setUpSyslogWriter(c *cli.Context) (_ io.Writer, e error) {
	return syslog.Dial(c.String("syslog-proto"), c.String("syslog-server"), syslog.LOG_INFO, c.String("syslog-tag"))
}
