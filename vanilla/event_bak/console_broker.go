package event_bak

import (
	"github.com/kfchen81/beego"
)

type ConsoleBroker struct {

}

func (this *ConsoleBroker) Send(asyncEvent *AsyncEvent){
	beego.Info(asyncEvent.Name, asyncEvent.Tag, asyncEvent.Data)
	beego.Info(asyncEvent.Name, " sent...")
}

func NewConsoleBroker() *ConsoleBroker{
	return new(ConsoleBroker)
}