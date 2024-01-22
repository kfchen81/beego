package beego

import (
	go_context "context"
	"fmt"
	"github.com/getsentry/raven-go"
	"github.com/kfchen81/beego/context"
	"github.com/kfchen81/beego/metrics"
	"net/http"
	"runtime/debug"
	"strings"
	
	"os"
)

var sentryChannel = make(chan map[string]interface{}, 2048)

func isEnableSentry() bool {
	return AppConfig.DefaultBool("sentry::ENABLE_SENTRY", false)
}

const SENTRY_CHANNEL_TIMEOUT = 50

// CapturePanicToSentry will collect error info then send to sentry
func CaptureErrorToSentry(ctx *context.Context, err string) {
	if !isEnableSentry() {
		beegoMode := os.Getenv("BEEGO_RUNMODE")
		if beegoMode == "prod" {
			Warn("Sentry is not enabled under prod mode, Please enable it!!!!")
		}
		return
	}
	
	data := make(map[string]interface{})
	data["err_msg"] = err
	data["service_name"] = AppConfig.String("appname")
	
	//skipFramesCount := AppConfig.DefaultInt("sentry::SKIP_FRAMES_COUNT", 3)
	//contextLineCount := AppConfig.DefaultInt("sentry::CONTEXT_LINE_COUNT", 5)
	//appRootPath := AppConfig.String("appname")
	//inAppPaths := []string{appRootPath}
	
	//var sStacktrace *raven.Stacktrace
	//var sError, ok = err.(error)
	//if ok {
	//	sStacktrace = raven.GetOrNewStacktrace(sError, skipFramesCount, contextLineCount, inAppPaths)
	//} else {
	//	sStacktrace = raven.NewStacktrace(skipFramesCount, contextLineCount, inAppPaths)
	//}
	//sStacktrace = raven.NewStacktrace(skipFramesCount, contextLineCount, inAppPaths)
	//sException := raven.NewException(sError, sStacktrace)
	//spew.Dump(sStacktrace)
	data["stack"] = string(debug.Stack())
	data["raven_http"] = raven.NewHttp(ctx.Request)
	data["http_request"] = ctx.Request
	
	select {
	case sentryChannel <- data:
	default:
		metrics.GetSentryChannelTimeoutCounter().Inc()
		Warn("[sentry] push timeout")
	//case <-time.After(time.Millisecond * SENTRY_CHANNEL_TIMEOUT):
	//	metrics.GetSentryChannelTimeoutCounter().Inc()
	//	Warn("[sentry] push timeout")
	}
	
}

func CaptureTaskErrorToSentry(ctx go_context.Context, errMsg string) {
	if !isEnableSentry() {
		beegoMode := os.Getenv("BEEGO_RUNMODE")
		if beegoMode == "prod" {
			Warn("Sentry is not enabled under prod mode, Please enable it!!!!")
		}
		return
	}
	
	data := make(map[string]interface{})
	data["err_msg"] = errMsg
	data["service_name"] = AppConfig.String("appname")
	
	data["stack"] = string(debug.Stack())
	
	select {
	case sentryChannel <- data:
	
	default:
		metrics.GetSentryChannelTimeoutCounter().Inc()
		Warn("[sentry] push timeout")
	}
}

func PushErrorToSentry(errMsg string, req *http.Request) {
	if !isEnableSentry() {
		return
	}
	
	data := make(map[string]interface{})
	data["err_msg"] = errMsg
	data["service_name"] = AppConfig.String("appname")
	
	stack := string(debug.Stack())
	data["stack"] = stack
	if req != nil {
		data["raven_http"] = raven.NewHttp(req)
		data["http_request"] = req
	}
	select {
	case sentryChannel <- data:
	
	default:
		metrics.GetSentryChannelTimeoutCounter().Inc()
		Warn("[sentry] push timeout")
	}
}

func PushErrorWithExtraDataToSentry(errMsg string, extra map[string]interface{}, req *http.Request) {
	if !isEnableSentry() {
		return
	}
	
	data := make(map[string]interface{})
	data["err_msg"] = errMsg
	data["service_name"] = AppConfig.String("appname")
	
	//stack := string(debug.Stack())
	data["stack"] = "ignore"
	data["extra"] = extra
	if req != nil {
		data["raven_http"] = raven.NewHttp(req)
		data["http_request"] = req
	}
	select {
	case sentryChannel <- data:
	
	default:
		metrics.GetSentryChannelTimeoutCounter().Inc()
		Warn("[sentry] push timeout")
	}
}

func sendSentryPacketV1(data map[string]interface{}) {
	var packet *raven.Packet
	errMsg := data["err_msg"].(string)
	packet = raven.NewPacket(errMsg)
	
	stack := data["stack"].(string)
	packet.Extra = map[string]interface{}{
		"stacktrace": stack,
	}
	
	tags := map[string]string{
		"service_name": data["service_name"].(string),
	}
	raven.Capture(packet, tags)
	Info("push v1 data to sentry success")
}

func sendSentryPacketV2(data map[string]interface{}) {
	var packet *raven.Packet
	errMsg := data["err_msg"].(string)
	
	//封装http request
	httpRequest, ok := data["http_request"].(*http.Request)
	if ok {
		ravenHttp := raven.NewHttp(httpRequest)
		
		method := strings.ToLower(httpRequest.Method)
		if method == "post" || method == "put" || method == "delete" {
			data := make(map[string]string)
			for key, _ := range httpRequest.PostForm {
				value := httpRequest.PostForm.Get(key)
				if len(value) >= 100 {
					value = value[:100] + "..."
				}
				data[key] = value
			}
			ravenHttp.Data = data
		}
		
		packet = raven.NewPacket(errMsg, ravenHttp)
	} else {
		packet = raven.NewPacket(errMsg)
	}
	
	//确定extra
	if extra, ok := data["extra"]; ok {
		packet.Extra = extra.(map[string]interface{})
	} else {
		packet.Extra = make(map[string]interface{})
	}
	
	//确定堆栈信息
	stack, ok := data["stack"].(string)
	if !ok {
		stack = "no stack"
	}
	packet.Extra["stacktrace"] = stack
	
	//其他Tag
	tags := map[string]string{
		"service_name": data["service_name"].(string),
	}
	
	//发送给Raven
	raven.Capture(packet, tags)
}

func runSentryWorker(ch chan map[string]interface{}) {
	Info("[sentry] push-worker is ready to receive message...")
	
	for {
		data := <-sentryChannel
		metrics.GetSentryChannelUnreadGuage().Set(float64(len(sentryChannel)))
		metrics.GetSentryChannelErrorCounter().Inc()
		sendSentryPacketV2(data)
	}
}

func startSentryWorker() {
	Info("[sentry] start push-worker")
	defer func() {
		if err := recover(); err != nil {
			stack := debug.Stack()
			fmt.Printf("\n>>>>>>>>>>>>>>>>>>>>\n%v\n%s\n<<<<<<<<<<<<<<<<<<<<\n", err, string(stack))
			//restart worker
			go startSentryWorker()
		}
	}()
	
	runSentryWorker(sentryChannel)
}

func init() {
	if isEnableSentry() {
		raven.SetDSN(AppConfig.String("sentry::SENTRY_DSN"))
		Info(fmt.Sprintf("[sentry] enable:%t, dsn:%s ", isEnableSentry(), AppConfig.String("sentry::SENTRY_DSN")))
		go startSentryWorker()
	} else {
		Warn("[sentry] sentry is DISABLED!!!")
	}
}
