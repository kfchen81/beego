package event

import (
	"encoding/json"
	"fmt"
	"github.com/kfchen81/beego"
	"github.com/kfchen81/beego/logs"
	"github.com/kfchen81/beego/vanilla/aliyun/mns"
	"strings"
)

const _MNS_QUEUE_RECEIVE_MESSAGE_TIMEOUT = 10
const _MNS_QUEUE_MESSAGE_VISIBILITY_TIMEOUT = 30
const _LOG_INTERVAL = 300

type mnsQueueConf struct{
	endpoint string
	accessId string
	accessKey string
	queue string
}

var queueConf *mnsQueueConf

type MessageHandler interface {
	Handle(map[string]interface{}) error
}

var event2handler = make(map[string]MessageHandler)

func handleMessage(queue mns.AliMNSQueue, resp *mns.MessageReceiveResponse) {
	data := make(map[string]interface{})
	messageData := make(map[string]interface{})
	event := "__default__"
	
	defer func(){
		if err := recover(); err!=nil{
			errMsg := fmt.Sprintf("handle event '%v' panic: %v", event, err)
			beego.PushErrorWithExtraDataToSentry(errMsg, map[string]interface{}{
				"message": data["Message"],
			}, nil)
			logs.Critical(err)
		}
	}()
	
	err := json.Unmarshal([]byte(resp.MessageBody), &data)
	if err != nil {
		beego.Error(err)
	}
	
	err = json.Unmarshal([]byte(data["Message"].(string)), &messageData)
	if err != nil {
		beego.Error(err)
	}
	
	event = messageData["_event_name"].(string)
	if ret, err := queue.ChangeMessageVisibility(resp.ReceiptHandle, _MNS_QUEUE_MESSAGE_VISIBILITY_TIMEOUT); err != nil {
		beego.Error(err)
	} else {
		//handle event
		canDeleteMessage := false
		if handler, ok := event2handler[event]; ok {
			err := handler.Handle(messageData)
			if err != nil {
				beego.Error(err)
			} else {
			}
			
			canDeleteMessage = true
		} else {
			beego.Error(fmt.Sprintf("[mns_queue_service] no handler for event '%s'", event))
			canDeleteMessage = true
		}
		
		//beego.Debug("delete it now: ", ret.ReceiptHandle)
		if canDeleteMessage {
			beego.Debug("[mns_queue_service] delete message now: ", ret.ReceiptHandle)
			if err := queue.DeleteMessage(ret.ReceiptHandle); err != nil {
				beego.Error(err)
			}
		}
	}
}

func RegisterEventHandler(event string, handler MessageHandler) {
	event2handler[event] = handler
}

type EventQueueService struct {
}

func NewEventQueueService() *EventQueueService {
	service := new(EventQueueService)
	return service
}

func (this *EventQueueService) Listen() {
	defer func(){
		if err := recover(); err!=nil{
			beego.Error(err)
		}
	}()
	
	client := mns.NewAliMNSClient(queueConf.endpoint, queueConf.accessId, queueConf.accessKey)
	queue := mns.NewMNSQueue(queueConf.queue, client)
	
	messageCount := 0
	fetchCount := 0
	
	respChan := make(chan mns.MessageReceiveResponse)
	errChan := make(chan error)
	go func() {
		defer func(){
			if err := recover(); err!=nil{
				beego.Error(err)
			}
		}()
		
		for {
			select {
			case resp := <-respChan:
				{
					messageCount += 1
					go handleMessage(queue, &resp)
				}
			case err := <-errChan:
				{
					if strings.Contains(err.Error(), "code: MessageNotExist") {
						beego.Debug("no message, continue receive...")
					} else {
						beego.Error(err)
					}
				}
			}
		}
	}()
	
	for {
		fetchCount += 1
		if fetchCount % _LOG_INTERVAL == 0 {
			beego.Warn(fmt.Sprintf("[mns_queue_service] receive for %d times, %d messages", fetchCount, messageCount))
		}
		queue.ReceiveMessage(respChan, errChan, _MNS_QUEUE_RECEIVE_MESSAGE_TIMEOUT)
	}
}

func init() {
	queueConf = new(mnsQueueConf)
	queueConf.accessId = beego.AppConfig.String("aliyun::MNS_ACCESS_ID")
	queueConf.accessKey = beego.AppConfig.String("aliyun::MNS_ACCESS_KEY")
	queueConf.endpoint = beego.AppConfig.String("aliyun::MNS_ENDPOINT")
	queueConf.queue = beego.AppConfig.String("aliyun::MNS_QUEUE")
}
