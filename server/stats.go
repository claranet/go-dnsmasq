// Copyright (c) 2014 The SkyDNS Authors. All rights reserved.
// Use of this source code is governed by The MIT License (MIT) that can be
// found in the LICENSE file.

package server

// Counter is the metric interface used by this package
type Counter interface {
	Inc(i int64)
	Count() int64
}

type nopCounter struct{}

func (nopCounter) Inc(_ int64)  {}
func (nopCounter) Count() int64 { return 0 }

var (
	StatsForwardCount     Counter = nopCounter{}
	StatsStubForwardCount Counter = nopCounter{}
	StatsLookupCount      Counter = nopCounter{}
	StatsRequestCount     Counter = nopCounter{}
	StatsDnssecOkCount    Counter = nopCounter{}
	StatsNameErrorCount   Counter = nopCounter{}
	StatsRefusedCount     Counter = nopCounter{}
	StatsNoDataCount      Counter = nopCounter{}
	StatsDnssecCacheMiss  Counter = nopCounter{}
	StatsCacheMiss        Counter = nopCounter{}
	StatsCacheHit         Counter = nopCounter{}
	StatsStaleCacheHit    Counter = nopCounter{}
	StatsRequestFail      Counter = nopCounter{}
)
