package event

import (
	"testing"
)

type testEvent struct {
	SimpleEvent
	Chan chan string
}

func (testEvent) EventType() string {
	return "testEvent"
}

func TestManager_HasEvent(t *testing.T) {
	tests := []struct {
		name string
		args string
		want bool
	}{
		{
			"good entry",
			"test1",
			true,
		},
		{
			"bad entry",
			"doesn't exist",
			false,
		},
		{
			"spaces in entry",
			"spaces are fun",
			true,
		},
		{
			"bad spaces in entry",
			"test string",
			false,
		},
	}
	m := Manager{}
	m.Attach("test1", nil, 20)
	m.Attach("spaces are fun", nil, 20)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.HasEvent(tt.args); got != tt.want {
				t.Errorf("Manager.HasEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_Attach(t *testing.T) {
	tests := []struct {
		name string
		args string
	}{
		{"test", "test"},
		{"test with spaces", "space test"},
	}

	m := Manager{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.Attach(tt.args, nil, -1)
			if !m.HasEvent(tt.args) {
				t.Error("Manager.Attach() did not correctly attach an event: Manager.HasEvent() != true")
			}
		})
	}
}

func TestManager_Detach(t *testing.T) {
	tests := []struct {
		name string
		args int
		want bool
	}{
		{
			"existent hook",
			0,
			true,
		},
		{
			"nonexistant hook",
			4247,
			false,
		},
		{
			"nonzero hooknum",
			1337,
			true,
		},
	}
	m := Manager{
		events: map[string]HandlerList{
			"test1": {
				Handler{
					Func:     nil,
					Priority: 0,
					ID:       0,
				},
			},
			"test2": {
				Handler{
					nil,
					500,
					1337,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.Detach(tt.args); got != tt.want {
				t.Errorf("Manager.Detach() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_Dispatch(t *testing.T) { //nolint:funlen // It contains test data
	tests := []struct {
		name            string
		eventName       string
		expectedResults []string
		handlers        []Handler
	}{
		{
			"single call",
			"test",
			[]string{"test"},
			[]Handler{
				{
					func(event Event) {
						event.(*testEvent).Chan <- "test"
					},
					1,
					0,
				},
			},
		},
		{
			"multiple calls",
			"test",
			[]string{"test 1", "test 2", "test 3", "test 4", "test 5"},
			[]Handler{
				{Func: createTestFunc("test 1")},
				{Func: createTestFunc("test 2")},
				{Func: createTestFunc("test 3")},
				{Func: createTestFunc("test 4")},
				{Func: createTestFunc("test 5")},
			},
		},
		{
			"multiple calls with priority",
			"test",
			[]string{"42", "47", "1337"},
			[]Handler{
				{Func: createTestFunc("47"), Priority: 1},
				{Func: createTestFunc("1337"), Priority: 2},
				{Func: createTestFunc("42"), Priority: 0},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataChan := make(chan string, 10)
			m := createManagerWithEvent(tt.eventName, tt.handlers...)
			m.Dispatch(&testEvent{SimpleEvent{&BaseEvent{Name_: tt.eventName}}, dataChan})
			count := 0

			for s := range dataChan {
				if count+1 > len(tt.expectedResults) {
					t.Errorf(
						"Manager.Dispatch() callbacks returned too much data: %d, want %d",
						count+1,
						len(tt.expectedResults),
					)
					break
				}

				if tt.expectedResults[count] != s {
					t.Errorf(
						"Manager.Dispatch() callback returned invalid data = %q, want %q",
						s,
						tt.expectedResults[count],
					)
					break
				}
				count++
				if count > len(tt.expectedResults)-1 {
					break
				}
			}
		})
	}
}

func TestManager_AttachOneShot(t *testing.T) {
	m := new(Manager)
	callCount := 0

	m.AttachOneShot("test", func(event Event) { callCount++ }, PriHighest)

	for i := 0; i < 5; i++ {
		m.Dispatch(NewSimpleEvent("test"))
	}

	if callCount != 1 {
		t.Errorf("Manager.AttachOneShot based hook called more than once: %d", callCount)
	}
}

func createManagerWithEvent(name string, handlers ...Handler) *Manager {
	m := Manager{}

	for _, h := range handlers {
		m.Attach(name, h.Func, h.Priority)
	}

	return &m
}

func createTestFunc(sendToChan string) HandlerFunc {
	return func(event Event) {
		event.(*testEvent).Chan <- sendToChan
	}
}
