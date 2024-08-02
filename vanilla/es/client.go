package es

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kfchen81/beego"
	"github.com/kfchen81/beego/metrics"
	"github.com/kfchen81/beego/vanilla"
	"github.com/olivere/elastic"
	"math"
	"reflect"
	"strings"
	"time"
)

type ESClient struct {
	vanilla.ServiceBase

	client *elastic.Client

	syncUpdate bool // 是否等待更新完成

	indexName string
	docType   string

	withNoHits bool

	pageInfo *vanilla.PageInfo

	searchResult *elastic.SearchResult
}

func (this *ESClient) Use(indexName string) *ESClient {
	this.indexName = indexName
	this.docType = indexName
	return this
}

func (this *ESClient) Select(docType string) *ESClient {
	this.docType = docType
	return this
}

func (this *ESClient) NoHits() *ESClient {
	this.withNoHits = true
	return this
}

func (this *ESClient) UseSyncUpdate(b bool) *ESClient {
	this.syncUpdate = b
	return this
}

func (this *ESClient) prepareUpdateData(updateService *elastic.UpdateByQueryService, data map[string]interface{}, filters map[string]interface{}) {
	// make query
	parser := NewQueryParser()
	query := parser.Parse(filters)
	updateService.Query(query)
	// make script
	script := make([]string, 0)
	for k, _ := range data {
		script = append(script, fmt.Sprintf("ctx._source.%s=params.%s", k, k))
	}
	scriptStr := strings.Join(script, ";")
	eScript := elastic.NewScript(scriptStr)
	eScript.Lang("painless").Params(data)
	updateService.Script(eScript)
	// make params
	updateService.Conflicts("proceed").Slices(50).WaitForCompletion(this.syncUpdate).Size(-1)
}

func (this *ESClient) Update(data map[string]interface{}, filters map[string]interface{}) error {
	startTime := time.Now()
	updateService := this.client.UpdateByQuery(this.indexName).Type(this.docType)
	this.prepareUpdateData(updateService, data, filters)
	// 失败后最多重试3次
	var err error
	for count := 3; count >= 0; count-- {
		_, err = updateService.Do(this.Ctx)
		if err == nil {
			break
		}
		time.Sleep(time.Millisecond * 100)
	}

	timeDur := time.Since(startTime)
	metrics.GetEsRequestTimer().WithLabelValues(this.indexName, "update").Observe(timeDur.Seconds())
	if err != nil {
		beego.Error(err)
	}
	return err
}

func (this *ESClient) Push(id string, data interface{}) {
	// Add a document
	startTime := time.Now()
	indexResult, err := this.client.Index().
		Index(this.indexName).
		Type(this.docType).
		Id(id).
		BodyJson(&data).
		Do(this.Ctx)
	timeDur := time.Since(startTime)
	metrics.GetEsRequestTimer().WithLabelValues(this.indexName, "push").Observe(timeDur.Seconds())
	if err != nil {
		errMsg := fmt.Errorf("es_push doc(id:%s) to index %s: %v", id, this.indexName, err)
		beego.Error(errMsg)
		panic(vanilla.NewSystemError("es_push:failed", errMsg.Error()))
	}
	if indexResult == nil {
		errMsg := fmt.Errorf("es_push doc(id:%s) to index %s: result is %v",
			id, this.indexName, indexResult)
		beego.Error(errMsg)
		panic(vanilla.NewSystemError("es_push:failed", errMsg.Error()))
	}
}

// Search 查询
// args: [pageInfo, sortAttrs, rawAggs, aggs]
func (this *ESClient) Search(filters map[string]interface{}, args ...map[string]interface{}) *ESClient {
	parser := NewQueryParser()
	query := parser.Parse(filters)
	searchService := this.client.Search().Index(this.indexName).Type(this.docType).MaxResponseSize(20 << 32)

	switch len(args) {
	case 1:
		param := args[0]
		if p, ok := param["pageInfo"]; ok && p != nil {
			// 分页
			pageInfo := p.(*vanilla.PageInfo)
			this.pageInfo = pageInfo
			searchService = searchService.From((pageInfo.Page - 1) * pageInfo.CountPerPage).Size(pageInfo.CountPerPage)
		}
		if p, ok := param["sortAttrs"]; ok && p != nil {
			// 排序
			sortAttrs := p.([]string)
			for _, sat := range sortAttrs {
				sps := strings.Split(sat, "-")
				var sortField string
				var asc bool
				if len(sps) == 1 {
					sortField = sps[0]
					asc = true
				} else {
					sortField = sps[1]
					asc = sps[0] == "+"
				}
				searchService = searchService.Sort(sortField, asc)
			}
		}
		if p, ok := param["rawAggs"]; ok && p != nil {
			// 聚合
			var aggs map[string]interface{}
			err := json.Unmarshal([]byte(p.(string)), &aggs)
			if err != nil {
				beego.Error(err)
				panic(vanilla.NewSystemError("es:invalid_aggs_format", "不合法的聚合查询参数"))
			}
			for name, aggData := range aggs {
				searchService = searchService.Aggregation(name, newAggregation(aggData))
			}
		}
		if p, ok := param["aggs"]; ok && p != nil {
			for _, agg := range p.([]*NamedAggregation) {
				searchService = searchService.Aggregation(agg.GetName(), agg.GetAggregation())
			}
		}
		if p, ok := param["routing"]; ok && p != nil {
			searchService = searchService.Routing(fmt.Sprintf("%v", p))
		}
	}

	// 不返回任何结果，用于只获取聚合数据
	if this.withNoHits {
		searchService = searchService.Size(0)
	}
	startTime := time.Now()
	result, err := searchService.Query(query).Do(this.Ctx)
	timeDur := time.Since(startTime)
	metrics.GetEsRequestTimer().WithLabelValues(this.indexName, "search").Observe(timeDur.Seconds())
	if err != nil {
		beego.Error(err)
	}
	this.searchResult = result
	return this
}

// BindRecords 将搜索记录绑定到一个包含struct的slice中
// container一定是某个slice的地址，如:
//		var orders []*Order
//		es.BindRecords(&orders)
func (this *ESClient) BindRecords(container interface{}) {
	if this.searchResult == nil || this.withNoHits {
		return
	}

	containerType := reflect.TypeOf(container).Elem()
	slice := reflect.Indirect(reflect.MakeSlice(containerType, 0, 0))

	elmType := containerType.Elem().Elem()
	hits := this.searchResult.Hits.Hits
	for _, hit := range hits {
		js, _ := hit.Source.MarshalJSON()
		elmIface := reflect.New(elmType).Interface()
		json.Unmarshal(js, elmIface)
		slice = reflect.Append(slice, reflect.ValueOf(elmIface))
	}
	reflect.Indirect(reflect.ValueOf(container)).Set(slice)
}

func (this *ESClient) GetSearchRecords() []*elastic.SearchHit {
	return this.searchResult.Hits.Hits
}

func (this *ESClient) GetSearchResult() *elastic.SearchResult {
	return this.searchResult
}

func (this *ESClient) GetPageResult() vanilla.INextPageInfo {
	if this.pageInfo != nil {
		return vanilla.MockPaginate(this.searchResult.TotalHits(), this.pageInfo)
	}
	return nil
}

func (this *ESClient) GetAggregation() elastic.Aggregations {
	if this.searchResult == nil {
		return nil
	}
	return this.searchResult.Aggregations
}

// GetSumAgg 聚合查询
func (this *ESClient) GetSumAgg(name string) float64 {
	result, ok := this.GetAggregation().Sum(name)
	if ok {
		return math.Trunc(*result.Value*1e2+0.5) * 1e-2 // 四舍五入保留两位小数
	}
	return 0
}

// GetSumAggWithFilter 带 filter 的聚合
func (this *ESClient) GetSumAggWithFilter(aggName string) float64 {
	filterAgg, iOK := this.GetAggregation().Filter(aggName)
	if iOK {
		result, iiOK := filterAgg.Sum(aggName)
		if iiOK {
			return math.Trunc(*result.Value*1e2+0.5) * 1e-2 // 四舍五入保留两位小数
		}
	}
	return 0
}

// GetNestedSumAgg 嵌套聚合查询
func (this *ESClient) GetNestedSumAgg(nestedName, aggName string) float64 {
	nestAgg, ok := this.GetAggregation().Nested(nestedName)
	if ok {
		result, innerOk := nestAgg.Sum(aggName)
		if innerOk {
			return math.Trunc(*result.Value*1e2+0.5) * 1e-2 // 四舍五入保留两位小数
		}
	}
	return 0
}

// GetNestedSumAggWithFilter 带filter的嵌套聚合查询
func (this *ESClient) GetNestedSumAggWithFilter(nestedName, aggName string) float64 {
	nestAgg, ok := this.GetAggregation().Nested(nestedName)
	if ok {
		filterAgg, iOk := nestAgg.Filter(aggName)
		if iOk {
			result, iiOk := filterAgg.Sum(aggName)
			if iiOk {
				return math.Trunc(*result.Value*1e2+0.5) * 1e-2 // 四舍五入保留两位小数
			}
		}
	}
	return 0
}

func NewESClient(ctx context.Context) *ESClient {

	host := beego.AppConfig.String("es::ES_SEARCH_HOST")
	port := beego.AppConfig.String("es::ES_SEARCH_PORT")
	user := beego.AppConfig.String("es::ES_AUTH_USER")
	pwd := beego.AppConfig.String("es::ES_AUTH_SECRET")

	beego.Info(host, port, user, pwd)

	optionFunc := func(c *elastic.Client) error {
		var err error
		if user != "" {
			err = elastic.SetBasicAuth(user, pwd)(c)
		}
		err = elastic.SetURL("http://" + host + ":" + port)(c)
		return err
	}

	client := new(ESClient)
	client.Ctx = ctx
	c, err := elastic.NewSimpleClient(optionFunc)
	if err != nil {
		beego.Error(err)
		panic(vanilla.NewSystemError("es:link_failed", "连接es服务失败"))
	}
	client.client = c
	return client
}
