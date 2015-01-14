// Copyright 2011 Aaron Jacobs. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Integration tests for the serial package.

package serial

import (
	"crypto/rand"
	"errors"
	"io"
	"os"
)

import "testing"
import "time"

var DEVICE string

//////////////////////////////////////////////////////
// Helpers
//////////////////////////////////////////////////////

// Read at least n bytes from an io.Reader, making sure not to block if it
// takes too long.
func readWithTimeout(r io.Reader, n int) ([]byte, error) {
	buf := make([]byte, n)
	done := make(chan error)
	readAndCallBack := func() {
		_, err := io.ReadAtLeast(r, buf, n)
		done <- err
	}

	go readAndCallBack()

	timeout := make(chan bool)
	sleepAndCallBack := func() { time.Sleep(2e9); timeout <- true }
	go sleepAndCallBack()

	select {
	case err := <-done:
		return buf, err
	case <-timeout:
		return nil, errors.New("Timed out.")
	}

	return nil, errors.New("Can't get here.")
}

//////////////////////////////////////////////////////
// Init
//////////////////////////////////////////////////////

func init() {
	DEVICE = os.Getenv("SERIAL_PORT")
	if DEVICE == "" {
		DEVICE = "/dev/tty.usbserial-A8008HlV"
	}
}

//////////////////////////////////////////////////////
// Tests
//////////////////////////////////////////////////////

// The device is assumed to be running the increment_and_echo program from the
// hardware directory.
func TestIncrementAndEcho(t *testing.T) {
	// Open the port.
	var options OpenOptions
	options.PortName = DEVICE
	options.BaudRate = 19200
	options.DataBits = 8
	options.StopBits = 1
	options.MinimumReadSize = 4

	circuit, err := Open(options)
	if err != nil {
		t.Fatal(err)
	}

	defer circuit.Close()

	// Pause for a few seconds to deal with the Arduino's annoying startup delay.
	time.Sleep(3e9)

	// Write some bytes.
	b := []byte{0x00, 0x17, 0xFE, 0xFF}

	n, err := circuit.Write(b)
	if err != nil {
		t.Fatal(err)
	}

	if n != 4 {
		t.Fatal("Expected 4 bytes written, got ", n)
	}

	// Check the response.
	b, err = readWithTimeout(circuit, 4)
	if err != nil {
		t.Fatal(err)
	}

	if b[0] != 0x01 {
		t.Error("Expected 0x01, got ", b[0])
	}
	if b[1] != 0x18 {
		t.Error("Expected 0x18, got ", b[1])
	}
	if b[2] != 0xFF {
		t.Error("Expected 0xFF, got ", b[2])
	}
	if b[3] != 0x00 {
		t.Error("Expected 0x00, got ", b[3])
	}
}

func TestLoopback(t *testing.T) {
	var options OpenOptions
	options.PortName = DEVICE
	options.BaudRate = 19200
	options.DataBits = 8
	options.StopBits = 1
	options.MinimumReadSize = 4

	circuit, err := Open(options)
	if err != nil {
		t.Fatal(err)
	}

	defer circuit.Close()

	randomCount := 100
	randomBytes := make([]byte, randomCount)
	_, err = rand.Read(randomBytes)
	if err != nil {
		t.Fatal(err)
	}

	n, err := circuit.Write(randomBytes)
	if err != nil {
		t.Fatal(err)
	}

	if n != randomCount {
		t.Fatalf("Expected %d bytes written, got %d", randomCount, n)
	}

	// Check the response.
	b, err := readWithTimeout(circuit, randomCount)
	if err != nil {
		t.Fatal(err)
	}

	for i, value := range b {
		if value != randomBytes[i]  {
			t.Errorf("Expected 0x%x, got 0x%x", randomBytes[i], value)
		}
	}

}

func TestList(t *testing.T) {
	list := List()
	for _, port := range list {
		t.Logf("%v", port)
	}
}
