package flags

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/urfave/cli/v2"
)

func httpServerFlags(expertMode bool) []cli.Flag {
	return []cli.Flag{
		// fiber-server settings
		&cli.StringFlag{
			Name:     "http-listen-addr",
			Category: "HTTP server settings",
			Usage:    "format - 127.0.0.1:8080, :8080",
			Value:    "127.0.0.1:8080",
		},
		&cli.StringFlag{
			Name:     "http-trusted-proxies",
			Category: "HTTP server settings",
			Usage:    "format - 192.168.0.0/16; can be separated by comma",
		},
		&cli.StringFlag{
			Name:     "http-realip-header",
			Category: "HTTP server settings",
			Value:    fiber.HeaderXForwardedFor,
		},
		&cli.BoolFlag{
			Name:     "http-prefork",
			Category: "HTTP server settings",
			Usage: `enables use of the SO_REUSEPORT socket option;
			if enabled, the application will need to be ran
			through a shell because prefork mode sets environment variables;
			EXPERIMENTAL! USE CAREFULLY!
			NOTICE: Prefork won't work if http-adv-* flags is used`,
			Hidden:             expertMode,
			DisableDefaultText: true,
		},
		&cli.DurationFlag{
			Name:     "http-read-timeout",
			Category: "HTTP server settings",
			Value:    10 * time.Second,
		},
		&cli.DurationFlag{
			Name:     "http-write-timeout",
			Category: "HTTP server settings",
			Value:    5 * time.Second,
		},
		&cli.DurationFlag{
			Name:     "http-idle-timeout",
			Category: "HTTP server settings",
			Value:    10 * time.Minute,
		},
		&cli.IntFlag{
			Name:     "http-concurrency-conns",
			Category: "HTTP server settings",
			Hidden:   expertMode,
			Value:    1 << 19, // 512k (fasthttp default: 256k)
		},
		&cli.BoolFlag{
			Name:               "http-pprof-enable",
			Category:           "HTTP server settings",
			Usage:              "enable golang http-pprof methods",
			DisableDefaultText: true,
		},
		&cli.StringFlag{
			Name:     "http-pprof-prefix",
			Category: "HTTP server settings",
			Usage:    "it should start with (but not end with) a slash. Example: '/test'",
			EnvVars:  []string{"PPROF_PREFIX"},
		},
		&cli.StringFlag{
			Name:     "http-pprof-secret",
			Category: "HTTP server settings",
			Usage:    "define static secret in x-pprof-secret header for avoiding unauthorized access",
			EnvVars:  []string{"PPROF_SECRET"},
		},
		&cli.StringFlag{
			Name:     "http-stats-secret",
			Category: "HTTP server settings",
			Usage:    "define static secret in Authorization header for avoiding unauthorized access",
			EnvVars:  []string{"STATS_SECRET"},
			Value:    "12de9f94ac51",
		},

		// fasthttp advanced settings
		&cli.BoolFlag{
			Name:               "http-adv-reuseport",
			Category:           "HTTP server settings",
			Usage:              "enables SO_REUSEPORT",
			DisableDefaultText: true,
			Hidden:             expertMode,
		},
		&cli.BoolFlag{
			Name:               "http-adv-deferaccept",
			Category:           "HTTP server settings",
			Usage:              "enables TCP_DEFER_ACCEPT",
			DisableDefaultText: true,
			Hidden:             expertMode,
		},
		&cli.BoolFlag{
			Name:               "http-adv-tcpfastopen",
			Category:           "HTTP server settings",
			Usage:              "enables TCP_FASTOPEN",
			DisableDefaultText: true,
			Hidden:             expertMode,
		},
		&cli.IntFlag{
			Name:     "http-adv-backlog",
			Category: "HTTP server settings",
			Usage:    "recommendation: 512 for common load, 2048 for highload",
			Hidden:   expertMode,
			Value:    0,
		},

		// fiber's limit request:
		&cli.IntFlag{
			Name:     "limit-request-maxrps",
			Category: "Limit Request",
			Usage:    "if 0 - disabled",
			Value:    20,
		},
		&cli.DurationFlag{
			Name:     "limit-request-expiration",
			Category: "Limit Request",
			Value:    10 * time.Second,
		},

		// !! LEGACY
		// !! LEGACY
		// !! LEGACY
		&cli.BoolFlag{
			Name:  "http-cors",
			Usage: "enable cors requests serving",
			Value: true,
		},
	}
}
