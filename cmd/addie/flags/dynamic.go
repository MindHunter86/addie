package flags

import (
	"github.com/MindHunter86/addie/internal/config"
	"github.com/urfave/cli/v2"
)

func dynamicFlags(_ bool) []cli.Flag {
	balancerFullBypass := config.NewDynamicString("111")

	return []cli.Flag{
		// generic flags for further mutations
		&cli.GenericFlag{
			Name:  "balancer-full-bypass",
			Usage: "",
			Value: balancerFullBypass,
		},
	}
}
