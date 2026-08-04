package flags

import (
	"time"

	"github.com/urfave/cli/v2"
)

func applicationFlags(_ bool) []cli.Flag {
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
			Name:     "anilibria-api-baseurl",
			Category: "App: AniLibria API",
			Usage:    "",
			Value:    "https://api.anilibria.tv/v2",
		},

		// balancing settings
		&cli.UintFlag{
			Name:     "balancer-server-max-fails",
			Category: "App: Balancer",
			Usage:    "max fails for one request; max value - 10",
			Value:    3,
		},
		&cli.BoolFlag{
			Name:     "balancer-full-bypass",
			Category: "App: Balancer",
			Usage:    "use X-Server header as a balance target",
		},
		&cli.BoolFlag{
			Name:     "balancer-highcost-zone",
			Category: "App: Balancer",
			Usage:    "enable all mitigation, migration and bypass methods configured in consul for this instance",
		},
		&cli.IntFlag{
			Name:     "balancer-softer-step",
			Category: "App: Balancer",
			Value:    99,
			Usage: `balancer 'soft' mode for soft witching between qualities;
			'step' - is a static variable with some 'starting' value; each tick it will be decreased by 1;
			a request's quality will be updated when 'hardcoded payload' mod 'step' == 0`,
		},
		&cli.DurationFlag{
			Name:     "balancer-softer-tick",
			Category: "App: Balancer",
			Value:    1 * time.Second,
			Usage: `balancer 'soft' mode for soft witching between qualities;
			'tick' - is a ticker duration; each tick, the step will be decreased by 1;
			a request's quality will be updated when 'hardcoded payload' mod 'step' == 0`,
		},
		&cli.StringFlag{
			Name:     "balancer-node-servers",
			Category: "App: Balancer",
			Usage:    "cache{1..2}.example.com",
			Value:    "cache{1..9}.libria.fun",
		},
		&cli.StringFlag{
			Name:     "balancer-cloud-servers",
			Category: "App: Balancer",
			Usage:    "cache-cloud{1..2}.example.com",
			Value:    "cache-cloud{1..18}.libria.fun",
		},
		&cli.DurationFlag{
			Name:     "balancer-server-check-timeout",
			Category: "App: Balancer",
			Value:    1 * time.Second,
		},
		&cli.DurationFlag{
			Name:     "balancer-server-check-interval",
			Category: "App: Balancer",
			Value:    10 * time.Second,
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
	}
}
