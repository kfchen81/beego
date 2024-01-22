// BatchConsumer 实现了批量同步的向接收端传送数据的功能
package thinkingdata

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type BatchConsumer struct {
	serverUrl string         // 接收端地址
	appId     string         // 项目 APP ID
	Timeout   time.Duration  // 网络请求超时时间, 单位毫秒
	ch        chan batchData // 数据传输信道
	wg        sync.WaitGroup
}

// 内部数据传输信道的数据结构, 内部 Go 程接收并处理
type batchDataType int
type batchData struct {
	t batchDataType // 数据类型
	d Data          // 数据，当 t 为 TYPE_DATA 时有效
}

const (
	DEFAULT_TIME_OUT   = 30000 // 默认超时时长 30 秒
	DEFAULT_BATCH_SIZE = 20    // 默认批量发送条数
	MAX_BATCH_SIZE     = 200   // 最大批量发送条数
	BATCH_CHANNEL_SIZE = 1000  // 数据缓冲区大小, 超过此值时会阻塞

	TYPE_DATA  batchDataType = 0 // 数据类型
	TYPE_FLUSH batchDataType = 1 // 立即发送数据
)

// 创建 BatchConsumer
func NewBatchConsumer(serverUrl string, appId string) (Consumer, error) {
	return initBatchConsumer(serverUrl, appId, DEFAULT_BATCH_SIZE, DEFAULT_TIME_OUT)
}

// 创建指定批量发送条数的 BatchConsumer
// serverUrl 接收端地址
// appId 项目的 APP ID
// batchSize 批量发送条数
func NewBatchConsumerWithBatchSize(serverUrl string, appId string, batchSize int) (Consumer, error) {
	return initBatchConsumer(serverUrl, appId, batchSize, DEFAULT_TIME_OUT)
}

func initBatchConsumer(serverUrl string, appId string, batchSize int, timeout int) (Consumer, error) {
	u, err := url.Parse(serverUrl)
	if err != nil {
		return nil, err
	}
	u.Path = "/logagent"

	if batchSize > MAX_BATCH_SIZE {
		batchSize = MAX_BATCH_SIZE
	}
	c := &BatchConsumer{
		serverUrl: u.String(),
		appId:     appId,
		Timeout:   time.Duration(timeout) * time.Millisecond,
		ch:        make(chan batchData, CHANNEL_SIZE),
	}

	c.wg.Add(1)

	go func() {
		buffer := make([]Data, 0, batchSize)

		defer func() {
			c.wg.Done()
		}()

		for {
			flush := false
			select {
			case rec, ok := <-c.ch:
				if !ok {
					return
				}

				switch rec.t {
				case TYPE_FLUSH:
					flush = true
				case TYPE_DATA:
					buffer = append(buffer, rec.d)
					if len(buffer) >= batchSize {
						flush = true
					}
				}
			}

			// 上传数据到服务端, 不会重试
			if flush && len(buffer) > 0 {
				jdata, err := json.Marshal(buffer)
				if err == nil {
					err = c.send(string(jdata))
					if err != nil {
						fmt.Println(err)
					}
				}
				buffer = buffer[:0]
				flush = false
			}
		}
	}()
	return c, nil
}

func (c *BatchConsumer) Add(d Data) error {
	c.ch <- batchData{
		t: TYPE_DATA,
		d: d,
	}
	return nil
}

func (c *BatchConsumer) Flush() error {
	c.ch <- batchData{
		t: TYPE_FLUSH,
	}
	return nil
}

func (c *BatchConsumer) Close() error {
	c.Flush()
	close(c.ch)
	c.wg.Wait()
	return nil
}

func (c *BatchConsumer) send(data string) error {
	encodedData, err := encodeData(data)
	if err != nil {
		return err
	}
	fmt.Println("------------------", encodedData)
	postData := bytes.NewBufferString(encodedData)

	var resp *http.Response
	req, _ := http.NewRequest("POST", c.serverUrl, postData)
	req.Header["appid"] = []string{c.appId}
	req.Header.Set("user-agent", "ta-go-sdk")
	req.Header.Set("version", "1.0.0")

	client := &http.Client{Timeout: c.Timeout}
	resp, err = client.Do(req)

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
		fmt.Println("==============", result.Code)

		if result.Code != 0 {
			return errors.New(fmt.Sprintf("send to receiver failed with return code: %d", result.Code))
		}
	} else {
		return errors.New(fmt.Sprintf("Unexpected Status Code: %d", resp.StatusCode))
	}
	return nil
}

// Gzip 压缩 + Base64 编码
func encodeData(data string) (string, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)

	_, err := gw.Write([]byte(data))
	if err != nil {
		gw.Close()
		return "", err
	}
	gw.Close()

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
