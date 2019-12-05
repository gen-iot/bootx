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
		this.midwares = make([]MiddlewareFunc, 0, len(m))
	}
	this.midwares = append(this.midwares, m...)
}

func (this *middlewares) Len() int {
	return len(this.midwares)
}

func (this *middlewares) buildChain(h HandlerFunc) HandlerFunc {
	return applyMiddleware(h, this.midwares...)
}

func applyMiddleware(h HandlerFunc, m ...MiddlewareFunc) HandlerFunc {
	std.Assert(h != nil, "applyMiddleware, h == nil")
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i](h)
	}
	return h
}
