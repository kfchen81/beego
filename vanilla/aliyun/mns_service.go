package aliyun

import (
	"github.com/kfchen81/beego"
)

type MNSService struct {
}

const endpoint string = "http://1644780993058984.mns.cn-beijing.aliyuncs.com/"

var mnsAccessKeyId = beego.AppConfig.String("system::ACCESS_KEY_ID")
var mnsAccessKeySecret = beego.AppConfig.String("system::ACCESS_KEY_SECRET")
//
//func (this *MNSService) send(response string) bool {
//	jsonObj := new(simplejson.Json)
//	err := jsonObj.UnmarshalJSON([]byte(response))
//	if err != nil {
//		beego.Error(err)
//		return false
//	}
//	beego.Debug(jsonObj)
//
//	code, _ := jsonObj.Get("code").Int()
//	if code != 200 {
//		msg, _ := jsonObj.Get("msg").String()
//		beego.Warn("[aliyun] green image check fail: ", msg)
//		return false
//	}
//
//	var shouldBlock bool = false
//	checkResults := jsonObj.Get("data").MustArray()
//	for _, checkResult := range checkResults {
//		checkResultMap := checkResult.(map[string]interface{})
//		code, _ := checkResultMap["code"].(json.Number).Int64()
//		if code == 200 {
//			sceneResults := checkResultMap["results"].([]interface{})
//			for _, sceneResult := range sceneResults {
//				sceneResultMap := sceneResult.(map[string]interface{})
//				beego.Notice(sceneResultMap)
//				suggestion := sceneResultMap["suggestion"].(string)
//				if suggestion == "block" {
//					shouldBlock = true
//					break
//				}
//
//				rate, _ := sceneResultMap["rate"].(json.Number).Float64()
//				if suggestion == "view" && rate > MAX_RATE {
//					shouldBlock = true
//					break
//				}
//			}
//		}
//	}
//
//	if shouldBlock {
//		return true
//	} else {
//		return false
//	}
//}

func (this *MNSService) Send(message string) (bool, error) {
	return this.SendWithTag([]byte(message), "normal")
}

func (this *MNSService) SendWithTag(message []byte, tag string) (bool, error) {
	//client := mns.NewAliMNSClient(endpoint,
	//	mnsAccessKeyId,
	//	mnsAccessKeySecret)
	//
	//msg := mns.TopicMessageSendRequest{
	//	MessageBody: []byte(message),
	//	MessageTag: tag,
	//}
	//
	//topic := mns.NewMNSTopic("DevTopic", client)
	//_, err := topic.SendMessage(msg)
	//if err != nil {
	//	beego.Error(err)
	//	return false, err
	//} else {
	//	beego.Notice("send message success")
	//	return true, nil
	//}
	panic("not implemented")
}


func NewMNSService() *MNSService {
	return new(MNSService)
}
