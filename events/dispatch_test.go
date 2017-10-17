package events

import (
	"errors"
	"testing"
)

func TestEventDispatcher(t *testing.T) {
	var primaryEvent TaskEvent = "PrimaryEvent"
	var secondaryEvent TaskEvent = "SecondaryEvent"
	var firstListenerExecutions, secondListenerExecutions int

	var firstListener EventListener = func(c EventContext) error {
		firstListenerExecutions++
		return nil
	}

	var secondListener EventListener = func(c EventContext) error {
		secondListenerExecutions++
		return nil
	}

	ed := NewEventDispatcher()
	ed.On(primaryEvent, firstListener)
	ed.On(secondaryEvent, firstListener)
	ed.On(secondaryEvent, secondListener)

	if err := ed.Fire(primaryEvent, EventContext{}); err != nil {
		t.Fatalf("expected no error from dispatcher, but got: %s", err)
	}
	if err := ed.Fire(secondaryEvent, EventContext{}); err != nil {
		t.Fatalf("expected no error from dispatcher, but got: %s", err)
	}

	if firstListenerExecutions != 2 {
		t.Fatalf("expected first listener to be executed 2 times, but got %d", firstListenerExecutions)
	}

	if secondListenerExecutions != 1 {
		t.Fatalf("expected second listener to be executed 1 time, but got %d", secondListenerExecutions)
	}
}

func TestEventDispatcher_Error_ExitEarly(t *testing.T) {
	var someEvent TaskEvent = "SomeEvent"
	var secondListenerExecutions int

	var firstListener EventListener = func(c EventContext) error {
		return errors.New("Some listener error")
	}

	var secondListener EventListener = func(c EventContext) error {
		secondListenerExecutions++
		return nil
	}

	ed := NewEventDispatcher()
	ed.On(someEvent, firstListener)
	ed.On(someEvent, secondListener)

	if err := ed.Fire(someEvent, EventContext{}); err == nil {
		t.Fatal("expected errornous handler to cause a returned error from dispatcher, but got none")
	}

	if secondListenerExecutions != 0 {
		t.Fatalf("expected second listener to not be executed, but got %d executions", secondListenerExecutions)
	}
}
