//package rest
//
//import (
//	"github.com/kfchen81/beego/vanilla"
//)
//
//type Console struct {
//	vanilla.RestResource
//}
//
//func (c *Console) Resource() string {
//	return "console.console"
//}
//
//func (c *Console) EnableHTMLResource() bool {
//	return true
//}
//
//func (c *Console) Get() {
//	resources := make([]string, 0)
//	for _, resource := range vanilla.RESOURCES {
//		//pos := strings.LastIndex(resource, ".")
//		//resource = resource[0:pos] + "-" + resource[pos+1:len(resource)]
//		resources = append(resources, resource)
//	}
//	//c.ViewPath = "./abc"
//	c.Data["Resources"] = resources
//	c.TplName = "service_console.tpl"
//
//	c.Render()
//}


package vanilla

import (
	"github.com/kfchen81/beego"
	"sort"
)

type ConsoleController struct {
	beego.Controller
}

func (c *ConsoleController) Get() {
	resources := make([]string, 0)
	for _, resource := range RESOURCES {
		resources = append(resources, resource)
	}
	sort.Strings(resources)
	
	serviceName := beego.AppConfig.String("appname")
	
	c.Data["ServiceName"] = serviceName
	c.Data["Resources"] = resources
	c.TplName = "service_console.tpl"

	c.Render()
}
