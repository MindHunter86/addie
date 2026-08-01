package flags

import (
	"strings"

	"github.com/MindHunter86/addie/internal/config"
	"github.com/urfave/cli/v2"
)

const DynamicConfigCategory = "Dynamic Config Defaults"

func dynamicFlags(_ bool) []cli.Flag {
	balancerProcessing := config.NewDynamicFlag(100)
	qualityProcessing := config.NewDynamicFlag(100)
	maxQuality := config.NewDynamicFlag(1080)

	return []cli.Flag{
		// dynamic config
		&cli.StringFlag{
			Name:     "dynamic-config-source",
			Category: DynamicConfigCategory[:strings.LastIndex(DynamicConfigCategory, " ")],
			Aliases:  []string{"c"},
			EnvVars:  []string{"DYNAMIC_CONFIG"},
			Usage:    "`FILE/URL` with/to config settings in YAML format (only for dynamic values)",
		},
		&cli.StringFlag{
			Name:     "balancer-schema-source",
			Category: DynamicConfigCategory[:strings.LastIndex(DynamicConfigCategory, " ")],
			Aliases:  []string{"s"},
			EnvVars:  []string{"BALANCER_SCHEMA"},
			Usage:    "`FILE/URL` with/to schema for balancers in YAML format",
		},

		// dynamic flag defaults for further mutations
		// by dynamic config source (above)
		&cli.GenericFlag{
			Name:     "balancer-traffic-processing",
			Category: DynamicConfigCategory,
			EnvVars:  []string{"BALANCER_PROCESSING"},
			Usage:    "",
			Value:    balancerProcessing,
		},
		&cli.GenericFlag{
			Name:     "balancer-quality-processing",
			Category: DynamicConfigCategory,
			EnvVars:  []string{"BALANCER_QUALITY_PROCESSING"},
			Usage:    "",
			Value:    qualityProcessing,
		},
		&cli.GenericFlag{
			Name:     "balancer-max-available-quality",
			Category: DynamicConfigCategory,
			EnvVars:  []string{"BALANCER_MAX_QUALITY"},
			Usage:    "",
			Value:    maxQuality,
		},
	}
}
