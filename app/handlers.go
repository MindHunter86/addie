package app

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/MindHunter86/anilibria-hlp-service/balancer"
	"github.com/MindHunter86/anilibria-hlp-service/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

func (*App) getBalancersClusterArg(ctx *fiber.Ctx) (cluster balancer.BalancerCluster, e error) {
	var buf string
	if buf = ctx.Query("cluster"); buf == "" {
		e = fiber.NewError(fiber.StatusBadRequest, "cluster could not be empty")
		return
	}

	switch buf = strings.TrimSpace(buf); buf {
	case "cache-nodes":
		cluster = balancer.BalancerClusterNodes
	case "cache-cloud":
		cluster = balancer.BalancerClusterCloud
	default:
		e = fiber.NewError(fiber.StatusBadRequest, "invalid cluster name")
	}

	return
}

func (m *App) fbHndApiBalancerStats(ctx *fiber.Ctx) (e error) {
	ctx.Type(fiber.MIMETextPlainCharsetUTF8)

	var cluster balancer.BalancerCluster
	if cluster, e = m.getBalancersClusterArg(ctx); e != nil {
		return
	}

	switch cluster {
	case balancer.BalancerClusterNodes:
		fmt.Fprint(ctx, m.bareBalancer.GetStats())
	case balancer.BalancerClusterCloud:
		fmt.Fprint(ctx, m.cloudBalancer.GetStats())
	}

	return ctx.SendStatus(fiber.StatusOK)
}

func (m *App) fbHndApiStatsReset(ctx *fiber.Ctx) (e error) {
	ctx.Type(fiber.MIMETextPlainCharsetUTF8)
	gLog.Debug().Msg("servers stats reset")

	var cluster balancer.BalancerCluster
	if cluster, e = m.getBalancersClusterArg(ctx); e != nil {
		return
	}

	switch cluster {
	case balancer.BalancerClusterNodes:
		m.bareBalancer.ResetStats()
	case balancer.BalancerClusterCloud:
		m.cloudBalancer.ResetStats()
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func (m *App) fbHndApiBalancerReset(ctx *fiber.Ctx) (e error) {
	ctx.Type(fiber.MIMETextPlainCharsetUTF8)
	gLog.Debug().Msg("upstream reset")

	var cluster balancer.BalancerCluster
	if cluster, e = m.getBalancersClusterArg(ctx); e != nil {
		return
	}

	switch cluster {
	case balancer.BalancerClusterNodes:
		m.bareBalancer.ResetUpstream()
	case balancer.BalancerClusterCloud:
		m.cloudBalancer.ResetUpstream()
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func (m *App) fbHndApiBlockIp(ctx *fiber.Ctx) error {
	ctx.Type(fiber.MIMETextHTMLCharsetUTF8)

	ip := ctx.Query("ip")
	if ip == "" {
		return fiber.NewError(fiber.StatusBadRequest, "given ip is empty")
	}

	if e := gConsul.addIpToBlocklist(ip); e != nil {
		return fiber.NewError(fiber.StatusInternalServerError, e.Error())
	}

	fmt.Fprintln(ctx, ip+" has been banned")
	return ctx.SendStatus(fiber.StatusOK)
}

func (m *App) fbHndApiUnblockIp(ctx *fiber.Ctx) error {
	ctx.Type(fiber.MIMETextHTMLCharsetUTF8)

	ip := ctx.Query("ip")
	if ip == "" {
		return fiber.NewError(fiber.StatusBadRequest, "given ip is empty")
	}

	if e := gConsul.removeIpFromBlocklist(ip); e != nil {
		return fiber.NewError(fiber.StatusInternalServerError, e.Error())
	}

	fmt.Fprintln(ctx, ip+" has been unbanned")
	return ctx.SendStatus(fiber.StatusOK)
}

func (m *App) fbHndApiBlockReset(ctx *fiber.Ctx) error {
	ctx.Type(fiber.MIMETextHTMLCharsetUTF8)

	if e := gConsul.resetIpsInBlocklist(); e != nil {
		return fiber.NewError(fiber.StatusInternalServerError, e.Error())
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func (m *App) fbHndApiBListSwitch(ctx *fiber.Ctx) error {
	ctx.Type(fiber.MIMETextHTMLCharsetUTF8)

	enabled := ctx.Query("enabled")
	switch enabled {
	case "0":
		fallthrough
	case "1":
		if e := gConsul.updateBlocklistSwitcher(enabled); e != nil {
			return fiber.NewError(fiber.StatusInternalServerError, e.Error())
		}
	default:
		return fiber.NewError(fiber.StatusBadRequest, "enabled query can be only 0 or 1")
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func (m *App) fbHndApiLimiterSwitch(ctx *fiber.Ctx) error {
	ctx.Type(fiber.MIMETextHTMLCharsetUTF8)

	enabled := ctx.Query("enabled")
	switch enabled {
	case "0":
		fallthrough
	case "1":
		if e := gConsul.updateLimiterSwitcher(enabled); e != nil {
			return fiber.NewError(fiber.StatusInternalServerError, e.Error())
		}
	default:
		return fiber.NewError(fiber.StatusBadRequest, "enabled query can be only 0 or 1")
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func (m *App) fbHndApiLoggerLevel(ctx *fiber.Ctx) error {
	lvl := ctx.Query("level")

	switch lvl {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		return fiber.NewError(fiber.StatusBadRequest, "unknown level sent")
	}

	gLog.Error().Msgf("[falsepositive]> new log level applied - %s", lvl)

	fmt.Fprintln(ctx, lvl+" logger level has been applied")
	return ctx.SendStatus(fiber.StatusOK)
}

func (*App) fbHndApiPreCondErr(ctx *fiber.Ctx) error {
	switch ctx.Locals("errors").(appMidError) {
	case errMidAppPreHeaderUri:
		gLog.Warn().Msg(errApiPreBadUri.Error())
		ctx.Set("X-Error", errApiPreBadUri.Error())
		ctx.SendString(errApiPreBadUri.Error())
	case errMidAppPreHeaderId:
		gLog.Warn().Msg(errApiPreBadId.Error())
		ctx.Set("X-Error", errApiPreBadId.Error())
		ctx.SendString(errApiPreBadId.Error())
	case errMidAppPreHeaderServer:
		gLog.Warn().Msg(errApiPreBadServer.Error())
		ctx.Set("X-Error", errApiPreBadServer.Error())
		ctx.SendString(errApiPreBadServer.Error())
	case errMidAppPreUriRegexp:
		gLog.Warn().Msg(errApiPreUriRegexp.Error())
		ctx.Set("X-Error", errApiPreUriRegexp.Error())
		ctx.SendString(errApiPreUriRegexp.Error())
	default:
		gLog.Warn().Msg("unknown error")
	}

	return ctx.SendStatus(fiber.StatusPreconditionFailed)
}

func (m *App) fbHndAppRequestSign(ctx *fiber.Ctx) (e error) {
	m.lapRequestTimer(ctx, utils.FbReqTmrReqSign)
	gLog.Trace().Msg("new 'sign request' request")

	srv, uri := ctx.Locals("srv").(string), ctx.Locals("uri").(string)
	expires, extra := m.getHlpExtra(
		uri,
		srv,
		ctx.Locals("uid").(string),
	)

	var rrl *url.URL
	if rrl, e = url.Parse(srv + uri); e != nil {
		gLog.Debug().Str("url_parse", srv+uri).Str("remote_addr", ctx.IP()).
			Msg("could not sign request; url.Parse error")
		return fiber.NewError(fiber.StatusInternalServerError, e.Error())
	}

	var rgs = &url.Values{}
	rgs.Add("expires", expires)
	rgs.Add("extra", extra)
	rrl.RawQuery, rrl.Scheme = rgs.Encode(), "https"

	gLog.Debug().Str("computed_request", rrl.String()).Str("remote_addr", ctx.IP()).
		Msg("request signing completed")
	ctx.Set(apiHeaderLocation, rrl.String())
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (m *App) fbHndBlcNodesBalance(ctx *fiber.Ctx) error {
	ctx.Type(fiber.MIMETextPlainCharsetUTF8)

	uri := ctx.Locals("uri").(*string)
	sub := m.chunkRegexp.FindSubmatch([]byte(*uri))

	buf := bytes.NewBuffer(sub[utils.ChunkTitleId])
	buf.Write(sub[utils.ChunkQualityLevel])

	_, server, e := m.bareBalancer.BalanceByChunk(buf.String(), string(sub[utils.ChunkName]))
	if errors.Is(e, balancer.ErrServerUnavailable) {
		gLog.Debug().Err(e).Msg("balancer soft error; fallback to random balancing")
		return ctx.Next()
	} else if e != nil {
		gLog.Warn().Err(e).Msg("balancer critical error; fallback to random balancing")
		return ctx.Next()
	}

	srv := strings.ReplaceAll(server.Name, "-node", "") + "." + gCli.String("consul-entries-domain")
	ctx.Set("X-Location", srv)

	return ctx.SendStatus(fiber.StatusNoContent)
}

func (m *App) fbHndBlcNodesBalanceFallback(ctx *fiber.Ctx) error {
	ctx.Type(fiber.MIMETextPlainCharsetUTF8)

	server, e := m.getServerFromRandomBalancer(ctx)
	if e != nil {
		return e
	}

	srv := strings.ReplaceAll(server.Name, "-node", "") + "." + gCli.String("consul-entries-domain")
	ctx.Set("X-Location", srv)

	return ctx.SendStatus(fiber.StatusNoContent)
}

func (m *App) getServerFromRandomBalancer(ctx *fiber.Ctx) (server *balancer.BalancerServer, e error) {
	reqid, force := ctx.Locals("requestid").(string), false

	for fails := 0; fails <= gCli.Int("balancer-server-max-fails"); fails++ {
		if fails == gCli.Int("balancer-server-max-fails") {
			gLog.Error().Str("req", reqid).Msg("internal balancer error; too many balance errors")
			e = fiber.NewError(fiber.StatusInternalServerError, "internal balancer error")
			return
		}

		_, server, e = m.bareBalancer.BalanceRandom(force)

		if errors.Is(e, balancer.ErrServerUnavailable) {
			gLog.Trace().Err(e).Int("fails", fails).Str("req", reqid).Msg("trying to roll new server...")
			continue
		} else if errors.Is(e, balancer.ErrUpstreamUnavailable) && !force {
			force = true
			gLog.Trace().Err(e).Int("fails", fails).Str("req", reqid).Msg("trying to force balancer")
			continue
		} else if e != nil {
			gLog.Error().Err(e).Str("req", reqid).Msg("could not balance the request")
			e = fiber.NewError(fiber.StatusInternalServerError, e.Error())
			return
		}

		return
	}

	return
}
