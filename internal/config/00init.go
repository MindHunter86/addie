package config

import (
	"context"

	"github.com/MindHunter86/addie/internal/utils"
)

func init() {
	utils.RegisterSubservice(utils.CtxConfig,
		func(c context.Context) (_ any, e error) {
			// var dc *DynamicConfig
			// if dc, e = NewDynamicConfig(c, utils.DynamicConfigCategory); e != nil {
			// 	return
			// }

			return nil, nil
		})
}
