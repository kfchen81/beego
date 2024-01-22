package vanilla

import (
	"context"
	"fmt"
	"github.com/kfchen81/beego/metrics"
	"net/http"
	"strconv"
	"strings"
	
	"github.com/kfchen81/beego"
	
	"encoding/json"
	"github.com/bitly/go-simplejson"
	"github.com/go-redsync/redsync"
	beego_context "github.com/kfchen81/beego/context"
	"github.com/kfchen81/beego/orm"
	"github.com/opentracing/opentracing-go"
)

var emptyStringArray = make([]string, 0)

type RestResourceInterface interface {
	beego.ControllerInterface
	Resource() string
	GetAlias() []string
	EnableHTMLResource() bool
	IsForDevTest() bool
	DisableTx() bool
	GetParameters() map[string][]string
	GetBusinessContext() context.Context
	SetBeegoController(ctx *beego_context.Context, data map[interface{}]interface{})
	GetLockKey() string
	GetLockOption() *LockOption
}

/*RestResource 扩展beego.Controller, 作为rest中各个资源的基类
 */
type RestResource struct {
	beego.Controller

	Name2JSON      map[string]map[string]interface{}
	Name2RAWJSON      map[string]*simplejson.Json
	Name2JSONArray map[string][]interface{}
	Filters        map[string]interface{}
}


// Init generates default values of controller operations.
func (c *RestResource) Init(ctx *beego_context.Context, controllerName, actionName string, app interface{}) {
	c.Controller.Init(ctx, controllerName, actionName, app)
}

// Get adds a request function to handle GET request.
func (c *RestResource) Get() {
	http.Error(c.Ctx.ResponseWriter, "Method Not Allowed", 405)
}

// Post adds a request function to handle POST request.
func (c *RestResource) Post() {
	http.Error(c.Ctx.ResponseWriter, "Method Not Allowed", 405)
}

// Delete adds a request function to handle DELETE request.
func (c *RestResource) Delete() {
	http.Error(c.Ctx.ResponseWriter, "Method Not Allowed", 405)
}

// Put adds a request function to handle PUT request.
func (c *RestResource) Put() {
	http.Error(c.Ctx.ResponseWriter, "Method Not Allowed", 405)
}

// Head adds a request function to handle HEAD request.
func (c *RestResource) Head() {
	http.Error(c.Ctx.ResponseWriter, "Method Not Allowed", 405)
}

// Patch adds a request function to handle PATCH request.
func (c *RestResource) Patch() {
	http.Error(c.Ctx.ResponseWriter, "Method Not Allowed", 405)
}

// Options adds a request function to handle OPTIONS request.
func (c *RestResource) Options() {
	http.Error(c.Ctx.ResponseWriter, "Method Not Allowed", 405)
}

// HandlerFunc call function with the name
func (c *RestResource) HandlerFunc(fnname string) bool {
	return c.Controller.HandlerFunc(fnname)
}

// URLMapping register the internal RestResource router.
func (c *RestResource) URLMapping() {}

// Mapping the method to function
func (c *RestResource) Mapping(method string, fn func()) {
	c.Controller.Mapping(method, fn)
}

/*Resource 返回resource名
 */
func (r *RestResource) Resource() string {
	return ""
}

/*GetAlias 返回别名集合
 */
func (r *RestResource) GetAlias() []string {
	return emptyStringArray
}

func (r *RestResource) SetBeegoController(ctx *beego_context.Context, data map[interface{}]interface{}) {
	r.Ctx = ctx
	r.Data = data
}

/*EnableHTMLResource 是否开启html资源
 */
func (r *RestResource) EnableHTMLResource() bool {
	return false
}

/*IsForDevTest 是否是开发时支持的资源
 */
func (r *RestResource) IsForDevTest() bool {
	return false
}

/*DisableTx 是否关闭事务支持
 */
func (r *RestResource) DisableTx() bool {
	method := r.Ctx.Input.Method()
	if method == "GET" {
		return true
	} else {
		return false
	}
}

/*GetLockKey 获取锁的key
 */
func (r *RestResource) GetLockKey() string {
	return ""
}

func (r *RestResource) GetLockOption() *LockOption{
	return nil
}

/*Parameters 获取需要检查的参数
 */
func (r *RestResource) GetParameters() map[string][]string {
	return nil
}

func (r *RestResource) GetBusinessContext() context.Context {
	data := r.Ctx.Input.GetData("bContext")
	if data == nil {
		return nil
	} else {
		bCtx := data.(context.Context)
		return bCtx
	}
}

func (r *RestResource) GetCorpToken() string {
	data := r.Ctx.Input.GetData("__corp_token")
	if data == nil {
		return ""
	} else {
		token := data.(string)
		return token
	}
}

//returnValidateParameterFailResponse 返回参数校验错误的response
func (r *RestResource) returnValidateParameterFailResponse(parameter string, paramType string, innerErrMsgs ...string) {
	innerErrMsg := ""
	if len(innerErrMsgs) > 0 {
		innerErrMsg = innerErrMsgs[0]
	}
	r.Data["json"] = &Response{
		500,
		nil,
		"rest:missing_argument",
		fmt.Sprintf("missing or invalid argument: %s(%s)", parameter, paramType),
		innerErrMsg,
		GetMachineInfo(),
	}
	r.ServeJSON()
}

func (r *RestResource) returnAcquireLockFailedResponse(lockKey string){
	r.Data["json"] = &Response{
		500,
		nil,
		"rest:acquire_lock_failed",
		fmt.Sprintf("acquire_lock_failed: %s", lockKey),
		"",
		GetMachineInfo(),
	}
	r.ServeJSON()
}

/*Prepare 实现beego.Controller的Prepare函数
 */
func (r *RestResource) Prepare() {
	
	method := r.Ctx.Input.Method()
	r.Name2JSON = make(map[string]map[string]interface{})
	r.Name2RAWJSON = make(map[string]*simplejson.Json)
	r.Name2JSONArray = make(map[string][]interface{})
	r.Filters = make(map[string]interface{})

	if app, ok := r.AppController.(RestResourceInterface); ok {
		//记录counter
		metrics.GetEndpointCounter().WithLabelValues(app.Resource(), method).Inc()
		source := r.Input().Get("__source_service")
		if source == "" {
			source = r.Input().Get("__source")
		}
		metrics.GetEndpointCallByServiceCounter().WithLabelValues(app.Resource(), method, source).Inc()
		
		// 记录local resource
		{
			v := r.Ctx.Input.GetData("bContext")
			switch v.(type) {
			case context.Context:
				bCtx := v.(context.Context)
				bCtx = context.WithValue(bCtx, "SOURCE_RESOURCE", app.Resource())
				bCtx = context.WithValue(bCtx, "SOURCE_METHOD", method)
				r.Ctx.Input.SetData("bContext", bCtx)
				
				o := bCtx.Value("orm")
				switch o.(type) {
				case orm.Ormer:
					ormer := o.(orm.Ormer)
					ormer.SetData("SOURCE_RESOURCE", app.Resource())
					ormer.SetData("SOURCE_METHOD", method)
				}
			default:
				beego.Warn("no business context")
			}
		}
		
		
		method2parameters := app.GetParameters()
		if method2parameters != nil {
			if parameters, ok := method2parameters[method]; ok {
				actualParams := r.Input()
				for _, param := range parameters {
					colonPos := strings.Index(param, ":")
					paramType := "string"
					if colonPos != -1 {
						paramType = param[colonPos+1 : len(param)]
						param = param[0:colonPos]
					}

					canMissParam := false
					if param[0] == '?' {
						canMissParam = true
						param = param[1:]
					}
					if _, ok := actualParams[param]; !ok {
						if !canMissParam {
							r.returnValidateParameterFailResponse(param, paramType, "no paramter provided")
						} else {
							continue
						}
					}
					if paramType == "string" {
						//value := r.GetString(param)
					} else if paramType == "int" {
						_, err := r.GetInt64(param)
						if err != nil {
							r.returnValidateParameterFailResponse(param, paramType, err.Error())
						} else {
							//requestData[param] = value
						}
					} else if paramType == "float" {
						_, err := r.GetFloat(param)
						if err != nil {
							r.returnValidateParameterFailResponse(param, paramType, err.Error())
						} else {
							//requestData[param] = value
						}
					} else if paramType == "bool" {
						value := r.GetString(param)
						_, err := strconv.ParseBool(value)
						if err != nil {
							r.returnValidateParameterFailResponse(param, paramType, err.Error())
						} else {
							//requestData[param] = result
						}
					} else if paramType == "json" {
						value := r.GetString(param)
						if value == "" && canMissParam == true {
							continue
						}
						js, err := simplejson.NewJson([]byte(value))
						if err != nil {
							r.returnValidateParameterFailResponse(param, paramType, err.Error())
						} else {
							data, err := js.Map()
							if err != nil {
								r.returnValidateParameterFailResponse(param, paramType, err.Error())
							} else {
								if param == "filters" {
									r.Filters = data
								} else {
									r.Name2JSON[param] = data
								}
							}
						}
					} else if paramType == "json-raw" {
						value := r.GetString(param)
						if value == "" && canMissParam == true {
							continue
						}
						js, err := simplejson.NewJson([]byte(value))
						if err != nil {
							r.returnValidateParameterFailResponse(param, paramType, err.Error())
						} else {
							r.Name2RAWJSON[param] = js
						}
					} else if paramType == "json-array" {
						value := r.GetString(param)
						if value == "" && canMissParam == true {
							continue
						}
						js, err := simplejson.NewJson([]byte(value))
						if err != nil {
							r.returnValidateParameterFailResponse(param, paramType, err.Error())
						} else {
							data, err := js.Array()
							if err != nil {
								r.returnValidateParameterFailResponse(param, paramType, err.Error())
							} else {
								r.Name2JSONArray[param] = data
							}
						}
					}
				}

				for key, _ := range actualParams {
					if strings.HasPrefix(key, "__f") {
						sps := strings.Split(key, "-")
						op := sps[2]
						switch op {
						case "in", "range", "notin":
							value := r.GetString(key)
							if value != ""{
								js, err := simplejson.NewJson([]byte(value))
								if err != nil {
									r.returnValidateParameterFailResponse(key, "__f", err.Error())
								} else {
									data, err := js.Array()
									if err != nil {
										r.returnValidateParameterFailResponse(key, "__f", err.Error())
									} else {
										r.Filters[key] = data
									}
								}
							}
						default:
							r.Filters[key] = r.GetString(key)
						}
					}
				}
			}
		}
		var lockOption *LockOption
		defaultKey := fmt.Sprintf("rest_api_lock_%s_%s", app.Resource(), method)
		customLockOption := app.GetLockOption()
		needLock := false
		if customLockOption != nil{
			lockOption = customLockOption
			if lockOption.key == ""{
				lockOption.key = defaultKey
			}
			needLock = true
		}else{
			lockKey := app.GetLockKey()
			if lockKey == "" {
				//do not lock
			} else {
				lockOption = NewLockOption(lockKey)
				needLock = true
			}
		}
		if needLock && lockOption != nil{
			mutex, err := Lock.Lock(lockOption.key, lockOption)
			if err != nil{
				r.returnAcquireLockFailedResponse(lockOption.key)
			}
			if mutex != nil {
				r.Ctx.Input.Data()["sessionRestMutex"] = mutex
			}
		}

		bCtx := r.GetBusinessContext()
		o := GetOrmFromContext(bCtx)
		r.Ctx.Input.Data()["sessionOrm"] = o
		if !r.Ctx.ResponseWriter.Started {
			if o != nil {
				if app, ok := r.AppController.(RestResourceInterface); ok {
					if !app.DisableTx() {
						o.Begin()
						beego.Debug("[ORM] start transaction")
					}
				}
			}
		}
	}
}

func (r *RestResource) Finish() {
	bCtx := r.GetBusinessContext()
	if bCtx != nil {
		o := bCtx.Value("orm")
		if o != nil {
			if app, ok := r.AppController.(RestResourceInterface); ok {
				if !app.DisableTx() {
					o.(orm.Ormer).Commit()
					beego.Debug("[ORM] commit transaction")
				}
			}
		}

		//提交open tracing的span
		span := opentracing.SpanFromContext(bCtx)
		if span != nil {
			beego.Debug("[Tracing] finish span in Controller.Finish")
			span.(opentracing.Span).Finish()
		}
		
		//释放锁
		//注意：锁的释放必须在数据库的事务提交之后进行
		if mutex, ok := r.Ctx.Input.Data()["sessionRestMutex"]; ok {
			if mutex != nil {
				beego.Debug("[lock] release resource lock @1")
				mutex.(*redsync.Mutex).Unlock()
			}
		}
	}
}

//GetJSONArray 与key对应的返回json array数据
func (r *RestResource) GetJSONArray(key string) []interface{} {
	if data, ok := r.Name2JSONArray[key]; ok {
		return data
	} else {
		return nil
	}
}

func (r *RestResource) GetIntArray(key string) []int {
	values := make([]int, 0)
	if datas, ok := r.Name2JSONArray[key]; ok {
		for _, data := range datas {
			switch d := data.(type) {
			case json.Number:
				intValue, _ := strconv.Atoi(d.String())
				values = append(values, intValue)
			default:
				fmt.Println(fmt.Sprintf("[warn] invalid number(%v) in int array", data))
			}
		}
		return values
	} else {
		return nil
	}
}

func (r *RestResource) GetStringArray(key string) []string {
	values := make([]string, 0)
	if datas, ok := r.Name2JSONArray[key]; ok {
		for _, data := range datas {
			values = append(values, data.(string))
		}
		return values
	} else {
		return nil
	}
}

//GetJSONArray 与key对应的返回json map数据
func (r *RestResource) GetJSON(key string) map[string]interface{} {
	if data, ok := r.Name2JSON[key]; ok {
		return data
	} else {
		return nil
	}
}

//GetRawJSON 与key对应的返回json数据
func (r *RestResource) GetRawJSON(key string) *simplejson.Json {
	if data, ok := r.Name2RAWJSON[key]; ok {
		return data
	} else {
		return nil
	}
}

func (r *RestResource) GetFillOptions(key string) FillOption {
	fillOption := FillOption{}
	if data, ok := r.Name2JSON[key]; ok {
		for key, _ := range data {
			fillOption[key] = true
		}
	} else {
		return fillOption
	}
	
	return fillOption
}

// 返回filters参数与__f标准查询的map数据
func (r *RestResource) GetFilters() map[string]interface{} {
	return r.Filters
}

/*ReturnJSON 返回json response*/
func (r *RestResource) ReturnJSON(response *Response) {
	r.Data["json"] = response
	r.ServeJSON()
}
