package flags

import (
	"github.com/urfave/cli/v2"
)

func databaseFlags(expertMode bool) []cli.Flag {
	return []cli.Flag{

		// TODO - rewrite to path-prefix
		// TODO - integrate settings from saeko

		// TODO bbolt - temporary disabled
		// bbolt database settings
		// &cli.StringFlag{
		// 	Name:     "database-prefix",
		// 	Category: "Database",
		// 	Value:    ".",
		// },

		// &cli.StringFlag{
		// 	Name:  "database-path",
		// 	Value: "data/saeko.db",
		// },
		// &cli.DurationFlag{
		// 	Name:     "database-open-timeout",
		// 	Category: "Database",
		// 	Hidden:   expertMode,
		// 	Value:    2 * time.Second,
		// },
		// &cli.BoolFlag{
		// 	Name:               "database-no-freelist-sync",
		// 	Category:           "Database",
		// 	Usage:              "This improves the database write performance under normal operation, but requires a full database re-sync during recovery.",
		// 	Hidden:             expertMode,
		// 	DisableDefaultText: true,
		// },
		// &cli.BoolFlag{
		// 	Name:               "database-no-sync",
		// 	Category:           "Database",
		// 	Hidden:             expertMode,
		// 	DisableDefaultText: true,
		// },
	}
}
