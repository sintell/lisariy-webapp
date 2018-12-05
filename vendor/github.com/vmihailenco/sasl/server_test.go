// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package sasl

func acceptAll(_ *Negotiator) bool {
	return true
}

var serverTestCases = testCases{
	name: "Server",
	cases: []saslTest{{
		machine: NewServer(plain, acceptAll),
		steps: []saslStep{
			{challenge: []byte("Ursel\x00Kurt\x00xipj3plmq\x00"), resp: nil, err: true, more: false},
		},
	}, {
		machine: NewServer(plain, acceptAll),
		steps: []saslStep{
			{challenge: []byte("\x00Ursel\x00Kurt\x00xipj3plmq"), resp: nil, err: true, more: false},
		},
	}, {
		machine: NewServer(plain, func(n *Negotiator) bool {
			user, pass, ident := n.Credentials()
			switch {
			case string(user) != "Kurt":
				return false
			case string(pass) != "xipj3plmq":
				return false
			case string(ident) != "Ursel":
				return false
			}
			return true
		}),
		steps: []saslStep{
			{challenge: plainResp, resp: plainResp, err: false, more: false},
		},
	}, {
		machine: NewServer(plain, func(n *Negotiator) bool {
			user, _, _ := n.Credentials()
			return string(user) == "FAIL"
		}),
		steps: []saslStep{
			{challenge: plainResp, resp: nil, err: true, more: false},
		},
	}, {
		machine: NewServer(plain, func(n *Negotiator) bool {
			_, pass, _ := n.Credentials()
			return string(pass) == "FAIL"
		}),
		steps: []saslStep{
			{challenge: plainResp, resp: nil, err: true, more: false},
		},
	}, {
		machine: NewServer(plain, func(n *Negotiator) bool {
			_, _, ident := n.Credentials()
			return string(ident) == "FAIL"
		}),
		steps: []saslStep{
			{challenge: plainResp, resp: nil, err: true, more: false},
		},
	}, {
		machine: NewServer(plain, nil),
		steps: []saslStep{
			{challenge: plainResp, resp: nil, err: true, more: false},
		},
	}, {
		machine: NewServer(scram("", nil), acceptAll),
		steps: []saslStep{
			{challenge: nil, resp: nil, err: true, more: false},
		},
	}, {
		machine: NewServer(scram("", nil), acceptAll),
		steps: []saslStep{
			{challenge: []byte{}, resp: nil, err: true, more: false},
		},
	}},
}
