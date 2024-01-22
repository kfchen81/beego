package metrics

import (
	"fmt"
	"github.com/kfchen81/beego/metrics"
	"github.com/kfchen81/beego/orm"
	"github.com/kfchen81/beego/logs"
	_ "github.com/prometheus/client_golang/prometheus/promhttp"
	"runtime/debug"
	"time"
)
const _DB_REPORT_INTERVAL = 3

func isEnableDBReportor() bool {
	return true
}

func runReportWorker() {
	logs.Info("[db_reportor] reportor-worker is running...")
	time.Sleep(5 * time.Second) // wait db init is finished
	dbNames := orm.GetAllDBNames()
	
	for {
		time.Sleep(_DB_REPORT_INTERVAL * time.Second)
		for _, dbName := range dbNames {
			db, _ := orm.GetDB(dbName)
			stats := db.Stats()
			metrics.GetDBConnectionPoolGauge().WithLabelValues(dbName, "max_open").Set(float64(stats.MaxOpenConnections))
			metrics.GetDBConnectionPoolGauge().WithLabelValues(dbName, "open").Set(float64(stats.OpenConnections))
			metrics.GetDBConnectionPoolGauge().WithLabelValues(dbName, "idle").Set(float64(stats.Idle))
			metrics.GetDBConnectionPoolGauge().WithLabelValues(dbName, "in_use").Set(float64(stats.InUse))
			metrics.GetDBConnectionPoolGauge().WithLabelValues(dbName, "wait").Set(float64(stats.WaitCount))
			metrics.GetDBConnectionPoolGauge().WithLabelValues(dbName, "wait_duration").Set(float64(stats.WaitDuration))
		}
	}
}

func startReportWorker() {
	logs.Info("[db_reportor] start report-worker")
	defer func() {
		if err := recover(); err != nil {
			stack := debug.Stack()
			fmt.Printf("\n>>>>>>>>>>>>>>>>>>>>\n%v\n%s\n<<<<<<<<<<<<<<<<<<<<\n", err, string(stack))
			//restart worker
			go startReportWorker()
		}
	}()
	
	runReportWorker()
}

func StartDBReportService() {
	if isEnableDBReportor() {
		logs.Info("[db_reportor] enbale report")
		go startReportWorker()
	} else {
		logs.Warn("[db_reportor] DISABLED!!!")
	}
}

func init() {
	StartDBReportService()
}
