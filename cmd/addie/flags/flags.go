package flags

import (
	"fmt"
	"time"

	"github.com/urfave/cli/v2"
)

func FlagsInitialization(em bool, aname string) []cli.Flag {
	f := []cli.Flag{}

	f = []cli.Flag{

		// http client settings
		&cli.BoolFlag{
			Name:  "http-client-insecure",
			Usage: "Flag for TLS certificate verification disabling",
		},
		&cli.DurationFlag{
			Name:  "http-client-timeout",
			Usage: "Internal HTTP client connection `TIMEOUT` (format: 1000ms, 1s)",
			Value: 3 * time.Second,
		},
		&cli.DurationFlag{
			Name:  "http-tcp-timeout",
			Usage: "",
			Value: 1 * time.Second,
		},
		&cli.DurationFlag{
			Name:  "http-tls-handshake-timeout",
			Usage: "",
			Value: 1 * time.Second,
		},
		&cli.DurationFlag{
			Name:  "http-idle-timeout",
			Usage: "",
			Value: 300 * time.Second,
		},
		&cli.DurationFlag{
			Name:  "http-keepalive-timeout",
			Usage: "",
			Value: 300 * time.Second,
		},
		&cli.IntFlag{
			Name:  "http-max-idle-conns",
			Usage: "",
			Value: 100,
		},
		&cli.BoolFlag{
			Name:  "http-debug",
			Usage: "",
		},

		// fiber (http server) settings
		&cli.StringFlag{
			Name:  "http-listen-addr",
			Usage: "Ex: 127.0.0.1:8080, :8080",
			Value: "127.0.0.1:8080",
		},
		&cli.StringFlag{
			Name:  "http-trusted-proxies",
			Usage: "Ex: 10.0.0.0/8; Separated by comma",
		},
		&cli.BoolFlag{
			Name: "http-prefork",
			Usage: `Enables use of the SO_REUSEPORT socket option;
			if enabled, the application will need to be ran
			through a shell because prefork mode sets environment variables`,
		},
		&cli.BoolFlag{
			Name:  "http-cors",
			Usage: "enable cors requests serving",
			Value: true,
		},
		&cli.BoolFlag{
			Name:  "http-pprof-enable",
			Usage: "enable golang http-pprof methods",
		},

		// limiter settings
		&cli.BoolFlag{
			Name:  "limiter-use-bbolt",
			Usage: "use bbolt key\value file database instead of memory database",
		},
		&cli.IntFlag{
			Name:  "limiter-max-req",
			Value: 200,
		},
		&cli.DurationFlag{
			Name:  "limiter-records-duration",
			Value: 5 * time.Minute,
		},

		// bbolt settings
		&cli.StringFlag{
			Name:  "database-prefix",
			Value: ".",
		},

		// anilibria settings
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

		// balancer
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

		// ...
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
			Value: fmt.Sprintf("anilibria/%s", aname),
		},
	}

	return f
}
