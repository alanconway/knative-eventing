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
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/cloudevents/sdk-go/pkg/cloudevents/codec"
	"github.com/knative/eventing/pkg/kncloudevents/kntransport"
	kntest "github.com/knative/eventing/pkg/kncloudevents/testing"
)

func assertNil(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestFileTransport(t *testing.T) {
	inRead, inWrite, err := os.Pipe()
	assertNil(t, err)
	outRead, outWrite, err := os.Pipe()
	assertNil(t, err)

	a := &kntransport.Adapter{
		Receiver: &FileTransport{Reader: inRead},
		Sender:   &FileTransport{Writer: outWrite},
	}
	assertNil(t, err)
	go func() { _ = a.Run(kntest.QuietContext()) }()
	testRun(t, inWrite, outRead)
	inWrite.Close()
}

func TestMain(t *testing.T) {
	// Start the adapter, reading stdin, writing stdout
	cmd := exec.Command("go", "run", "main.go", "-debug")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf(`%v={"input":"stdin"}`, kntransport.ReceiverEnv),
		fmt.Sprintf(`%v={"output":"stdout"}`, kntransport.SenderEnv),
	)
	in, err := cmd.StdinPipe()
	assertNil(t, err)
	out, err := cmd.StdoutPipe()
	assertNil(t, err)
	assertNil(t, cmd.Start())
	defer func() { _ = cmd.Process.Kill() }()

	testRun(t, in, out)
}

func testRun(t *testing.T, in io.Writer, out io.Reader) {
	r := bufio.NewReader(out)
	for _, v := range []string{"a", "b", "c"} {
		e1 := kntest.MakeEvent(v)
		b, err := codec.JsonEncodeV02(e1)
		assertNil(t, err)
		_, err = in.Write(append(b, '\n'))
		assertNil(t, err)

		b, err = r.ReadBytes('\n')
		assertNil(t, err)
		e2, err := codec.JsonDecodeV02(b)
		assertNil(t, err)

		if !reflect.DeepEqual(e1.String(), e2.String()) {
			t.Errorf("%s != %s", e1, e2)
		}
	}
}
