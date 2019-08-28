package middleware

import (
	"github.com/gen-iot/bootx"
)

type (
	DumpConfig struct {
		Skipper Skipper
		Handler DumpHandler
	}
	DumpHandler func(ctx bootx.Context)
)

var (
	// DefaultBodyDumpConfig is the default BodyDump middleware config.
	DefaultDumpConfig = DumpConfig{
		Skipper: DefaultSkipper,
	}
)

func Dump(handler DumpHandler) bootx.MiddlewareFunc {
	conf := DefaultDumpConfig
	conf.Handler = handler
	return DumpWithConfig(conf)
}

func DumpWithConfig(config DumpConfig) bootx.MiddlewareFunc {
	// Defaults
	if config.Handler == nil {
		panic("bootx: dump middleware requires a handler function")
	}
	if config.Skipper == nil {
		config.Skipper = DefaultDumpConfig.Skipper
	}
	return func(next bootx.HandlerFunc) bootx.HandlerFunc {
		return func(ctx bootx.Context) {
			if config.Skipper(ctx) {
				next(ctx)
				return
			}
			config.Handler(ctx)
		}
	}
}
