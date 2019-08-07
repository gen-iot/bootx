package web

import (
	"bufio"
	"bytes"
	"github.com/gen-iot/log"
	"github.com/labstack/echo/v4"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
)

func PrintHandler(ctx echo.Context, reqData []byte, resData []byte) {
	req := ctx.Request()
	if req.ContentLength == 0 {
		log.DEBUG.Printf("%s %s %s %s \nresponse: %s",
			logTag, req.RemoteAddr, req.Method, req.RequestURI, resData,
		)
	} else {
		ctype := req.Header.Get(echo.HeaderContentType)
		//only dump json
		if strings.HasPrefix(ctype, echo.MIMEApplicationJSON) {
			log.DEBUG.Printf("%s %s %s %s \nrequest:%s \nresponse: %s",
				logTag, req.RemoteAddr, req.Method, req.RequestURI, reqData, resData,
			)
		} else {
			log.DEBUG.Printf("%s %s %s %s \nrequest:%s \nresponse: %s",
				logTag, req.RemoteAddr, req.Method, req.RequestURI, ctype, resData,
			)
		}
	}
}

func Dump() echo.MiddlewareFunc {
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
			PrintHandler(c, reqBody, resBody.Bytes())
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
