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
	"bytes"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/cloudevents/sdk-go/pkg/cloudevents/transport"
	cehttp "github.com/cloudevents/sdk-go/pkg/cloudevents/transport/http"
)

// SenderFactory is a TransportFactory for a HTTP client.
// This is the sender type for Event Importers
type SenderFactory struct {
	// SinkURI is used to send events.
	SinkURI string `json:"sinkURI"`
}

// New creates a HTTP client transport.
func (c *SenderFactory) New() (transport.Transport, error) {
	// TODO(alanconway) Control over event encoding?
	return cehttp.New(cehttp.WithTarget(c.SinkURI), cehttp.WithEncoding(cehttp.BinaryV02))
}

// ReceiverFactory is configuration for a HTTP server receiver.
type ReceiverFactory struct {
	// ListenAddr is "host:port" for the HTTP listener
	ListenAddr string `json:"listenAddr"`
}

func splitHostPortInt(network, hostport string) (string, int, error) {
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		return "", -1, err
	}
	n, err := net.LookupPort(network, port)
	if err != nil {
		return "", -1, err
	}
	return host, n, nil
}

// New creates a Transport configured with the HTTPReceiver settings
func (f *ReceiverFactory) New() (transport.Transport, error) {
	_, port, err := splitHostPortInt("tcp", f.ListenAddr)
	if err != nil {
		return nil, err
	}
	return cehttp.New(cehttp.WithPort(port))
}

func copyHeaders(from, to http.Header) {
	for header, values := range from {
		for _, value := range values {
			to.Add(header, value)
		}
	}
}

// TODO(alanconway) PR for cloudevents, should be a http.Message method

// MessageRequest make HTTP request from HTTP message
func MessageRequest(m cehttp.Message) http.Request {
	var r http.Request
	copyHeaders(m.Header, r.Header)
	r.Body = ioutil.NopCloser(bytes.NewBuffer(m.Body))
	r.ContentLength = int64(len(m.Body))
	r.Method = http.MethodPost
	return r
}
