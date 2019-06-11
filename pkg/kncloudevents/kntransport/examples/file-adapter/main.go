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

// EXAMPLE only: this example shows how to build a trivial custom
// transport to read/write events to/from files and how to use it in
// an adapter.
//
package main

import (
	"bufio"
	"context"
	"io"
	"os"

	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/codec"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/transport"
	"github.com/knative/eventing/pkg/kncloudevents/kntransport"
)

// NOTE: In a real application the transport, factories and
// main() would typically be in separate packages.

// ================ STEP 1: implement a transport.
// This is a trivial transport that just reads/writes JSON encoded
// events to/from files

// FileTransport reads/writes JSON-encoded events to/from files
type FileTransport struct {
	// Writer used by Send
	Writer io.Writer
	// Reader used by StartReceiver
	Reader io.Reader
	// Receiver called by StartReceiver
	Receiver transport.Receiver
}

// Send implements transport.Send
func (t *FileTransport) Send(ctx context.Context, e cloudevents.Event) (*cloudevents.Event, error) {
	if b, err := codec.JsonEncodeV02(e); err != nil {
		return nil, err
	} else if _, err := t.Writer.Write(append(b, '\n')); err != nil {
		return nil, err
	}
	return nil, nil
}

// SetReceiver implements transport.SetReceiver
func (t *FileTransport) SetReceiver(r transport.Receiver) { t.Receiver = r }

// StartReceiver implements transport.StartReceiver
func (t *FileTransport) StartReceiver(ctx context.Context) (err error) {
	r := bufio.NewReader(t.Reader)
	for {
		if b, err := r.ReadBytes('\n'); err != nil {
			if len(b) == 0 && err == io.EOF {
				return nil // EOF with no partial data is not an error.
			}
			return err
		} else if e, err := codec.JsonDecodeV02(b); err != nil {
			return err
		} else if err := t.Receiver.Receive(ctx, *e, nil); err != nil {
			return err
		}
	}
}

// ================ STEP 2: implement configuration factories
// A factory is a struct holding the configuration values for your
// transport, that can be marshalled/unmarshalled as JSON.
//
// It must provide a New() method to create your transport from
// the values in the struct.
//
// You can provide more than one factory for the same transport if it
// can be used in multiple modes (client/server, sender/receiver etc.)
//
// When the transport is used as part of an importer or channel,
// the controller will assemble a JSON object from the CRD YAML
// spec and controller status and pass it to your factory.

// SenderFactory configures a FileTransport for sending
type SenderFactory struct {
	// Output is "stdout" or an output file name.
	Output string `json:"output"`
}

// New creates a sender transport based on the configuration in struct f.
func (f *SenderFactory) New() (t transport.Transport, err error) {
	out := os.Stdout
	if f.Output != "stdout" {
		if out, err = os.Create(f.Output); err != nil {
			return nil, err
		}
	}
	t = &FileTransport{Writer: out}
	return t, nil
}

// ReceiverFactory configures a FileTransport for receiving.
type ReceiverFactory struct {
	// Input is "stdin" or an input file name
	Input string `json:"input"`
}

// New creates a  receiver transport based on the configuration in struct f.
func (f *ReceiverFactory) New() (t transport.Transport, err error) {
	in := os.Stdin
	if f.Input != "stdin" {
		if in, err = os.Open(f.Input); err != nil {
			return nil, err
		}
	}
	t = &FileTransport{Reader: in}
	return t, nil
}

func main() {
	kntransport.AdapterMain(&ReceiverFactory{}, &SenderFactory{})
}
