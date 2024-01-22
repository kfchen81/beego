# ESClient - golang es api 封装

### 配置项
```
[es]
ES_SEARCH_HOST = 127.0.0.1 es服务器地址
ES_SEARCH_PORT = 9200   es服务端口
ES_AUTH_USER = ""   es服务认证用户
ES_AUTH_SECRET = "" es服务认证密码
```

### 使用方式

#### 创建实例
```
    // ctx 必须
    esClient := NewESClient(ctx).Use(indexName).Select(docType) // 当indexName和docType一样时，Select可以省略
```

#### 向es提交新数据
```
    esClient.Push(id, data) // id为字符串， data必须是能够序列化的数据，比如data中如果存在decimal数据类型则会报错
```

#### 更新数据
```
    esClient.Update(data, filters) // data为要更新的数据；filters为定位更新文档的查询条件
```

#### 查询
```
    params := map[string]interface{}{
        "pageInfo": vanilla.ExtractPageInfoFromRequest(ctx),
        "sortAttrs": []string{"-id"},
        "rawAggs": "xxx", // 聚合查询参数的字符串形式，一般使用此参数作为聚合查询
        "aggs": ..., // elasticSDK中提供的聚合实现，高阶用法，使用此参数代表你熟悉elasticSDK中的相关实现
    }
    var records []*EsRecords // 注意，此处的EsRecords需要按照预期es查询返回的数据定义，详细参考es_test.go
    // BindRecords 方法将es返回的数据绑定到传入的参数，注意参数一定要是指针值
    esClient.Search(filters, params).BindRecords(&records) // filters为查询条件，params为其他辅助参数，比如分页、排序等
    nestPageInfo := esClient.GetPageResult() // 获取分页结果
```

#### 聚合查询
```
    aggStr = `
        {
            "total_order_money": {
                "sum": {
                    "field": "final_money"
                }
            },
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
                }
            }
        }
    `
    esClient.NoHits() // 如果只需要聚合查询的数据，则调用此方法后search的result即为空
    esClient.Search(filters, map[string]interface{}{
        "rawAggs": aggStr,
    })
    totalOrderMoney := esClient.GetSumAgg("total_order_money")
    totalMoney := esClient.GetNestedSumAgg("total_money", "total_money")
    totalArtistMoney := esClient.GetNestedSumAggWithFilter("total_money", "total_artist_money")

    // 此处只实现了sum聚合，更多的聚合查询可以自己实现，参考上述方法的源码
```

#### 辅助方法
```
    // 为了支持更高的自由度，esClient暴露了几个elasticSDK的实现
    agg := esClient.GetAggregation() // 获取聚合查询数据
    searchResult := esClient.GetSearchResult() // 获取search查询的结果
    rawRecords := esClient.GetSearchRecords() // 获取查询结果集
```

#### 注意es文档字段类型 keyword和text
- text类型的英文用contains查询时，因为分词的原因，英文以空格为分届符，因此contains只可以查找到被空格分开的词或整个字符串；而中文可以任意查询
- 不要用term查询text类型的field,只要term的value符合其中一个分词就会被搜索到，而不是预期的精确匹配

#### 上述例子都可以在[es_test.go](./es_test.go)中查看到