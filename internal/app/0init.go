package app

import (
	"context"

	"github.com/MindHunter86/addie/internal/utils"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

func init() {
	utils.RegisterSubservice(utils.CtxApp,
		func(c context.Context) (_ any, _ error) {
			log := utils.ContextValueExtract[*zerolog.Logger](c, utils.CtxZeroLogger)
			cli := utils.ContextValueExtract[*cli.Context](c, utils.CtxCliContext)

			// TODO: REMOVE: legacy code compatibility
			gCtx, gLog, gCli = c, log, cli

			// TODO: FIX: syslogWrite should be dropped
			app := NewApp(nil)

			// callbacks
			utils.RegisterCallback(utils.OnServiceBootstrap, app.bootstrap)

			return app, nil
		})
}
