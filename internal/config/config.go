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

func test() {
	// cli := utils.ContextValueExtract[*cli.Context](c, utils.CtxCliContext)
	// for _, fl := range cli.App.Flags {
	// 	fl.Names()

	// }

	// for _, cat := range cli.App.VisibleFlagCategories() {
	// }

	// lala := dc.cli.Generic("1").(DynamicFlag[int])
	// if abc := *lala.Load(); abc != 1 {
	// 	panic("")
	// }
}

func NewDynamicConfig(c context.Context, catname string) (dc *DynamicConfig, e error) {
	dc = new(DynamicConfig)
	dc.cli = utils.ContextValueExtract[*cli.Context](c, utils.CtxCliContext)

	if dc.keys = dc.lookupForConfigKeys(catname); dc.keys == nil {
		return nil, errors.New("BUG: could not find dynamic config values in cli.Flags")
	}

	return
}

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

// YAML config parsing
type (
	ExternalSource struct {
		Balancer *ExternalSourceBalancer
	}
	ExternalSourceBalancer struct {
		Config  map[string]any
		Routing map[string]*BalancerRouting
		Regions map[string]*BalancerRegion
	}
	BalancerConfig struct {
	}
	BalancerRouting struct {
		Countries []string

		Primary   []string
		Secondary []string
		Backup    []string
	}
	BalancerRegion struct {
		Dynamic bool

		Orgin    *RegionOrigin
		Upstream []*RegionUpstream
	}
	RegionOrigin struct {
		Region string
		Server string
	}
	RegionUpstream struct {
		Server    string
		Bandwidth uint
	}
)

func (m *DynamicConfig) fetchSourceFromURL() ([]byte, error) {
	return nil, nil
}

func (m *DynamicConfig) fetchSourceFromFile() ([]byte, error) {
	return nil, nil
}

func (m *DynamicConfig) lookupForSources() {

}
