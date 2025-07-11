// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package orm

import (
	"context"
	"fmt"
	"github.com/kfchen81/beego"
	"github.com/kfchen81/beego/logs"
	"github.com/kfchen81/beego/metrics"
)

var _ENABLE_DB_ACCESS_TRACE bool = false
var _ENABLE_DB_READ_CHECK bool = false
var _ENABLE_SQL_RESOURCE_COMMENT = false

type colValue struct {
	value int64
	opt   operator
}

type colFloatValue struct {
	value float64
	opt   operator
}

type operator int

// define Col operations
const (
	ColAdd operator = iota
	ColMinus
	ColMultiply
	ColExcept
)

// ColValue do the field raw changes. e.g Nums = Nums + 10. usage:
//
//	Params{
//		"Nums": ColValue(Col_Add, 10),
//	}
func ColValue(opt operator, value interface{}) interface{} {
	switch opt {
	case ColAdd, ColMinus, ColMultiply, ColExcept:
	default:
		panic(fmt.Errorf("orm.ColValue wrong operator"))
	}
	v, err := StrTo(ToStr(value)).Int64()
	if err != nil {
		panic(fmt.Errorf("orm.ColValue doesn't support non string/numeric type, %s", err))
	}
	var val colValue
	val.value = v
	val.opt = opt
	return val
}

func ColFloatValue(opt operator, value interface{}) interface{} {
	switch opt {
	case ColAdd, ColMinus, ColMultiply, ColExcept:
	default:
		panic(fmt.Errorf("orm.ColFloatValue wrong operator"))
	}
	v, err := StrTo(ToStr(value)).Float64()
	if err != nil {
		panic(fmt.Errorf("orm.ColFloatValue doesn't support non string/numeric type, %s", err))
	}
	var val colFloatValue
	val.value = v
	val.opt = opt
	return val
}

// real query struct
type querySet struct {
	mi         *modelInfo
	cond       *Condition
	related    []string
	relDepth   int
	limit      int64
	offset     int64
	groups     []string
	orders     []string
	distinct   bool
	forupdate  bool
	orm        *orm
	ctx        context.Context
	forContext bool
	index      string // 索引
}

var _ QuerySeter = new(querySet)

// add condition expression to QuerySeter.
func (o querySet) Filter(query interface{}, args ...interface{}) QuerySeter {
	if o.cond == nil {
		o.cond = NewCondition()
	}

	if len(args) > 0 {
		expr := query.(string)
		o.cond = o.cond.And(expr, args...)
	} else {
		conditions := query.(map[string]interface{})
		for k, v := range conditions {
			o.cond = o.cond.And(k, v)
		}
	}
	return &o
}

// add raw sql to querySeter.
func (o querySet) FilterRaw(expr string, sql string) QuerySeter {
	if o.cond == nil {
		o.cond = NewCondition()
	}
	o.cond = o.cond.Raw(expr, sql)
	return &o
}

// add NOT condition to querySeter.
func (o querySet) Exclude(expr string, args ...interface{}) QuerySeter {
	if o.cond == nil {
		o.cond = NewCondition()
	}
	o.cond = o.cond.AndNot(expr, args...)
	return &o
}

// set offset number
func (o *querySet) setOffset(num interface{}) {
	o.offset = ToInt64(num)
}

// add LIMIT value.
// args[0] means offset, e.g. LIMIT num,offset.
func (o querySet) Limit(limit interface{}, args ...interface{}) QuerySeter {
	o.limit = ToInt64(limit)
	if len(args) > 0 {
		o.setOffset(args[0])
	}
	return &o
}

// add OFFSET value
func (o querySet) Offset(offset interface{}) QuerySeter {
	o.setOffset(offset)
	return &o
}

// add GROUP expression
func (o querySet) GroupBy(exprs ...string) QuerySeter {
	o.groups = exprs
	return &o
}

// add ORDER expression.
// "column" means ASC, "-column" means DESC.
func (o querySet) OrderBy(exprs ...string) QuerySeter {
	o.orders = exprs
	return &o
}

// add DISTINCT to SELECT
func (o querySet) Distinct() QuerySeter {
	o.distinct = true
	return &o
}

// add FOR UPDATE to SELECT
func (o querySet) ForUpdate() QuerySeter {
	o.forupdate = true
	return &o
}

// set relation model to query together.
// it will query relation models and assign to parent model.
func (o querySet) RelatedSel(params ...interface{}) QuerySeter {
	if len(params) == 0 {
		o.relDepth = DefaultRelsDepth
	} else {
		for _, p := range params {
			switch val := p.(type) {
			case string:
				o.related = append(o.related, val)
			case int:
				o.relDepth = val
			default:
				panic(fmt.Errorf("<QuerySeter.RelatedSel> wrong param kind: %v", val))
			}
		}
	}
	return &o
}

// set condition to QuerySeter.
func (o querySet) SetCond(cond *Condition) QuerySeter {
	o.cond = cond
	return &o
}

// get condition from QuerySeter
func (o querySet) GetCond() *Condition {
	return o.cond
}

func (o querySet) recordMetrics(dbMethod string) {
	dbType := o.orm.alias.Name
	o.recordMetricsWithDbType(dbMethod, dbType)
}

func (o querySet) recordMetricsWithDbType(dbMethod string, dbType string) {
	localResource := o.orm.GetData("SOURCE_RESOURCE")
	localMethod := o.orm.GetData("SOURCE_METHOD")
	if _ENABLE_DB_ACCESS_TRACE {
		metrics.GetDBTableAccessCounter().WithLabelValues(localMethod, localResource, dbType, dbMethod, o.mi.table).Inc()
	}

	if _ENABLE_SQL_RESOURCE_COMMENT {
		txid := o.orm.txid
		switch o.orm.db.(type) {
		case *dbQueryTracable:
			comment := fmt.Sprintf("/* %s:%s:%s */", localMethod, localResource, txid)
			o.orm.db.(*dbQueryTracable).SetComment(comment)
		default:

		}
	}
}

// return QuerySeter execution result number
func (o *querySet) Count() (int64, error) {
	partnerSlave := o.orm.partnerSlave
	if partnerSlave != nil {
		slaveOrm := partnerSlave.(*orm)
		o.recordMetricsWithDbType("COUNT", slaveOrm.alias.Name)
		logs.Notice("[orm-25] use partner slave : ", slaveOrm.alias.Name)
		return o.orm.alias.DbBaser.Count(slaveOrm.db, o, o.mi, o.cond, o.orm.alias.TZ)
	} else {
		o.recordMetrics("COUNT")
		return o.orm.alias.DbBaser.Count(o.orm.db, o, o.mi, o.cond, o.orm.alias.TZ)
	}
}

// check result empty or not after QuerySeter executed
func (o *querySet) Exist() bool {
	o.recordMetrics("COUNT")
	cnt, _ := o.orm.alias.DbBaser.Count(o.orm.db, o, o.mi, o.cond, o.orm.alias.TZ)
	return cnt > 0
}

// execute update with parameters
func (o *querySet) Update(values Params) (int64, error) {
	o.recordMetrics("UPDATE")
	return o.orm.alias.DbBaser.UpdateBatch(o.orm.db, o, o.mi, o.cond, values, o.orm.alias.TZ)
}

// execute delete
func (o *querySet) Delete() (int64, error) {
	o.recordMetrics("DELETE")
	return o.orm.alias.DbBaser.DeleteBatch(o.orm.db, o, o.mi, o.cond, o.orm.alias.TZ)
}

// return a insert queryer.
// it can be used in times.
// example:
//
//	i,err := sq.PrepareInsert()
//	i.Add(&user1{},&user2{})
func (o *querySet) PrepareInsert() (Inserter, error) {
	return newInsertSet(o.orm, o.mi)
}

// query all data and map to containers.
// cols means the columns when querying.
func (o *querySet) All(container interface{}, cols ...string) (int64, error) {
	partnerSlave := o.orm.partnerSlave
	if partnerSlave != nil {
		slaveOrm := partnerSlave.(*orm)
		o.recordMetricsWithDbType("SELECT", slaveOrm.alias.Name)
		logs.Notice("[orm-25] use partner slave : ", slaveOrm.alias.Name)
		cnt, err := o.orm.alias.DbBaser.ReadBatch(slaveOrm.db, o, o.mi, o.cond, container, o.orm.alias.TZ, cols)

		return cnt, err
	} else {
		o.recordMetrics("SELECT")
		cnt, err := o.orm.alias.DbBaser.ReadBatch(o.orm.db, o, o.mi, o.cond, container, o.orm.alias.TZ, cols)

		return cnt, err
	}
}

// query one row data and map to containers.
// cols means the columns when querying.
func (o *querySet) One(container interface{}, cols ...string) error {
	o.limit = 1

	var num int64 = 0
	var err error
	partnerSlave := o.orm.partnerSlave
	if partnerSlave != nil {
		slaveOrm := partnerSlave.(*orm)
		o.recordMetricsWithDbType("SELECT", slaveOrm.alias.Name)
		logs.Notice("[orm-25] use partner slave : ", slaveOrm.alias.Name)
		num, err = o.orm.alias.DbBaser.ReadBatch(slaveOrm.db, o, o.mi, o.cond, container, o.orm.alias.TZ, cols)
	} else {
		o.recordMetrics("SELECT")
		num, err = o.orm.alias.DbBaser.ReadBatch(o.orm.db, o, o.mi, o.cond, container, o.orm.alias.TZ, cols)
	}

	if err != nil {
		return err
	}
	if num == 0 {
		return ErrNoRows
	}

	if num > 1 {
		return ErrMultiRows
	}
	return nil
}

// query all data and map to []map[string]interface.
// expres means condition expression.
// it converts data to []map[column]value.
func (o *querySet) Values(results *[]Params, exprs ...string) (int64, error) {
	if o.orm.partnerSlave != nil {
		slaveOrm := o.orm.partnerSlave.(*orm)
		o.recordMetricsWithDbType("SELECT", slaveOrm.alias.Name)
		logs.Notice("[orm-25] use partner slave : ", slaveOrm.alias.Name)
		return o.orm.alias.DbBaser.ReadValues(slaveOrm.db, o, o.mi, o.cond, exprs, results, o.orm.alias.TZ)
	} else {
		o.recordMetrics("SELECT")
		return o.orm.alias.DbBaser.ReadValues(o.orm.db, o, o.mi, o.cond, exprs, results, o.orm.alias.TZ)
	}
}

// query all data and map to [][]interface
// it converts data to [][column_index]value
func (o *querySet) ValuesList(results *[]ParamsList, exprs ...string) (int64, error) {
	if o.orm.partnerSlave != nil {
		slaveOrm := o.orm.partnerSlave.(*orm)
		o.recordMetricsWithDbType("SELECT", slaveOrm.alias.Name)
		logs.Notice("[orm-25] use partner slave : ", slaveOrm.alias.Name)
		return o.orm.alias.DbBaser.ReadValues(slaveOrm.db, o, o.mi, o.cond, exprs, results, o.orm.alias.TZ)
	} else {
		o.recordMetrics("SELECT")
		return o.orm.alias.DbBaser.ReadValues(o.orm.db, o, o.mi, o.cond, exprs, results, o.orm.alias.TZ)
	}
}

// query all data and map to []interface.
// it's designed for one row record set, auto change to []value, not [][column]value.
func (o *querySet) ValuesFlat(result *ParamsList, expr string) (int64, error) {
	if o.orm.partnerSlave != nil {
		slaveOrm := o.orm.partnerSlave.(*orm)
		o.recordMetricsWithDbType("SELECT", slaveOrm.alias.Name)
		logs.Notice("[orm-25] use partner slave : ", slaveOrm.alias.Name)
		return o.orm.alias.DbBaser.ReadValues(slaveOrm.db, o, o.mi, o.cond, []string{expr}, result, o.orm.alias.TZ)
	} else {
		o.recordMetrics("SELECT")
		return o.orm.alias.DbBaser.ReadValues(o.orm.db, o, o.mi, o.cond, []string{expr}, result, o.orm.alias.TZ)
	}
}

// query all rows into map[string]interface with specify key and value column name.
// keyCol = "name", valueCol = "value"
// table data
// name  | value
// total | 100
// found | 200
//
//	to map[string]interface{}{
//		"total": 100,
//		"found": 200,
//	}
func (o *querySet) RowsToMap(result *Params, keyCol, valueCol string) (int64, error) {
	panic(ErrNotImplement)
}

// query all rows into struct with specify key and value column name.
// keyCol = "name", valueCol = "value"
// table data
// name  | value
// total | 100
// found | 200
//
//	to struct {
//		Total int
//		Found int
//	}
func (o *querySet) RowsToStruct(ptrStruct interface{}, keyCol, valueCol string) (int64, error) {
	panic(ErrNotImplement)
}

func (o querySet) UseIndex(index string) QuerySeter {
	o.index = index
	return &o
}

func (o querySet) UseTableShard(suffix string) QuerySeter {
	o.mi.table = fmt.Sprintf("%s_%s", o.mi.origTable, suffix)
	o.mi.shard = true
	logs.Notice("[merge_orm] switch table_shard: ", o.mi.table)
	return &o
}

func (o querySet) RestoreTable() QuerySeter {
	o.mi.table = o.mi.origTable
	o.mi.shard = false
	logs.Notice("[merge_orm] restore table: ", o.mi.table)
	return &o
}

func (o querySet) UseShardModelInfo() QuerySeter {
	newMi := modelInfo{
		pkg:         o.mi.pkg,
		name:        o.mi.name,
		fullName:    o.mi.fullName,
		table:       o.mi.table,
		origTable:   o.mi.origTable,
		tableShards: o.mi.tableShards,
		model:       o.mi.model,
		fields:      o.mi.fields,
		manual:      o.mi.manual,
		addrField:   o.mi.addrField,
		uniques:     o.mi.uniques,
		isThrough:   o.mi.isThrough,
		shard:       o.mi.shard,
	}
	o.mi = &newMi
	logs.Notice("[merge_orm] create new modelInfo as shard modelInfo")
	return &o
}

func (o querySet) GetTableShards() []string {
	return o.mi.tableShards
}

func (o querySet) GetModelInfoTableName() string {
	return o.mi.table
}

// set context to QuerySeter.
func (o querySet) WithContext(ctx context.Context) QuerySeter {
	o.ctx = ctx
	o.forContext = true
	return &o
}

// create new QuerySeter.
func newQuerySet(orm *orm, mi *modelInfo) QuerySeter {
	o := new(querySet)
	o.mi = mi
	o.orm = orm
	return o
}

func init() {
	_ENABLE_DB_ACCESS_TRACE = beego.AppConfig.DefaultBool("system::ENABLE_DB_ACCESS_TRACE", false)
	beego.Info("[init] use _ENABLE_DB_ACCESS_TRACE: ", _ENABLE_DB_ACCESS_TRACE)

	_ENABLE_DB_READ_CHECK = beego.AppConfig.DefaultBool("system::ENABLE_DB_READ_CHECK", false)
	beego.Info("[init] use _ENABLE_DB_READ_CHECK: ", _ENABLE_DB_READ_CHECK)

	_ENABLE_SQL_RESOURCE_COMMENT = beego.AppConfig.DefaultBool("system::ENABLE_SQL_RESOURCE_COMMENT", false)
	beego.Info("[init] use _ENABLE_SQL_RESOURCE_COMMENT: ", _ENABLE_SQL_RESOURCE_COMMENT)
}
