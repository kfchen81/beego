package event_bak

type IBroker interface {
	Send(asyncEvent *AsyncEvent)
}