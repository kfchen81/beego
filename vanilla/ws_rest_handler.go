package vanilla

import (
	"encoding/json"
	"fmt"
	"github.com/kfchen81/beego"
	beecontext "github.com/kfchen81/beego/context"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

type WsResponse struct {
	*Response
	Rid string `json:"rid"`
}

type fakeResponseWriter struct{}

func (f *fakeResponseWriter) Header() http.Header {
	return http.Header{}
}
func (f *fakeResponseWriter) Write(b []byte) (int, error) {
	return 0, nil
}
func (f *fakeResponseWriter) WriteHeader(n int) {}

func handleRequest(restReq RestRequest, rawCtx *beecontext.Context) (resp WsResponse) {
	ctx := beecontext.NewContext()
	defer func() {
		err := recover()
		if err != nil {
			resp = WsRestRecoverPanic(err, ctx, restReq)
		}
	}()
	resp = mockContext(rawCtx, ctx, restReq)
	if resp.Rid != "" {
		return
	}

	cr := beego.BeeApp.Handlers
	controllerInfo, findRouter := cr.FindRouter(ctx)
	if !findRouter {
		resp = WsResponse{
			Response: &Response{
				Code: 404,
				Data: Map{
					"endpoint": restReq.Path,
				},
				ErrMsg:  "",
				ErrCode: "restws:404",
			},
			Rid: restReq.Rid,
		}
		return
		//exception("404", context)
	}
	execController := controllerInfo.Init()
	execController.Init(ctx, "restws.request", ctx.Input.Method(), execController)

	//call prepare function
	execController.Prepare()
	switch ctx.Input.Method() {
	case "GET":
		execController.Get()
	case "POST":
		execController.Post()
	case "PUT":
		execController.Put()
	case "DELETE":
		execController.Delete()
	}
	execController.Finish()
	vcData := reflect.ValueOf(execController).Elem().FieldByName("Data")
	respData := vcData.Interface().(map[interface{}]interface{})["json"]
	resp = WsResponse{respData.(*Response), restReq.Rid}
	return
}

func mockContext(rawCtx *beecontext.Context, ctx *beecontext.Context, restReq RestRequest) (resp WsResponse) {
	restReq.Method = strings.ToUpper(restReq.Method)
	if !strings.HasPrefix(restReq.Path, "/") {
		restReq.Path = fmt.Sprintf("/%s", restReq.Path)
	}
	if !strings.HasSuffix(restReq.Path, "/") {
		restReq.Path = fmt.Sprintf("%s/", restReq.Path)
	}
	rawRequest := rawCtx.Request
	req := &http.Request{
		URL:    &url.URL{Scheme: rawRequest.URL.Scheme, Host: rawRequest.URL.Host, Path: restReq.Path},
		Method: restReq.Method,
	}
	ctx.Reset(&fakeResponseWriter{}, req)

	formData := url.Values{}
	data := make(map[string]interface{}, 0)
	json.Unmarshal([]byte(restReq.Params), &data)
	for k, v := range data {
		formData.Set(k, fmt.Sprint(v))
	}
	req.Form = formData

	token := rawCtx.Input.Query("token")
	jwtToken := rawCtx.Input.Query("_jwt")
	ctx.Request.Form.Set("token", token)
	ctx.Request.Form.Set("_jwt", jwtToken)

	cr := beego.BeeApp.Handlers
	var urlPath = ctx.Input.URL()
	if cr.ExecFilter(ctx, urlPath, beego.BeforeRouter){
		if token != "" {
			response := MakeErrorResponse(500, "corp_token:invalid_corp", fmt.Sprintf("无效的corp token 1 - [%s]", token))
			resp = WsResponse{
				Response: response,
				Rid: restReq.Rid,
			}
			return
		} else {
			response := MakeErrorResponse(500, "jwt:invalid_jwt_token", fmt.Sprintf("无效的jwt token 5 - [%s]", jwtToken))
			resp = WsResponse{
				Response: response,
				Rid: restReq.Rid,
			}
			return
		}
	}
	return
}
