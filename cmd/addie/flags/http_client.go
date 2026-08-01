package flags

import (
	"time"

	"github.com/urfave/cli/v2"
)

func httpClientFlags(expertMode bool) []cli.Flag {
	return []cli.Flag{

		// http client commons
		&cli.BoolFlag{
			Name:               "http-client-ssl-insecure",
			Category:           "Http Client Commons",
			Usage:              "TLS certificate verification disabling",
			Hidden:             expertMode,
			DisableDefaultText: true,
		},
		&cli.IntFlag{
			Name:     "http-client-max-conns",
			Category: "Http Client Commons",
			Hidden:   expertMode,
			Value:    32,
		},
		&cli.DurationFlag{
			Name:     "http-client-timeout-read",
			Category: "Http Client Commons",
			Hidden:   expertMode,
			Value:    3 * time.Second,
		},
		&cli.DurationFlag{
			Name:     "http-client-timeout-write",
			Category: "Http Client Commons",
			Hidden:   expertMode,
			Value:    3 * time.Second,
		},
		&cli.DurationFlag{
			Name:     "http-client-timeout-idle",
			Category: "Http Client Commons",
			Usage:    "idle keep-alive connections are closed after this duration",
			Hidden:   expertMode,
			Value:    5 * time.Minute,
		},
		&cli.DurationFlag{
			Name:     "http-client-timeout-conn",
			Category: "Http Client Commons",
			Usage:    "keep-alive connections are closed after this duration",
			Hidden:   expertMode,
			Value:    10 * time.Minute,
		},
		&cli.DurationFlag{
			Name:     "http-client-timeout-conn-wait",
			Category: "Http Client Commons",
			Usage:    "maximum duration for waiting for a free connection",
			Hidden:   expertMode,
			Value:    3 * time.Second,
		},
		&cli.IntFlag{
			Name:     "http-client-tcpdial-concurr",
			Category: "Http Client Commons",
			Usage:    "concurrency controls the maximum number of concurrent Dials that can be performed using this object. Setting this to 0 means unlimited",
			Hidden:   expertMode,
			Value:    0,
		},
		&cli.DurationFlag{
			Name:     "http-client-dnscache-dur",
			Category: "Http Client Commons",
			Usage:    "this may be used to override the default DNS cache duration",
			Hidden:   expertMode,
			Value:    1 * time.Minute,
		},

		// !! LEGACY
		// !! LEGACY
		// !! LEGACY
		// http client settings
		&cli.DurationFlag{
			Name:     "http-client-timeout",
			Category: "Http Client Legacy",
			Hidden:   expertMode,
			Usage:    "Internal HTTP client connection `TIMEOUT` (format: 1000ms, 1s)",
			Value:    3 * time.Second,
		},
		&cli.DurationFlag{
			Name:     "http-tcp-timeout",
			Category: "Http Client Legacy",
			Hidden:   expertMode,
			Usage:    "",
			Value:    1 * time.Second,
		},
		&cli.DurationFlag{
			Name:     "http-tls-handshake-timeout",
			Category: "Http Client Legacy",
			Hidden:   expertMode,
			Usage:    "",
			Value:    1 * time.Second,
		},
		&cli.DurationFlag{
			Name:     "http-keepalive-timeout",
			Category: "Http Client Legacy",
			Hidden:   expertMode,
			Usage:    "",
			Value:    300 * time.Second,
		},
		&cli.IntFlag{
			Name:     "http-max-idle-conns",
			Category: "Http Client Legacy",
			Hidden:   expertMode,
			Usage:    "",
			Value:    100,
		},
		&cli.BoolFlag{
			Name:     "http-debug",
			Category: "Http Client Legacy",
			Hidden:   expertMode,
			Usage:    "",
		},
	}
}
