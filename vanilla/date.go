package vanilla

import (
	"time"
	"strings"
	"github.com/kfchen81/beego"
)

var locShanghai, _ = time.LoadLocation("Asia/Shanghai")

func ParseTime(strTime string) time.Time {
	if strings.Count(strTime, ":") == 1 {
		strTime += ":00"
	}
	
	timeVal, err := time.ParseInLocation("2006-01-02 15:04:05", strTime, locShanghai)
	if err != nil {
		beego.Error(err)
	}
	return timeVal
}

