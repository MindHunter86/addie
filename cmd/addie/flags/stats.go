package flags

import (
	"regexp"
	"time"

	v "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/urfave/cli/v2"
)

func statsFlags(expertMode bool) []cli.Flag {
	prototest, err := regexp.Compile("(udp|tcp)[46]?")
	if err != nil {
		panic(err)
	}

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
			Action: validate[string]("stats-graphite-proto", nil,
				v.Match(prototest)),
		},
		&cli.StringFlag{
			Name:     "stats-graphite-server",
			Category: "Stats Collection",
			Usage:    "127.0.0.1:2003",
			EnvVars:  []string{"GRAPHITE_SERVER"},
			Action: validate[string]("stats-graphite-server", nil,
				is.DialString),
		},
		&cli.DurationFlag{
			Name:     "stats-metrics-interval",
			Category: "Stats Collection",
			Usage:    "`INTERVAL` in seconds for stats collector; must be a multiple of 5 seconds",
			Hidden:   expertMode,
			Value:    5 * time.Second,
			Action: validate("stats-metrics-interval", toSecondsCeil,
				v.Required, v.Min(5)),
			// TODO : MINOR : add mod(v) == 0 check
		},
		&cli.DurationFlag{
			Name:     "stats-metrics-loop-warn",
			Category: "Stats Collection",
			Usage:    "warning for too long loop metrics collection",
			Hidden:   expertMode,
			Value:    100 * time.Millisecond,
			Action: validate("stats-metrics-loop-warn", time.Duration.Milliseconds,
				v.Required, v.Min(10)),
		},
		&cli.IntFlag{
			Name:     "stats-queue-buffer-size",
			Category: "Stats Collection",
			Hidden:   expertMode,
			Value:    1 << 8, // 256
			Action: validate[int]("stats-queue-buffer-size", nil,
				v.Required, v.Min(1<<5)),
		},
	}
}
