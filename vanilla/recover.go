package vanilla

import (
	"bytes"
	go_context "context"
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

func RecoverPanic(ctx *context.Context) {
	if err := recover(); err != nil {
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
		var resp Map
		if be, ok := err.(*BusinessError); ok {
			resp = Map{
				"code":        500,
				"data":        nil,
				"errCode":     be.ErrCode,
				"errMsg":      be.ErrMsg,
				"innerErrMsg": "",
			}
		} else {
			endpoint := ctx.Request.RequestURI
			pos := strings.Index(endpoint, "?")
			if pos != -1 {
				endpoint = endpoint[:pos]
			}
			resp = Map{
				"code": 531,
				"data": Map{
					"endpoint": endpoint,
				},
				"errCode":     "system:exception",
				"errMsg":      fmt.Sprintf("%s", err),
				"innerErrMsg": "",
			}
		}
		ctx.Output.JSON(resp, true, true)
	}
}

// RecoverFromCronTaskPanic crontask的recover
func RecoverFromCronTaskPanic(ctx go_context.Context) {
	o := GetOrmFromContext(ctx)
	if err := recover(); err!=nil{
		beego.Info("recover from cron task panic...")
		if o != nil{
			o.Rollback()
			beego.Warn("[ORM] rollback transaction for cron task")
		}
		
		{
			// 推送日志到sentry
			errMsg := err.(error).Error()
			if be, ok := err.(*BusinessError); ok{
				errMsg = fmt.Sprintf("%s - %s", be.ErrCode, be.ErrMsg)
			}
			logs.Critical(errMsg)
			beego.CaptureTaskErrorToSentry(ctx, errMsg)
		}
		
	}
}

func init() {
}
