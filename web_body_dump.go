package bootx

import (
	"bufio"
	"bytes"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
)

/*
* copy from https://github.com/labstack/echo/blob/master/middleware/body_dump.go
* just fix some error
 */
type (
	BodyDumpConfig struct {
		Skipper middleware.Skipper
		Handler BodyDumpHandler
	}

	// BodyDumpHandler receives the request and response payload.
	BodyDumpHandler func(echo.Context, []byte, []byte)
)

var (
	// DefaultBodyDumpConfig is the default BodyDump middleware config.
	DefaultBodyDumpConfig = BodyDumpConfig{
		Skipper: middleware.DefaultSkipper,
	}
	DefaultBodyDumpHandler = func(ctx echo.Context, reqData []byte, resData []byte) {
		req := ctx.Request()
		if req.ContentLength == 0 {
			logger.Printf("%s %s %s \nresponse: %s",
				req.RemoteAddr, req.Method, req.RequestURI, resData,
			)
		} else {
			ctype := req.Header.Get(echo.HeaderContentType)
			//only dump json
			if strings.HasPrefix(ctype, echo.MIMEApplicationJSON) {
				logger.Printf("%s %s %s \nrequest:%s \nresponse: %s",
					req.RemoteAddr, req.Method, req.RequestURI, reqData, resData,
				)
			} else {
				logger.Printf("%s %s %s \nrequest:%s \nresponse: %s",
					req.RemoteAddr, req.Method, req.RequestURI, ctype, resData,
				)
			}
		}
	}
)

func BodyDump(handler BodyDumpHandler) echo.MiddlewareFunc {
	c := DefaultBodyDumpConfig
	c.Handler = handler
	return BodyDumpWithConfig(c)
}

func BodyDumpWithConfig(config BodyDumpConfig) echo.MiddlewareFunc {
	// Defaults
	if config.Handler == nil {
		panic("bootx: body-dump middleware requires a handler function")
	}
	if config.Skipper == nil {
		config.Skipper = DefaultBodyDumpConfig.Skipper
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			// Req
			reqBody := make([]byte, 0)
			if c.Request().Body != nil { // Read
				reqBody, _ = ioutil.ReadAll(c.Request().Body)
			}
			c.Request().Body = ioutil.NopCloser(bytes.NewBuffer(reqBody)) // Reset
			// Resp
			resBody := new(bytes.Buffer)
			mw := io.MultiWriter(c.Response().Writer, resBody)
			writer := &bodyDumpResponseWriter{Writer: mw, ResponseWriter: c.Response().Writer}
			c.Response().Writer = writer
			err = next(c)
			config.Handler(c, reqBody, resBody.Bytes())
			return
		}
	}
}

type bodyDumpResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *bodyDumpResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

func (w *bodyDumpResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *bodyDumpResponseWriter) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}

func (w *bodyDumpResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}
