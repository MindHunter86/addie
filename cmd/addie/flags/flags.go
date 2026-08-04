package flags

import "github.com/urfave/cli/v2"

func FlagsInitialization(em bool) (f []cli.Flag) {
	f = append(f, commonFlags(em)...)
	f = append(f, httpClientFlags(em)...)
	f = append(f, httpServerFlags(em)...)
	f = append(f, applicationFlags(em)...)
	f = append(f, databaseFlags(em)...)
	f = append(f, dynamicFlags(em)...)
	f = append(f, statsFlags(em)...)

	return f
}
