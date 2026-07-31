package config

import "github.com/urfave/cli/v2"

type Config interface{}

type DynamicConfig struct {
	cli *cli.Context
}
