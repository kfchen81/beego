package metrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	_ "github.com/prometheus/client_golang/prometheus/promhttp"
)

var startServiceCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "a_service_start",
	Help: "service start info",
},
	[]string{"time"},
)

var endpointCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "endpoint_call_total",
	Help: "total counts for panic",
},
	[]string{"endpoint", "method"},
)

var endpointCallByServiceCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "endpoint_call_by_service_total",
	Help: "total counts for endpoint call by service",
},
	[]string{"endpoint", "method", "service"},
)

var remoteEndpointCallCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "remote_endpoint_call_total",
	Help: "total counts for remote endpoint call",
},
	[]string{"local_method", "local_resource", "remote_method", "remote_service", "remote_endpoint"},
)

var dbTableAccessCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "db_table_access_total",
	Help: "total counts for db table access",
},
	[]string{"local_method", "local_resource", "db_type", "db_method", "db_table"},
)

var dbReadBatchGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "read_batch_gauge",
	Help: "read batch gauge",
}, []string{"metric"})

/*
var endpointSummary = promauto.NewSummaryVec(
	prometheus.SummaryOpts{
		Name:       "endpoint_durations_seconds",
		Help:       "endpoint latency distributions.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	},
	[]string{"endpoint", "method"},
)*/

//var normDomain = 0.0002
//var normMean = 0.00001
var endpointHistogram = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "endpoint_durations_histogram_seconds",
	Help: "endpoint latency distributions.",
	//Buckets: prometheus.LinearBuckets(normMean-5*normDomain, .5*normDomain, 20),
},
	[]string{"endpoint", "method"},
)

var panicCounter = promauto.NewCounter(prometheus.CounterOpts{
	Name: "panic_total",
	Help: "total counts for panic",
})

var sentryChannelErrorCounter = promauto.NewCounter(prometheus.CounterOpts{
	Name: "sentry_channel_error_total",
	Help: "total error counts for sentry channel",
})

var sentryChannelUnreadGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "sentry_channel_unread",
	Help: "unread counts for sentry channel",
})

var sentryChannelTimeoutCounter = promauto.NewCounter(prometheus.CounterOpts{
	Name: "sentry_channel_timeout_total",
	Help: "timeout counts for sentry channel",
})

var resourceRetryCounter = promauto.NewCounter(prometheus.CounterOpts{
	Name: "resource_retry_total",
	Help: "total counts for resource's retry",
})

var businessErrorCounter = promauto.NewCounter(prometheus.CounterOpts{
	Name: "business_error_total",
	Help: "total counts for business error",
})

var restwsGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "restws_connection",
	Help: "Number of rest proxy websocket connection is active",
})

var restwsErrorCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "restws_error_total",
	Help: "total counts for rest proxy error",
},
	[]string{"option"},
)

var errorJwtInCacheCounter = promauto.NewCounter(prometheus.CounterOpts{
	Name: "error_jwt_in_cache_count",
	Help: "count of get error jwt from cache",
})

var lruCacheCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "lru_cache_counter",
	Help: "Number of operations on the lru cache",
},
	[]string{"name", "operation"},
)

var esRequestTimer = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "es_request_timer",
	Help: "the time of a es request",
}, []string{"index", "action"})

var taChannelIsFullCounter = promauto.NewCounter(prometheus.CounterOpts{
	Name: "ta_channel_is_full_counter",
	Help: "count when ta channel is full",
})

var taTracedDataCounter = promauto.NewCounter(prometheus.CounterOpts{
	Name: "ta_traced_data_counter",
	Help: "data count that ta traced",
})

var taConsumerCounter = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "ta_consumer_counter",
	Help: "count of ta consumer",
})

var taServerPushCounter = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "ta_server_push_counter",
	Help: "count of ta pushed times and failed times",
}, []string{"name"})

var taServerPushTimer = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "ta_server_push_timer",
	Help: "using time of ta server push",
})

var dbConnectionPoolGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "db_connection_pool_gauge",
	Help: "count of ta pushed times and failed times",
}, []string{"db", "type"})

var usingAntsPoolCounter = promauto.NewCounter(prometheus.CounterOpts{
	Name: "count_ants_pool_usage",
	Help: "count of ants pool usage",
})

func GetEsRequestTimer() *prometheus.HistogramVec {
	return esRequestTimer
}

func GetTaChannelIsFullCounter() prometheus.Counter {
	return taChannelIsFullCounter
}

func GetTaTracedDataCounter() prometheus.Counter {
	return taTracedDataCounter
}

func GetTaConsumerCounter() prometheus.Gauge {
	return taConsumerCounter
}

func GetTaServerPushCounter() *prometheus.GaugeVec {
	return taServerPushCounter
}

func GetTaServerPushTimer() prometheus.Histogram {
	return taServerPushTimer
}

func GetLRUCacheCounter() *prometheus.CounterVec {
	return lruCacheCounter
}

func GetErrorJwtInCacheCounter() prometheus.Counter {
	return errorJwtInCacheCounter
}

func GetRestwsGauge() prometheus.Gauge {
	return restwsGauge
}

func GetStartServiceCounter() *prometheus.CounterVec {
	return startServiceCounter
}

func GetRestwsErrorCounter() *prometheus.CounterVec {
	return restwsErrorCounter
}

func GetEndpointCounter() *prometheus.CounterVec {
	return endpointCounter
}

func GetEndpointCallByServiceCounter() *prometheus.CounterVec {
	return endpointCallByServiceCounter
}

func GetRemoteEndpointCallCounter() *prometheus.CounterVec {
	return remoteEndpointCallCounter
}

func GetDBTableAccessCounter() *prometheus.CounterVec {
	return dbTableAccessCounter
}

func GetDBReadBatchGauge() *prometheus.GaugeVec {
	return dbReadBatchGauge
}

func GetEndpointSummary() *prometheus.SummaryVec {
	return nil
}

func GetEndpointHistogram() *prometheus.HistogramVec {
	return endpointHistogram
}

func GetPanicCounter() prometheus.Counter {
	return panicCounter
}

func GetBusinessErrorCounter() prometheus.Counter {
	return businessErrorCounter
}

func GetResourceRetryCounter() prometheus.Counter {
	return resourceRetryCounter
}

func GetSentryChannelErrorCounter() prometheus.Counter {
	return sentryChannelErrorCounter
}

func GetSentryChannelUnreadGuage() prometheus.Gauge {
	return sentryChannelUnreadGauge
}

func GetSentryChannelTimeoutCounter() prometheus.Counter {
	return sentryChannelTimeoutCounter
}

func GetDBConnectionPoolGauge() *prometheus.GaugeVec {
	return dbConnectionPoolGauge
}

func GetAntsPoolUsageCounter() prometheus.Counter {
	return usingAntsPoolCounter
}

func init() {
	fmt.Println("in metrics init")
}
