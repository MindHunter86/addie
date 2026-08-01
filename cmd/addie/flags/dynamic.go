package flags

import (
	"strings"

	"github.com/MindHunter86/addie/internal/config"
	"github.com/MindHunter86/addie/internal/utils"
	"github.com/urfave/cli/v2"
)

func dynamicFlags(_ bool) []cli.Flag {
	balancerProcessing := config.NewDynamicFlag(100)
	qualityProcessing := config.NewDynamicFlag(100)
	maxQuality := config.NewDynamicFlag(1080)

	return []cli.Flag{
		// dynamic config
		&cli.StringFlag{
			Name:     "dynamic-config-source",
			Category: utils.DynamicConfigCategory[:strings.LastIndex(utils.DynamicConfigCategory, " ")],
			Aliases:  []string{"c"},
			EnvVars:  []string{"DYNAMIC_CONFIG"},
			Usage:    "`FILE/URL` with/to config settings in YAML format (only for dynamic values)",
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
	}
}
