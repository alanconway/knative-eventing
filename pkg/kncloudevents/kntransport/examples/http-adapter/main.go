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

// EXAMPLE only: this example shows how to build a HTTP-to-HTTP adapter
// executable using the kntransport tools.
//
package main

import (
	"github.com/cloudevents/sdk-go/pkg/cloudevents/transport"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/transport/http"
	"github.com/knative/eventing/pkg/kncloudevents/kntransport"
)

// Note that standard HTTP transport factories are available in
// package kncloudevents/knhttp, the ones here are for illustration.

// SenderFactory is a TransportFactory for a HTTP client.
// This is the sender type for Event Importers
type SenderFactory struct {
	// SinkURI is used to send events.
	SinkURI string `json:"sinkURI"`
}

// New creates a HTTP client transport.
func (c *SenderFactory) New() (transport.Transport, error) {
	// TODO(alanconway) Control over event encoding?
	return http.New(http.WithTarget(c.SinkURI), http.WithEncoding(http.BinaryV02))
}

// ReceiverFactory is configuration for a HTTP server receiver.
type ReceiverFactory struct {
	// Port is the port for the HTTP listener
	Port int `json:"port"`
}

// New creates a Transport configured with the HTTPReceiver settings
func (f *ReceiverFactory) New() (transport.Transport, error) {
	return http.New(http.WithPort(f.Port))
}

func main() {
	kntransport.AdapterMain(&ReceiverFactory{}, &SenderFactory{})
}
