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
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"path"

	ce "github.com/cloudevents/sdk-go/pkg/cloudevents"
	cecontext "github.com/cloudevents/sdk-go/pkg/cloudevents/context"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/transport"
	"github.com/knative/pkg/signals"
	"go.uber.org/zap"
)

// Adapter tranfers events from one cloudevents Transport to another.
type Adapter struct {
	Receiver, Sender transport.Transport
}

// NewAdapter creates a new Adapter using receiverFactory and senderFactory
func NewAdapter(receiverFactory, senderFactory Factory) (*Adapter, error) {
	if receiver, err := receiverFactory.New(); err != nil {
		return nil, err
	} else if sender, err := senderFactory.New(); err != nil {
		return nil, err
	} else {
		return &Adapter{receiver, sender}, nil
	}
}

// Run calls receiver.StartReceiver(ctx) and sends received
// events with sender.Send(ctx, event).
// It returns the return value of receiver.StartReceiver(ctx)
func (a *Adapter) Run(ctx context.Context) error {
	logger := cecontext.LoggerFrom(ctx).Desugar() // Used on critical path
	receiver := ReceiveFunc(
		func(ctx context.Context, e ce.Event, er *ce.EventResponse) error {
			logger.Debug("sending event", zap.Stringer("event", e))
			resp, err := a.Sender.Send(ctx, e)
			if err != nil {
				logger.Error("send error", zap.Error(err))
				er.Error(http.StatusInternalServerError, err.Error())
			} else if resp != nil {
				logger.Debug("sender got response", zap.Stringer("event", e))
				er.RespondWith(http.StatusAccepted, resp)
			}
			return err // Let the transport observer logic process the error.
		})
	a.Receiver.SetReceiver(receiver)
	logger.Info("starting receiver receiver")
	return a.Receiver.StartReceiver(ctx)
}

// TODO(alanconway) PR ReceiveFunc to cloudevents library

// ReceiveFunc wraps a function as a transport.Receiver
type ReceiveFunc func(ctx context.Context, e ce.Event, er *ce.EventResponse) error

// Receive implements transport.Receiver.Receive
func (f ReceiveFunc) Receive(ctx context.Context, e ce.Event, er *ce.EventResponse) error {
	return f(ctx, e, er)
}

const (
	// SenderEnv is name of environment variable for sender JSON configuration.
	SenderEnv = "ADAPTER_SENDER"
	// ReceiverEnv is name of environment variable for receiver JSON configuration.
	ReceiverEnv = "ADAPTER_RECEIVER"
)

// TODO(alanconway) Compare against existing importer adapters for missing features.

// AdapterMain is a main() function that conventional command line
// arguments and environment variables to configure and run an Adapter
// with the given receiverFactory and senderFactory. It does the following:
//
// - Configures receiverFactory and senderFactory from environment variables
//   name by ReceiverEnv and SenderEnv.
// - Handles default command line flags and usage message.
//   (You can create additional flags before calling AdapterMain())
// - Sets up a cloudevents.Context with logging and signal handling,
//   which is passed to receiver and sender transports.
// - Logs errors and progress info while setting up the adapter.
// - Runs the Adapter
// - Returns a value suitable for os.Exit (0 on success, non-0 on error)
//
func AdapterMain(receiverFactory, senderFactory Factory) int {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `Usage of %s:
Transfers messages from receiver to sender.

Environment variables:

%s: configuration for receiver (incoming) transport.
%s: JSON configuration for sender (outgoing) transport.

`, os.Args[0], ReceiverEnv, SenderEnv)
		flag.PrintDefaults()
	}

	debug := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	var l *zap.Logger
	var err error
	if *debug {
		l, err = zap.NewDevelopment()
	} else {
		l, err = zap.NewProduction()
	}
	if err != nil {
		log.Printf("cannot create logger: %v", err)
		return 1
	}
	defer func() { _ = l.Sync() }()
	logger := l.Named(path.Base(os.Args[0])).Sugar()
	receiver := newEnvTransport(receiverFactory, ReceiverEnv, logger)
	sender := newEnvTransport(senderFactory, SenderEnv, logger)
	a := Adapter{Receiver: receiver, Sender: sender}
	err = a.Run(cecontext.WithLogger(signals.NewContext(), logger))
	if err != nil {
		logger.Errorw("adapter run error", zap.Error(err))
		return 1
	}
	return 0
}

func newEnvTransport(factory Factory, env string, logger *zap.SugaredLogger) transport.Transport {
	config := os.Getenv(env)
	if config == "" {
		logger.Fatalf("no value for environment variable: %v", env)
	}
	if err := json.Unmarshal([]byte(config), factory); err != nil {
		logger.Fatalf("invalid configuration: %v: %v", env, err)
	}
	logger.Infof("creating transport %s=%v", env, config)
	t, err := factory.New()
	if err != nil {
		logger.Fatalf("error creating transport %v: %v", env, err)
	}
	return t
}
