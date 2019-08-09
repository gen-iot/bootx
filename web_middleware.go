package bootx

import (
	"github.com/gen-iot/std"
)

type HandlerFunc = func(ctx Context)
type MiddlewareFunc = func(next HandlerFunc) HandlerFunc

type middlewares struct {
	midwares []MiddlewareFunc
}

func (this *middlewares) Use(m ...MiddlewareFunc) {
	if this.midwares == nil {
		this.midwares = make([]MiddlewareFunc, 0, 4)
	}
	this.midwares = append(this.midwares, m...)
}

func (this *middlewares) Len() int {
	return len(this.midwares)
}

func (this *middlewares) buildChain(h HandlerFunc) HandlerFunc {
	std.Assert(h != nil, "buildMiddleware, h == nil")
	for i := len(this.midwares) - 1; i >= 0; i-- {
		h = this.midwares[i](h)
	}
	return h
}
