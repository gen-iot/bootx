package middleware

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/labstack/echo/v4"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

/*
* copy from https://github.com/labstack/echo/blob/master/middleware/body_dump.go
* just fix some error
 */
type (
	BodyDumpConfig struct {
		Skipper Skipper
		Handler BodyDumpHandler
	}

	// BodyDumpHandler receives the request and response payload.
	BodyDumpHandler func(echo.Context, []byte, []byte, int64)
)

var (
	// DefaultBodyDumpConfig is the default BodyDump middleware config.
	DefaultBodyDumpConfig = BodyDumpConfig{
		Skipper: DefaultSkipper,
	}
)

type BodyDumpOption uint64

const (
	DumpNone BodyDumpOption = 1 << iota
	DumpHeader
	DumpForm
	DumpMultipartForm
	DumpJson
	DumpHtml
	DumpTextPlain
	DumpXml
	DumpJs
	DumpAll
)

func DefaultBodyDumpHandler(option BodyDumpOption) BodyDumpHandler {
	check := func(opt BodyDumpOption, ctype string) bool {
		if (option&DumpAll != 0 ||
			option&DumpForm != 0 && strings.Contains(ctype, echo.MIMEApplicationForm)) ||
			(option&DumpMultipartForm != 0 && strings.Contains(ctype, echo.MIMEMultipartForm)) ||
			(option&DumpJson != 0 && strings.Contains(ctype, echo.MIMEApplicationJSON)) ||
			(option&DumpHtml != 0 && strings.Contains(ctype, echo.MIMETextHTML)) ||
			(option&DumpTextPlain != 0 && strings.Contains(ctype, echo.MIMETextPlain)) ||
			(option&DumpXml != 0 && strings.Contains(ctype, echo.MIMETextXML)) ||
			(option&DumpJs != 0 && strings.Contains(ctype, echo.MIMEApplicationJavaScript)) {
			return true
		}
		return false
	}
	return func(ctx echo.Context, reqData []byte, resData []byte, latency int64) {
		ctxReq := ctx.Request()
		reqCtype := ctxReq.Header.Get(echo.HeaderContentType)
		respHeader := ctx.Response().Header()
		resCtype := respHeader.Get(echo.HeaderContentType)
		buf := bytes.Buffer{}
		buf.WriteString(fmt.Sprintf("%s %s %s     latency : %d ms\n",
			ctxReq.RemoteAddr, ctxReq.Method, ctxReq.RequestURI, latency))
		if option&DumpNone != 0 {
			return
		}
		if option&DumpHeader != 0 {
			buf.WriteString(fmt.Sprintf("request headers: %v\n", ctxReq.Header))
		}
		if check(option, reqCtype) {
			buf.WriteString(fmt.Sprintf("request :\n%s\n", reqData))
		}
		if option&DumpHeader != 0 {
			buf.WriteString(fmt.Sprintf("response headers: %v\n", respHeader))
		}
		if check(option, resCtype) {
			buf.WriteString(fmt.Sprintf("response :\n%s", resData))
		}
		log.Println(buf.String())
	}
}

func BodyDump(option BodyDumpOption) echo.MiddlewareFunc {
	return BodyDumpWithHandler(DefaultBodyDumpHandler(option))
}

func BodyDumpWithHandler(handler BodyDumpHandler) echo.MiddlewareFunc {
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
		return func(c echo.Context) error {
			start := time.Now()
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
			err := next(c)
			stop := time.Now()
			l := stop.Sub(start).Milliseconds()
			config.Handler(c, reqBody, resBody.Bytes(), l)
			return err
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
