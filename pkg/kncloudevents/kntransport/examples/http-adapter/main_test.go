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

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/cloudevents/sdk-go/pkg/cloudevents/transport/http"
	"github.com/knative/eventing/pkg/kncloudevents"
	"github.com/knative/eventing/pkg/kncloudevents/kntransport"
	kntest "github.com/knative/eventing/pkg/kncloudevents/testing"
)

func checkFatal(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func TestMain(t *testing.T) {
	ctx, cancel := context.WithCancel(kntest.QuietContext())

	// Start the back-end server
	server, err := http.New(http.WithPort(kntest.GetFreePort()))
	serverURL := fmt.Sprintf("http://:%d", server.GetPort())
	checkFatal(t, err)
	server.SetReceiver(make(kntest.MockReceiver, 10))
	serverErr := make(chan error)
	go func() { serverErr <- server.StartReceiver(ctx) }()

	// Start the adapter, sender to server
	receiverPort := kntest.GetFreePort()
	cmd := exec.Command("go", "run", "main.go", "-debug")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf(`%v={"port":%v}`, kntransport.ReceiverEnv, receiverPort),
		fmt.Sprintf(`%v={"sinkURI":"%v"}`, kntransport.SenderEnv, serverURL))
	err = cmd.Start()
	checkFatal(t, err)
	defer func() { _ = cmd.Process.Kill() }()
	checkFatal(t, kntest.WaitAddr("tcp", fmt.Sprintf(":%d", receiverPort)))

	// Start the client, send to adapter receiver
	client, err := kncloudevents.NewDefaultClient(fmt.Sprintf("http://:%d", receiverPort))
	checkFatal(t, err)
	for _, v := range []string{"a", "b", "c"} {
		e1 := kntest.MakeEvent(v)
		_, err := client.Send(ctx, e1)
		checkFatal(t, err)
		e2 := <-server.Receiver.(kntest.MockReceiver)
		if !reflect.DeepEqual(e1.String(), e2.String()) {
			t.Errorf("%v != %v", e1, e2)
		}
	}
	cancel()
}
