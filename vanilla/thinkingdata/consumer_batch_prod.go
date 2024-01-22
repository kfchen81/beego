package thinkingdata

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kfchen81/beego"
	"github.com/kfchen81/beego/metrics"
	"io/ioutil"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DEFAULT_PROD_TIME_OUT   = 10 // 默认超时时长 10 秒
	DEFAULT_PROD_BATCH_SIZE = 20    // 默认批量发送条数
	MAX_PROD_BATCH_SIZE     = 200   // 最大批量发送条数
	PROD_CHANNEL_SIZE		= 1000  // 数据通道缓冲
	DEFAULT_CONSUMER_COUNT  = 3		// 默认消费者数量
)

var taAnalyser TDAnalytics
var taSwitchOn bool

var once sync.Once
var prodBatchConsumer *ProdBatchConsumer

type ProdBatchConsumer struct {
	serverUrl string         // 接收端地址
	appId     string         // 项目 APP ID
	Timeout   time.Duration  // 网络请求超时时间, 单位毫秒
	ch        chan *batchData // 数据传输信道

	batchSize int
	consumerCount int // 消费者数量
	tmpConsumerCount int32 // 临时线程数
}

// runConsumer
// 每个消费者持有一个计时器，实现每隔1小时将buffer中的数据推到服务端
func (this *ProdBatchConsumer) runConsumer(tmp bool){
	metrics.GetTaConsumerCounter().Inc()
	ticker := time.NewTicker(time.Hour) //计时器
	go func() {
		buffer := make([]Data, 0, this.batchSize)
		flush := func() {
			if len(buffer) > 0{
				this.push(buffer)
				buffer = buffer[:0]
			}
		}

		go func() {
			for _ = range ticker.C{
				beego.Info("tok...")
				if len(buffer) > 0{
					this.ch <- &batchData{
						t: TYPE_FLUSH,
						d: Data{},
					}
				}
			}
		}()

		defer func() {
			ticker.Stop() // 停止计时器
			if tmp{
				atomic.AddInt32(&this.tmpConsumerCount, -1)
				metrics.GetTaConsumerCounter().Dec()
			}

			if err := recover(); err != nil{
				beego.Error(err)
				if !tmp{ // 临时线程不会被重启
					this.runConsumer(tmp)
				}
			}
		}()

		for {
			bData, ok := <- this.ch
			if !ok {
				// 此时channel已关闭
				if len(buffer) > 0{
					flush()
				}
				return
			}

			switch bData.t {
			case TYPE_DATA:
				buffer = append(buffer, bData.d)
				if len(buffer) < this.batchSize {
					continue
				}
				fallthrough
			case TYPE_FLUSH:
				flush()
				if tmp{
					// 临时线程在完成一次push后即退出
					return
				}
			}
		}
	}()
}

func (this *ProdBatchConsumer)run(){
	beego.Info("[ta]: consumer running...")
	for i:=0; i<this.consumerCount; i++{
		this.runConsumer(false)
	}
}

func (this *ProdBatchConsumer) send(dataStr string) error{
	buffer := bytes.NewBufferString(dataStr)
	var resp *http.Response
	req, _ := http.NewRequest("POST", this.serverUrl, buffer)
	req.Header["appid"] = []string{this.appId}
	req.Header.Set("user-agent", "ta-go-sdk")
	req.Header.Set("version", "1.0.0")

	client := &http.Client{Timeout: this.Timeout}
	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		var result struct {
			Code int
		}

		err = json.Unmarshal(body, &result)
		if err != nil {
			return err
		}

		if result.Code != 0 {
			return errors.New(fmt.Sprintf("send to receiver failed with return code: %d", result.Code))
		}
	} else {
		return errors.New(fmt.Sprintf("Unexpected Status Code: %d", resp.StatusCode))
	}
	return nil
}

// push 上传数据到服务端
// 数据错误：丢弃数据
// 请求错误：重试3次
func (this *ProdBatchConsumer) push(datas []Data) error{
	startTime := time.Now()
	encodedData, err := this.encodeData(datas)
	if err != nil {
		return err
	}
	for i:=0; i<3; i++{
		err = this.send(encodedData)
		metrics.GetTaServerPushCounter().WithLabelValues("push").Inc()
		if err != nil{
			metrics.GetTaServerPushCounter().WithLabelValues("push_failed").Inc()
		}else{
			break
		}
	}
	if err != nil{
		beego.Error(err)
	}
	timeDur := time.Since(startTime)
	metrics.GetTaServerPushTimer().Observe(timeDur.Seconds())
	return err
}

// Gzip 压缩 + Base64 编码
func (this *ProdBatchConsumer) encodeData(datas []Data) (string, error) {
	jdata, err := json.Marshal(datas)
	if err != nil{
		return "", err
	}

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)

	_, err = gw.Write(jdata)
	if err != nil {
		gw.Close()
		return "", err
	}
	gw.Close()
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func (this *ProdBatchConsumer) Add(d Data) error {
	if beego.BConfig.RunMode == "dev"{
		return nil
	}
	select {
	case this.ch <- &batchData{
		t: TYPE_DATA,
		d: d,
	}:
		metrics.GetTaTracedDataCounter().Inc()
	default:
		beego.Warn("[ta]: channel is full")
		metrics.GetTaChannelIsFullCounter().Inc()
		// 信道满时策略
		// 1、新增临时线程处理，临时线程数不超过 2*当前持久consumer数
		// 2、持久线程已达上限，抛弃数据并启动新的consumer
		// 注意，当多个协程同时达到default逻辑块时，以下逻辑会使得只有一个协程能启动新的consumer，
		// 		结果就是其他协程的消息丢失
		if tcc := atomic.LoadInt32(&this.tmpConsumerCount); tcc < 2 * int32(this.consumerCount){
			if atomic.CompareAndSwapInt32(&this.tmpConsumerCount, tcc, tcc + 1){
				this.runConsumer(true)
			}
		}
	}
	return nil
}

// Flush 不支持flush
func (this *ProdBatchConsumer) Flush() error {
	return nil
}

func (this *ProdBatchConsumer) Close() error {
	if beego.BConfig.RunMode == "dev"{
		return nil
	}
	close(this.ch)
	return nil
}

// NewProdBatchConsumer 创建consumer, 单例模式
// args:
//		0: batchSize
//		1: consumerCount
//		2: timeout
//
// 生产级别，生产-消费者模型
// 1、？确保内存中数据能够被推送到服务端
// 		服务端如果失去响应，客户端消息就会堆积并最终被抛弃
// 		在数据量非常大的情况下，临时消费者也来不及消费时，数据也会被抛弃
// 	 因此最终的解决方案是将溢出的数据推送到额外存储介质中，再使用一个线程去消费这个介质中的数据
// 2、消费者出错能自动重启，保证持久消费者数量
// 3、消费能力伸缩，使用临时消费者处理溢出的数据，处理完后即关闭
// 4、指标监控
//		消息数
//		推送次数
//		推送失败次数
//		消费者数量
//		推送耗时

func GetProdBatchConsumer(serverUrl string, appId string, args ...int) Consumer {
	if beego.BConfig.RunMode == "dev"{
		return &ProdBatchConsumer{}
	}
	once.Do(func() {
		batchSize := DEFAULT_PROD_BATCH_SIZE
		timeout := DEFAULT_PROD_TIME_OUT
		consumerCount := DEFAULT_CONSUMER_COUNT
		l := len(args)
		if l >= 1{
			batchSize = args[0]
			if batchSize > MAX_PROD_BATCH_SIZE{
				batchSize = MAX_PROD_BATCH_SIZE
			}
		}
		if l >= 2{
			consumerCount = args[1]
		}
		if l >= 3{
			timeout = args[2]
		}
		prodBatchConsumer = &ProdBatchConsumer{
			serverUrl: fmt.Sprintf("%s/logagent", serverUrl),
			appId:     appId,
			Timeout:   time.Duration(timeout) * time.Second,
			ch:        make(chan *batchData, PROD_CHANNEL_SIZE),
			batchSize: batchSize,
			consumerCount: consumerCount,
		}
		atomic.StoreInt32(&prodBatchConsumer.tmpConsumerCount, 0)
		prodBatchConsumer.run()
	})
	return prodBatchConsumer
}

func GetTaAnalyst(args ...int) TDAnalytics{
	host := beego.AppConfig.DefaultString("ta::TA_HOST", "")
	appid := beego.AppConfig.DefaultString("ta::TA_APPID", "")
	consumer := GetProdBatchConsumer(host, appid, args...)
	return New(consumer)
}

func Track(eventName, accountId, distinctId string, data map[string]interface{}){
	if taSwitchOn{
		err := taAnalyser.Track(accountId, distinctId, eventName, data)
		if err != nil{
			beego.Error(err)
		}
	}else{
		beego.Info("ta_analyze is closed")
	}
}

func init(){
	bufferSize := beego.AppConfig.DefaultInt("ta::TA_BUFFER_SIZE", DEFAULT_PROD_BATCH_SIZE)
	consumerCount := beego.AppConfig.DefaultInt("ta::TA_CONSUMER_COUNT", DEFAULT_CONSUMER_COUNT)
	beego.Info(fmt.Sprintf("init ta in %s mode with %d buffer_size and %d consumers...", beego.BConfig.RunMode, bufferSize, consumerCount))
	taAnalyser = GetTaAnalyst(bufferSize, consumerCount)
	taSwitchOn = beego.AppConfig.DefaultString("ta::TA_SWITCH", "off") == "on" // 功能开关
}