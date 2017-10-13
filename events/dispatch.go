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
	mapLock sync.RWMutex
	events  map[TaskEvent][]EventListener
}

func NewEventDispatcher() *EventDispatcher {
	return &EventDispatcher{
		events: make(map[TaskEvent][]EventListener),
	}
}

func (ed *EventDispatcher) On(e TaskEvent, l EventListener) {
	ed.mapLock.Lock()
	defer ed.mapLock.Unlock()

	ed.events[e] = append(ed.events[e], l)
}

func (ed *EventDispatcher) Fire(e TaskEvent, c EventContext) error {
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
