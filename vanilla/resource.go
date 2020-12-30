package vanilla

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bitly/go-simplejson"
	"github.com/kfchen81/beego"
	"github.com/kfchen81/beego/logs"
	"github.com/kfchen81/beego/metrics"
	"github.com/kfchen81/beego/orm"
	"github.com/kfchen81/beego/vanilla/cache"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var _PLATFORM_SECRET string
var _USER_LOGIN_SECRET string
var _ENABLE_RESOURCE_LOGIN_CACHE bool

const _RETRY_COUNT = 3
var _SERVICE_NAME string

// Login Cache: 登录信息的缓存机制
var _RESOURCE_LOGIN_CACHE_SIZE int

var v2kOption = cache.WithValue2Key()

var corpTTLOption = cache.WithTTL(time.Duration(24) * time.Hour)
var corpLoginCache cache.Cache

var userTTLOption = cache.WithTTL(time.Duration(24) * time.Hour)
var userLoginCache cache.Cache

// http client 参数
var _HTTP_DIAL_TIMEOUT = 5
var _HTTP_DIAL_KEEPALIVE = 60
var _HTTP_IdleConnsPerHost = 1000
var _HTTP_MaxIdleConns = 2000
var _HTTP_IdleConnTimeout = 60
var _HTTP_REUSE_TRANSPORT = false

const InvalidJwtError = "jwt:invalid_jwt_token"


//func PrintLocalDial(ctx context.Context, network, addr string) (net.Conn, error) {
//	dial := net.Dialer{
//		Timeout:   30 * time.Second,
//		KeepAlive: 30 * time.Second,
//	}
//
//	conn, err := dial.Dial(network, addr)
//	if err != nil {
//		return conn, err
//	}
//
//	fmt.Println("connect done, use", conn.LocalAddr().String())
//
//	return conn, err
//}

var globalNetTransport = &http.Transport{
	DialContext: (&net.Dialer{
		Timeout: time.Duration(_HTTP_DIAL_TIMEOUT) * time.Second,
		KeepAlive: time.Duration(_HTTP_DIAL_KEEPALIVE) * time.Second,
	}).DialContext,
	MaxIdleConnsPerHost:   _HTTP_IdleConnsPerHost,
	MaxIdleConns:          _HTTP_MaxIdleConns,
	IdleConnTimeout:       time.Duration(_HTTP_IdleConnTimeout) * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 2 * time.Second,
}

// Request

type ResourceResponse struct {
	RespData *simplejson.Json
}

func (this *ResourceResponse) IsSuccess() bool {
	code, _ := this.RespData.Get("code").Int()
	return code == 200
}

func (this *ResourceResponse) Data() *simplejson.Json {
	return this.RespData.Get("data")
}

// Bind 将respData映射到struct，container一定要是指针类型
func (this *ResourceResponse) Bind(container interface{}) error{
	bs, err := this.Data().MarshalJSON()
	if err != nil{
		return err
	}
	err = json.Unmarshal(bs, container)
	if err != nil{
		return err
	}
	return nil
}

/*RestResource 扩展beego.Controller, 作为rest中各个资源的基类
 */
type Resource struct {
	Ctx            context.Context
	CustomJWTToken string
	disableRetry   bool
}

func (this *Resource) request(method string, service string, resource string, data Map) (respData *ResourceResponse, err error, httpErr error) {
	var jwtToken string
	if this.CustomJWTToken != "" {
		jwtToken = this.CustomJWTToken
	} else {
		var ok bool
		if jwtToken, ok = this.Ctx.Value("jwt").(string); ok {
		
		} else {
			jwtToken = ""
		}
	}
	
	//hasBid := false
	//bid := ""
	//if bidData, ok := data["bid"]; ok {
	//	bid = bidData.(string)
	//	hasBid = true
	//}
	
	usePeanutPure := os.Getenv("USE_PEANUT_PURE")
	if usePeanutPure == "1" && service == "peanut" {
		service = "peanut_pure"
	}
	
	apiServerHost := beego.AppConfig.String("api::API_SERVER_HOST")
	
	//if hasBid {
	//	beego.Warn(fmt.Sprintf("[bid] @resource before create client bid(%s)", bid))
	//}
	//创建client
	//var perNetTransport = &http.Transport{
	//	DialContext: (&net.Dialer{
	//		Timeout: time.Duration(_HTTP_DIAL_TIMEOUT) * time.Second,
	//		KeepAlive: time.Duration(_HTTP_DIAL_KEEPALIVE) * time.Second,
	//	}).DialContext,
	//	MaxIdleConnsPerHost:   _HTTP_IdleConnsPerHost,
	//	MaxIdleConns:          _HTTP_MaxIdleConns,
	//	IdleConnTimeout:       time.Duration(_HTTP_IdleConnTimeout) * time.Second,
	//	TLSHandshakeTimeout:   10 * time.Second,
	//	ExpectContinueTimeout: 2 * time.Second,
	//}
	var netTransport *http.Transport
	if _HTTP_REUSE_TRANSPORT {
		netTransport = globalNetTransport
	} else {
		netTransport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: time.Duration(_HTTP_DIAL_TIMEOUT) * time.Second,
				KeepAlive: time.Duration(_HTTP_DIAL_KEEPALIVE) * time.Second,
			}).DialContext,
			MaxIdleConnsPerHost:   _HTTP_IdleConnsPerHost,
			MaxIdleConns:          _HTTP_MaxIdleConns,
			IdleConnTimeout:       time.Duration(_HTTP_IdleConnTimeout) * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 2 * time.Second,
		}
	}
	
	var netClient = &http.Client{
		Timeout:   time.Second * 20,
		Transport: netTransport,
	}
	
	//if hasBid {
	//	beego.Warn(fmt.Sprintf("[bid] @resource after create client bid(%s)", bid))
	//}

	//构建url.Values
	params := url.Values{"_v": {"1"}, "__source_service": {_SERVICE_NAME}}

	//处理resource
	pos := strings.LastIndexByte(resource, '.')
	resource = fmt.Sprintf("%s/%s", resource[:pos], resource[pos+1:])

	//构建request
	//bytes, _ := json.Marshal(ids)
	apiUrl := fmt.Sprintf("http://%s/%s/%s/", apiServerHost, service, resource)
	var req *http.Request
	if method == "GET" {
		for k, v := range data {
			value := ""
			switch t := v.(type) {
			case int:
				value = fmt.Sprintf("%d", v)
			case bool:
				value = fmt.Sprintf("%t", v)
			case string:
				value = v.(string)
			case float64:
				value = fmt.Sprintf("%f", v)
			default:
				beego.Warn("unknown type: ", t)
			}
			params.Set(k, value)
		}
		apiUrl += "?" + params.Encode()
		beego.Warn("apiUrl: ", apiUrl)
		//strings.NewReader(values.Encode())

		req, err = http.NewRequest("GET", apiUrl, nil)
	} else {
		if method == "PUT" {
			params.Set("_method", "put")
		} else if method == "DELETE" {
			params.Set("_method", "delete")
		}
		apiUrl += "?" + params.Encode()
		beego.Warn("apiUrl: ", apiUrl)

		values := url.Values{}
		for k, v := range data {
			value := ""
			switch t := v.(type) {
			case int:
				value = fmt.Sprintf("%d", v)
			case bool:
				value = fmt.Sprintf("%t", v)
			case string:
				value = v.(string)
			case float64:
				value = fmt.Sprintf("%f", v)
			default:
				beego.Warn("unknown type: ", t)
			}
			values.Set(k, value)
		}

		req, err = http.NewRequest("POST", apiUrl, strings.NewReader(values.Encode()))
	}
	//req, err := http.NewRequest("GET", apiUrl, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err, err
	}

	if method != "GET" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.Header.Set("AUTHORIZATION", jwtToken)
	modeIf := this.Ctx.Value(REQUEST_MODE_CTX_KEY)
	if modeIf != nil{
		req.Header.Set(REQUEST_HEADER_FORMAT, strings.ToUpper(modeIf.(string)))
	}

	//inject open tracing
	span := opentracing.SpanFromContext(this.Ctx)
	if span != nil {
		ext.SpanKindRPCClient.Set(span)
		ext.HTTPUrl.Set(span, apiUrl)
		ext.HTTPMethod.Set(span, method)
		span.Tracer().Inject(
			span.Context(),
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(req.Header),
		)
	}

	//执行request，获得response
	resp, err := netClient.Do(req)

	if err != nil {
		return nil, err, err
	}
	
	defer resp.Body.Close()

	//获取response的内容
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err, err
	}

	jsonObj := new(simplejson.Json)
	err = jsonObj.UnmarshalJSON(body)
	if err != nil {
		return nil, err, err
	}

	resourceResp := new(ResourceResponse)
	resourceResp.RespData = jsonObj
	//fmt.Println(string(body))

	if resourceResp.IsSuccess() {
		return resourceResp, nil, nil
	} else {
		logs.Critical(jsonObj)
		errCode := jsonObj.Get("errCode")
		if errCode == nil {
			return resourceResp, errors.New("remote_service_error"), nil
		} else {
			this.handleJWTError(errCode.MustString())
			return resourceResp, errors.New(errCode.MustString()), nil
		}
	}
}

func (this *Resource) handleJWTError(errCode string) {
	if !_ENABLE_RESOURCE_LOGIN_CACHE {
		return
	}
	if errCode != InvalidJwtError {
		return
	}
	if this.CustomJWTToken == "" {
		return
	}
	corpLoginCache.DelByValue(this.CustomJWTToken)
	userLoginCache.DelByValue(this.CustomJWTToken)
	metrics.GetErrorJwtInCacheCounter().Inc()
}

func (this *Resource) requestWithRetry(method string, service string, resource string, data Map) (resp *ResourceResponse, err error) {
	var httpErr error
	for i := 0; i < _RETRY_COUNT; i++ {
		if i > 0 {
			metrics.GetResourceRetryCounter().Inc()
		}
		resp, err, httpErr = this.request(method, service, resource, data)
		
		if httpErr == nil {
			break
		}
		
		if this.disableRetry {
			break
		}
		
		if i >= _RETRY_COUNT-1 {
			break
		}
		
		hasBid := false
		bid := ""
		if bidData, ok := data["bid"]; ok {
			bid = bidData.(string)
			hasBid = true
		}
		if hasBid {
			beego.Warn(fmt.Sprintf("[bid] retry error(%s), bid(%s)", httpErr.Error(), bid))
		}
	}
	
	return resp, err
}

func (this *Resource) Get(service string, resource string, data Map) (resp *ResourceResponse, err error) {
	return this.requestWithRetry("GET", service, resource, data)
}

func (this *Resource) Put(service string, resource string, data Map) (resp *ResourceResponse, err error) {
	return this.requestWithRetry("PUT", service, resource, data)
}

func (this *Resource) Post(service string, resource string, data Map) (resp *ResourceResponse, err error) {
	return this.requestWithRetry("POST", service, resource, data)
}

func (this *Resource) Delete(service string, resource string, data Map) (resp *ResourceResponse, err error) {
	return this.requestWithRetry("DELETE", service, resource, data)
}

func (this *Resource) LoginAs(username string) *Resource {
	if _PLATFORM_SECRET == "" {
		beego.Error("_PLATFORM_SECRET is '', Please set _PLATFORM_SECRET in your *.conf file")
		return nil
	}
	
	if _ENABLE_RESOURCE_LOGIN_CACHE {
		if jwt, ok := corpLoginCache.Get(username); ok {
			this.CustomJWTToken = jwt.(string)
			return this
		}
	}
	
	resp, err := this.Put("gskep", "login.logined_corp_user", Map{
		"username": username,
		"password": _PLATFORM_SECRET,
	})
	if err != nil {
		logs.Critical(err)
		return nil
	}
	
	respData := resp.Data()
	jwt, _ := respData.Get("sid").String()
	if _ENABLE_RESOURCE_LOGIN_CACHE {
		corpLoginCache.Set(username, jwt)
	}
	this.CustomJWTToken = jwt
	return this
}

func (this *Resource) LoginAsUser(unionid string) *Resource {
	if _USER_LOGIN_SECRET == "" {
		beego.Error("_USER_LOGIN_SECRET is '', Please set _USER_LOGIN_SECRET in your *.conf file")
		return nil
	}
	
	if _ENABLE_RESOURCE_LOGIN_CACHE {
		if jwt, ok := userLoginCache.Get(unionid); ok {
			this.CustomJWTToken = jwt.(string)
			return this
		}
	}
	
	resp, err := this.Put("gskep", "login.logined_h5_user", Map{
		"unionid": unionid,
		"secret": _USER_LOGIN_SECRET,
	})
	if err != nil {
		logs.Critical(err)
		return nil
	}
	
	respData := resp.Data()
	jwt, _ := respData.Get("sid").String()
	if _ENABLE_RESOURCE_LOGIN_CACHE {
		userLoginCache.Set(unionid, jwt)
	}
	this.CustomJWTToken = jwt
	return this
}

func (this *Resource) LoginAsManager() *Resource {
	return this.LoginAs("manager")
}

func (this *Resource) DisableRetry() *Resource {
	this.disableRetry = true
	return this
}

func CronLogin(o orm.Ormer) (*Resource, error) {
	apiServerHost := beego.AppConfig.String("api::API_SERVER_HOST")
	apiUrl := fmt.Sprintf("http://%s/skep/account/logined_corp_user", apiServerHost)
	params := url.Values{"_v": {"1"}}
	params.Set("_method", "put")
	apiUrl += "?" + params.Encode()
	
	values := url.Values{}
	
	values.Set("username", "manager")
	values.Set("password", "dc120c3e372d9ba9998a52c9d8edcdcb")
	
	resp, err := http.PostForm(apiUrl, values)
	
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	jsonObj := new(simplejson.Json)
	err = jsonObj.UnmarshalJSON(body)
	
	jsonData, err := jsonObj.Map()
	
	if err != nil {
		return nil, err
	}
	
	dataMap := jsonData["data"].(map[string]interface{})
	jwt := dataMap["sid"].(string)
	
	ctx := context.Background()
	span := opentracing.StartSpan("CRONTAB")
	ctx = opentracing.ContextWithSpan(ctx, span)
	ctx = context.WithValue(ctx, "jwt", jwt)
	ctx = context.WithValue(ctx, "orm", o)
	resource := NewResource(ctx)
	resource.CustomJWTToken = jwt
	
	return resource, nil
}

func NewResource(ctx context.Context) *Resource {
	resource := new(Resource)
	resource.Ctx = ctx
	resource.disableRetry = false
	return resource
}

func ToJsonString(obj interface{}) string {
	bytes, _ := json.Marshal(obj)
	return string(bytes)
}

func init() {
	_PLATFORM_SECRET = beego.AppConfig.String("system::PLATFORM_SECRET")
	_USER_LOGIN_SECRET = beego.AppConfig.String("system::USER_LOGIN_SECRET")
	_SERVICE_NAME = beego.AppConfig.String("appname")
	_ENABLE_RESOURCE_LOGIN_CACHE = beego.AppConfig.DefaultBool("system::ENABLE_RESOURCE_LOGIN_CACHE", true)
	_RESOURCE_LOGIN_CACHE_SIZE = beego.AppConfig.DefaultInt("system::RESOURCE_LOGIN_CACHE_SIZE", 100)
	beego.Info("[init] use _PLATFORM_SECRET: ", _PLATFORM_SECRET)
	beego.Info("[init] use _USER_LOGIN_SECRET: ", _USER_LOGIN_SECRET)
	beego.Info("[init] use _ENABLE_RESOURCE_LOGIN_CACHE: ", _ENABLE_RESOURCE_LOGIN_CACHE)
	beego.Info("[init] use _RESOURCE_LOGIN_CACHE_SIZE: ", _RESOURCE_LOGIN_CACHE_SIZE)
	
	_HTTP_DIAL_TIMEOUT = beego.AppConfig.DefaultInt("httpclient::DIAL_TIMEOUT", 5)
	_HTTP_DIAL_KEEPALIVE = beego.AppConfig.DefaultInt("httpclient::DIAL_KEEPALIVE", 60)
	_HTTP_IdleConnsPerHost = beego.AppConfig.DefaultInt("httpclient::MAX_IDLE_CONNS_PER_HOST", 1000)
	_HTTP_MaxIdleConns = beego.AppConfig.DefaultInt("httpclient::MAX_IDLE_CONNS", 2000)
	_HTTP_IdleConnTimeout = beego.AppConfig.DefaultInt("httpclient::IDLE_CONN_TIMEOUT", 60)
	_HTTP_REUSE_TRANSPORT = beego.AppConfig.DefaultBool("httpclient::REUSE_TRANSPORT", false)
	msg := fmt.Sprintf("[init] use http parameters dial_timeout(%d), dial_keepalive(%d), maxIdleConnsPerHost(%d), maxIdleConns(%d), idleConnTimeout(%d), reuse_transport(%v)", _HTTP_DIAL_TIMEOUT, _HTTP_DIAL_KEEPALIVE, _HTTP_IdleConnsPerHost, _HTTP_MaxIdleConns, _HTTP_IdleConnTimeout, _HTTP_REUSE_TRANSPORT)
	beego.Info(msg)

	userLoginCache = cache.NewLRUCache("user_jwt_token", _RESOURCE_LOGIN_CACHE_SIZE, userTTLOption, v2kOption)
	corpLoginCache = cache.NewLRUCache("corp_jwt_token", _RESOURCE_LOGIN_CACHE_SIZE, corpTTLOption, v2kOption)
}
