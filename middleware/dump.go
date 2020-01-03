package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gen-iot/bootx"
	"log"
	"time"
)

type (
	DumpConfig struct {
		Skipper Skipper
		Handler DumpHandler
	}
	DumpHandler func(ctx bootx.Context, in interface{}, out interface{}, latency int64)
)

var (
	// DefaultBodyDumpConfig is the default BodyDump middleware config.
	DefaultDumpConfig = DumpConfig{
		Skipper: DefaultSkipper,
	}
)

func DefaultDumpHandler(ctx bootx.Context, in interface{}, out interface{}, latency int64) {
	ctxReq := ctx.Request()
	buf := bytes.Buffer{}
	buf.WriteString(fmt.Sprintf("\n< %s >    %s %s %s   latency : %d ms\n",
		ctx.FuncName(), ctxReq.RemoteAddr, ctxReq.Method, ctxReq.RequestURI, latency))
	buf.WriteString("in :\n")
	if in != nil {
		bt, err := json.MarshalIndent(in, "", "  ")
		if err == nil {
			buf.Write(bt)
			buf.WriteString("\n")
		}
	}
	buf.WriteString("out :\n")
	if out != nil {
		bt, err := json.MarshalIndent(out, "", "  ")
		if err == nil {
			buf.Write(bt)
			buf.WriteString("\n")
		}
	}
	log.Println(buf.String())
}

func Dump() bootx.MiddlewareFunc {
	return DumpWithHandler(DefaultDumpHandler)
}

func DumpWithHandler(handler DumpHandler) bootx.MiddlewareFunc {
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
			start := time.Now()
			next(ctx)
			stop := time.Now()
			l := stop.Sub(start).Milliseconds()
			config.Handler(ctx, ctx.Req(), ctx.Resp(), l)
		}
	}
}
