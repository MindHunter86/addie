package config

import (
	"context"
	"errors"

	"github.com/MindHunter86/addie/internal/utils"
	"github.com/urfave/cli/v2"
)

type DynamicConfig struct {
	cli *cli.Context

	keys []string
}

func NewDynamicConfig(c context.Context, cat string) (dc *DynamicConfig, e error) {
	dc = new(DynamicConfig)
	dc.cli = utils.ContextValueExtract[*cli.Context](c, utils.CtxCliContext)

	if dc.keys = dc.lookupForConfigKeys(cat); dc.keys == nil {
		return nil, errors.New("BUG: could not find dynamic config values in cli.Flags")
	}

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

	return
}

func (m *DynamicConfig) lookupForRemoteConfig(url string) {}

func (m *DynamicConfig) lookupForConfigKeys(catname string) (keys []string) {
	keys = make([]string, 0, 32)

	for _, cat := range m.cli.App.VisibleFlagCategories() {
		if cat.Name() != catname {
			continue
		}

		for _, flag := range cat.Flags() {
			keys = append(m.keys, flag.Names()...)
		}
	}

	if len(keys) == 0 {
		return nil
	}

	return
}

type ConfigSet struct {
}
