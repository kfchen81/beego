package event_bak

type AsyncEvent struct{
	Name string
	Tag string
	Data map[string]interface{}
}

func NewAsyncEvent(name string, tag string) *AsyncEvent{
	instance := new(AsyncEvent)
	instance.Name = name
	instance.Tag = tag
	return instance
}