package bootx

import (
	"fmt"
	"github.com/gen-iot/std"
	"github.com/labstack/echo/v4"
	"net/http"
	"reflect"
	"runtime"
	"strings"
)

type Context interface {
	echo.Context
	Id() string

	SetUserAuthData(data interface{})
	UserAuthData() interface{}

	SetReq(in interface{})
	Req() interface{}

	SetResp(out interface{})
	Resp() interface{}

	SetHttpStatusCode(code int)
	HttpStatusCode() int

	SetError(err error)
	Err() error
}

type contextImpl struct {
	echo.Context
	AuthData interface{}
	in       interface{}
	out      interface{}
	code     int
	err      error
	message  string
}

func (c *contextImpl) reset() {
	c.Context = nil
	c.AuthData = nil
	c.in = nil
	c.out = nil
	c.code = http.StatusOK
	c.err = nil
	c.message = ""
}

func (c *contextImpl) init(echoCtx echo.Context) {
	c.Context = echoCtx
	c.AuthData = nil
	c.code = http.StatusOK
}

func (c *contextImpl) SetUserAuthData(data interface{}) {
	c.AuthData = data
}

func (c *contextImpl) UserAuthData() interface{} {
	return c.AuthData
}

func (c *contextImpl) Id() string {
	return c.Request().Header.Get(echo.HeaderXRequestID)
}

func (c *contextImpl) SetReq(in interface{}) {
	c.in = in
}

func (c *contextImpl) Req() interface{} {
	return c.in
}

func (c *contextImpl) SetResp(out interface{}) {
	c.out = out
}

func (c *contextImpl) Resp() interface{} {
	return c.out
}

func (c *contextImpl) SetHttpStatusCode(code int) {
	c.code = code
}

func (c *contextImpl) HttpStatusCode() int {
	return c.code
}

func (c *contextImpl) SetError(err error) {
	c.err = err
}

func (c *contextImpl) Err() error {
	return c.err
}

func (c *contextImpl) BindAndValidate(in interface{}) error {
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

func (c *contextImpl) End() error {
	err := c.Err()
	if err != nil {
		return err
	}
	return c.JSONPretty(c.code, c.Resp(), jsonIndent)
}

func (this *WebX) customContextMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(echoCtx echo.Context) error {
		ctx := this.grabCtx()
		defer func() {
			ctx.reset()
			this.releaseCtx(ctx)
		}()
		ctx.init(echoCtx)
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
	fv, ok := handler.(reflect.Value)
	if !ok {
		fv = reflect.ValueOf(handler)
	}
	std.Assert(fv.Kind() == reflect.Func, "handler not func!")
	ft := fv.Type()
	fName := getFuncName(fv)
	inType, inFlags := checkInParam(fName, ft)
	_, outFlags := checkOutParam(fName, ft)
	flags := inFlags | outFlags
	mid := middlewares{}
	mid.Use(m...)
	return ConvertFromEchoCtx(func(ctx Context) error {
		//has req data
		if flags&handlerHasReqData == handlerHasReqData {
			elementType := inType
			isPtr := false
			if inType.Kind() == reflect.Ptr {
				elementType = inType.Elem()
				isPtr = true
			}
			req := reflect.New(elementType).Interface()
			//bind
			err := ctx.Bind(req)
			if err != nil {
				ctx.SetHttpStatusCode(http.StatusBadRequest)
				ctx.SetError(echo.NewHTTPError(http.StatusBadRequest, err.Error()))
			}
			if !isPtr {
				req = reflect.ValueOf(req).Elem().Interface()
			}
			ctx.SetReq(req)
		}
		fn := mid.buildChain(buildInvoke(fv, flags))
		fn(ctx)
		//if has rsp & no error need write response,otherwise err handler will handle
		if !ctx.Response().Committed && ctx.Resp() != nil && ctx.Err() == nil {
			return ctx.JSONPretty(ctx.HttpStatusCode(), ctx.Resp(), jsonIndent)
		}
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
			if !(outs[rspDataIdx]).IsNil() {
				rsp := outs[rspDataIdx].Interface()
				ctx.SetResp(rsp)
			}
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

func checkInParam(name string, t reflect.Type) (reflect.Type, uint32) {
	var handlerFlags uint32 = 0
	var inParamType reflect.Type = nil
	inNum := t.NumIn()
	std.Assert(inNum >= 0 && inNum <= 2,
		fmt.Sprintf("'%s' not valid : inNum len must be 0() or 1(req any) or 2(ctx Context,req Any)", name))
	//
	switch inNum {
	case 0:
		//func()
	case 1:
		// func foo(Context)
		if t.In(0) == typeOfContext {
			handlerFlags = handlerFlags | handlerHasCtx
		} else {
			handlerFlags = handlerFlags | handlerHasReqData
			in1 := t.In(0)
			inParamType = in1
		}
	case 2:
		// func foo(Context,param1)
		in0 := t.In(0)
		std.Assert(in0 == typeOfContext,
			fmt.Sprintf("'%s' not valid :first in param must be Context", name))
		in1 := t.In(1)
		inParamType = in1
		handlerFlags = handlerFlags | handlerHasCtx | handlerHasReqData
	default:
		std.Assert(false, fmt.Sprintf("'%s' not valid :illegal func in params num", name))
	}
	return inParamType, handlerFlags
}

func checkOutParam(name string, t reflect.Type) (reflect.Type, uint32) {
	var handlerFlags uint32 = 0
	var outParamType reflect.Type = nil
	outNum := t.NumOut()
	std.Assert(outNum > 0 && outNum <= 2,
		fmt.Sprintf("'%s' not valid :outNum len must be 1(error) or 2(any,error)", name))
	lastOut := t.Out(outNum - 1)
	std.Assert(lastOut == typeOfError,
		fmt.Sprintf("'%s' not valid :the last out param must be 'error'", name))
	switch outNum {
	case 1:
		//fun(xxx)error
	case 2:
		outParamType = t.Out(0)
		handlerFlags = handlerFlags | handlerHasRsp
	default:
		std.Assert(false, fmt.Sprintf("'%s' not valid :illegal func return params num", name))
	}
	return outParamType, handlerFlags
}

func getFuncName(fv reflect.Value) string {
	fname := runtime.FuncForPC(reflect.Indirect(fv).Pointer()).Name()
	idx := strings.LastIndex(fname, ".")
	if idx != -1 {
		fname = fname[idx+1:]
	}
	idx = strings.LastIndex(fname, "-")
	if idx != -1 {
		fname = fname[:idx]
	}
	return fname
}
