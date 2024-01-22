package event_bak

import (
	"github.com/kfchen81/beego"
)

type AsyncEventService struct{

}

var mode2broker = map[string]IBroker{
	"dev": NewConsoleBroker(),
	"prod": NewMNSBroker(),
}

func (this *AsyncEventService) Send(asyncEvent *AsyncEvent, data map[string]interface{}){
	asyncEvent.Data = data
	broker := mode2broker[beego.BConfig.RunMode]
	broker.Send(asyncEvent)
}

func NewAsyncEventService() *AsyncEventService{
	return new(AsyncEventService)
}