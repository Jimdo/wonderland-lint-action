package events

import (
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go/service/ecs"
)

type DispatchableEvent string

type EventListener func(c EventContext) error

// TODO: The context should not have knowledge about ECS tasks
type EventContext struct {
	Target string
	Task   *ecs.Task
}

type EventDispatcher struct {
	mapLock sync.RWMutex
	events  map[DispatchableEvent][]EventListener
}

func NewEventDispatcher() *EventDispatcher {
	return &EventDispatcher{
		events: make(map[DispatchableEvent][]EventListener),
	}
}

func (ed *EventDispatcher) On(e DispatchableEvent, l EventListener) {
	ed.mapLock.Lock()
	defer ed.mapLock.Unlock()

	ed.events[e] = append(ed.events[e], l)
}

func (ed *EventDispatcher) Fire(e DispatchableEvent, c EventContext) error {
	ed.mapLock.RLock()
	defer ed.mapLock.RUnlock()

	if listeners, ok := ed.events[e]; ok {
		for _, listener := range listeners {
			if err := listener(c); err != nil {
				return fmt.Errorf("error executing listener for event %s: %s", e, err)
			}
		}
	}
	return nil
}
