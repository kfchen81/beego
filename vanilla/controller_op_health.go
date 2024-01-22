package vanilla

import (
	"github.com/kfchen81/beego"
	"io/ioutil"
	"os"
	"time"
)

type OpHealthController struct {
	beego.Controller
}

func (c *OpHealthController) Get() {
	content, err := ioutil.ReadFile("./image.version")
	if err != nil {
		//panic(err)
		content = []byte("no_image")
	}
	
	beegoMode := os.Getenv("BEEGO_RUNMODE")
	k8sEnv := os.Getenv("_K8S_ENV")
	now := time.Now().Format("2006-01-02 15:04:05")
	serviceName := beego.AppConfig.String("appname")
	resp := MakeResponse(Map{
		"service":   serviceName,
		"is_online": true,
		"time":      now,
		"image":     string(content),
		"mode": beegoMode,
		"k8s_env": k8sEnv,
	})
	c.Data["json"] = resp
	c.ServeJSON()
}
