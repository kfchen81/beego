package event


type Event struct {
	Name string
	Tag string
}

func NewEvent(name, tag string) *Event{
	event := new(Event)
	event.Name = name
	event.Tag = tag
	return event
}

var DEV_TEST *Event

func init(){
	DEV_TEST = NewEvent("dev:test", "any")
}