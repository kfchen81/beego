package es

import (
	"context"
	"testing"
	"time"
)

const TIME_LAYOUT = "2006-01-02 15:04:05"

type EsTransferTest struct{
	UserId int `json:"user_id"`
	UserType string `json:"user_type"`
	Money float64 `json:"money"`
	Name string `json:"name"`

}

type EsOrderTest struct{
	Bid string `json:"bid"`
	UserId int `json:"user_id"`
	FinalMoney float64 `json:"final_money"`
	ProductName string `json:"product_name"`
	IsCleared bool `json:"is_cleared"`
	CreatedAt string `json:"created_at"`
	ExtraData map[string]interface{} `json:"extra_data"`
	Transfers []*EsTransferTest `json:"transfers"`

}

var ctx = context.Background()
var indexName = "es_test_order"
var esClient = NewESClient(ctx).Use(indexName)

// prepareData 执行此方法需要首先创建响应的索引，可以执行jarvis中的index.prepare_for_es_test脚本。
// 注意 es的push方法是异步化的，即push的数据在当前请求返回前并不会立马生效，而会在请求后的一小段时间后生效，
// 因此，初次执行该方法不应该立马验证数据
func prepareData(){
	orders := []map[string]interface{}{
		map[string]interface{}{
			"bid": "1908011221039182213",
			"user_id": 111,
			"final_money": 12.3,
			"product_name": "usurp_screen",
			"is_cleared": true,
			"created_at": time.Now().Format(TIME_LAYOUT),
			"extra_data": map[string]interface{}{
				"app": "usurp_screen",
				"bank_name": "华夏银行",
			},
			"transfers": []map[string]interface{}{
				map[string]interface{}{
					"user_id": 112,
					"user_type": "platform",
					"money": 12.3,
					"name": "",
				},
				map[string]interface{}{
					"user_id": 113,
					"user_type": "artist",
					"money": 1.3,
					"name": "艺人1",
				},
				map[string]interface{}{
					"user_id": 114,
					"user_type": "partner",
					"money": 5.5,
					"name": "合伙人1",
				},
				map[string]interface{}{
					"user_id": 115,
					"user_type": "corp",
					"money": 5.5,
					"name": "门店1",
				},
			},
		},
		map[string]interface{}{
			"bid": "19080111203123910235",
			"user_id": 211,
			"final_money": 50,
			"product_name": "largess",
			"is_cleared": true,
			"created_at": time.Now().Format(TIME_LAYOUT),
			"extra_data": map[string]interface{}{
				"app": "largess",
				"bank_name": "南京银行",
			},
			"transfers": []map[string]interface{}{
				map[string]interface{}{
					"user_id": 112,
					"user_type": "platform",
					"money": 10,
					"name": "",
				},
				map[string]interface{}{
					"user_id": 113,
					"user_type": "artist",
					"money": 10,
					"name": "艺人1",
				},
				map[string]interface{}{
					"user_id": 114,
					"user_type": "partner",
					"money": 15,
					"name": "合伙人1",
				},
				map[string]interface{}{
					"user_id": 115,
					"user_type": "corp",
					"money": 35,
					"name": "门店1",
				},
			},
		},
		map[string]interface{}{
			"bid": "1908011827301928312",
			"user_id": 311,
			"final_money": 100,
			"product_name": "lens",
			"is_cleared": false,
			"created_at": time.Now().Format(TIME_LAYOUT),
			"extra_data": map[string]interface{}{
				"app": "lens",
				"label": "keep up",
				"bank_name": "江苏银行",
			},
			"transfers": []map[string]interface{}{
				map[string]interface{}{
					"user_id": 112,
					"user_type": "platform",
					"money": 25,
					"name": "",
				},
				map[string]interface{}{
					"user_id": 113,
					"user_type": "artist",
					"money": 5,
					"name": "艺人1",
				},
				map[string]interface{}{
					"user_id": 114,
					"user_type": "partner",
					"money": 10,
					"name": "合伙人1",
				},
				map[string]interface{}{
					"user_id": 116,
					"user_type": "corp",
					"money": 60,
					"name": "门店2",
				},
			},
		},
	}

	for _, order := range orders{
		esClient.Push(order["bid"].(string), order)
	}
}

// TestEsSearch_1 测试查询功能
func TestEsSearch_1(t *testing.T){
	prepareData()
	filters := map[string]interface{}{
		"bid": "19080111203123910235",
	}
	var records []*EsOrderTest
	esClient.Search(filters).BindRecords(&records)

	t.Log(len(records) == 1 && records[0].Bid == "19080111203123910235")
}

// TestEsSearch_2 测试查询功能
// text类型的英文用contains查询时，因为分词的原因，英文以空格为分届符，因此contains只可以查找到被空格分开的词或整个字符串
func TestEsSearch_2(t *testing.T){
	prepareData()
	filters := map[string]interface{}{
		"product_name__contains": "screen",
	}
	var records []*EsOrderTest
	esClient.Search(filters).BindRecords(&records)

	t.Log(len(records) == 1 && records[0].Bid == "1908011221039182213" && records[0].ProductName == "usurp_screen") // false

	filters = map[string]interface{}{
		"extra_data>label__contains": "up",
	}
	esClient.Search(filters).BindRecords(&records)
	t.Log(len(records) == 1 && records[0].Bid == "1908011827301928312" && records[0].ProductName == "lens") // true

}

// TestEsSearch_3 测试查询功能
// text类型的英文无法用contains查询，因为分词的原因，而中文可以
// 不要用term查询text类型的field,只要term的value符合其中一个分词就会被搜索到，而不是预期的精确匹配
func TestEsSearch_3(t *testing.T){
	prepareData()
	filters := map[string]interface{}{
		"extra_data>bank_name": "银",
	}
	var records []*EsOrderTest
	esClient.Search(filters).BindRecords(&records)

	t.Log(len(records) == 3) // true

	filters = map[string]interface{}{
		"extra_data>bank_name__contains": "京银",
	}
	esClient.Search(filters).BindRecords(&records)
	t.Log(len(records) == 1 && records[0].Bid == "19080111203123910235") // true
}

// TestEsSearch_4 nest查询
func TestEsSearch_4(t *testing.T){
	prepareData()
	filters := map[string]interface{}{
		"transfers.user_id": 112,
		"transfers.money": 25,
	}
	var records []*EsOrderTest
	esClient.Search(filters).BindRecords(&records)
	t.Log(len(records) == 1 && records[0].Bid == "1908011827301928312") // true

	filters = map[string]interface{}{
		"transfers.user_id": 112,
		"bid": "1908011827301928312",
	}
	esClient.Search(filters).BindRecords(&records)
	t.Log(len(records) == 1 && records[0].ProductName == "lens") // true
}

// TestEsSearch_5 聚合查询, 不返回查询记录，只返回聚合查询记录
func TestEsSearch_5(t *testing.T){
	prepareData()
	filters := map[string]interface{}{}
	aggsStr := `
		{
			"total_money": {
				"sum": {
					"field": "final_money"
				}
			}
		}
	`
	esClient.NoHits()
	esClient.Search(filters, map[string]interface{}{
		"rawAggs": aggsStr,
	})
	totalMoney := esClient.GetSumAgg("total_money")
	t.Log(totalMoney == 162.30) // true
}

// TestEsSearch_6 嵌套聚合查询
func TestEsSearch_6(t *testing.T){
	prepareData()
	filters := map[string]interface{}{}
	aggsStr := `
		{
			"total_money": {
				"nested": {
					"path": "transfers"
				},
				"aggs": {
					"total_money": {
						"sum": {
							"field": "transfers.money"
						}
					}
				}
			}
		}
	`
	esClient.NoHits()
	esClient.Search(filters, map[string]interface{}{
		"rawAggs": aggsStr,
	})
	totalMoney := esClient.GetNestedSumAgg("total_money", "total_money")
	t.Log(totalMoney, totalMoney == 194.60) // 194.6, true
}

// TestEsSearch_7 带filter的嵌套聚合查询
func TestEsSearch_7(t *testing.T){
	prepareData()
	filters := map[string]interface{}{}
	aggsStr := `
		{
			"total_money": {
				"nested": {
					"path": "transfers"
				},
				"aggs": {
					"total_money": {
						"sum": {
							"field": "transfers.money"
						}
					},
					"total_artist_money": {
						"filter": {
							"term": {
								"transfers.user_type": "artist"
							}
						},
						"aggs": {
							"total_artist_money": {
								"sum": {
									"field": "transfers.money"
								}
							}
						}
					},
					"total_corp_user_money": {
						"filter": {
							"term": {
								"transfers.user_id": 115
							}
						},
						"aggs": {
							"total_corp_user_money": {
								"sum": {
									"field": "transfers.money"
								}
							}
						}
					}
				}
			}
		}
	`
	esClient.NoHits()
	esClient.Search(filters, map[string]interface{}{
		"rawAggs": aggsStr,
	})
	totalMoney := esClient.GetNestedSumAgg("total_money", "total_money")
	t.Log(totalMoney, totalMoney == 194.60) // 194.6, true

	totalArtistMoney := esClient.GetNestedSumAggWithFilter("total_money", "total_artist_money")
	t.Log(totalArtistMoney, totalArtistMoney == 16.30) // 16.3, true

	totalCorpUserMoney := esClient.GetNestedSumAggWithFilter("total_money", "total_corp_user_money")
	t.Log(totalCorpUserMoney, totalCorpUserMoney == 40.50) // 40.5, true
}