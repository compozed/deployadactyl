package mocks

import (
	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
)

// EventManager handmade mock for tests.
type EventManager struct {
	AddHandlerCall struct {
		Received struct {
			Handler   I.Handler
			EventType string
		}
		Returns struct {
			Error error
		}
	}
	EmitCall struct {
		TimesCalled int
		Received    struct {
			Events []S.Event
		}
		Returns struct {
			Error []error
		}
	}
}

// AddHandler mock method.
func (e *EventManager) AddHandler(handler I.Handler, eventType string) error {
	e.AddHandlerCall.Received.Handler = handler
	e.AddHandlerCall.Received.EventType = eventType

	return e.AddHandlerCall.Returns.Error
}

// Emit mock method.
func (e *EventManager) Emit(event S.Event) error {
	defer func() { e.EmitCall.TimesCalled++ }()

	e.EmitCall.Received.Events = append(e.EmitCall.Received.Events, event)

	if len(e.EmitCall.Returns.Error) > e.EmitCall.TimesCalled {
		return e.EmitCall.Returns.Error[e.EmitCall.TimesCalled]
	}

	return nil
}
