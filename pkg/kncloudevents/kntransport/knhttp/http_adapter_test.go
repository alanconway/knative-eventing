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

package knhttp

import (
	"fmt"
	"net/url"
	"reflect"
	"testing"

	"github.com/cloudevents/sdk-go/pkg/cloudevents/transport/http"
	"github.com/knative/eventing/pkg/kncloudevents"
	"github.com/knative/eventing/pkg/kncloudevents/kntransport"
	kntest "github.com/knative/eventing/pkg/kncloudevents/testing"
)

func assertNil(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// Send events thru a HTTPReceiver-HTTPSender adapter.
func TestHTTPAdapter(t *testing.T) {
	ctx := kntest.QuietContext()

	// Start the back-end server
	server, err := http.New(http.WithPort(kntest.GetFreePort()))
	serverURL := &url.URL{Scheme: "http", Host: fmt.Sprintf(":%d", server.GetPort())}
	assertNil(t, err)
	server.SetReceiver(make(kntest.MockReceiver, 10))
	serverErr := make(chan error)
	go func() { serverErr <- server.StartReceiver(ctx) }()
	assertNil(t, kntest.WaitAddr("tcp", serverURL.Host))

	// Start the adapter, sender sends to server
	receiverURL := &url.URL{Scheme: "http", Host: fmt.Sprintf(":%d", kntest.GetFreePort())}
	a, err := kntransport.NewAdapter(
		&ReceiverFactory{ListenAddr: receiverURL.Host},
		&SenderFactory{SinkURI: serverURL.String()})
	assertNil(t, err)
	adapterErr := make(chan error)
	go func() { adapterErr <- a.Run(ctx) }()
	assertNil(t, kntest.WaitAddr("tcp", receiverURL.Host))

	// Start the front-end client, send to adapter receiver
	client, err := kncloudevents.NewDefaultClient(receiverURL.String())
	assertNil(t, err)
	for _, v := range []string{"a", "b", "c"} {
		e1 := kntest.MakeEvent(v)
		_, err := client.Send(ctx, e1)
		assertNil(t, err)
		e2 := <-server.Receiver.(kntest.MockReceiver)
		if !reflect.DeepEqual(e1.String(), e2.String()) {
			t.Errorf("%v != %v", e1, e2)
		}
	}
}
