/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package testing

import (
	"context"
	"strconv"
	"time"

	"github.com/cloudevents/sdk-go/pkg/cloudevents"
	ce "github.com/cloudevents/sdk-go/pkg/cloudevents"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/transport"
	"go.uber.org/atomic"
)

// Generate readable sequential IDs for tests.
var idCounter atomic.Int32

// MakeEvent sets all required event values to make a valid event for test purposes.
func MakeEvent(data interface{}) cloudevents.Event {
	e := cloudevents.New("0.2")
	e.SetType("test.event.type")
	e.SetSource("test/event/source")
	e.SetID(strconv.Itoa(int(idCounter.Add(1))))
	e.SetTime(time.Now())
	e.SetDataContentType("application/json")
	if err := e.SetData(data); err != nil {
		panic(err)
	}
	return e
}

// MockReceiver is a channel that implements transport.Receiver by receiving to the channel.
type MockReceiver chan ce.Event

// Receive an event to a MockReceiver channel
func (r MockReceiver) Receive(ctx context.Context, e ce.Event, er *ce.EventResponse) error {
	r <- e
	return nil
}

// MockTransport is a dummy in-memory transport that sends and receives events from
// a channel.
type MockTransport struct {
	// Sent receives events from MockTransport.Send()
	Sent chan ce.Event
	// Received receives events to be passed to the Receiver.
	Received chan ce.Event
	Receiver transport.Receiver
}

// Send puts the event on t.Sent
func (t *MockTransport) Send(ctx context.Context, e ce.Event) (*ce.Event, error) {
	t.Sent <- e
	return nil, nil
}

// SetReceiver sets the receiver
func (t *MockTransport) SetReceiver(r transport.Receiver) { t.Receiver = r }

// StartReceiver starts processing t.Received by calling  t.Receiver.Received
func (t *MockTransport) StartReceiver(ctx context.Context) error {
	// Only process Received there is a Receiver
	var recv chan ce.Event
	if t.Receiver != nil {
		recv = t.Received
	}
	for {
		select {
		case e, ok := <-recv:
			if !ok {
				return nil
			} else if err := t.Receiver.Receive(ctx, e, nil); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// MockTransportFactory creates a Transport with the given capacity.
type MockTransportFactory struct {
	Capacity int
}

// New makes a MockTransport
func (f *MockTransportFactory) New() (transport.Transport, error) {
	return &MockTransport{
		Sent:     make(chan ce.Event, f.Capacity),
		Received: make(chan ce.Event, f.Capacity),
	}, nil
}
