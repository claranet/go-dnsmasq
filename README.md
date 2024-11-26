# go-dnsmasq
[![Latest Version](https://img.shields.io/github/release/claranet/go-dnsmasq.svg?maxAge=60)][release]
[![Github All Releases](https://img.shields.io/github/downloads/claranet/go-dnsmasq/total.svg?maxAge=86400)]()
[![License](https://img.shields.io/github/license/claranet/go-dnsmasq.svg?maxAge=86400)]()

[release]: https://github.com/claranet/go-dnsmasq/releases

go-dnsmasq is a lightweight DNS caching server/forwarder with minimal filesystem and runtime overhead.

### Application examples:

- Caching DNS server/forwarder in a local network
- Container/Host DNS cache
- DNS proxy providing DNS `search` capabilities to `musl-libc` based clients, particularly Alpine Linux

### Features

* Automatically set upstream `nameservers` and `search` domains from resolv.conf
* Insert itself into the host's /etc/resolv.conf on start
* Serve static A/AAAA records from a hosts file
* Provide DNS response caching
* Replicate the `search` domain treatment not supported by `musl-libc` based Linux distributions
* Supports virtually unlimited number of `search` paths and `nameservers` ([related Kubernetes article](https://github.com/kubernetes/kubernetes/tree/master/cluster/addons/dns#known-issues))
* Configure stubzones (different nameserver for specific domains)
* Round-robin of DNS records
* Send server metrics to Graphite and StatHat
* Configuration through both command line flags and environment variables
* Retain stale records. If TTL expires and all upstream servers are not available, then the state record will be served, if it not older than StaleTTL seconds
* Only cache non negative. Allows only positive records to be stored in cache. A positive record is a record whose client returned with `state: NOERROR`
* Use TTL from response. Allows to extract the lowest TTL from anwsers in response and use that value, for that record

### Resolve logic

DNS queries are resolved in the style of the GNU libc resolver:
* The first nameserver (as listed in resolv.conf or configured by `--nameservers`) is always queried first, additional servers are considered fallbacks
* Multiple `search` domains are tried in the order they are configured.
* Single-label queries (e.g.: "redis-service") are always qualified with the `search` domains
* Multi-label queries (ndots >= 1) are first tried as absolute names before qualifying them with the `search` domains
* Serve stale records if upstream is not available, if `rstale-ttl` is set to above 0

### Command-line options / environment variables

| Flag                           | Description                                                                   | Default       | Environment vars     |
| ------------------------------ | ----------------------------------------------------------------------------- | ------------- | -------------------- |
| --listen, -l                   | Address to listen on  `host[:port]`                                           | 127.0.0.1:53  | $DNSMASQ_LISTEN      |
| --default-resolver, -d         | Update resolv.conf to make go-dnsmasq the host's nameserver                   | False         | $DNSMASQ_DEFAULT     |
| --nameservers, -n              | Comma delimited list of nameservers `host[:port]`. IPv6 literal address must be enclosed in brackets. (supersedes etc/resolv.conf) | -  | $DNSMASQ_SERVERS     |
| --stubzones, -z                | Use different nameservers for given domains. Can be passed multiple times. `domain[,domain]/host[:port][,host[:port]]`   | -  |$DNSMASQ_STUB        |
| --hostsfile, -f                | Path to a hosts file (e.g. ‘/etc/hosts‘)                                      | -             | $DNSMASQ_HOSTSFILE   |
| --hostsfile-poll, -p           | How frequently to poll hosts file for changes (seconds, ‘0‘ to disable)       | 0             | $DNSMASQ_POLL        |
| --search-domains, -s           | Comma delimited list of search domains `domain[,domain]` (supersedes /etc/resolv.conf) | -             | $DNSMASQ_SEARCH_DOMAINS      |
| --enable-search, -search       | Qualify names with search domains to resolve queries                          | False         | $DNSMASQ_ENABLE_SEARCH      |
| --rcache, -r                   | Capacity of the response cache (‘0‘ disables caching)                         | 0             | $DNSMASQ_RCACHE      |
| --rcache-ttl                   | TTL for entries in the response cache                                         | 60            | $DNSMASQ_RCACHE_TTL  |
| --rcache-ttl-from-resp         | Use TTL from response. If multiple anwsers, lowest value is used; `rcache-tll` and `rcache-tll-max` are used as min and max values                                         | False            | $GO_DNSMASQ_RSTALE_TTL_FROM_RESP  |
| --rcache-ttl-max               | Used with `rcache-ttl-from-resp`. If ttl from response is higher than max, max is used                         | 3600         | $GO_DNSMASQ_RCACHE_TTL_MAX       |
| --rstale-ttl                   | Stale retention in `seconds` for response cache entries. Stale retention keeps cache after regular TTL if name server are not reachable                        | 0         | $GO_DNSMASQ_RSTALE_TTL       |
| --rcache-non-negative          | Cache only non negative responses                                             | False         | $GO_DNSMASQ_CACHE_NON_NEGATIVE       |
| --no-rec                       | Disable forwarding of queries to upstream nameservers                         | False         | $DNSMASQ_NOREC       |
| --fwd-ndots                    | Number of dots a name must have before the query is forwarded                 | 0 | $DNSMASQ_FWD_NDOTS   |
| --ndots                        | Number of dots a name must have before making an initial absolute query (supersedes /etc/resolv.conf) | 1  | $DNSMASQ_NDOTS |
| --round-robin                  | Enable round robin of A/AAAA records                                          | False         | $DNSMASQ_RR          |
| --systemd                      | Bind to socket(s) activated by Systemd (ignores --listen)                     | False         | $DNSMASQ_SYSTEMD     |
| --verbose                      | Enable verbose logging                                                        | False         | $DNSMASQ_VERBOSE     |
| --syslog                       | Enable syslog logging                                                         | False         | $DNSMASQ_SYSLOG      |
| --multithreading               | Enable multithreading (experimental)                                          | False         |                      |
| --help, -h                     | Show help                                                                     |               |                      |
| --version, -v                  | Print the version                                                             |               |                      |

#### Enable Graphite/StatHat metrics

EnvVar: **GRAPHITE_SERVER**
Default: ` `
Set to the `host:port` of the Graphite server

EnvVar: **GRAPHITE_PREFIX**
Default: `go-dnsmasq`
Set a custom prefix for Graphite metrics

EnvVar: **STATHAT_USER**
Default: ` `
Set to your StatHat account email address

### Usage

#### Run from the command line

Download the binary for your OS from the [releases page](https://github.com/claranet/go-dnsmasq/releases/latest).

go-dnsmasq is available in two versions. The minimal version (`go-dnsmasq-min`) has a lower memory footprint but doesn't have caching, stats reporting and systemd support.

```sh
   sudo ./go-dnsmasq [options]
```

#### Get stats from local http

- `curl -s http://127.0.0.1:8053/ping`: Ping, Pong
- `curl -s http://127.0.0.1:8053/stats`: Get the current stats in JSON format. It is suitable to be requested continuously, as this operation should be cheap.
- `curl -s http://127.0.0.1:8053/dump`: Get the current cache table alongside some statistic such as hits, stale hits, expiration times and question type. It is **not** suitable to be requested continuously, as this operation should be **expensive**.

#### Serving A/AAAA records from a hosts file
The `--hostsfile` parameter expects a standard plain text [hosts file](https://en.wikipedia.org/wiki/Hosts_(file)) with the only difference being that a wildcard `*` in the left-most label of hostnames is allowed. Wildcard entries will match any subdomain that is not explicitly defined.
For example, given a hosts file with the following content:

```
192.168.0.1 db1.db.local
192.168.0.2 *.db.local
```

Queries for `db2.db.local` would be answered with an A record pointing to 192.168.0.2, while queries for `db1.db.local` would yield an A record pointing to 192.168.0.1.

### Acknowledgements

- Initial implementation by [janeczku](http://github.com/janeczku)
