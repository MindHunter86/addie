package config

import (
	"context"

	"github.com/MindHunter86/addie/internal/utils"
)

func init() {
	utils.RegisterSubservice(utils.CtxConfig,
		func(c context.Context) (_ any, e error) {
			var dc *DynamicConfig
			if dc, e = NewDynamicConfig(c, utils.DynamicConfigCategory); e != nil {
				return
			}

			// callbacks
			utils.RegisterCallback(utils.OnServiceBootstrap, dc.onServiceBootstrap)
			utils.RegisterCallback(utils.OnServiceDestruct, dc.onServiceDestruct)
			utils.RegisterCallback(utils.OnServiceTicker1sec, dc.onServiceTicker1sec)

			return dc, nil
		})
}
