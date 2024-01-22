package middleware

import (
	"fmt"
	"github.com/kfchen81/beego"
	"github.com/kfchen81/beego/context"
	"github.com/kfchen81/beego/vanilla"

	go_context "context"
)

var RequestHeaderDetectFilter = func(ctx *context.Context) {
	reqMode := ctx.Input.Header(vanilla.REQUEST_HEADER_FORMAT)
	if reqMode != ""{
		v := ctx.Input.GetData("bContext")
		var bCtx go_context.Context
		if v == nil{
			bCtx = go_context.Background()
		}else{
			bCtx = v.(go_context.Context)
		}

		bCtx = go_context.WithValue(bCtx, "REQUEST_MODE", reqMode)
		ctx.Input.SetData("bContext", bCtx)
		beego.Info(fmt.Sprintf("set request mod: %s", reqMode))
	}
}