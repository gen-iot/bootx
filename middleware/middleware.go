package middleware

import (
	"github.com/gen-iot/bootx"
)

type (
	Skipper func(ctx bootx.Context) bool
)

func DefaultSkipper(bootx.Context) bool {
	return false
}
