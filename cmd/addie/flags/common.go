package flags

import (
	"github.com/urfave/cli/v2"
)

func commonFlags(expertMode bool) []cli.Flag {
	return []cli.Flag{
		// common settings
		&cli.StringFlag{
			Name:    "log-level",
			Value:   "info",
			Usage:   "levels: trace, debug, info, warn, err, panic, disabled",
			Aliases: []string{"l"},
			EnvVars: []string{"LOG_LEVEL"},
		},
		&cli.BoolFlag{
			Name:               "expert-mode",
			Usage:              "show hidden flags",
			DisableDefaultText: true,
		},

		// common settings : syslog
		&cli.StringFlag{
			Name:     "syslog-server",
			Category: "Syslog settings",
			Usage:    "syslog server (optional); syslog sender is not used if value is empty",
			Value:    "",
			EnvVars:  []string{"SYSLOG_ADDRESS"},
		},
		&cli.StringFlag{
			Name:     "syslog-proto",
			Category: "Syslog settings",
			Usage:    "syslog protocol (optional); tcp or udp is possible",
			Value:    "tcp",
			EnvVars:  []string{"SYSLOG_PROTO"},
		},
		&cli.StringFlag{
			Name:     "syslog-tag",
			Category: "Syslog settings",
			Usage:    "optional setting; more information in syslog RFC",
			Value:    "",
			Hidden:   expertMode,
		},
	}
}
