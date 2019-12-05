package middleware

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gen-iot/bootx"
	"github.com/labstack/echo/v4"
	"net/http"
	"reflect"
	"strings"
)

/*
* copy from https://github.com/labstack/echo/blob/master/middleware/jwt.go
* just do some wrapper
 */

type (
	JWTConfig struct {
		// Skipper defines a function to skip middleware.
		Skipper Skipper

		// BeforeFunc defines a function which is executed just before the middleware.
		BeforeFunc BeforeFunc

		// SuccessHandler defines a function which is executed for a valid token.
		SuccessHandler JWTSuccessHandler

		// ErrorHandlerWithContext is almost identical to ErrorHandler, but it's passed the current context.
		ErrorHandlerWithContext JWTErrorHandlerWithContext

		// Signing key to validate token. Used as fallback if SigningKeys has length 0.
		// Required. This or SigningKeys.
		SigningKey interface{}

		// Map of signing keys to validate token with kid field usage.
		// Required. This or SigningKey.
		SigningKeys map[string]interface{}

		// Signing method, used to check token signing method.
		// Optional. Default value HS256.
		SigningMethod string

		// Context key to store user information from the token into context.
		// Optional. Default value "user".
		ContextKey string

		// Claims are extendable claims data defining token content.
		// Optional. Default value jwt.MapClaims
		Claims jwt.Claims

		// TokenLookup is a string in the form of "<source>:<name>" that is used
		// to extract token from the request.
		// Optional. Default value ["header:Authorization"].
		// Possible values:
		// - "header:<name>"
		// - "query:<name>"
		// - "param:<name>"
		// - "cookie:<name>"
		TokenLookups []string

		// AuthScheme to be used in the Authorization header.
		// Optional. Default value "Bearer".
		AuthScheme string

		KeyFunc jwt.Keyfunc
	}

	BeforeFunc func(bootx.Context)

	// JWTSuccessHandler defines a function which is executed for a valid token.
	JWTSuccessHandler func(bootx.Context)

	//JWTErrorHandlerWithContext defines a function which is executed for an invalid token. passed the current context.
	JWTErrorHandlerWithContext func(error, bootx.Context)

	jwtExtractor func(bootx.Context) (string, error)
)

// Algorithms
const (
	AlgorithmHS256 = "HS256"
)

// Errors
var (
	ErrJWTMissing = echo.NewHTTPError(http.StatusBadRequest, "missing or malformed jwt")
	ErrJWTInvalid = echo.NewHTTPError(http.StatusBadRequest, "invalid or expired jwt")
)

const (
	JWTContextKey    = "JWT"
	JWTHeaderKeyName = bootx.HeaderAuthorization
	JWTCookieKeyName = "token"
	JWTQueryKeyName  = "token"
	JWTParamKeyName  = "token"
)

const (
	TokenFromHeader = "header:" + JWTHeaderKeyName
	TokenFromCookie = "cookie:" + JWTCookieKeyName
	TokenFromQuery  = "query:" + JWTQueryKeyName
	TokenFromParam  = "param:" + JWTParamKeyName
)

var (
	// DefaultJWTConfig is the default JWT auth middleware config.
	DefaultJWTConfig = JWTConfig{
		Skipper:       DefaultSkipper,
		SigningMethod: AlgorithmHS256,
		ContextKey:    "user",
		TokenLookups:  []string{"header:" + echo.HeaderAuthorization},
		AuthScheme:    "Bearer",
		Claims:        jwt.MapClaims{},
	}
)

// JWT returns a JSON Web Token (JWT) auth middleware.
//
// For valid token, it sets the user in context and calls next handler.
// For invalid token, it returns "401 - Unauthorized" error.
// For missing token, it returns "400 - Bad Request" error.
//
// See: https://jwt.io/introduction
// See `JWTConfig.TokenLookup`

func JWT(key interface{}) bootx.MiddlewareFunc {
	c := DefaultJWTConfig
	c.SigningKey = key
	return JWTWithConfig(c)
}

func JWT1(keys map[string]interface{}) bootx.MiddlewareFunc {
	c := DefaultJWTConfig
	c.SigningKeys = keys
	return JWTWithConfig(c)
}
func JWT2(keyFunc jwt.Keyfunc) bootx.MiddlewareFunc {
	c := DefaultJWTConfig
	c.KeyFunc = keyFunc
	return JWTWithConfig(c)
}

func JWTWithConfig(config JWTConfig) bootx.MiddlewareFunc {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = DefaultJWTConfig.Skipper
	}
	if config.SigningKey == nil && len(config.SigningKeys) == 0 && config.KeyFunc == nil {
		panic("echo: jwt middleware requires signing key")
	}
	if config.SigningMethod == "" {
		config.SigningMethod = DefaultJWTConfig.SigningMethod
	}
	if config.ContextKey == "" {
		config.ContextKey = DefaultJWTConfig.ContextKey
	}
	if config.Claims == nil {
		config.Claims = DefaultJWTConfig.Claims
	}
	if len(config.TokenLookups) == 0 {
		config.TokenLookups = DefaultJWTConfig.TokenLookups
	}
	if config.AuthScheme == "" {
		config.AuthScheme = DefaultJWTConfig.AuthScheme
	}
	if config.KeyFunc == nil {
		config.KeyFunc = func(t *jwt.Token) (interface{}, error) {
			// Check the signing method
			if t.Method.Alg() != config.SigningMethod {
				return nil, fmt.Errorf("unexpected jwt signing method=%v", t.Header["alg"])
			}
			if len(config.SigningKeys) > 0 {
				if kid, ok := t.Header["kid"].(string); ok {
					if key, ok := config.SigningKeys[kid]; ok {
						return key, nil
					}
				}
				return nil, fmt.Errorf("unexpected jwt key id=%v", t.Header["kid"])
			}
			return config.SigningKey, nil
		}
	}
	// Initialize
	var extractors []jwtExtractor
	for _, lookupStr := range config.TokenLookups {
		parts := strings.Split(lookupStr, ":")
		if len(parts) != 2 {
			panic(fmt.Sprintf("invalid token lookup string %s", lookupStr))
		}
		var extractor jwtExtractor = nil
		switch parts[0] {
		case "header":
			extractor = jwtFromHeader(parts[1], config.AuthScheme)
		case "query":
			extractor = jwtFromQuery(parts[1])
		case "param":
			extractor = jwtFromParam(parts[1])
		case "cookie":
			extractor = jwtFromCookie(parts[1])
		default:
			panic(fmt.Sprintf("invalid token lookup string %s", lookupStr))
		}
		extractors = append(extractors, extractor)
	}
	if len(extractors) == 0 {
		extractors = []jwtExtractor{jwtFromHeader(JWTHeaderKeyName, config.AuthScheme)}
	}

	return func(next bootx.HandlerFunc) bootx.HandlerFunc {
		return func(c bootx.Context) {
			if config.Skipper(c) {
				next(c)
				return
			}
			if config.BeforeFunc != nil {
				config.BeforeFunc(c)
			}
			auth := ""
			var err error = nil
			for _, extractor := range extractors {
				auth, err = extractor(c)
				if err != nil {
					//try next
					continue
				}
			}
			if auth == "" {
				if config.ErrorHandlerWithContext != nil {
					config.ErrorHandlerWithContext(ErrJWTMissing, c)
					return
				}
				c.SetError(ErrJWTMissing)
				return
			}
			token := new(jwt.Token)
			// Issue #647, #656
			if _, ok := config.Claims.(jwt.MapClaims); ok {
				token, err = jwt.Parse(auth, config.KeyFunc)
			} else {
				t := reflect.ValueOf(config.Claims).Type().Elem()
				claims := reflect.New(t).Interface().(jwt.Claims)
				token, err = jwt.ParseWithClaims(auth, claims, config.KeyFunc)
			}
			if err == nil && token.Valid {
				// Store user information from token into context.
				c.Set(config.ContextKey, token)
				if config.SuccessHandler != nil {
					config.SuccessHandler(c)
				}
				next(c)
				return
			}
			if config.ErrorHandlerWithContext != nil {
				config.ErrorHandlerWithContext(err, c)
				return
			}
			c.SetError(ErrJWTInvalid)
			return
		}
	}
}

// jwtFromHeader returns a `jwtExtractor` that extracts token from the request header.
func jwtFromHeader(header string, authScheme string) jwtExtractor {
	return func(c bootx.Context) (string, error) {
		auth := c.Request().Header.Get(header)
		l := len(authScheme)
		if len(auth) > l+1 && auth[:l] == authScheme {
			return auth[l+1:], nil
		}
		return "", ErrJWTMissing
	}
}

// jwtFromQuery returns a `jwtExtractor` that extracts token from the query string.
func jwtFromQuery(param string) jwtExtractor {
	return func(c bootx.Context) (string, error) {
		token := c.QueryParam(param)
		if token == "" {
			return "", ErrJWTMissing
		}
		return token, nil
	}
}

// jwtFromParam returns a `jwtExtractor` that extracts token from the url param string.
func jwtFromParam(param string) jwtExtractor {
	return func(c bootx.Context) (string, error) {
		token := c.Param(param)
		if token == "" {
			return "", ErrJWTMissing
		}
		return token, nil
	}
}

// jwtFromCookie returns a `jwtExtractor` that extracts token from the named cookie.
func jwtFromCookie(name string) jwtExtractor {
	return func(c bootx.Context) (string, error) {
		cookie, err := c.Cookie(name)
		if err != nil {
			return "", ErrJWTMissing
		}
		return cookie.Value, nil
	}
}
