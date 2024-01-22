package middleware

import (
	"fmt"
	go_context "context"
	"github.com/bitly/go-simplejson"
	"github.com/kfchen81/beego/context"
	"github.com/kfchen81/beego/orm"
	"github.com/kfchen81/beego/vanilla"
	"github.com/kfchen81/beego/vanilla/encrypt"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"strconv"
)

func decodeToken(token string) (corpId int, err error) {
	corpid, _, err := encrypt.DecodeToken(token)
	if err != nil {
		return 0, err
	}
	corpId, _ = strconv.Atoi(corpid)
	return
}

var CorpTokenAuthFilter = func(ctx *context.Context) {
	token := ctx.Input.Query("token")
	if token == "" {
		return
	}
	
	corpId, err := decodeToken(token)
	
	if err != nil {
		response := vanilla.MakeErrorResponse(500, "corp_token:invalid_corp", fmt.Sprintf("无效的corp token 1 - [%s]", token))
		ctx.Output.JSON(response, true, false)
		return
	}
	
	if gBContextFactory != nil {
		jsonData := simplejson.New()
		jsonData.Set("corp_id", corpId)
		jsonData.Set("__source", "corp_token_auth")
		bCtx := gBContextFactory.NewContext(go_context.Background(), ctx.Request, 0, "", jsonData) //bCtx is for "business context"
		
		//add tracing span
		spanCtx, _ := vanilla.Tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(ctx.Request.Header))
		uri := ctx.Request.URL.Path
		operationName := fmt.Sprintf("%s %s", ctx.Request.Method, uri)
		span := vanilla.Tracer.StartSpan(operationName, ext.RPCServerOption(spanCtx))
		bCtx = opentracing.ContextWithSpan(bCtx, span)
		
		//add orm
		o := orm.NewOrmWithSpan(span)
		bCtx = go_context.WithValue(bCtx, "orm", o)
		
		ctx.Input.SetData("bContext", bCtx)
		ctx.Input.SetData("span", opentracing.SpanFromContext(bCtx))
	}
}

func init() {

}