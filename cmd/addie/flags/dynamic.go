package flags

import (
	"github.com/MindHunter86/addie/internal/config"
	"github.com/urfave/cli/v2"
)

func dynamicFlags(_ bool) []cli.Flag {
	// balancer-handling
	balancerHandling := config.NewDynamicFlag(1)

	// balancer-max-quality
	// balancer-quality-handling

	// balancer-force-ru-migration

	return []cli.Flag{
		// generic flags for further mutations
		&cli.GenericFlag{
			Name:     "balancer-handling",
			Category: "Dynamic Config Defaults",
			Usage:    "",
			Value:    balancerHandling,
		},
	}
}
