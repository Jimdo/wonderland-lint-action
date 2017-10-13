package events

import (
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go/service/ecs"
)

type TaskEvent string

type EventContext struct {
	CronName string
	Task     *ecs.Task
}

type EventListener func(c EventContext) error

type EventDispatcher struct {
	mapLock           sync.RWMutex
	listenersByEvents *sync.Map
}

func NewEventDispatcher() *EventDispatcher {
	return &EventDispatcher{
		listenersByEvents: &sync.Map{},
	}
}

func (ed *EventDispatcher) getListenersForEvent(e TaskEvent) []EventListener {
	var listeners []EventListener
	val, ok := ed.listenersByEvents.Load(e)
	if ok {
		listeners = val.([]EventListener)
	} else {
		listeners = []EventListener{}
	}
	return listeners
}

func (ed *EventDispatcher) On(e TaskEvent, l EventListener) {
	listeners := ed.getListenersForEvent(e)
	listeners = append(listeners, l)
	ed.listenersByEvents.Store(e, listeners)
}

func (ed *EventDispatcher) Fire(e TaskEvent, c EventContext) error {
	for _, listener := range ed.getListenersForEvent(e) {
		if err := listener(c); err != nil {
			return fmt.Errorf("error executing listener for event %s: %s", e, err)
		}
	}
	return nil
}
