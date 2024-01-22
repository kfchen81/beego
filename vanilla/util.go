package vanilla

import (
	"fmt"
	"github.com/kfchen81/beego"
	"net/url"
	"strconv"
	"strings"
)

func HostDomain(host string) string {
	u :=  url.URL{
		Host: host,
	}
	parts := strings.Split(u.Hostname(), ".")
	if len(parts) < 2 {
		return host
	}
	domain := parts[len(parts)-2] + "." +  parts[len(parts)-1]
	return domain
}

func ExtractUniqueIds(datas []IIDable, idType string) []int {
	id2bool := make(map[int]bool)

	for _, data := range datas {
		id := data.GetId(idType)
		id2bool[id] = true
	}
	
	ids := make([]int, 0)
	for id := range id2bool {
		ids = append(ids, id)
	}
	
	return ids
}

func Decimal(value float64) float64 {
	value, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	return value
}

const SERVICE_MODE_REST = "rest"
const SERVICE_MODE_CRON = "cron"
const SERVICE_MODE_EVENT = "event"
func GetServiceMode() string {
	serviceMode := beego.AppConfig.String("system::SERVICE_MODE")
	if serviceMode != SERVICE_MODE_CRON && serviceMode != SERVICE_MODE_REST && serviceMode != SERVICE_MODE_EVENT {
		panic(fmt.Sprintf("[CRITICAL] invalid service mode '%s'", serviceMode))
	}
	
	enableCronMode := beego.AppConfig.DefaultBool("system::ENABLE_CRON_MODE", false)
	if enableCronMode {
		serviceMode = SERVICE_MODE_CRON
	}
	
	return serviceMode
}


// RunInGoroutine
func runAsGorutione(task func ()) {
	defer func() {
		if err := recover(); err != nil {
			beego.Error(err)
		}
	}()
	
	task()
}


func RunInGoroutine(task func ()) {
	go runAsGorutione(task)
}

func init() {
}