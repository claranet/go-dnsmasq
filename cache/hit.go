// Copyright (c) 2014 The SkyDNS Authors. All rights reserved.
// Use of this source code is governed by The MIT License (MIT) that can be
// found in the LICENSE file.

package cache

import (
	"time"

	"github.com/miekg/dns"
)

// Hit returns a dns message from the cache. If the message's TTL is expired nil
// is returned and the message is removed from the cache.
func (c *Cache) Hit(question dns.Question, dnssec, tcp bool, msgid uint16, keepStale bool, returnStale bool) *dns.Msg {
	key := Key(question, dnssec, tcp)
	m1, exp, staleExp, hit := c.Search(key)
	valid := time.Since(exp) < 0
	if hit {
		// Cache hit! \o/
		if valid || returnStale{
			m1.Id = msgid
			m1.Compress = true
			// Even if something ended up with the TC bit *in* the cache, set it to off
			m1.Truncated = false
			if valid {
				c.m[key].hits++
			} else {
				c.m[key].staleHits++
			}
			// Remove if stale expired
			if time.Since(staleExp) > 0 {
				c.Remove(key)
			}
			return m1
		}
		// Expired! /o\
		if !keepStale {
			c.Remove(key)
		}
	}
	return nil
}
