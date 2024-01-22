package vanilla

import (
	"github.com/kfchen81/beego"
)

type IndexController struct {
	beego.Controller
}

func (c *IndexController) Get() {
	c.Redirect("/console/console/", 302)
}
