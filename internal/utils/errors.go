package utils

import (
	"fmt"

	"github.com/pkg/errors"
)

func ExtraErrorWrapper(e error, dscr string, params ...interface{}) error {
	if len(params) == 0 {
		return errors.Wrap(e, dscr)
	}

	return errors.Wrap(e, fmt.Sprintf(dscr, params...))
}
