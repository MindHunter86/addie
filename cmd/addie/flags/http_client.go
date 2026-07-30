package flags

import (
	"time"

	"github.com/urfave/cli/v2"
)

func httpClientFlags(expertMode bool) []cli.Flag {
	return []cli.Flag{

		// http client commons
		&cli.DurationFlag{
			Name:     "http-client-read-timeout",
			Category: "Http Client Commons",
			Hidden:   expertMode,
			Value:    10 * time.Second,
		},
		&cli.DurationFlag{
			Name:     "http-client-write-timeout",
			Category: "Http Client Commons",
			Hidden:   expertMode,
			Value:    5 * time.Second,
		},
		&cli.DurationFlag{
			Name:     "http-client-conn-timeout",
			Category: "Http Client Commons",
			Usage:    "force connection rotation after this `time`",
			Hidden:   expertMode,
			Value:    10 * time.Minute,
		},
		&cli.DurationFlag{
			Name:     "http-client-idle-timeout",
			Category: "Http Client Commons",
			Hidden:   expertMode,
			Value:    5 * time.Minute,
		},
		&cli.IntFlag{
			Name:     "http-client-max-idle-conn",
			Category: "Http Client Commons",
			Hidden:   expertMode,
			Value:    256,
		},
		&cli.DurationFlag{
			Name:     "http-client-ssl-timeout",
			Category: "Http Client Commons",
			Usage:    "tls handshake timeout",
			Hidden:   expertMode,
			Value:    30 * time.Second,
		},
		&cli.IntFlag{
			Name:     "http-client-max-conns-per-host",
			Category: "Http Client Commons",
			Hidden:   expertMode,
			Value:    256,
		},
		&cli.DurationFlag{
			Name:     "http-client-dns-cache-dur",
			Category: "Http Client Commons",
			Hidden:   expertMode,
			Value:    1 * time.Minute,
		},
		&cli.IntFlag{
			Name:     "http-client-tcpdial-concurr",
			Category: "Http Client Commons",
			Usage:    "0 - unlimited",
			Hidden:   expertMode,
			Value:    0,
		},
		&cli.BoolFlag{
			Name:               "http-client-insecure",
			Category:           "Http Client Commons",
			Usage:              "TLS certificate verification disabling",
			Hidden:             expertMode,
			DisableDefaultText: true,
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
