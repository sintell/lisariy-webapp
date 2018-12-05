// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package sasl

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"testing"
)

type saslStep struct {
	challenge []byte
	resp      []byte
	more      bool
	err       bool
}

type saslTest struct {
	machine *Negotiator
	steps   []saslStep
}

type testCases struct {
	name  string
	cases []saslTest
}

func getStepName(n *Negotiator) string {
	switch n.State() & StepMask {
	case Initial:
		return "Initial"
	case AuthTextSent:
		return "AuthTextSent"
	case ResponseSent:
		return "ResponseSent"
	case ValidServerResponse:
		return "ValidServerResponse"
	default:
		panic("Step part of state byte apparently has too many bits")
	}
}

func TestSASL(t *testing.T) {
	doTests(t, []testCases{clientTestCases, serverTestCases}, func(t *testing.T, test saslTest) {
		// Run each test twice to make sure that Reset actually sets the state back
		// to the initial state.
		for run := 1; run < 3; run++ {
			// Reset the nonce to the one used by all of our test vectors.
			test.machine.nonce = []byte("fyko+d2lbbFgONRv9qkxdawL")
			for _, step := range test.steps {
				more, resp, err := test.machine.Step(
					[]byte(base64.StdEncoding.EncodeToString(step.challenge)),
				)
				switch {
				case err != nil && test.machine.State()&Errored != Errored:
					t.Logf("Run %d, Step %s", run, getStepName(test.machine))
					t.Fatalf("State machine internal error state was not set, got error: %v", err)
				case err == nil && test.machine.State()&Errored == Errored:
					t.Logf("Run %d, Step %s", run, getStepName(test.machine))
					t.Fatal("State machine internal error state was set, but no error was returned")
				case err == nil && step.err:
					// There was no error, but we expect one
					t.Logf("Run %d, Step %s", run, getStepName(test.machine))
					t.Fatal("Expected SASL step to error")
				case err != nil && !step.err:
					// There was an error, but we didn't expect one
					t.Logf("Run %d, Step %s", run, getStepName(test.machine))
					t.Fatalf("Got unexpected SASL error: %v", err)
				case base64.StdEncoding.EncodeToString(step.resp) != string(resp):
					t.Logf("Run %d, Step %s", run, getStepName(test.machine))
					decoded, _ := base64.StdEncoding.DecodeString(string(resp))
					t.Fatalf("Got invalid response text:\nexpected `%s'\n     got `%s'", step.resp, decoded)
				case more != step.more:
					t.Logf("Run %d, Step %s", run, getStepName(test.machine))
					t.Fatalf("Got unexpected value for more: %v", more)
				}
			}
			test.machine.Reset()
		}
	})
}

func BenchmarkScram(b *testing.B) {
	for n := 0; n < b.N; n++ {
		c := NewClient(
			scram("SCRAM-SHA-1", sha1.New),
			Credentials(func() ([]byte, []byte, []byte) {
				return []byte("user"), []byte("pencil"), []byte{}
			}),
		)
		for _, step := range clientTestCases.cases[0].steps {
			more, _, _ := c.Step(step.challenge)
			if !more {
				break
			}
		}
	}
}

func BenchmarkPlain(b *testing.B) {
	for n := 0; n < b.N; n++ {
		c := NewClient(
			plain,
			Credentials(func() ([]byte, []byte, []byte) {
				return []byte("user"), []byte("pencil"), []byte{}
			}),
		)
		for _, step := range clientTestCases.cases[0].steps {
			more, _, _ := c.Step(step.challenge)
			if !more {
				break
			}
		}
	}
}

func doTests(t *testing.T, cases []testCases, fn func(t *testing.T, tc saslTest)) {
	for _, g := range cases {
		for i, tc := range g.cases {
			name := fmt.Sprintf("%s:%d", g.name, i)
			t.Run(name, func(t *testing.T) {
				fn(t, tc)
			})
		}
	}
}
