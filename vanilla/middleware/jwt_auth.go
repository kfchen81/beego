package middleware

import (
	"github.com/kfchen81/beego/context"
	"net/url"
	
	go_context "context"
	"fmt"
	"strings"
	
	"github.com/kfchen81/beego/vanilla"
	
	"github.com/kfchen81/beego"
	"github.com/kfchen81/beego/orm"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

var SALT string = "030e2cf548cf9da683e340371d1a74ee"
var SKIP_JWT_CHECK_URLS []string = make([]string, 0)

var JWTAuthFilter = func(ctx *context.Context) {
	if _, ok := ctx.Input.Data()["bContext"]; ok {
		//already has a bContext, skip
		return
	}
	uri := ctx.Request.RequestURI
	
	if uri == "/" {
		if gBContextFactory != nil {
			bCtx := gBContextFactory.NewContext(go_context.Background(), ctx.Request, 0, "", nil) //bCtx is for "business context"
			o := orm.NewOrm()
			bCtx = go_context.WithValue(bCtx, "orm", o)
			ctx.Input.SetData("bContext", bCtx)
		}
		return
	}
	
	for _, skipUrl := range SKIP_JWT_CHECK_URLS {
		if strings.Contains(uri, skipUrl) {
			if strings.Contains(uri, "/logined_microapp_corp_user") || strings.Contains(uri, "/logined_angler_user") || strings.Contains(uri, "/user_reflection") {
			
			} else {
				beego.Debug("[jwt_middleware] skip jwt check", "url", skipUrl)
				if gBContextFactory != nil {
					bCtx := gBContextFactory.NewContext(go_context.Background(), ctx.Request, 0, "", nil) //bCtx is for "business context"
					o := orm.NewOrm()
					bCtx = go_context.WithValue(bCtx, "orm", o)
					//尽管跳过jwt处理，但仍然传递jwt，以方便在业务代码中调用其他的data service
					/*
					jwtToken := ctx.Input.Header("AUTHORIZATION")
					if jwtToken != "" {
						bCtx = go_context.WithValue(bCtx, "jwt", jwtToken)
					}
					*/
					
					ctx.Input.SetData("bContext", bCtx)
				}
				return
			}
		}
	}

	jwtToken := ctx.Input.Header("AUTHORIZATION")
	
	if jwtToken == "" {
		jwtToken = ctx.Input.Query("_jwt")
	}
	
	if jwtToken == "" {
		cookie, err := ctx.Request.Cookie("_jwt")
		if err != nil {
			jwtToken = ""
		} else {
			jwtToken = cookie.Value
			jwtToken, err = url.QueryUnescape(jwtToken)
			if err != nil {
				beego.Error(err)
			}
		}
	}
	
	if jwtToken != "" {
		js, err := vanilla.DecodeJWT(jwtToken)

		if err != nil{
			response := vanilla.MakeErrorResponse(500, "jwt:invalid_jwt_token", err.Error())
			ctx.Output.JSON(response, true, false)
			return
		}

		userId, authUserId, err := vanilla.ParseUserIdFromJwtData(js)
		if err != nil{
			response := vanilla.MakeErrorResponse(500, "jwt:invalid_jwt_token", err.Error())
			ctx.Output.JSON(response, true, false)
			return
		}
		
		bCtx := gBContextFactory.NewContext(go_context.Background(), ctx.Request, userId, jwtToken, js) //bCtx is for "business context"
		bCtx = go_context.WithValue(bCtx, "user_id", userId)
		bCtx = go_context.WithValue(bCtx, "uid", authUserId)
		//enhance business context
		{
			//add tracing span
			spanCtx, _ := vanilla.Tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(ctx.Request.Header))
			uri := ctx.Request.URL.Path
			operationName := fmt.Sprintf("%s %s", ctx.Request.Method, uri)
			span := vanilla.Tracer.StartSpan(operationName, ext.RPCServerOption(spanCtx))
			bCtx = opentracing.ContextWithSpan(bCtx, span)
			
			//add orm
			bCtx = go_context.WithValue(bCtx, "jwt", jwtToken)
			o := orm.NewOrmWithSpan(span)
			bCtx = go_context.WithValue(bCtx, "orm", o)
		}
		
		// 识别user location
		location := ctx.Input.Header("X-VXIAOCHENG-Loc")
		bCtx = go_context.WithValue(bCtx, "user_loc", location)
		
		ctx.Input.SetData("bContext", bCtx)
		ctx.Input.SetData("span", opentracing.SpanFromContext(bCtx))
	} else {
		response := vanilla.MakeErrorResponse(500, "jwt:invalid_jwt_token", fmt.Sprintf("无效的jwt token 5 - [%s]", jwtToken))
		ctx.Output.JSON(response, true, false)
		return
	}

}

func init() {
	skipUrls := beego.AppConfig.String("SKIP_JWT_CHECK_URLS")
	if skipUrls == "" {
		beego.Info("SKIP_JWT_CHECK_URLS is empty")
	} else {
		SKIP_JWT_CHECK_URLS = strings.Split(skipUrls, ";")
	}
	
	beego.Info("SKIP_JWT_CHECK_URLS: ", SKIP_JWT_CHECK_URLS)
}