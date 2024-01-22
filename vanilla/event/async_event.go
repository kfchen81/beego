package event

import (
	"fmt"
	"github.com/kfchen81/beego"
	"github.com/kfchen81/beego/vanilla/event/engine"
	"time"
)

// 异步消息
type asyncEvent struct{}

func (ae *asyncEvent) Send(event *Event, data map[string]interface{}){
	data["_time"] = time.Now().Format("2006-01-02 15:04:05")
	messageData := map[string]interface{}{
		"_event_name": event.Name,
		"data": data,
	}
	engineType := beego.AppConfig.String("event::ASYNC_EVENT_ENGINE")
	if validEngine, ok := engine.Type2Engine[engineType]; ok{
		validEngine.Send(messageData, event.Tag)
	}else{
		fmt.Printf("[Event] NO ENGINE FOUND")
	}
}

var AsyncEvent *asyncEvent

func init()  {
	AsyncEvent = new(asyncEvent)
}