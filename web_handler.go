package bootx

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"net/http"
)

//统一异常处理
func (this *WebX) defaultErrorHandler(err error, ctx echo.Context) {
	switch cause := errors.Cause(err).(type) {
	case *echo.HTTPError:
		{
			//http error
			msg := fmt.Sprintf("%v", cause.Message)
			e := ctx.JSONPretty(cause.Code,
				map[string]interface{}{
					"code":    cause.Code,
					"message": msg,
				}, jsonIndent)
			if e != nil {
				logger.Printf("error while response :%v", err)
			}
		}
	default:
		{
			logger.Printf("unexpect error :%v", err)
			msg := "unexpect error "
			if this.conf.Debug {
				msg = fmt.Sprintf("%v", err)
			}
			//500
			e := ctx.JSONPretty(http.StatusOK,
				map[string]interface{}{
					"code":    -1,
					"message": msg,
				}, jsonIndent)
			if e != nil {
				logger.Printf("error while response :%v", err)
			}
		}
	}
}
