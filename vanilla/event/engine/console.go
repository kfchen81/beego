package engine

import (
	"fmt"
)

type consoleEngine struct{
	engineType string
}

func newConsoleEngine() *consoleEngine{
	eg := new(consoleEngine)
	eg.engineType = "console"
	return eg
}

func (this *consoleEngine) Send(data map[string]interface{}, tag string){
	eventName := data["_event_name"]
	fmt.Printf("[Event] CONSOLE ENGINE: receive event %s with tag: %s", eventName, tag)
}

func init(){
	eg := newConsoleEngine()
	registerEngine(eg.engineType, eg)
}