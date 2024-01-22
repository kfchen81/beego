package testing

import (
	"github.com/kfchen81/beego"
	"github.com/kfchen81/beego/logs"
	"os"
	"path/filepath"
	"runtime"
)

func init() {
	var dir string
	dir = os.Getenv("BEEGO_TEST_CONF")
	if dir == "" {
		_, file, _, ok := runtime.Caller(0)
		if ok {
			dir = filepath.Dir(file)
		}
	}

	if dir == "" {
		logs.Critical("Cannot find current dir to set test conf")
	}
	beego.TestBeegoInit(dir)
}
