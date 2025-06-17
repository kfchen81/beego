package es

import (
	"github.com/kfchen81/beego"
	"github.com/kfchen81/beego/vanilla"
	"github.com/olivere/elastic"
	"strings"
)

type Query struct {
	data map[string]interface{}
}

func (this *Query) SetData(data map[string]interface{}) {
	this.data = data
}

func (this *Query) RemoveKey(key string) {
	delete(this.data, key)
}

func (this *Query) Source() (interface{}, error) {
	return this.data, nil
}

// aggregation 内部使用，接收外部拼装好的聚合查询字符串
type aggregation struct {
	data interface{}
}

func (this *aggregation) Source() (interface{}, error) {
	return this.data, nil
}

func newAggregation(data interface{}) *aggregation {
	instance := new(aggregation)
	instance.data = data
	return instance
}

// NamedAggregation 供外部使用，可自由组合elastic提供的各类聚合
type NamedAggregation struct {
	name string
	agg  elastic.Aggregation
}

func (this *NamedAggregation) GetName() string {
	return this.name
}

func (this *NamedAggregation) GetAggregation() elastic.Aggregation {
	return this.agg
}

func NewNamedAggregation(name string, aggregation elastic.Aggregation) *NamedAggregation {
	instance := new(NamedAggregation)
	instance.name = name
	instance.agg = aggregation

	return instance
}

// 将rest filter语法转换成es query语法、sort语法
type QueryParser struct {
	vanilla.ServiceBase

	query        *Query
	mustArray    []map[string]interface{}
	mustNotArray []map[string]interface{}
}

/**
处理 [range, icontains, gte, lte, range, notin, not, in, gt, lt]
	形如：{
		'bid': '1242464682341',
		'createdAt__range': ['2018-07-08 12:12:22', '2018-07-09 13:12:22'],
		'finalMoney__gte': 50,
		'corp_name__contains': u'小',
		'status__in': [1, 3],
	} ===>>>

	{
		'bool': {
			'must': [{
				'term': {
					'bid': '1242464682341'
				}
			}, {
				'range': {
					'created_at': {
						'gte': '2018-07-08 12:12:22',
						'lte': '2018-07-09 13:12:22'
					}
				}
			}, {
				'range': {
					'final_money': {
						'gte': 50
					}
				}
			}, {
				'match': {
					'corp.name': u'小'
				}
			}, {
				'terms': {
					'status': [1, 3]
				}
			}]
		}
	}
*/
func (this *QueryParser) Parse(filters map[string]interface{}) *Query {
	if filters == nil {
		this.query.RemoveKey("bool")
	} else {
		this.query.RemoveKey("match_all")
	}

	nestPath2Query := make(map[string]map[string][]interface{})
	for k, v := range filters {
		realKey := strings.Split(k, "__")[0]
		isNested := len(strings.Split(realKey, ".")) > 1

		if len(strings.Split(k, ">")) > 1 {
			k = strings.Replace(k, ">", ".", -1)
		}

		mustNode, mustNotNode := this.makeQuery(k, v)

		var nestedPath string
		if isNested {
			nestedPath = strings.Split(realKey, ".")[0]
			if _, ok := nestPath2Query[nestedPath]; !ok {
				nestPath2Query[nestedPath] = map[string][]interface{}{
					"must":     make([]interface{}, 0),
					"must_not": make([]interface{}, 0),
				}
			}
		}

		if mustNode != nil {
			if !isNested {
				this.mustArray = append(this.mustArray, mustNode)
			} else {
				nestPath2Query[nestedPath]["must"] = append(nestPath2Query[nestedPath]["must"], mustNode)
			}
		}

		if mustNotNode != nil {
			if !isNested {
				this.mustNotArray = append(this.mustNotArray, mustNotNode)
			} else {
				nestPath2Query[nestedPath]["must_not"] = append(nestPath2Query[nestedPath]["must_not"], mustNotNode)
			}
		}
	}

	for nestPath, nestQuery := range nestPath2Query {
		if len(nestQuery["must"]) > 0 {
			this.mustArray = append(this.mustArray, map[string]interface{}{
				"nested": map[string]interface{}{
					"path": nestPath,
					"query": map[string]interface{}{
						"bool": map[string][]interface{}{
							"must": nestQuery["must"],
						},
					},
				},
			})
		}
		if len(nestQuery["must_not"]) > 0 {
			this.mustNotArray = append(this.mustNotArray, map[string]interface{}{
				"nested": map[string]interface{}{
					"path": nestPath,
					"query": map[string]interface{}{
						"bool": map[string][]interface{}{
							"must_not": nestQuery["must_not"],
						},
					},
				},
			})
		}
	}

	return this.query
}

func (this *QueryParser) makeQuery(k string, v interface{}) (map[string]interface{}, map[string]interface{}) {
	splits := strings.Split(k, "__")
	var mustNode map[string]interface{}
	var mustNotNode map[string]interface{}

	if len(splits) == 1 {
		mustNode = map[string]interface{}{
			"term": map[string]interface{}{
				k: v,
			},
		}
	} else if len(splits) == 2 {
		key := splits[0]
		op := splits[1]
		switch op {
		case "exact":
			mustNode = map[string]interface{}{
				"term": map[string]interface{}{
					key: v,
				},
			}
		case "icontain", "contains", "contain":
			mustNode = map[string]interface{}{
				"match_phrase": map[string]interface{}{
					key: v,
				},
			}
		case "match":
			mustNode = map[string]interface{}{
				"match": map[string]interface{}{
					key: v,
				},
			}
		case "startswith", "start_with":
			mustNode = map[string]interface{}{
				"match_phrase_prefix": map[string]interface{}{
					key: v,
				},
			}
		case "range":
			gte := v.([]interface{})[0]
			lte := v.([]interface{})[1]
			mustNode = map[string]interface{}{
				"range": map[string]map[string]interface{}{
					key: {
						"gte": gte,
						"lte": lte,
					},
				},
			}
		case "regexp":
			mustNode = map[string]interface{}{
				"regexp": map[string]map[string]interface{}{
					key: {
						"value": v,
					},
				},
			}
		case "wildcard":
			mustNode = map[string]interface{}{
				"wildcard": map[string]interface{}{
					key + ".keyword": "*" + v.(string) + "*",
				},
			}
		case "in":
			mustNode = map[string]interface{}{
				"terms": map[string]interface{}{
					key: v.([]interface{}),
				},
			}
		case "lt", "gt", "lte", "gte":
			mustNode = map[string]interface{}{
				"range": map[string]map[string]interface{}{
					key: {
						op: v,
					},
				},
			}
		case "notin":
			mustNotNode = map[string]interface{}{
				"terms": map[string]interface{}{
					key: v,
				},
			}
		case "not":
			mustNotNode = map[string]interface{}{
				"term": map[string]interface{}{
					key: v,
				},
			}
		default:
			beego.Warn("[es:query_parser]: no filter op matched -", op)
		}
	}

	return mustNode, mustNotNode
}

func NewQueryParser() *QueryParser {
	parser := new(QueryParser)
	parser.mustArray = make([]map[string]interface{}, 0)
	parser.mustNotArray = make([]map[string]interface{}, 0)
	parser.query = new(Query)
	parser.query.SetData(map[string]interface{}{
		"match_all": map[string]interface{}{},
		"bool": map[string]interface{}{
			"filter": map[string]interface{}{
				"bool": map[string]interface{}{
					"must":     &parser.mustArray,
					"must_not": &parser.mustNotArray,
				},
			},
		},
	})
	return parser
}
