package stats

import (
	"context"

	"github.com/MindHunter86/addie/internal/utils"
)

func init() {
	utils.RegisterSubservice(utils.CtxStats,
		func(c context.Context) (any, error) {
			sts := NewStatsService(c)

			// callbacks
			utils.RegisterCallback(utils.OnServiceBootstrap, sts.onServiceBootstrap)
			utils.RegisterCallback(utils.OnServiceTicker1sec, sts.onServiceTicker1sec)

			return sts, nil
		})
}
