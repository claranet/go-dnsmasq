// Copyright (c) 2014 The SkyDNS Authors. All rights reserved.
// Use of this source code is governed by The MIT License (MIT) that can be
// found in the LICENSE file.

package cache

import (
	"testing"
	"time"

	"github.com/miekg/dns"
)

const testTTL = 2
const testStaleTTL = 4

type testcase struct {
	msg         *dns.Msg
	dnssec, tcp bool
}

func newMsg(zone string, typ uint16) *dns.Msg {
	msg := &dns.Msg{}
	msg.SetQuestion(zone, typ)
	return msg
}

func TestInsertMessage(t *testing.T) {
	cch := New(10, testTTL, testStaleTTL, false, 0)

	testcases := []testcase{
		{newMsg("example.com.", dns.TypeA), false, false},
		{newMsg("example.net.", dns.TypeAAAA), false, false},
		{newMsg("example.org.", dns.TypeCNAME), true, false},
		{newMsg("example.com.", dns.TypeMX), true, false},
		{newMsg("example.net.", dns.TypeNS), true, false},
		{newMsg("example.org.", dns.TypeTXT), true, false},
		{newMsg("example.com.", dns.TypeSOA), true, false},
	}

	for _, tc := range testcases {
		cch.InsertMessage(Key(tc.msg.Question[0], tc.dnssec, tc.tcp), tc.msg)

		cMsg := cch.Hit(tc.msg.Question[0], tc.dnssec, tc.tcp, tc.msg.Id, false, false)

		if cMsg.Question[0].Qtype != tc.msg.Question[0].Qtype {
			t.Fatalf("bad Qtype, expected %d, got %d:", tc.msg.Question[0].Qtype, cMsg.Question[0].Qtype)
		}
		if cMsg.Question[0].Name != tc.msg.Question[0].Name {
			t.Fatalf("bad Qtype, expected %s, got %s:", tc.msg.Question[0].Name, cMsg.Question[0].Name)
		}

		cMsg = cch.Hit(tc.msg.Question[0], !tc.dnssec, tc.tcp, tc.msg.Id, false, false)
		if cMsg != nil {
			t.Fatalf("bad cache hit, expected <nil>, got %s:", cMsg)
		}
		cMsg = cch.Hit(tc.msg.Question[0], !tc.dnssec, !tc.tcp, tc.msg.Id, false, false)
		if cMsg != nil {
			t.Fatalf("bad cache hit, expected <nil>, got %s:", cMsg)
		}
		cMsg = cch.Hit(tc.msg.Question[0], tc.dnssec, !tc.tcp, tc.msg.Id, false, false)
		if cMsg != nil {
			t.Fatalf("bad cache hit, expected <nil>, got %s:", cMsg)
		}
	}
}

func TestExpireMessage(t *testing.T) {
	cch := New(10, testTTL-1, testStaleTTL-1, false, 0)

	testcases := testcase{newMsg("example.com.", dns.TypeA), false, false}

	cch.InsertMessage(Key(testcases.msg.Question[0], testcases.dnssec, testcases.tcp), testcases.msg)

	cMsg := cch.Hit(testcases.msg.Question[0], testcases.dnssec, testcases.tcp, testcases.msg.Id, false, false)
	if cMsg.Question[0].Qtype != testcases.msg.Question[0].Qtype {
		t.Fatalf("bad Qtype, expected %d, got %d:", testcases.msg.Question[0].Qtype, cMsg.Question[0].Qtype)
	}
	if cMsg.Question[0].Name != testcases.msg.Question[0].Name {
		t.Fatalf("bad Qtype, expected %s, got %s:", testcases.msg.Question[0].Name, cMsg.Question[0].Name)
	}

	time.Sleep(testTTL * time.Second)

	cMsg = cch.Hit(testcases.msg.Question[0], testcases.dnssec, testcases.tcp, testcases.msg.Id, false, false)

	if cMsg != nil {
		t.Fatalf("expected nil message from expired cache, got %v", cMsg.Answer)
	}

}

func TestExpireMessageWithStale(t *testing.T) {
	cch := New(10, testTTL-1, testStaleTTL-1, false, 0)

	testcases := testcase{newMsg("example.com.", dns.TypeA), false, false}

	cch.InsertMessage(Key(testcases.msg.Question[0], testcases.dnssec, testcases.tcp), testcases.msg)

	cMsg := cch.Hit(testcases.msg.Question[0], testcases.dnssec, testcases.tcp, testcases.msg.Id, true, true)
	if cMsg.Question[0].Qtype != testcases.msg.Question[0].Qtype {
		t.Fatalf("bad Qtype, expected %d, got %d:", testcases.msg.Question[0].Qtype, cMsg.Question[0].Qtype)
	}
	if cMsg.Question[0].Name != testcases.msg.Question[0].Name {
		t.Fatalf("bad Qtype, expected %s, got %s:", testcases.msg.Question[0].Name, cMsg.Question[0].Name)
	}

	time.Sleep(testTTL * time.Second)

	cMsg = cch.Hit(testcases.msg.Question[0], testcases.dnssec, testcases.tcp, testcases.msg.Id, true, true)

	if cMsg == nil {
		t.Fatal("expected stale message from expired cache, got nil")
	}

}

func TestExpireMessageWithExpiredStale(t *testing.T) {
	cch := New(10, testTTL-1, testStaleTTL-1, false, 0)

	testcases := testcase{newMsg("example.com.", dns.TypeA), false, false}

	cch.InsertMessage(Key(testcases.msg.Question[0], testcases.dnssec, testcases.tcp), testcases.msg)

	cMsg := cch.Hit(testcases.msg.Question[0], testcases.dnssec, testcases.tcp, testcases.msg.Id, true, true)
	if cMsg.Question[0].Qtype != testcases.msg.Question[0].Qtype {
		t.Fatalf("bad Qtype, expected %d, got %d:", testcases.msg.Question[0].Qtype, cMsg.Question[0].Qtype)
	}
	if cMsg.Question[0].Name != testcases.msg.Question[0].Name {
		t.Fatalf("bad Qtype, expected %s, got %s:", testcases.msg.Question[0].Name, cMsg.Question[0].Name)
	}

	time.Sleep(testStaleTTL * time.Second)

	// Call twice because stale feature returns the message one last time after it is stale expired,
	// only removing it after
	_ = cch.Hit(testcases.msg.Question[0], testcases.dnssec, testcases.tcp, testcases.msg.Id, false, true)
	cMsg = cch.Hit(testcases.msg.Question[0], testcases.dnssec, testcases.tcp, testcases.msg.Id, false, true)

	if cMsg != nil {
		t.Fatalf("expected nil message from stale expired cache, got %v", cMsg)
	}

}
