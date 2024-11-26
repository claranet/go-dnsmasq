package control

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/claranet/go-dnsmasq/server"
	"github.com/claranet/go-dnsmasq/cache"
	log "github.com/sirupsen/logrus"
)

const defaultControlAddr = "127.0.0.1"

type control struct {
	port int
	cch *cache.Cache
}

type PingResponse struct {
	Ping string `json:"ping"`
}

type StatsResponse struct {
	StatsForwardCount     int64 `json:"forwardCount"`
	StatsStubForwardCount int64 `json:"stubForwardCount"`
	StatsLookupCount      int64 `json:"lookupCount"`
	StatsRequestCount     int64 `json:"requestCount"`
	StatsDnssecOkCount    int64 `json:"dnssecOkCount"`
	StatsNameErrorCount   int64 `json:"nameErrorCount"`
	StatsNoDataCount      int64 `json:"noDataCount"`
	StatsDnssecCacheMiss int64 `json:"dnssecCacheMiss"`
	StatsCacheMiss      int64 `json:"cacheMiss"`
	StatsCacheHit       int64 `json:"cacheHit"`
	StatsRequestFail int64 `json:"requestFail"`
	StatsStaleCacheHit  int64 `json:"staleCacheHit"`
	StatsCacheSize		int `json:"cacheSize"`
	StatsCacheCapacity		int `json:"cacheCapacity"`
	StatsCacheHitRate		float64 `json:"cacheHitRate"`
}

func writeResponse(w http.ResponseWriter, rsp []byte, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(rsp)
	if err != nil {
		log.Error("Write response fail")
	}
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	response := PingResponse{Ping: "pong"}
	jsonResponse, err := json.Marshal(response)
	writeResponse(w, jsonResponse, err)
}

func (c *control) statsHandler(w http.ResponseWriter, r *http.Request) {

	hitRate := 0.0
	hitAndMiss := server.StatsCacheHit.Count() + server.StatsCacheMiss.Count()
	if hitAndMiss > 0 {
		hitRate = float64(server.StatsCacheHit.Count()) / float64(hitAndMiss)
	}

	response := StatsResponse{
		StatsForwardCount: server.StatsForwardCount.Count(),
		StatsStubForwardCount: server.StatsStubForwardCount.Count(),
		StatsLookupCount: server.StatsLookupCount.Count(),
		StatsRequestCount: server.StatsRequestCount.Count(),
		StatsDnssecOkCount: server.StatsDnssecOkCount.Count(),
		StatsNameErrorCount: server.StatsNameErrorCount.Count(),
		StatsNoDataCount: server.StatsNoDataCount.Count(),
		StatsDnssecCacheMiss: server.StatsDnssecCacheMiss.Count(),
		StatsCacheMiss: server.StatsCacheMiss.Count(),
		StatsCacheHit: server.StatsCacheHit.Count(),
		StatsRequestFail: server.StatsRequestFail.Count(),
		StatsStaleCacheHit: server.StatsStaleCacheHit.Count(),
		StatsCacheSize:		c.cch.CacheSize(),
		StatsCacheCapacity:	c.cch.Capacity(),
		StatsCacheHitRate: hitRate,
	}

	jsonResponse, err := json.Marshal(response)
	writeResponse(w, jsonResponse, err)
}

func (c *control) dumpHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(c.cch.DumpCache()))
	if err != nil {
		log.Error("Write response fail")
	}
}

func getAddr(port int) string {
	return fmt.Sprintf("%s:%d", defaultControlAddr, port)
}

func New(port int, cch *cache.Cache) *control {
	return &control{
		port: port,
		cch: cch,
	}
}

func (c *control) Run() error {
	addr := getAddr(c.port)
	http.HandleFunc("/ping", pingHandler)
	http.HandleFunc("/stats", c.statsHandler)
	http.HandleFunc("/dump", c.dumpHandler)

	log.Infof("Control server listening on http://%s", addr)
	return http.ListenAndServe(addr, nil)
}
