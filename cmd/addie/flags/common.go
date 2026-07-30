package flags

import (
	"github.com/urfave/cli/v2"
)

func commonFlags(expertMode bool) []cli.Flag {
	return []cli.Flag{
		// common settings
		&cli.StringFlag{
			Name:  "log-level",
			Value: "debug",
			Usage: `levels: trace, debug, info, warn, err, panic, disabled;
					used only for system messages (also see accesslog)`,
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
			Category: "Advanced logging settings",
			Usage:    "syslog server (optional); syslog sender is not used if value is empty",
			EnvVars:  []string{"SYSLOG_ADDRESS"},
		},
		&cli.StringFlag{
			Name:     "syslog-proto",
			Category: "Advanced logging settings",
			Usage:    "syslog protocol (optional); tcp or udp is possible",
			Value:    "tcp",
			EnvVars:  []string{"SYSLOG_PROTO"},
		},
		&cli.StringFlag{
			Name:     "syslog-tag",
			Category: "Advanced logging settings",
			Usage:    "optional setting; more information in syslog RFC",
			Hidden:   expertMode,
		},

		// common settings : http access levels
		&cli.StringFlag{
			Name:     "accesslog-level",
			Category: "Advanced logging settings",
			Usage:    "accesslog to stdout; disabled if 'disabled'; uses log-level if empty",
			EnvVars:  []string{"ACCESSLOG_LEVEL"},
		},
		&cli.StringFlag{
			Name:     "accesslog-syslog-level",
			Category: "Advanced logging settings",
			Usage:    "acceslog to syslog; uses accesslog-level if empty; disabled if 'disabled'; if enabled, stdout accesslog will be skipped",
			EnvVars:  []string{"ACCESSLOG_SYSLOG_LEVEL"},

			// Action: func(ctx *cli.Context, s string) error {
			// 	return nil
			// },
		},

		// GOGC
		&cli.UintFlag{
			Name:     "runtime-gogc",
			Category: "Go Runtime",
			Value:    11300,
			EnvVars:  []string{"RUNTIME_GOGC"},
			Hidden:   expertMode,
		},
	}
}
