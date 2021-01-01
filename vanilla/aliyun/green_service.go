package aliyun

import (
	"fmt"
	"github.com/kfchen81/beego/vanilla/aliyun/green"
	"github.com/kfchen81/beego/vanilla/uuid"
	"github.com/bitly/go-simplejson"
	"github.com/kfchen81/beego"
	"encoding/json"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
)

const MAX_RATE = 90.0

type GreenService struct {
}

const seed string = "VpJz55jhJnsX9tcCZcXC"

var enableContentCheck, _ = beego.AppConfig.Bool("system::ENABLE_CONTENT_CHECK")
var accessKeyId = beego.AppConfig.String("system::ACCESS_KEY_ID")
var accessKeySecret = beego.AppConfig.String("system::ACCESS_KEY_SECRET")

func (this *GreenService) shouldBlock(response string) bool {
	jsonObj := new(simplejson.Json)
	err := jsonObj.UnmarshalJSON([]byte(response))
	if err != nil {
		beego.Error(err)
		return false
	}
	beego.Debug(jsonObj)

	code, _ := jsonObj.Get("code").Int()
	if code != 200 {
		msg, _ := jsonObj.Get("msg").String()
		beego.Warn("[aliyun] green image check fail: ", msg)
		return false
	}

	var shouldBlock bool = false
	checkResults := jsonObj.Get("data").MustArray()
	for _, checkResult := range checkResults {
		checkResultMap := checkResult.(map[string]interface{})
		code, _ := checkResultMap["code"].(json.Number).Int64()
		if code == 200 {
			sceneResults := checkResultMap["results"].([]interface{})
			for _, sceneResult := range sceneResults {
				sceneResultMap := sceneResult.(map[string]interface{})
				suggestion := sceneResultMap["suggestion"].(string)
				if suggestion == "block" {
					shouldBlock = true
					break
				}

				rate, _ := sceneResultMap["rate"].(json.Number).Float64()
				if suggestion == "view" && rate > MAX_RATE {
					shouldBlock = true
					break
				}
			}
		}
	}

	if shouldBlock {
		return true
	} else {
		return false
	}
}

func (this *GreenService) ShouldBlockImage(url string) bool {
	if !enableContentCheck {
		return false
	}

	profile := green.Profile{AccessKeyId:accessKeyId, AccessKeySecret:accessKeySecret}
	path := "/green/image/scan";

	clientInfo := green.ClinetInfo{Ip:"127.0.0.1"}
	bizType := "Green"
	scenes := []string{"porn", "terrorism"}

	task := green.Task{DataId:uuid.Rand().Hex(), Url:url}
	tasks := []green.Task{task}

	bizData := green.BizData{ bizType, scenes, tasks}

	var client green.IAliYunClient = green.DefaultClient{Profile:profile}

	resp := client.GetResponse(path, clientInfo, bizData)

	shouldBlock := this.shouldBlock(resp)
	if shouldBlock {
		return true
	}

	return this.ShouldBlockOCRImages([]string{url})
}

func (this *GreenService) ShouldBlockOCRImages(urls []string) bool {
	if !enableContentCheck {
		return false
	}

	profile := green.Profile{AccessKeyId:accessKeyId, AccessKeySecret:accessKeySecret}
	path := "/green/image/scan";

	clientInfo := green.ClinetInfo{Ip:"127.0.0.1"}
	bizType := "Green"
	scenes := []string{"ocr"}

	tasks := make([]green.Task, 0)
	for _, url := range urls {
		task := green.Task{DataId:uuid.Rand().Hex(), Url:url}
		tasks = append(tasks, task)
	}

	bizData := green.BizData{ bizType, scenes, tasks}
	var client green.IAliYunClient = green.DefaultClient{Profile:profile}
	resp := client.GetResponse(path, clientInfo, bizData)

	ocrText := this.getOCRText(resp)
	if ocrText == "" {
		return false
	}

	return this.ShouldBlockText(ocrText)
}

func (this *GreenService) getOCRText(response string) (ocrText string) {
	jsonObj := new(simplejson.Json)
	err := jsonObj.UnmarshalJSON([]byte(response))
	if err != nil {
		beego.Error(err)
		return
	}

	code, _ := jsonObj.Get("code").Int()
	if code != 200 {
		msg, _ := jsonObj.Get("msg").String()
		beego.Warn("[aliyun] get ocr text fail: ", msg)
		return
	}

	respData := jsonObj.Get("data").MustArray()
	for _, data := range respData {
		dataMap := data.(map[string]interface{})
		code, _ := dataMap["code"].(json.Number).Int64()
		if code == 200 {
			results := dataMap["results"].([]interface{})
			for _, result := range results {
				resultMap := result.(map[string]interface{})
				label := resultMap["label"].(string)
				if label == "normal" {
					continue
				}
				ocrLocations := resultMap["ocrLocations"].([]interface{})
				for _, item := range ocrLocations {
					itemMap := item.(map[string]interface{})
					ocrText += itemMap["text"].(string)
				}
			}
		}
	}

	return
}

func (this *GreenService) ShouldBlockImages(urls []string) bool {
	if !enableContentCheck {
		return false
	}

	profile := green.Profile{AccessKeyId:accessKeyId, AccessKeySecret:accessKeySecret}
	path := "/green/image/scan";

	clientInfo := green.ClinetInfo{Ip:"127.0.0.1"}
	bizType := "Green"
	scenes := []string{"porn", "terrorism"}

	tasks := make([]green.Task, 0)

	for _, url := range urls {
		task := green.Task{DataId:uuid.Rand().Hex(), Url:url}
		tasks = append(tasks, task)
	}

	bizData := green.BizData{ bizType, scenes, tasks}

	var client green.IAliYunClient = green.DefaultClient{Profile:profile}

	resp := client.GetResponse(path, clientInfo, bizData)

	shouldBlock := this.shouldBlock(resp)
	if shouldBlock {
		return true
	}

	return this.ShouldBlockOCRImages(urls)
}

func (this *GreenService) ShouldBlockText(content string) bool {
	if !enableContentCheck {
		return false
	}

	profile := green.Profile{AccessKeyId:accessKeyId, AccessKeySecret:accessKeySecret}
	path := "/green/text/scan";

	clientInfo := green.ClinetInfo{Ip:"127.0.0.1"}
	bizType := "Green"
	scenes := []string{"antispam"}

	task := green.Task{DataId:uuid.Rand().Hex(), Content:content}
	tasks := []green.Task{task}

	bizData := green.BizData{ bizType, scenes, tasks}

	var client green.IAliYunClient = green.DefaultClient{Profile:profile}

	resp := client.GetResponse(path, clientInfo, bizData)

	return this.shouldBlock(resp)
}

func (this *GreenService) ShouldBlockVideo(url string) bool {
	if !enableContentCheck {
		return false
	}

	profile := green.Profile{AccessKeyId: accessKeyId, AccessKeySecret: accessKeySecret}
	path := "/green/image/scan"

	clientInfo := green.ClinetInfo{Ip: "127.0.0.1"}
	bizType := "Green"
	scenes := []string{"porn"}
	times := []string{"1000", "5000", "10000", "15000", "20000"}
	imageUrls := make([]string, 0)
	for _, time := range times {
		imageUrls = append(imageUrls, fmt.Sprintf("%s?x-oss-process=video/snapshot,t_%s,f_jpg,w_0,h_0,m_fast", url, time))
		beego.Notice("检测图片:", fmt.Sprintf("%s?x-oss-process=video/snapshot,t_%s,f_jpg,w_0,h_0,m_fast", url, time))
	}

	tasks := make([]green.Task, 0)
	for _, url := range imageUrls {
		task := green.Task{DataId: uuid.Rand().Hex(), Url: url}
		tasks = append(tasks, task)
	}

	bizData := green.BizData{bizType, scenes, tasks}

	var client green.IAliYunClient = green.DefaultClient{Profile: profile}

	resp := client.GetResponse(path, clientInfo, bizData)

	return this.shouldBlock(resp)
}

func (this *GreenService) SubmitVoiceCheckTask(url string) string {
	if !enableContentCheck {
		return "pass"
	}

	client, err := sdk.NewClientWithAccessKey("cn-shanghai", accessKeyId, accessKeySecret)
	if err != nil {
		beego.Error("创建aliyun_client失败", err)
		return ""
	}

	request := requests.NewCommonRequest()
	request.Method = "POST"
	request.Domain = "green.cn-shanghai.aliyuncs.com"
	request.Version = "2018-05-09"
	request.PathPattern = "/green/voice/asyncscan"

	scenes := []string{"antispam"}
	task := green.Task{Url: url, Type: "file"}
	tasks := []green.Task{task}
	callback := "https://api.vxiaocheng.com/honeycomb/green/voice_scan_result"

	data := green.VoiceTask{scenes, callback, seed, tasks}
	dataObj, _ := json.Marshal(data)
	body := string(dataObj)
	request.Content = []byte(body)

	response, err := client.ProcessCommonRequest(request)
	if err != nil {
		beego.Error("提交声音检测任务失败", err)
		return ""
	}

	respString := response.GetHttpContentString()
	respJson, _ := simplejson.NewJson([]byte(respString))
	code, _ := respJson.Get("code").Int()
	errData, _ := respJson.Get("data").Array()
	dataMap := errData[0].(map[string]interface{})
	if code != 200 {
		errMsg := dataMap["msg"].(string)
		beego.Error("提交声音检测任务失败", errMsg)
		return ""
	} else {
		taskId := dataMap["taskId"].(string)
		return taskId
	}
}

func (this *GreenService) ShouldBlockVoice(response string) (bool, string) {
	jsonObj := new(simplejson.Json)
	_ = jsonObj.UnmarshalJSON([]byte(response))

	code, _ := jsonObj.Get("code").Int()
	taskId, _ := jsonObj.Get("taskId").String()
	if code != 200 {
		msg, _ := jsonObj.Get("msg").String()
		beego.Warn("[aliyun] green voice check fail: ", msg)
		return false, taskId
	}

	var shouldBlock bool
	results := jsonObj.Get("results").MustArray()
	for _, result := range results {
		resMap := result.(map[string]interface{})
		suggestion := resMap["suggestion"].(string)
		if suggestion == "block" {
			shouldBlock = true
			break
		}

		rate, _ := resMap["rate"].(json.Number).Float64()
		if suggestion == "review" && rate > MAX_RATE {
			shouldBlock = true
			break
		}
	}

	if shouldBlock {
		return true, taskId
	} else {
		return false, taskId
	}
}

func NewGreenService() *GreenService {
	return new(GreenService)
}
