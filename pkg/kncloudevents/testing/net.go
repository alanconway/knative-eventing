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
	"net"
	"time"
)

// GetFreePort gets a free port for listening.
//
// NOTE: This is a work-around for https://github.com/cloudevents/sdk-go/issues/131
// You should normally call Listen("tcp", ":0") and l.Addr() directly.
// This workaround is potentially racy.
//
// Panic on error for ease of use. If your machine is out of ephemeral
// ports you are deep trouble anyway.
func GetFreePort() int {
	if l, err := net.Listen("tcp", "localhost:0"); err != nil {
		panic(err)
	} else {
		defer func() { _ = l.Close() }()
		return l.Addr().(*net.TCPAddr).Port
	}
}

// Redial dials and re-dials until a connection is established
// or a timeout is reached. See net.Dial.
func Redial(network, addr string) (net.Conn, error) {
	interval := time.Millisecond * 10
	max := time.Millisecond * 1000
	timeout := time.Second * 10

	d := net.Dialer{Deadline: time.Now().Add(timeout)}
	c, err := d.Dial(network, addr)
	for err != nil {
		time.Sleep(interval)
		c, err = d.Dial(network, addr)
		if time.Now().After(d.Deadline) {
			break
		}
		if interval < max {
			interval *= 2
		}
	}
	return c, err
}

// WaitAddr attempts to connect using Redial, then closes the connection.
// Used to wait until a listening port is ready to accepts connections.
func WaitAddr(network, addr string) error {
	c, err := Redial("tcp", addr)
	if err != nil {
		return err
	}
	c.Close()
	return nil
}
