package flags

import (
	"fmt"
	"math"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/urfave/cli/v2"
)

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

func validate[T comparable](field string, t transformer[T], rules ...validation.Rule) func(*cli.Context, T) error {
	return func(_ *cli.Context, v T) (e error) {
		var val any
		if val = v; t != nil {
			val = t(v)
		}

		if e = validation.Validate(val, rules...); e == nil {
			return
		}

		return fmt.Errorf("\nargument is invalid for %s : %s\n\tuse --help for more information",
			field, e.Error())
	}
}

type transformer[T comparable] func(T) int64

func toSecondsCeil(val time.Duration) int64 {
	return int64(math.Ceil(val.Seconds()))
}
