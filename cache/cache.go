// Copyright (c) 2014 The SkyDNS Authors. All rights reserved.
// Use of this source code is governed by The MIT License (MIT) that can be
// found in the LICENSE file.

package cache

// Cache that holds RRs and for DNSSEC an RRSIG.

// TODO(miek): there is a lot of copying going on to copy myself out of data
// races. This should be optimized.

import (
	"crypto/sha1"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
)

// Elem hold an answer and additional section that returned from the cache.
// The signature is put in answer, extra is empty there. This wastes some memory.
type elem struct {
	expiration      time.Time // time added + TTL, after this the elem is invalid
	msg             *dns.Msg
	staleExpiration time.Time
	hits            uint
	staleHits       uint
	ttlSeconds      uint32
}

// Cache is a cache that holds on the a number of RRs or DNS messages. The cache
// eviction is randomized.
type Cache struct {
	sync.RWMutex

	capacity      int
	m             map[string]*elem
	ttl           time.Duration
	staleTtl      time.Duration
	ttlFromResp   bool
	ttlMax        time.Duration
	ttlMinSeconds uint32
	ttlMaxSeconds uint32
}

var qTypeToName = map[uint16]string{
	1:  "A",
	28: "AAAA",
	5:  "CNAME",
	15: "MX",
	2:  "NS",
	16: "TXT",
	6:  "SOA",
}

func getRecordTypeName(qType uint16) string {
	val, ok := qTypeToName[qType]
	if !ok {
		val = strconv.FormatUint(uint64(qType), 10)
	}
	return val
}

func (c *Cache) Capacity() int {
	return c.capacity
}

func (c *Cache) CacheSize() int {
	return len(c.m)
}

func (c *Cache) DumpCache() string {
	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 1, 1, 1, ' ', 0)
	now := time.Now()
	fmt.Fprintln(w, "=== BEGIN CACHE DUMP ===")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Capacity: %d\n", c.capacity)
	fmt.Fprintf(w, "Current Size: %d\n", len(c.m))
	fmt.Fprintf(w, "Default: %v\n", c.ttl)
	fmt.Fprintf(w, "Stale TTL: %v\n", c.staleTtl)
	fmt.Fprintf(w, "Min TTL (s): %v\n", c.ttlMinSeconds)
	fmt.Fprintf(w, "Max TTL (s): %v\n", c.ttlMaxSeconds)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Dumped at: %v\n", now.Format(time.RFC3339))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "QType\tExpired\tStaleExpired\tTTL(s)\tExpire In\tStaleExpire In\tQuestion\tHits\tStaleHits")
	for _, v := range c.m {
		var sb strings.Builder
		qType := uint16(0)
		if len(v.msg.Question) == 1 {
			qType = v.msg.Question[0].Qtype
		}
		for i := range v.msg.Question {
			sb.WriteString(v.msg.Question[i].Name)
			sb.WriteString(",")
		}
		fmt.Fprintf(w, "%s\t%t\t%t\t%d\t%v\t%v\t%s\t%d\t%d\n", getRecordTypeName(qType), time.Since(v.expiration) > 0, time.Since(v.staleExpiration) > 0, v.ttlSeconds, v.expiration.Sub(now).Truncate(time.Second), v.staleExpiration.Sub(now).Truncate(time.Second), strings.Trim(sb.String(), ","), v.hits, v.staleHits)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "=== END CACHE DUMP ===")
	w.Flush()
	return sb.String()
}

// New returns a new cache with the capacity and the ttl and stale ttl specified.
func New(capacity, ttl int, staleTtl int, ttlFromResp bool, ttlMax int) *Cache {
	c := new(Cache)
	c.m = make(map[string]*elem)
	c.capacity = capacity
	c.ttl = time.Duration(ttl) * time.Second
	c.staleTtl = time.Duration(staleTtl) * time.Second
	c.ttlFromResp = ttlFromResp
	c.ttlMax = time.Duration(ttlMax) * time.Second
	c.ttlMinSeconds = uint32(ttl)
	c.ttlMaxSeconds = uint32(ttlMax)
	return c
}

func (c *Cache) Remove(s string) {
	c.Lock()
	delete(c.m, s)
	c.Unlock()
}

// Identical to EvictRandom() but can evict only stale stale records
// Created to keep compatibility with calls to EvictRandom()
func (c *Cache) evictRandomInternal(onlyStale bool) {
	clen := len(c.m)
	if clen < c.capacity {
		return
	}
	i := c.capacity - clen
	for k, _ := range c.m {

		if i == 0 {
			if onlyStale {
				stale := time.Since(c.m[k].staleExpiration) > 0
				if stale {
					delete(c.m, k)
					log.Debug("Evicted stale record")
				}
			} else {
				delete(c.m, k)
				log.Debug("Evicted record")
			}
		}
		i--
	}
}

// EvictRandom removes a random member a the cache.
// Must be called under a write lock.
func (c *Cache) EvictRandom() {
	c.evictRandomInternal(true)  // First evict only stale records
	c.evictRandomInternal(false) // Then the rest. Should only loop if capacity still bigger than set
}

// InsertMessage inserts a message in the Cache. We will cache it for ttl seconds, which
// should be a small (60...300) integer.
func (c *Cache) InsertMessage(s string, msg *dns.Msg) {
	if c.capacity <= 0 {
		return
	}

	c.Lock()
	renew := false
	_, ok := c.m[s]
	if ok {
		renew = time.Since(c.m[s].expiration) > 0 && time.Since(c.m[s].staleExpiration) < 0
	}
	if !ok || renew {
		exp := time.Now().UTC().Add(c.ttl)
		ttlSeconds := uint32(0) //c.ttl
		if c.ttlFromResp {
			lowestTll := getLowestTtl(msg, c.ttlMinSeconds, c.ttlMaxSeconds)
			log.Debugf("Found lowest ttl: %d\n", lowestTll)
			ttlD := time.Duration(lowestTll) * time.Second
			exp = time.Now().UTC().Add(ttlD)
			ttlSeconds = lowestTll
		}
		c.m[s] = &elem{exp, msg.Copy(), time.Now().UTC().Add(c.staleTtl), 0, 0, ttlSeconds}
		logMsg := fmt.Sprintf("Insert into cache: %v", msg.Answer)
		if renew {
			logMsg = fmt.Sprintf("Renew entry: %v", msg.Answer)
		}
		log.Debug(logMsg)

	}
	c.EvictRandom()
	c.Unlock()
}

// Search returns a dns.Msg, the expiration time and a boolean indicating if we found something
// in the cache.
func (c *Cache) Search(s string) (*dns.Msg, time.Time, time.Time, bool) {
	if c.capacity <= 0 {
		return nil, time.Time{}, time.Time{}, false
	}
	c.RLock()
	if e, ok := c.m[s]; ok {
		e1 := e.msg.Copy()
		c.RUnlock()
		return e1, e.expiration, e.staleExpiration, true
	}
	c.RUnlock()
	return nil, time.Time{}, time.Time{}, false
}

// Key creates a hash key from a question section. It creates a different key
// for requests with DNSSEC.
func Key(q dns.Question, dnssec, tcp bool) string {
	h := sha1.New()
	i := append([]byte(q.Name), packUint16(q.Qtype)...)
	if dnssec {
		i = append(i, byte(255))
	}
	if tcp {
		i = append(i, byte(254))
	}
	return string(h.Sum(i))
}

// Key uses the name, type and rdata, which is serialized and then hashed as the key for the lookup.
func KeyRRset(rrs []dns.RR) string {
	h := sha1.New()
	i := []byte(rrs[0].Header().Name)
	i = append(i, packUint16(rrs[0].Header().Rrtype)...)
	for _, r := range rrs {
		switch t := r.(type) { // we only do a few type, serialize these manually
		case *dns.SOA:
			// We only fiddle with the serial so store that.
			i = append(i, packUint32(t.Serial)...)
		case *dns.SRV:
			i = append(i, packUint16(t.Priority)...)
			i = append(i, packUint16(t.Weight)...)
			i = append(i, packUint16(t.Weight)...)
			i = append(i, []byte(t.Target)...)
		case *dns.A:
			i = append(i, []byte(t.A)...)
		case *dns.AAAA:
			i = append(i, []byte(t.AAAA)...)
		case *dns.NSEC3:
			i = append(i, []byte(t.NextDomain)...)
		case *dns.DNSKEY:
		case *dns.NS:
		case *dns.TXT:
		}
	}
	return string(h.Sum(i))
}

// return the lowest ttl from all the records in message
func getLowestTtl(r *dns.Msg, min uint32, max uint32) uint32 {
	var ttls []uint32
	for _, m := range r.Answer {
		ttls = append(ttls, m.Header().Ttl)
	}
	// deal with no anwser
	if len(ttls) == 0 {
		ttls = append(ttls, 0)
	}
	value := slices.Min(ttls)
	if value < min {
		value = min
	} else if value > max {
		value = max
	}
	return value
}

func packUint16(i uint16) []byte { return []byte{byte(i >> 8), byte(i)} }
func packUint32(i uint32) []byte { return []byte{byte(i >> 24), byte(i >> 16), byte(i >> 8), byte(i)} }
