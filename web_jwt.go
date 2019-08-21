package bootx

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"net/http"
)

const (
	cookieKey4Token = "token"
	jwtContextKey   = "jwtUser"
)

type JwtClaims struct {
	*jwt.StandardClaims
}

var missTokenError = echo.NewHTTPError(http.StatusBadRequest, "miss auth token")
var invalidTokenError = echo.NewHTTPError(http.StatusUnauthorized, "invalid auth token")

var DefaultJwtConfig = middleware.JWTConfig{
	Claims:      &JwtClaims{},
	ContextKey:  jwtContextKey,
	TokenLookup: "header:" + echo.HeaderAuthorization,
	AuthScheme:  "Bearer",
	ErrorHandler: func(e error) error {
		logger.Println(logTag, "jwt auth error :", e)
		if e == middleware.ErrJWTMissing {
			return missTokenError
		}
		return invalidTokenError
	},
}

func Jwt(key interface{}) echo.MiddlewareFunc {
	c := DefaultJwtConfig
	c.SigningKey = key
	return JwtWithConfig(c)
}

func Jwt1(keys map[string]interface{}) echo.MiddlewareFunc {
	c := DefaultJwtConfig
	c.SigningKeys = keys
	return JwtWithConfig(c)
}

func JwtWithConfig(config middleware.JWTConfig) echo.MiddlewareFunc {
	oldFun := config.BeforeFunc
	config.BeforeFunc = func(ctx echo.Context) {
		//将cookie中的accessToken同步到header:Authorization中
		authFromCookie := jwtFromCookie(cookieKey4Token, ctx)
		if authFromCookie != "" {
			authFromHeader := jwtFromHeader(echo.HeaderAuthorization, config.AuthScheme, ctx)
			if authFromHeader == "" {
				ctx.Request().Header.Set(echo.HeaderAuthorization,
					fmt.Sprintf("%s %s", config.AuthScheme, authFromCookie))
			}
		}
		if oldFun != nil {
			oldFun(ctx)
		}
	}
	return middleware.JWTWithConfig(config)
}

func jwtFromCookie(name string, c echo.Context) string {
	cookie, err := c.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func jwtFromHeader(header string, authScheme string, c echo.Context) string {
	auth := c.Request().Header.Get(header)
	l := len(authScheme)
	if len(auth) > l+1 && auth[:l] == authScheme {
		return auth[l+1:]
	}
	return ""
}
