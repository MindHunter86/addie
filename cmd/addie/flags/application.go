package flags

import (
	"fmt"
	"time"

	"github.com/urfave/cli/v2"
)

func applicationFlags(_ bool, appname string) []cli.Flag {
	// validate := validator.New()

	return []cli.Flag{
		// web settings page
		// &cli.BoolFlag{
		// 	Name:     "settingspage-enable",
		// 	Category: "Web Internals",
		// 	Action: func(c *cli.Context, _ bool) error {
		// 		u, p := c.String("settingspage-auth-username"), c.String("settingspage-auth-password")
		// 		if validate.Var(u, "required") != nil || validate.Var(p, "required") != nil {
		// 			return errors.New("settingspage-auth-username, settingspage-auth-password flags required since settingspage-enable is used")
		// 		}
		// 		return nil
		// 	},
		// },

		// anilibria APIs settings
		&cli.StringFlag{
			Name:  "anilibria-baseurl",
			Usage: "",
			Value: "https://www.anilibria.tv",
		},
		&cli.StringFlag{
			Name:  "anilibria-api-baseurl",
			Usage: "",
			Value: "https://api.anilibria.tv/v2",
		},

		// balancing settings
		&cli.UintFlag{
			Name:  "balancer-server-max-fails",
			Usage: "max fails for one request; max value - 10",
			Value: 3,
		},
		&cli.BoolFlag{
			Name:  "balancer-full-bypass",
			Usage: "use X-Server header as a balance target",
		},
		&cli.BoolFlag{
			Name:  "balancer-highcost-zone",
			Usage: "enable all mitigation, migration and bypass methods configured in consul for this instance",
		},
		&cli.IntFlag{
			Name:  "balancer-softer-step",
			Value: 99,
			Usage: `balancer 'soft' mode for soft witching between qualities;
			'step' - is a static variable with some 'starting' value; each tick it will be decreased by 1;
			a request's quality will be updated when 'hardcoded payload' mod 'step' == 0`,
		},
		&cli.DurationFlag{
			Name:  "balancer-softer-tick",
			Value: 1 * time.Second,
			Usage: `balancer 'soft' mode for soft witching between qualities;
			'tick' - is a ticker duration; each tick, the step will be decreased by 1;
			a request's quality will be updated when 'hardcoded payload' mod 'step' == 0`,
		},

		// link generation settings
		&cli.DurationFlag{
			Name:    "link-expiration",
			Usage:   "",
			Value:   10 * time.Second,
			EnvVars: []string{"LINK_EXPIRATION"},
		},
		&cli.StringFlag{
			Name:        "link-secret",
			Usage:       "",
			Value:       "TZj3Ts1Lsvk",
			EnvVars:     []string{"SIGN_SECRET"},
			DefaultText: "CHANGE DEFAULT SECRET",
		},

		// consul settings
		&cli.BoolFlag{
			Name: "consul-ignore-errors",
		},
		&cli.StringFlag{
			Name:    "consul-address",
			Usage:   "consul API uri",
			Value:   "http://127.0.0.1:8500",
			EnvVars: []string{"CONSUL_ADDRESS"},
		},
		&cli.StringFlag{
			Name:  "consul-service-nodes",
			Usage: "service name (id) with cache-nodes used for balancing",
			Value: "cache-node-internal",
		},
		&cli.StringFlag{
			Name:  "consul-service-cloud",
			Usage: "service name (id) with cache-clouds used for balancing",
			Value: "cache-cloud-ingress",
		},
		&cli.StringFlag{
			Name:  "consul-entries-domain",
			Usage: "add domain for all service entries",
			Value: "libria.fun",
		},
		&cli.StringFlag{
			Name:  "consul-kv-prefix",
			Value: fmt.Sprintf("anilibria/%s", appname),
		},
	}
}
