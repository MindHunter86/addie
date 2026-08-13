package balancerv2

import (
	"context"
	"sync"

	"github.com/MindHunter86/addie/internal/config"
	"github.com/MindHunter86/addie/internal/utils"
)

type BalancerV2 struct {
	mu      sync.RWMutex
	routing map[string]*config.BalancerRouting
	regions map[string]*config.BalancerRegion
}

func NewBalancerV2(c context.Context) *BalancerV2 {
	// cli := utils.ContextValueExtract[*cli.Context](c, utils.CtxCliContext)

	cfg := utils.ContextValueExtract[*config.DynamicConfig](c, utils.CtxConfig)
	src := cfg.LoadSource()

	return &BalancerV2{
		routing: src.Balancer.Routing,
		regions: src.Balancer.Regions,
	}
}
