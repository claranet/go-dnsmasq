// Copyright (c) 2014 The SkyDNS Authors. All rights reserved.
// Use of this source code is governed by The MIT License (MIT) that can be
// found in the LICENSE file.

// Package stats may be imported to record statistics about a DNS server.
// If the GRAPHITE_SERVER environment variable is set, the statistics can
// be periodically reported to that server.
package stats

import (
	"net"
	"os"
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/stathat"

	"github.com/claranet/go-dnsmasq/server"
	log "github.com/sirupsen/logrus"
)

var (
	graphiteServer = os.Getenv("GRAPHITE_SERVER")
	graphitePrefix = os.Getenv("GRAPHITE_PREFIX")
	stathatUser    = os.Getenv("STATHAT_USER")
)

var counters = map[string]*server.Counter{
	"go-dnsmasq-forward-requests": &server.StatsForwardCount,
	"go-dnsmasq-stub-forward-requests": &server.StatsStubForwardCount,
	"go-dnsmasq-dnssecok-requests": &server.StatsDnssecOkCount,
	"go-dnsmasq-dnssec-cache-miss": &server.StatsDnssecCacheMiss,
	"go-dnsmasq-internal-lookups": &server.StatsLookupCount,
	"go-dnsmasq-requests": &server.StatsRequestCount,
	"go-dnsmasq-nameerror-responses": &server.StatsNameErrorCount,
	"go-dnsmasq-refused": &server.StatsRefusedCount,
	"go-dnsmasq-nodata-responses": &server.StatsNoDataCount,
	"go-dnsmasq-cache-miss": &server.StatsCacheMiss,
	"go-dnsmasq-cache-hit": &server.StatsCacheHit,
	"go-dnsmasq-stale-cache-hit": &server.StatsStaleCacheHit,
	"go-dnsmasq-stale-request-fail": &server.StatsRequestFail,
}

func init() {
	if graphitePrefix == "" {
		graphitePrefix = "go-dnsmasq"
	}


	for k, v := range counters {
		*v = metrics.NewCounter()
		err := metrics.Register(k, *v)
		if err != nil {
			log.Errorf("Failed to register %s", k)
		}
		log.Debugf("Register counter %s", k)
	}
}

func Collect() {
	if graphiteServer != "" {
		addr, err := net.ResolveTCPAddr("tcp", graphiteServer)
		if err == nil {
			go metrics.Graphite(metrics.DefaultRegistry, 10e9, graphitePrefix, addr)
		}
	}

	if stathatUser != "" {
		go stathat.Stathat(metrics.DefaultRegistry, 10e9, stathatUser)
	}
}
