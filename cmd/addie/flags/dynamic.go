package flags

import (
	"strings"
	"time"

	"github.com/MindHunter86/addie/internal/config"
	"github.com/MindHunter86/addie/internal/utils"
	"github.com/urfave/cli/v2"
)

func dynamicFlags(expertMode bool) []cli.Flag {
	balancerProcessing := config.NewDynamicFlag(100)
	qualityProcessing := config.NewDynamicFlag(100)
	maxQuality := config.NewDynamicFlag(1080)

	balancerRouting := config.NewDynamicFlag(&config.BalancerRoutingMap{})
	balancerRegions := config.NewDynamicFlag(&config.BalancerRegionMap{})

	return []cli.Flag{
		// dynamic config
		&cli.StringFlag{
			Name:     "dynamic-config-source",
			Category: utils.DynamicConfigCategory[:strings.LastIndex(utils.DynamicConfigCategory, " ")],
			Aliases:  []string{"c"},
			EnvVars:  []string{"DYNAMIC_CONFIG"},
			Usage:    "`FILE/URL` with/to config settings in YAML format (only for dynamic values)",
		},
		&cli.StringFlag{
			Name:     "dynamic-config-source-tmp",
			Category: utils.DynamicConfigCategory[:strings.LastIndex(utils.DynamicConfigCategory, " ")],
			Usage:    "temporary `DIR` for downloading 'dynamic-config-source' file",
			Hidden:   expertMode,
		},
		&cli.DurationFlag{
			Name:     "dynamic-config-fetch-interval",
			Category: utils.DynamicConfigCategory[:strings.LastIndex(utils.DynamicConfigCategory, " ")],
			Usage:    "every `INTERVAL` check update for remote changes, if dynamic-config-source point to URL",
			Hidden:   expertMode,
			Value:    30 * time.Second,
		},
		&cli.Int64Flag{
			Name:     "dynamic-config-max-size",
			Category: utils.DynamicConfigCategory[:strings.LastIndex(utils.DynamicConfigCategory, " ")],
			Usage:    "file more than `SIZE` in bytes will be rejected",
			Hidden:   expertMode,
			Value:    1 << 16, // 64Kib
		},

		// dynamic flag defaults for further mutations
		// by dynamic config source (above)
		&cli.GenericFlag{
			Name:     "balancer-traffic-processing",
			Category: utils.DynamicConfigCategory,
			EnvVars:  []string{"BALANCER_PROCESSING"},
			Usage:    "",
			Value:    balancerProcessing,
		},
		&cli.GenericFlag{
			Name:     "balancer-quality-processing",
			Category: utils.DynamicConfigCategory,
			EnvVars:  []string{"BALANCER_QUALITY_PROCESSING"},
			Usage:    "",
			Value:    qualityProcessing,
		},
		&cli.GenericFlag{
			Name:     "balancer-max-available-quality",
			Category: utils.DynamicConfigCategory,
			EnvVars:  []string{"BALANCER_MAX_QUALITY"},
			Usage:    "",
			Value:    maxQuality,
		},

		// internal values for yaml parsers, should be hidden from --help!
		&cli.GenericFlag{
			Name:     "balancer-routng",
			Category: utils.DynamicConfigCategory,
			Hidden:   true,
			Value:    balancerRouting,
		},
		&cli.GenericFlag{
			Name:     "balancer-regions",
			Category: utils.DynamicConfigCategory,
			Hidden:   true,
			Value:    balancerRegions,
		},
	}
}
