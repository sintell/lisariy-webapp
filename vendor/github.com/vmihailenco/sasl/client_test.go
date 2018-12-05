// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package sasl

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
)

var plainResp = []byte("Ursel\x00Kurt\x00xipj3plmq")

var clientTestCases = testCases{
	name: "Client",
	cases: []saslTest{{
		machine: NewClient(plain,
			Credentials(func() ([]byte, []byte, []byte) {
				return []byte("Kurt"), []byte("xipj3plmq"), []byte("Ursel")
			})),
		steps: []saslStep{
			{challenge: []byte{}, resp: plainResp, err: false, more: false},
			{challenge: nil, resp: nil, err: true, more: false},
		},
	}, {
		machine: NewClient(
			scram("SCRAM-SHA-1", sha1.New),
			Credentials(func() ([]byte, []byte, []byte) {
				return []byte("user"), []byte("pencil"), []byte{}
			}),
		),
		steps: []saslStep{
			{
				challenge: nil,
				resp:      []byte(`n,,n=user,r=fyko+d2lbbFgONRv9qkxdawL`),
				err:       false, more: true,
			},
			{
				challenge: []byte(`r=fyko+d2lbbFgONRv9qkxdawL3rfcNHYJY1ZVvWVs7j,s=QSXCR+Q6sek8bf92,i=4096`),
				resp:      []byte(`c=biws,r=fyko+d2lbbFgONRv9qkxdawL3rfcNHYJY1ZVvWVs7j,p=v0X8v3Bz2T0CJGbJQyF0X+HI4Ts=`),
				err:       false, more: true,
			},
			{
				challenge: []byte(`v=rmF9pqV8S7suAoZWja4dJRkFsKQ=`),
				resp:      nil,
				err:       false, more: false,
			},
		},
	}, {
		// Mechanism is not SCRAM-SHA-1-PLUS, but has connstate and remote mechanisms.
		machine: NewClient(
			scram("SCRAM-SHA-1", sha1.New),
			Credentials(func() ([]byte, []byte, []byte) {
				return []byte("user"), []byte("pencil"), []byte{}
			}),
			RemoteMechanisms("SCRAM-SHA-1-PLUS", "SCRAM-SHA-1"),
			TLSState(tls.ConnectionState{TLSUnique: []byte{0, 1, 2, 3, 4}}),
		),
		steps: []saslStep{
			{
				challenge: nil,
				resp:      []byte(`n,,n=user,r=fyko+d2lbbFgONRv9qkxdawL`),
				err:       false, more: true,
			},
			{
				challenge: []byte(`r=fyko+d2lbbFgONRv9qkxdawL3rfcNHYJY1ZVvWVs7j,s=QSXCR+Q6sek8bf92,i=4096`),
				resp:      []byte(`c=biws,r=fyko+d2lbbFgONRv9qkxdawL3rfcNHYJY1ZVvWVs7j,p=v0X8v3Bz2T0CJGbJQyF0X+HI4Ts=`),
				err:       false, more: true,
			},
			{
				challenge: []byte(`v=rmF9pqV8S7suAoZWja4dJRkFsKQ=`),
				resp:      nil,
				err:       false, more: false,
			},
		},
	}, {
		machine: NewClient(
			scram("SCRAM-SHA-1-PLUS", sha1.New),
			Credentials(func() ([]byte, []byte, []byte) {
				return []byte("user"), []byte("pencil"), []byte{}
			}),
			RemoteMechanisms("SCRAM-SHA-1-PLUS"),
			TLSState(tls.ConnectionState{TLSUnique: []byte{0, 1, 2, 3, 4}}),
		),
		steps: []saslStep{
			{
				challenge: nil,
				resp:      []byte(`p=tls-unique,,n=user,r=fyko+d2lbbFgONRv9qkxdawL`),
				err:       false, more: true,
			},
			{
				challenge: []byte(`r=fyko+d2lbbFgONRv9qkxdawL16090868851744577,s=QSXCR+Q6sek8bf92,i=4096`),
				resp:      []byte(`c=cD10bHMtdW5pcXVlLCwAAQIDBA==,r=fyko+d2lbbFgONRv9qkxdawL16090868851744577,p=kD6Wfe1kGICYN08YH7oONG2Enb0=`),
				err:       false, more: true,
			},
			{
				challenge: []byte(`v=QI0Ihj/QJv+VSyezLtd/d5PrYy0=`),
				resp:      nil,
				err:       false, more: false,
			},
		},
	}, {
		machine: NewClient(
			scram("SCRAM-SHA-256", sha256.New),
			Credentials(func() ([]byte, []byte, []byte) {
				return []byte("user"), []byte("pencil"), []byte{}
			}),
		),
		steps: []saslStep{
			{
				challenge: []byte{},
				resp:      []byte("n,,n=user,r=fyko+d2lbbFgONRv9qkxdawL"),
				err:       false, more: true,
			},
			{
				challenge: []byte(`r=fyko+d2lbbFgONRv9qkxdawL%hvYDpWUa2RaTCAfuxFIlj)hNlF$k0,s=W22ZaJ0SNY7soEsUEjb6gQ==,i=4096`),
				resp:      []byte(`c=biws,r=fyko+d2lbbFgONRv9qkxdawL%hvYDpWUa2RaTCAfuxFIlj)hNlF$k0,p=2FUSN0pPcS7P8hBhsxBJOiUDbRoW4KVNGZT0LxVnSek=`),
				err:       false, more: true,
			},
			{
				challenge: []byte(`v=zJZjsVp2g+W9jd01vgbsshippfH1sM0tLdBvs+e3DF4=`),
				resp:      nil,
				err:       false, more: false,
			},
		},
	}, {
		machine: NewClient(
			scram("SCRAM-SHA-256-PLUS", sha256.New),
			Credentials(func() ([]byte, []byte, []byte) {
				return []byte("user"), []byte("pencil"), []byte("admin")
			}),
			RemoteMechanisms("SCRAM-SOMETHING", "SCRAM-SHA-256-PLUS"),
			TLSState(tls.ConnectionState{TLSUnique: []byte{0, 1, 2, 3, 4}}),
		),
		steps: []saslStep{
			{
				challenge: []byte{},
				resp:      []byte("p=tls-unique,a=admin,n=user,r=fyko+d2lbbFgONRv9qkxdawL"),
				err:       false, more: true,
			},
			{
				challenge: []byte(`r=fyko+d2lbbFgONRv9qkxdawL,s=W22ZaJ0SNY7soEsUEjb6gQ==,i=4096`),
				resp:      []byte(`c=cD10bHMtdW5pcXVlLGE9YWRtaW4sAAECAwQ=,r=fyko+d2lbbFgONRv9qkxdawL,p=USNVS9hYD1JWfBOQwzc8o/9vFPQ7kA4CKsocmko/8yU=`),
				err:       false, more: true,
			},
			{
				challenge: []byte(`v=zjC1aKz20rqp7P92qtiJD1+gihbP5dKzIUFlBWgOuss=`),
				resp:      nil,
				err:       false, more: false,
			},
		},
	}, {
		machine: NewClient(
			scram("SCRAM-SHA-1-PLUS", sha1.New),
			Credentials(func() ([]byte, []byte, []byte) {
				return []byte(",=,="), []byte("password"), []byte{}
			}),
			RemoteMechanisms("SCRAM-SHA-1-PLUS"),
			TLSState(tls.ConnectionState{TLSUnique: []byte("finishedmessage")}),
		),
		steps: []saslStep{
			{
				challenge: []byte{},
				resp:      []byte("p=tls-unique,,n==2C=3D=2C=3D,r=fyko+d2lbbFgONRv9qkxdawL"),
				err:       false, more: true,
			},
			{
				challenge: []byte(`r=fyko+d2lbbFgONRv9qkxdawLtheirnonce,s=W22ZaJ0SNY7soEsUEjb6gQ==,i=4096`),
				resp:      []byte(`c=cD10bHMtdW5pcXVlLCxmaW5pc2hlZG1lc3NhZ2U=,r=fyko+d2lbbFgONRv9qkxdawLtheirnonce,p=8t6BJnSAd7Vi+mGZEi+Oqwci11c=`),
				err:       false, more: true,
			},
			{
				challenge: []byte(`v=8IDvl31piL1lkn6XLCqqFVS4EJM=`),
				resp:      nil,
				err:       false, more: false,
			},
		},
	},
	},
}
