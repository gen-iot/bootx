package bootx

import (
	"github.com/gen-iot/std"
	"github.com/labstack/echo/v4"
	"net/http"
	"reflect"
)

type Context interface {
	echo.Context
	Id() string

	SetUserAuthData(data interface{})
	UserAuthData() interface{}

	SetReq(in interface{})
	Req() interface{}

	SetResponse(out interface{})
	Resp() interface{}

	SetHttpStatusCode(code int)
	HttpStatusCode() int

	SetError(err error)
	Err() error

	BindAndValidate(in interface{}) error

	End() error
}

type context struct {
	echo.Context
	AuthData     interface{}
	in           interface{}
	out          interface{}
	code         int
	err          error
	message      string
	handlerFlags uint32
	handlerValue reflect.Value
}

func (c *context) SetUserAuthData(data interface{}) {
	c.AuthData = data
}

func (c *context) UserAuthData() interface{} {
	return c.AuthData
}

func (c *context) Id() string {
	return c.Request().Header.Get(echo.HeaderXRequestID)
}

func (c *context) SetReq(in interface{}) {
	c.in = in
}

func (c *context) Req() interface{} {
	return c.in
}

func (c *context) SetResponse(out interface{}) {
	c.out = out
}

func (c *context) Resp() interface{} {
	return c.out
}

func (c *context) SetHttpStatusCode(code int) {
	c.code = code
}

func (c *context) HttpStatusCode() int {
	return c.code
}

func (c *context) SetError(err error) {
	c.err = err
}

func (c *context) Err() error {
	return c.err
}

func (c *context) BindAndValidate(in interface{}) error {
	err := c.Bind(in)
	if err != nil {
		return err
	}
	lang := c.Request().Header.Get("Accept-Language")
	err = std.ValidateStructWithLanguage(std.Str2Lang(lang), in)
	if err != nil {
		return err
	}
	return nil
}

func (c *context) End() error {
	err := c.Err()
	if err != nil {
		return err
	}
	return c.JSONPretty(c.code, c.Resp(), jsonIndent)
}

func CustomContextMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(echoCtx echo.Context) error {
		ctx := &context{
			Context:  echoCtx,
			AuthData: nil,
			code:     http.StatusOK,
		}
		return next(ctx)
	}
}

func ConvertFromEchoCtx(h func(Context) (err error)) echo.HandlerFunc {
	return func(echoCtx echo.Context) error {
		ctx, ok := echoCtx.(Context)
		if !ok {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		return h(ctx)
	}
}

//noinspection ALL
func BuildHttpHandler(handler interface{}, m ...MiddlewareFunc) echo.HandlerFunc {
	hv, ok := handler.(reflect.Value)
	if !ok {
		hv = reflect.ValueOf(handler)
	}
	std.Assert(hv.Kind() == reflect.Func, "handler not func!")
	hvType := hv.Type()
	hvType.NumIn()
	inType, inFlags := checkInParam(hvType)
	_, outFlags := checkOutParam(hvType)
	flags := inFlags | outFlags
	mid := middlewares{}
	mid.Use(m...)
	return ConvertFromEchoCtx(func(ctx Context) (err error) {
		//has req data
		if flags&handlerHasReqData == handlerHasReqData {
			elementType := inType
			isPtr := false
			if inType.Kind() == reflect.Ptr {
				elementType = inType.Elem()
				isPtr = true
			}
			req := reflect.New(elementType).Interface()
			//bind and validate
			if err = ctx.BindAndValidate(req); err != nil {
				ctx.SetHttpStatusCode(http.StatusBadRequest)
				ctx.SetError(echo.NewHTTPError(http.StatusBadRequest, err.Error()))
				return ctx.End()
			}
			if !isPtr {
				req = reflect.ValueOf(req).Elem().Interface()
			}
			ctx.SetReq(req)
		}
		fn := mid.buildChain(buildInvoke(hv, flags))
		fn(ctx)
		return ctx.Err()
	})
}

func buildInvoke(handlerV reflect.Value, flags uint32) HandlerFunc {
	return func(ctx Context) {
		inParams := make([]reflect.Value, 0)
		//has Ctx
		if flags&handlerHasCtx == handlerHasCtx {
			inParams = append(inParams, reflect.ValueOf(ctx))
		}
		//has req data
		if flags&handlerHasReqData == handlerHasReqData {
			inParams = append(inParams, reflect.ValueOf(ctx.Req()))
		}
		outs := handlerV.Call(inParams)
		rspErrIdx := -1
		rspDataIdx := -1
		//has rsp data
		if flags&handlerHasRsp == handlerHasRsp {
			rspErrIdx = 1
			rspDataIdx = 0
		} else {
			rspErrIdx = 0
		}
		if !outs[rspErrIdx].IsNil() { // check error
			err := outs[rspErrIdx].Interface().(error)
			ctx.SetError(err)
		}
		if rspDataIdx != -1 {
			rsp := outs[rspDataIdx].Interface()
			ctx.SetResponse(rsp)
		}
	}
}

const (
	handlerHasCtx uint32 = 1 << iota
	handlerHasReqData
	handlerHasRsp
)

var typeOfError = reflect.TypeOf((*error)(nil)).Elem()
var typeOfContext = reflect.TypeOf((*Context)(nil)).Elem()

func checkInParam(t reflect.Type) (reflect.Type, uint32) {
	var handlerFlags uint32 = 0
	var inParamType reflect.Type = nil
	inNum := t.NumIn()
	std.Assert(inNum >= 0 && inNum <= 2, "inNum len must be 0() or 1(req any) or 2(ctx Context,req Any)")
	//
	switch inNum {
	case 0:
		//func()
	case 1:
		// func foo(context)
		if t.In(0) == typeOfContext {
			handlerFlags = handlerFlags | handlerHasCtx
		} else {
			handlerFlags = handlerFlags | handlerHasReqData
		}
	case 2:
		// func foo(context,param1)
		in0 := t.In(0)
		std.Assert(in0 == typeOfContext, "first in param must be Context")
		in1 := t.In(1)
		inParamType = in1
		handlerFlags = handlerFlags | handlerHasCtx | handlerHasReqData
	default:
		std.Assert(false, "illegal func in params num")
	}
	return inParamType, handlerFlags
}

func checkOutParam(t reflect.Type) (reflect.Type, uint32) {
	var handlerFlags uint32 = 0
	var outParamType reflect.Type = nil
	outNum := t.NumOut()
	std.Assert(outNum > 0 && outNum <= 2, "outNum len must be 1(error) or 2(any,error)")
	lastOut := t.Out(outNum - 1)
	std.Assert(lastOut == typeOfError, "the last out param must be 'error'")
	switch outNum {
	case 1:
		//fun(xxx)error
	case 2:
		outParamType = t.Out(0)
		handlerFlags = handlerFlags | handlerHasRsp
	default:
		std.Assert(false, "illegal func return params num")
	}
	return outParamType, handlerFlags
}
