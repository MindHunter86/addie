package app

import (
	"bytes"
	"errors"
	"math/rand"
	"net/url"
	"regexp"
	"strings"

	"github.com/MindHunter86/addie/internal/balancer"
	"github.com/MindHunter86/addie/internal/runtime"
	"github.com/MindHunter86/addie/utils"
	"github.com/gofiber/fiber/v2"
)

var (
	// errApiPreBadHeaders = errors.New("could not parse required headers")
	errApiPreBadUri    = errors.New("invalid uri")
	errApiPreBadId     = errors.New("invalid id")
	errApiPreBadServer = errors.New("invalid server")
	errApiPreUriRegexp = errors.New("regexp matching failure")
)

const (
	apiHeaderUri      = "X-Client-Uri"
	apiHeaderId       = "X-Client-Id"
	apiHeaderServer   = "X-Cache-Server"
	apiHeaderLocation = "X-Location"
)

type appMidError uint8

const (
	errMidAppPreHeaderUri appMidError = 1 << iota
	errMidAppPreHeaderId
	errMidAppPreHeaderServer
	errMidAppPreUriRegexp
)

// func (m *App) hlpHandler(ctx *fasthttp.RequestCtx) {
// 	// client IP parsing
// 	cip := string(ctx.Request.Header.Peek(fasthttp.HeaderXForwardedFor))
// 	if cip == "" || cip == "127.0.0.1" {
// 		rlog(ctx).Debug().Str("remote_addr", ctx.RemoteIP().String()).Str("x_forwarded_for", cip).Msg("")
// 		m.hlpRespondError(&ctx.Response, errHlpBadIp)
// 		return
// 	}
// }

func (m *App) fbMidAppPreCond(ctx *fiber.Ctx) (skip bool) {
	m.lapRequestTimer(ctx, utils.FbReqTmrBlcPreCond)
	rlog(ctx).Trace().Interface("hdrs", ctx.GetReqHeaders()).Msg("cache-XXX-internal precond balancer")

	var errs appMidError
	switch h := ctx.GetReqHeaders(); {
	case strings.TrimSpace(h[apiHeaderUri][0]) == "":
		errs = errs | errMidAppPreHeaderUri
		ctx.Locals("errors", errs)
		return
	case strings.TrimSpace(h[apiHeaderId][0]) == "":
		errs = errs | errMidAppPreHeaderId
		ctx.Locals("errors", errs)
		return
	case strings.TrimSpace(h[apiHeaderServer][0]) == "":
		errs = errs | errMidAppPreHeaderServer
		ctx.Locals("errors", errs)
		return
	}

	ctx.Locals("uid", strings.TrimSpace(ctx.Get(apiHeaderId)))
	ctx.Locals("srv", strings.TrimSpace(ctx.Get(apiHeaderServer)))

	// match uri
	if !m.chunkRegexp.Match([]byte(ctx.Get(apiHeaderUri))) {
		ctx.Locals("errors", errs|errMidAppPreUriRegexp)
		return
	}

	return true
}

// fake quality check
func (m *App) fbMidAppFakeQuality(ctx *fiber.Ctx) error {
	m.lapRequestTimer(ctx, utils.FbReqTmrFakeQuality)
	rlog(ctx).Trace().Msg("fake quality check")

	uri := ctx.Get(apiHeaderUri)
	tsr := NewTitleSerieRequest(uri)

	if !tsr.isValid() {
		ctx.Locals("uri", uri)
		return ctx.Next()
	}

	origin := ctx.Get("Origin", "")
	if origin != "" && m.runtime.Config.Get(runtime.ParamQualityBypass) != nil {
		reg := m.runtime.Config.Get(runtime.ParamQualityBypass).(*regexp.Regexp)
		if reg.MatchString(origin) {
			rlog(ctx).Trace().Msg("request bypassed due matching with bypass regexp from consul")
			ctx.Locals("uri", uri)
			return ctx.Next()
		}
	}

	mitigation := m.runtime.Config.Get(runtime.ParamForceRUMitigate).(string)
	if mitigation != "" {
		rrl, e := url.Parse(mitigation)
		if e != nil {
			rlog(ctx).Warn().Msg("could not migrate this requests because of mitigate-to URL parsing error")
			ctx.Locals("uri", uri)
			return ctx.Next()
		}

		rrl.Path = uri
		rlog(ctx).Trace().Msg("request redirected to " + rrl.String())

		ctx.Response().Header.Set("Location", rrl.String())
		return fiber.NewError(fiber.StatusTemporaryRedirect)
	}

	quality := m.runtime.Config.Get(runtime.ParamQuality).(utils.TitleQuality)
	rlog(ctx).Debug().Uint16("tsr", uint16(tsr.getTitleQuality())).Uint16("coded", uint16(quality)).
		Msg("quality check")
	if tsr.getTitleQuality() <= quality {
		ctx.Locals("uri", uri)
		return ctx.Next()
	}

	// precondition finished; quality cool down
	ctx.Locals("uri", m.getUriWithFakeQuality(ctx, tsr, uri, quality))
	return ctx.Next()
}

// if return value == true - Balance() will be skipped
func (m *App) fbMidAppBalancerLottery(_ *fiber.Ctx) bool {
	return gCli.Bool("balancer-full-bypass") || m.runtime.Config.Get(runtime.ParamLottery).(int) < rand.Intn(99)+1 // skipcq: GSC-G404 math/rand is enough
}

func (m *App) fbMidAppBalance(ctx *fiber.Ctx) (e error) {
	m.lapRequestTimer(ctx, utils.FbReqTmrConsulLottery)
	rlog(ctx).Trace().Msg("consul lottery winner, rewriting destination server...")

	var server *balancer.BalancerServer
	uri, reqid := []byte(ctx.Locals("uri").(string)), ctx.Locals("requestid").(string)
	// uri := []byte(ctx.Locals("uri").(string))

	prefixbuf := bytes.NewBuffer(m.chunkRegexp.FindSubmatch(uri)[utils.ChunkTitleId])
	prefixbuf.Write(m.chunkRegexp.FindSubmatch(uri)[utils.ChunkEpisodeId])
	prefixbuf.Write(m.chunkRegexp.FindSubmatch(uri)[utils.ChunkQualityLevel])

	// chunkname, prefix := string(m.chunkRegexp.FindSubmatch(uri)[utils.ChunkName]), prefixbuf.String()

	// for _, cluster := range []balancer.Balancer{m.cloudBalancer, m.bareBalancer} {
	// 	// TODO
	// 	// ? do we need the failover with RandomBalancing ???
	// 	// var fallback bool

	// 	// get all servers for balancing
	// 	var status *balancer.Status
	// 	if e = cluster.Balance(chunkname, prefix); e == nil {
	// 		rlog(ctx).Error().Msg("there is no status with payload and error from balancer")
	// 	}

	// 	if errors.As(e, &status) {
	// 		if e = status.Err(); e != nil {
	// 			rlog(ctx).Error().Err(e).Interface("cluster", status.Cluster()).Msg(status.Descr())
	// 			continue
	// 		}
	// 	} else {
	// 		rlog(ctx).Error().Err(e).Msg("undefined error from balancer")
	// 	}

	// 	// parse given servers
	// 	for _, server := range status.Servers {
	// 		// if all ok (if no errors) - save destination and go to the next fiber handler:
	// 		ctx.Locals("srv",
	// 			strings.ReplaceAll(server.Name, "-node", "")+"."+gCli.String("consul-entries-domain"))

	// 		return ctx.Next()
	// 	}
	// }

	for _, cluster := range []balancer.Balancer{m.cloudBalancer, m.bareBalancer} {
		var fallback bool

		for fails := 0; fails <= gCli.Int("balancer-server-max-fails"); fails++ {

			// so if fails limit reached - use new cluster or fallback to baremetal random balancing
			if fails == gCli.Int("balancer-server-max-fails") {
				if fallback {
					rlog(ctx).Error().Str("req", reqid).Str("cluster", cluster.GetClusterName()).
						Msg("internal balancer error; too many balance errors; using fallback func()...")
					return m.fbMidAppBalanceFallback(ctx)
				} else {
					fallback = true
					rlog(ctx).Error().Str("req", reqid).Str("cluster", cluster.GetClusterName()).
						Msg("internal balancer error; too many balance errors; using next cluster...")
					break
				}
			}

			// trying to balance with giver cluster
			_, server, e = cluster.BalanceByChunk(
				prefixbuf.String(),
				string(m.chunkRegexp.FindSubmatch(uri)[utils.ChunkName]), fails)

			if errors.Is(e, balancer.ErrServerUnavailable) {
				rlog(ctx).Info().Err(e).Int("fails", fails).Str("req", reqid).
					Str("cluster", cluster.GetClusterName()).Msg("trying to roll new server...")
				continue
			} else if errors.Is(e, balancer.ErrUpstreamUnavailable) {
				rlog(ctx).Trace().Err(e).Int("fails", fails).Str("req", reqid).Msg("temporary upstream error")
				continue
			} else if e != nil {
				rlog(ctx).Error().Err(e).Str("req", reqid).
					Str("cluster", cluster.GetClusterName()).Msg("could not balance; undefined error")
				break
			}

			// if all ok (if no errors) - save destination and go to the next fiber handler:
			ctx.Locals("srv",
				strings.ReplaceAll(server.Name, "-node", "")+"."+gCli.String("consul-entries-domain"))

			return ctx.Next()
		}
	}

	// if we here - no alive balancers, so return error
	return fiber.NewError(fiber.StatusInternalServerError, e.Error())
}

func (m *App) fbMidAppBalanceFallback(ctx *fiber.Ctx) error {
	server, e := m.getServerFromRandomBalancer(ctx)
	if e != nil {
		return e
	}

	ctx.Locals("srv",
		strings.ReplaceAll(server.Name, "-node", "")+"."+gCli.String("consul-entries-domain"))
	return ctx.Next()
}

// balancer api
func (m *App) fbMidBlcPreCond(ctx *fiber.Ctx) bool {
	m.lapRequestTimer(ctx, utils.FbReqTmrBlcPreCond)
	rlog(ctx).Trace().Interface("hdrs", ctx.GetReqHeaders()).Msg("cache-node-internal balancer")

	var errs appMidError

	if huri := strings.TrimSpace(ctx.Get(apiHeaderUri)); huri == "" {
		errs = errs | errMidAppPreHeaderUri
	} else if !m.chunkRegexp.Match([]byte(huri)) {
		errs = errs | errMidAppPreUriRegexp
	} else {
		ctx.Locals("uri", &huri)
	}

	ctx.Locals("errors", errs)
	return errs == 0
}
