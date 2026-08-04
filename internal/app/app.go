package app

import (
	"context"
	"io"
	"regexp"

	"github.com/MindHunter86/addie/internal/balancer"
	"github.com/MindHunter86/addie/internal/runtime"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

var (
	gCli *cli.Context
	gLog *zerolog.Logger
	gCtx context.Context

	gAniApi     *ApiClient
	gController *Controller
)

type App struct {
	// fb    *fiber.App
	// abort context.CancelFunc

	cache   *CachedTitlesBucket
	runtime *runtime.Runtime

	cloudBalancer balancer.Balancer
	bareBalancer  balancer.Balancer

	chunkRegexp *regexp.Regexp

	syslogWriter io.Writer
}

func NewApp(s io.Writer) (app *App) {
	app = &App{}
	app.syslogWriter = s

	// api controller init
	gController = NewController()

	return app
}

func (m *App) bootstrap(_ context.Context) (e error) {
	// BOOTSTRAP SECTION:
	// common
	const chunksplit = `^(\/[^\/]+\/[^\/]+\/[^\/]+\/)([^\/]+)\/([^\/]+)\/([^\/]+)\/([^.\/]+)\.ts$`
	m.chunkRegexp = regexp.MustCompile(chunksplit)

	// anilibria API
	gLog.Info().Msg("starting anilibria api client...")
	if gAniApi, e = NewApiClient(); e != nil {
		return
	}

	// fake quality cooler cache
	gLog.Info().Msg("starting fake quality cache buckets...")
	m.cache = NewCachedTitlesBucket()

	// runtime
	if m.runtime, e = runtime.NewRuntime(gCtx); e != nil {
		return
	}

	// balancer V2
	gLog.Info().Msg("bootstrap balancer_v2 subsystems...")

	m.bareBalancer = balancer.NewClusterBalancer(gCtx, balancer.BalancerClusterNodes)
	m.cloudBalancer = balancer.NewClusterBalancer(gCtx, balancer.BalancerClusterCloud)

	var bservers, cservers []string
	if bservers, e = m.bareBalancer.GetFQDNsByBrace(gCli.String("balancer-node-servers")); e != nil {
		return
	}
	if cservers, e = m.cloudBalancer.GetFQDNsByBrace(gCli.String("balancer-cloud-servers")); e != nil {
		return
	}

	m.bareBalancer.UpdateServersByFQDN(bservers)
	m.cloudBalancer.UpdateServersByFQDN(cservers)

	// update API controller after balancers initialization
	gController.WithContext(gCtx, m.bareBalancer, m.cloudBalancer)
	gController.SetReady()

	// another subsystems
	// ...

	return
}

func (m *App) rsyslog(c *fiber.Ctx) (l *zerolog.Logger) {
	return c.Locals("syslogger").(*zerolog.Logger)
}

func rlog(c *fiber.Ctx) *zerolog.Logger {
	return c.Locals("logger").(*zerolog.Logger)
}
