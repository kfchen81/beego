package vanilla

import (
	"context"
	"os"
	"strings"
)

var (
	REQUEST_MODE_PROD = "PROD"
	REQUEST_MODE_TEST = "TEST"
	REQUEST_HEADER_FORMAT = "Request-Mode"
	REQUEST_MODE_CTX_KEY = "REQUEST_MODE"
)

type requestMode struct {
	define string
}

func (this *requestMode) String() string{
	return strings.ToUpper(this.define)
}

func (this *requestMode) IsTest() bool{
	return strings.HasSuffix(this.String(), "TEST")
}

func (this *requestMode) IsProd() bool{
	return strings.HasSuffix(this.String(), "PROD")
}

// GetRequestModeFromCtx 获取请求模式
// 首先从ctx中获取，如果没有则进行以下判断
// 		生产环境：默认prod
// 		测试环境：如果请求gaia出错，则默认为test
func GetRequestModeFromCtx(ctx context.Context) *requestMode{
	mode := new(requestMode)
	mode.define = REQUEST_MODE_PROD
	modeIf := ctx.Value(REQUEST_MODE_CTX_KEY)
	if modeIf == nil && os.Getenv("_K8S_ENV") == "test"{
		resp, err := NewResource(ctx).Get("gaia", "system.request_mode", Map{})
		if err == nil && resp.IsSuccess(){
			mode.define = resp.Data().MustString()
		}else{
			mode.define = REQUEST_MODE_TEST
		}
		return mode
	}
	if modeIf != nil{
		mode.define = modeIf.(string)
	}
	return mode
}