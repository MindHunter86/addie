package config

import (
	"context"

	"github.com/MindHunter86/addie/internal/utils"
	"github.com/urfave/cli/v2"
)

type DynamicConfig struct {
	cli *cli.Context
}

func NewDynamicConfig(c context.Context) (dc *DynamicConfig) {
	dc = new(DynamicConfig)
	dc.cli = utils.ContextValueExtract[*cli.Context](c, utils.CtxCliContext)

	// cli := utils.ContextValueExtract[*cli.Context](c, utils.CtxCliContext)
	// for _, fl := range cli.App.Flags {
	// 	fl.Names()

	// }

	// for _, cat := range cli.App.VisibleFlagCategories() {
	// }

	lala := dc.cli.Generic("1").(DynamicFlag[int])
	if abc := *lala.Load(); abc != 1 {
		panic("")
	}

	return dc
}

type ConfigSet struct {
}
