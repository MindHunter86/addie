package app

import (
	"bytes"
	"errors"
	"net/url"
	"strings"

	"github.com/MindHunter86/addie/internal/balancer"
	"github.com/MindHunter86/addie/internal/utils"
	"github.com/gofiber/fiber/v2"
)

var (
	errFbApiInvalidMode    = errors.New("mode argument is invalid; values soft, hard are permited only")
	errFbApiInvalidQuality = errors.New("quality argument is invalid; 480, 720, 1080 values are permited only")
)

func (*App) fbHndApiPreCondErr(ctx *fiber.Ctx) error {
	switch ctx.Locals("errors").(appMidError) {
	case errMidAppPreHeaderUri:
		rlog(ctx).Warn().Msg(errApiPreBadUri.Error())
		ctx.Set("X-Error", errApiPreBadUri.Error())
		ctx.SendString(errApiPreBadUri.Error())
	case errMidAppPreHeaderId:
		rlog(ctx).Warn().Msg(errApiPreBadId.Error())
		ctx.Set("X-Error", errApiPreBadId.Error())
		ctx.SendString(errApiPreBadId.Error())
	case errMidAppPreHeaderServer:
		rlog(ctx).Warn().Msg(errApiPreBadServer.Error())
		ctx.Set("X-Error", errApiPreBadServer.Error())
		ctx.SendString(errApiPreBadServer.Error())
	case errMidAppPreUriRegexp:
		rlog(ctx).Warn().Msg(errApiPreUriRegexp.Error())
		ctx.Set("X-Error", errApiPreUriRegexp.Error())
		ctx.SendString(errApiPreUriRegexp.Error())
	default:
		rlog(ctx).Warn().Msg("unknown error")
	}

	return ctx.SendStatus(fiber.StatusPreconditionFailed)
}

func (m *App) fbHndAppRequestSign(ctx *fiber.Ctx) (e error) {
	m.lapRequestTimer(ctx, utils.FbReqTmrReqSign)
	rlog(ctx).Trace().Msg("new 'sign request' request")

	srv, uri := ctx.Locals("srv").(string), ctx.Locals("uri").(string)
	expires, extra := m.getHlpExtra(
		ctx,
		uri,
		srv,
		ctx.Locals("uid").(string),
	)

	var rrl *url.URL
	if rrl, e = url.Parse(srv + uri); e != nil {
		rlog(ctx).Debug().Str("url_parse", srv+uri).Str("remote_addr", ctx.IP()).
			Msg("could not sign request; url.Parse error")
		return fiber.NewError(fiber.StatusInternalServerError, e.Error())
	}

	var rgs = &url.Values{}
	if ctx.Get("X-Ru-Cluster") != "" {
		if core := ctx.Locals("core"); core != nil {
			rgs.Add("core", core.(string))
		}
	}

	rgs.Add("expires", expires)
	rgs.Add("extra", extra)
	rrl.RawQuery, rrl.Scheme = rgs.Encode(), "https"

	rlog(ctx).Debug().Str("computed_request", rrl.String()).Str("remote_addr", ctx.IP()).
		Msg("request signing completed")
	ctx.Set(apiHeaderLocation, rrl.String())
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (m *App) fbHndApiCoreBalance(ctx *fiber.Ctx) (e error) {
	if ctx.Get("X-Ru-Cluster") == "" {
		return ctx.Next()
	}

	uri := ctx.Locals("uri").(string)
	sub := m.chunkRegexp.FindSubmatch([]byte(uri))

	buf := bytes.NewBuffer(sub[utils.ChunkTitleId])
	buf.Write(sub[utils.ChunkEpisodeId])
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
	ctx.Locals("core", srv)

	return ctx.Next()
}

func (m *App) fbHndBlcNodesBalance(ctx *fiber.Ctx) error {
	ctx.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)

	uri := ctx.Locals("uri").(*string)
	sub := m.chunkRegexp.FindSubmatch([]byte(*uri))

	buf := bytes.NewBuffer(sub[utils.ChunkTitleId])
	buf.Write(sub[utils.ChunkEpisodeId])
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
	reqid := ctx.Locals("requestid").(string)

	for fails := 0; fails <= gCli.Int("balancer-server-max-fails"); fails++ {
		if fails == gCli.Int("balancer-server-max-fails") {
			gLog.Error().Str("req", reqid).Msg("internal balancer error; too many balance errors")
			e = fiber.NewError(fiber.StatusInternalServerError, "internal balancer error")
			return
		}

		_, server, e = m.bareBalancer.BalanceRandom()

		if errors.Is(e, balancer.ErrServerUnavailable) {
			gLog.Trace().Err(e).Int("fails", fails).Str("req", reqid).Msg("trying to roll new server...")
			continue
		} else if errors.Is(e, balancer.ErrUpstreamUnavailable) {
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
