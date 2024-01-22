package vanilla

import (
	"bytes"
	"fmt"
	"github.com/kfchen81/beego/context"
	"github.com/kfchen81/beego/logs"
	"github.com/kfchen81/beego/metrics"
	"github.com/opentracing/opentracing-go"
	"github.com/go-redsync/redsync"
	"runtime"
	"strings"

	"github.com/kfchen81/beego"
	"github.com/kfchen81/beego/orm"
)

func WsRestRecoverPanic(err interface{}, ctx *context.Context, restReq RestRequest) (resp WsResponse) {
	//err := recover()
	if err != nil {
		//rollback tx
		o := ctx.Input.Data()["sessionOrm"]
		if o != nil {
			o.(orm.Ormer).Rollback()
			beego.Warn("[ORM] rollback transaction")
		}

		//finish span
		span := ctx.Input.GetData("span")
		if span != nil {
			beego.Info("[Tracing] finish span in recoverPanic")
			span.(opentracing.Span).Finish()
		}

		//释放锁
		if mutex, ok := ctx.Input.Data()["sessionRestMutex"]; ok {
			if mutex != nil {
				beego.Debug("[lock] release resource lock @2")
				mutex.(*redsync.Mutex).Unlock()
			}
		}

		//记录到sentry
		{
			errMsg := ""
			if be, ok := err.(*BusinessError); ok {
				errMsg = fmt.Sprintf("%s - %s", be.ErrCode, be.ErrMsg)
			} else {
				errMsg = fmt.Sprint(err)
			}
			beego.CaptureErrorToSentry(ctx, errMsg)
		}

		//记录panic counter
		//1. 非BusinessError需要记录
		//2. IsPanicError为true的BusinessError需要记录
		if be, ok := err.(*BusinessError); ok {
			if be.IsPanicError() {
				metrics.GetPanicCounter().Inc()
			} else {
				metrics.GetBusinessErrorCounter().Inc()
			}
		} else {
			metrics.GetPanicCounter().Inc()
		}

		if err == beego.ErrAbort {
			return
		}

		errMsg := ""
		if be, ok := err.(*BusinessError); ok {
			errMsg = fmt.Sprintf("%s:%s", be.ErrCode, be.ErrMsg)
		} else {
			errMsg = fmt.Sprintf("%s", err)
		}

		//log error info
		var buffer bytes.Buffer
		buffer.WriteString(fmt.Sprintf("[Unprocessed_Exception] %s\n", errMsg))
		buffer.WriteString(fmt.Sprintf("Request URL: %s\n", ctx.Input.URL()))
		for i := 1; ; i++ {
			_, file, line, ok := runtime.Caller(i)
			if !ok {
				break
			}
			buffer.WriteString(fmt.Sprintf("%s:%d\n", file, line))
		}
		if beego.BConfig.RunMode == "dev" {
			logs.Critical(buffer.String())
		} else {
			logs.Critical(strings.Replace(buffer.String(), "\n", ";  ", -1))
		}

		//return error response
		if be, ok := err.(*BusinessError); ok {
			resp = WsResponse{
				Response: &Response{
					Code: 500,
					Data: nil,
					ErrMsg: be.ErrMsg,
					ErrCode: be.ErrCode,
				},
				Rid: restReq.Rid,
			}
		} else {
			endpoint := ctx.Request.RequestURI
			pos := strings.Index(endpoint, "?")
			if pos != -1 {
				endpoint = endpoint[:pos]
			}
			resp = WsResponse{
				Response: &Response{
					Code: 541,
					Data: Map{
						"endpoint": restReq.Path,
					},
					ErrMsg: fmt.Sprintf("%s", err),
					ErrCode: "restws:exception",
				},
				Rid: restReq.Rid,
			}
		}
		return
	}
	resp = WsResponse{
		Response: &Response{
			Code: 541,
			Data: Map{
				"endpoint": restReq.Path,
			},
			ErrMsg: "unknown",
			ErrCode: "restws:exception",
		},
		Rid: restReq.Rid,
	}
	return
}
