package app

import (
	"bytes"
	"errors"
	"strings"

	"github.com/MindHunter86/addie/balancer"
	"github.com/MindHunter86/addie/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

func (m *App) balanceFiberRequest(ctx *fiber.Ctx, balancers []balancer.Balancer) (e error) {
	var server *balancer.BalancerServer
	uri := []byte(ctx.Locals("uri").(string))

	prefixbuf := bytes.NewBuffer(m.chunkRegexp.FindSubmatch(uri)[utils.ChunkTitleId])
	prefixbuf.Write(m.chunkRegexp.FindSubmatch(uri)[utils.ChunkEpisodeId])
	prefixbuf.Write(m.chunkRegexp.FindSubmatch(uri)[utils.ChunkQualityLevel])

	try, retries := uint8(balancer.MaxTries), uint8(0)

	// TODO
	// ? do we need the failover with RandomBalancing

	// simplify the upstream switching (using) with constant slice
	// minifies code and removes copy-paste blocks
loop:
	for _, cluster := range balancers {

		// common && backup balancing with one upstream ("balancer")
		// loop used for retries after errors and for fallback to a backup server
		// err count has limited by `balancer.BalancerMaxRetries`
		for ; try != 0; try-- {

			// trying to balance with giver cluster
			// `try` used as a collision for slice with available servers
			// works like (`X` * maxTries) * -1
			_, server, e = cluster.BalanceByChunkname(
				prefixbuf.String(),
				string(m.chunkRegexp.FindSubmatch(uri)[utils.ChunkName]),
				try,
			)

			var berr *balancer.BalancerError
			if e == nil {
				// if all ok (if no errors) - save destination and go to the next fiber handler:
				ctx.Locals("srv",
					strings.ReplaceAll(server.Name, "-node", "")+"."+gCli.String("consul-entries-domain"))

				rlog(ctx).Trace().Msgf("all cool, server %s", server.Name)

				return
			} else if !errors.As(e, &berr) {
				panic("balancer internal error - undefined error")
			}

			if zerolog.GlobalLevel() <= zerolog.DebugLevel && berr.HasError() {
				gLog.Debug().Err(berr).Msg("error")
			}

			// here we accept only 3 retry for one server in current "balance session"
			// if TryLock in balancer always fails, balancer session in sum has 6 tries for request
			// (if MaxTries == 3)
			if berr.Has(balancer.IsRetriable) {
				rlog(ctx).Trace().Uint8("try", try).Msg("undefined error")
				if retries += 1; retries < balancer.MaxTries {
					try++
				}
				continue
			} else if berr.Has(balancer.IsBackupable) {
				rlog(ctx).Trace().Uint8("try", try).Msg("IsNextServerRouted")
				// ! use backup server
				continue
			} else if berr.Has(balancer.IsReroutable) {
				rlog(ctx).Trace().Uint8("try", try).Msg("IsNextClusterRouted")
				// ! use next cluster
				break
			}

			break loop
		}
	}

	// if we here - no alive balancers, so return error
	return fiber.NewError(fiber.StatusInternalServerError, e.Error())
}
