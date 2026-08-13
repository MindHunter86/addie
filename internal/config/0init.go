package config

import (
	"context"

	"github.com/MindHunter86/addie/internal/utils"
)

func init() {
	utils.RegisterSubservice(utils.CtxConfig,
		func(c context.Context) (any, error) {
			dc, e := NewDynamicConfig(c, utils.DynamicConfigCategory)

			// subservice callbacks
			utils.RegisterCallback(utils.OnServiceBootstrap, dc.onServiceBootstrap)
			utils.RegisterCallback(utils.OnServiceDestruct, dc.onServiceDestruct)

			if dc.hasRemoteSource() {
				utils.RegisterCallback(utils.OnServiceTicker1sec, dc.onServiceTicker1sec)
			}

			return dc, e
		})
}
