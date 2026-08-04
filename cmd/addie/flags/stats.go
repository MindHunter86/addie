package flags

import (
	"time"

	"github.com/urfave/cli/v2"
)

func statsFlags(expertMode bool) []cli.Flag {
	return []cli.Flag{
		// stats settings
		&cli.StringFlag{
			Name:     "stats-graphite-prefix",
			Category: "Stats Collection",
			Value:    "mh00net.golang.saeko",
		},
		&cli.StringFlag{
			Name:     "stats-graphite-proto",
			Category: "Stats Collection",
			Hidden:   expertMode,
			Value:    "udp4",
		},
		&cli.StringFlag{
			Name:     "stats-graphite-server",
			Category: "Stats Collection",
			Usage:    "127.0.0.1:2003",
			EnvVars:  []string{"GRAPHITE_SERVER"},
			Value:    "",
		},
		&cli.DurationFlag{
			Name:     "stats-metrics-interval",
			Category: "Stats Collection",
			Usage:    "must be a multiple of 5 seconds",
			Hidden:   expertMode,
			Value:    5 * time.Second,
		},
		&cli.DurationFlag{
			Name:     "stats-metrics-loop-warn",
			Category: "Stats Collection",
			Usage:    "warning for too long loop metrics collection",
			Hidden:   expertMode,
			Value:    100 * time.Millisecond,
		},
		&cli.IntFlag{
			Name:     "stats-queue-buffer-size",
			Category: "Stats Collection",
			Hidden:   expertMode,
			Value:    1 << 8, // 256
		},
	}
}
