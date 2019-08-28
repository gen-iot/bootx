package middleware

import (
	"github.com/gen-iot/bootx"
)

type (
	ValidateConfig struct {
		Skipper   Skipper
		Validator Validator
	}
	Validator interface {
		Validate(i interface{}) error
	}
)

var (
	// DefaultBodyDumpConfig is the default BodyDump middleware config.
	DefaultValidateConfig = ValidateConfig{
		Skipper: DefaultSkipper,
	}
)

func Validate(validator Validator) bootx.MiddlewareFunc {
	c := DefaultValidateConfig
	c.Validator = validator
	return ValidateWithConfig(c)
}

func ValidateWithConfig(config ValidateConfig) bootx.MiddlewareFunc {
	if config.Validator == nil {
		panic("bootx: validate middleware requires a validator")
	}
	return func(next bootx.HandlerFunc) bootx.HandlerFunc {
		return func(ctx bootx.Context) {
			if config.Skipper(ctx) {
				next(ctx)
				return
			}
			err := config.Validator.Validate(ctx.Req())
			if err != nil {
				ctx.SetError(err)
			}
			next(ctx)
		}
	}
}
