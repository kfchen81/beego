package vanilla

import (
	"fmt"
	"strings"

	"reflect"

	"github.com/kfchen81/beego"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"os"
)

//RESOURCES 所有资源名的集合
var RESOURCES = make([]string, 0, 100)

var enableDevTestResource = (os.Getenv("ENABLE_DEV_TEST_RESOURCE") == "1")

//Router 添加路由
func Router(r RestResourceInterface) {
	//check whether is dev RESOURCE
	if r.IsForDevTest() && !enableDevTestResource {
		return
	}
	
	resource := r.Resource()
	RESOURCES = append(RESOURCES, resource)

	items := strings.Split(resource, ".")
	
	// html url
	if r.EnableHTMLResource() {
		url := fmt.Sprintf("/%s/", strings.Join(items, "/"))
		beego.Info(fmt.Sprintf("[resource]: %s -> %s", url, reflect.TypeOf(r)))
		beego.Router(url, r)
		return
	}
	
	//standard url
	{
		url := fmt.Sprintf("/%s/", strings.Join(items, "/"))
		beego.Info(fmt.Sprintf("[resource]: %s -> %s", url, reflect.TypeOf(r)))
		beego.Router(url, r)
	}
	
	//alias url
	{
		for _, alias := range r.GetAlias() {
			url := alias
			if url[0] != '/' {
				url = "/" + url
			}
			if url[len(url)-1] != '/' {
				url = url + "/"
			}
			beego.Info(fmt.Sprintf("[resource alias]: %s -> %s", url, reflect.TypeOf(r)))
			beego.Router(url, r)
		}
	}

	// api url
	{
		lastIndex := len(items) - 1
		lastItem := items[lastIndex]
		items[lastIndex] = "api"

		itemSclie := items[:]
		itemSlice := append(itemSclie, lastItem)
		url := fmt.Sprintf("/%s/", strings.Join(itemSlice, "/"))
		//beego.Info(fmt.Sprintf("[resource]: %s -> %s", url, reflect.TypeOf(r)))
		beego.Router(url, r)
	}
	
	// python eaglet protocol url
	{
		items := strings.Split(resource, ".")
		if len(items) > 2 {
			appItems := items[0 : len(items)-1]
			resourceItem := items[len(items)-1]
			url := fmt.Sprintf("/%s/%s/", strings.Join(appItems, "."), resourceItem)
			beego.Router(url, r)
		}
	}
}

func init() {
	beego.Router("/console/console/", &ConsoleController{})
	beego.Router("/op/health/", &OpHealthController{})
	beego.Handler("/metrics", promhttp.Handler())
	beego.Router("/", &IndexController{})
	Router(&RestProxy{})
}