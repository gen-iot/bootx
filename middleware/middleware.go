package middleware

import (
	"github.com/labstack/echo/v4"
)

type (
	Skipper func(ctx echo.Context) bool
)

func DefaultSkipper(echo.Context) bool {
	return false
}
