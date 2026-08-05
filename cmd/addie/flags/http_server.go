package flags

import (
	"fmt"
	"time"

	v "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
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
			Action: validate[string]("http-listen-addr", nil,
				v.Required, is.DialString),
		},
		&cli.StringFlag{
			Name:     "http-trusted-proxies",
			Category: "HTTP server settings",
			Usage:    "format - 192.168.0.0/16; can be separated by comma",
			// TODO : MINOR : custom validator
		},
		&cli.StringFlag{
			Name:     "http-realip-header",
			Category: "HTTP server settings",
			Value:    fiber.HeaderXForwardedFor,
			Hidden:   expertMode,
			// TODO : MINOR : custom validator
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

			// TODO : MINOR : custom validator
			Action: func(ctx *cli.Context, b bool) error {
				return fmt.Errorf("temporary could not be manualy switched")
			},
		},
		&cli.DurationFlag{
			Name:     "http-timeout-read",
			Category: "HTTP server settings",
			Value:    10 * time.Second,
			Action: validate("http-timeout-read", toSeconds,
				v.Required, is.Int),
		},
		&cli.DurationFlag{
			Name:     "http-timeout-write",
			Category: "HTTP server settings",
			Value:    5 * time.Second,
			Action: validate("http-timeout-write", toSeconds,
				v.Required, is.Int),
		},
		&cli.DurationFlag{
			Name:     "http-timeout-idle",
			Category: "HTTP server settings",
			Value:    10 * time.Minute,
			Action: validate("http-timeout-idle", toSeconds,
				v.Required, is.Int),
		},
		&cli.IntFlag{
			Name:     "http-concurrency-conns",
			Category: "HTTP server settings",
			Hidden:   expertMode,
			Value:    1 << 19, // 512k (fasthttp default: 256k)
			Action: validate[int]("http-concurrency-conns", nil,
				v.Required, is.Int),
		},
		&cli.BoolFlag{
			Name:               "http-pprof-enable",
			Category:           "HTTP server settings",
			Usage:              "enable golang http-pprof methods",
			DisableDefaultText: true,
			Hidden:             expertMode,
			// TODO : MINOR : custom validator
		},
		&cli.StringFlag{
			Name:     "http-pprof-prefix",
			Category: "HTTP server settings",
			Usage:    "it should start with (but not end with) a slash. Example: '/test'",
			EnvVars:  []string{"PPROF_PREFIX"},
			Hidden:   expertMode,
			Action: validate[string]("http-pprof-prefix", nil,
				v.Required, is.RequestURI),
		},
		&cli.StringFlag{
			Name:     "http-pprof-secret",
			Category: "HTTP server settings",
			Usage:    "define static secret in x-pprof-secret header for avoiding unauthorized access",
			EnvVars:  []string{"PPROF_SECRET"},
			Hidden:   expertMode,
			Action: validate[string]("http-pprof-secret", nil,
				v.Required, v.Length(5, 64)),
		},
		&cli.StringFlag{
			Name:     "http-stats-secret",
			Category: "HTTP server settings",
			Usage:    "define static secret in Authorization header for avoiding unauthorized access",
			EnvVars:  []string{"STATS_SECRET"},
			Value:    "12de9f94ac51",
			Hidden:   expertMode,
			Action: validate[string]("http-stats-secret", nil,
				v.Required, v.Length(5, 64)),
		},

		// fasthttp advanced settings
		&cli.BoolFlag{
			Name:               "http-adv-reuseport",
			Category:           "HTTP server settings",
			Usage:              "enables SO_REUSEPORT",
			DisableDefaultText: true,
			Hidden:             expertMode,
			// TODO : MINOR : custom validator
		},
		&cli.BoolFlag{
			Name:               "http-adv-deferaccept",
			Category:           "HTTP server settings",
			Usage:              "enables TCP_DEFER_ACCEPT",
			DisableDefaultText: true,
			Hidden:             expertMode,
			// TODO : MINOR : custom validator
		},
		&cli.BoolFlag{
			Name:               "http-adv-tcpfastopen",
			Category:           "HTTP server settings",
			Usage:              "enables TCP_FASTOPEN",
			DisableDefaultText: true,
			Hidden:             expertMode,
			// TODO : MINOR : custom validator
		},
		&cli.IntFlag{
			Name:     "http-adv-backlog",
			Category: "HTTP server settings",
			Usage:    "recommendation: 512 for common load, 2048 for highload",
			Hidden:   expertMode,
			Value:    0,
			Action: validate[int]("http-adv-backlog", nil,
				is.Int, v.Min(128), v.Max(8192)),
		},

		// fiber's limit request:
		&cli.IntFlag{
			Name:     "limit-request-maxrps",
			Category: "Limit Request",
			Usage:    "if 0 - disabled",
			Value:    20,
			Action: validate[int]("limit-request-maxrps", nil,
				is.Int),
		},
		&cli.DurationFlag{
			Name:     "limit-request-expiration",
			Category: "Limit Request",
			Value:    10 * time.Second,
			Action: validate("limit-request-expiration", toSeconds,
				v.Required, is.Int),
		},

		// !! LEGACY
		// !! LEGACY
		// !! LEGACY
		&cli.BoolFlag{
			Name:     "http-cors",
			Category: "Legacy",
			Usage:    "enable cors requests serving",
			Value:    true,
		},
	}
}
