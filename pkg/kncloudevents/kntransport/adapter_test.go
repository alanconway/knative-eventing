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

package kntransport

import (
	"context"
	"reflect"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go"
	kntest "github.com/knative/eventing/pkg/kncloudevents/testing"
)

func checkFatal(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func roundTrip(t *testing.T, a *Adapter, e cloudevents.Event) {
	t.Helper()
	a.Receiver.(*kntest.MockTransport).Received <- e
	e2 := <-a.Sender.(*kntest.MockTransport).Sent
	if !reflect.DeepEqual(e, e2) {
		t.Errorf("%v != %v", e, e2)
	}
}

// TestAdapter round-trip events with dummy transports.
func TestAdapter(t *testing.T) {
	f := &kntest.MockTransportFactory{Capacity: 10}
	a, err := NewAdapter(f, f)
	checkFatal(t, err)
	done := make(chan error)
	go func() { done <- a.Run(kntest.QuietContext()) }()
	for _, v := range []string{"a", "b", "c"} {
		roundTrip(t, a, kntest.MakeEvent(v))
	}
	close(a.Receiver.(*kntest.MockTransport).Received)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

// TestCancel verify that cancelling the context stops the adapter
func TestCancel(t *testing.T) {
	f := &kntest.MockTransportFactory{Capacity: 10}
	a, err := NewAdapter(f, f)
	checkFatal(t, err)
	done := make(chan error)
	ctx, cancel := context.WithCancel(kntest.QuietContext())
	go func() { done <- a.Run(ctx) }()
	roundTrip(t, a, kntest.MakeEvent("x"))
	cancel()
	if err := <-done; err != context.Canceled {
		t.Fatal("expected context.Canceled", err)
	}
}
