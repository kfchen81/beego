package event_bak

import (
	"encoding/json"
	"github.com/kfchen81/beego"
	"github.com/kfchen81/beego/vanilla/aliyun"
)

type MNSBroker struct {

}

func (this *MNSBroker) Send(asyncEvent *AsyncEvent){
	beego.Info("sending event_bak: ", asyncEvent.Name)

	messageData := asyncEvent.Data
	jsonBytes, err := json.Marshal(messageData)

	if err != nil {
		beego.Error(err)
	}else{
		aliyun.NewMNSService().SendWithTag(jsonBytes, asyncEvent.Tag)
	}
}

func NewMNSBroker() *MNSBroker{
	return new(MNSBroker)
}