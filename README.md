# Overview
Astera is a custom implementation of the Go module proxy. Instead of serving modules directly from the filesystem, Astera stores them in **SQLite3** and exposes them through the Go proxy HTTP protocol.

The origin of the Astera started when I moved to an area with poor internet connectivity, building Go projects became painful. Fetching dependencies would often fail after long waits. The obvious solution was to cache dependencies locally:

- Either share the `$GOPATH/pkg/mod` directory across builds (docker builds)
- Run an HTTP server serving from that shared cache

However, both approaches were clunky and incomplete — they didn’t fully solve the problem and tbh were quite boring ;).

This led me to the idea of building my own **Go proxy** that caches responses from `proxy.golang.org`. With it, every machine on my local network could fetch modules quickly, without hitting the internet.

These was my requirements for Astera
- Support Go proxy protocol (without sumdb). The `go mod tidy` and `go get -u ./...` had to work.
- Private repository support but limited to Git.
- Handle high load ;) by high load I mean 30-40 `go mod tidy` runs in parallels on dev machines
- Astera will be hosted on RaspberryPi 5 8GB Ram.

The initial implementation works but fail under high load because there were a lot of redundant requests to sqlite which created a massive IO pressure on the system. The whole Raspberr Pi 5 was unresponsive.

The solution was to use single flight and in memory cache with weak pointers. If multiple requests was asking for the same module only one was actually query the database, others waits for the first one to return with the data. In addition to that we also cache the response from the database for some time (we had the data in memory anyway) using weak pointers so after some time the data was gone. This leads to creation of [weakcache](https://github.com/tmwalaszek/weakcache) which is use by Astera.

Using [weakcache](https://github.com/tmwalaszek/weakcache) I can't kill my small Raspberry Pi 5 8GB Ram, my dev machines can't handle that many `go mod` operations :).

# Usage
It's very simple, we just need to compile it and run.

We can also tell astera to populate the database using golang module cache.

```
 go build cmd/astera.go
 ./astera -h
Usage of ./astera:
  -addr string
        listen address (default ":8080")
  -db string
        database file (default "astera.db")
  -import-local-cache
        import local cache
  -local-cache-dir string
        local cache directory (default "/Users/tmwl/go/pkg/mod/cache/download")
  -pprof
        enable pprof
```
