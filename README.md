# Benchttp

[![GoDoc](https://godoc.org/github.com/siadat/benchttp/benchttp?status.svg)](https://godoc.org/github.com/siadat/benchttp/benchttp)
[![Build Status](https://travis-ci.org/siadat/benchttp.svg?branch=master)](https://travis-ci.org/siadat/benchttp)


Benchttp implements the most commonly used features of ApacheBench.

Benchmarks are limited with either `-n number-of-requests` or `-d total-duration`.

## Install

    go get -u github.com/siadat/benchttp/cmd/benchmark

## Usage

Benchmark 1000 requests

    benchttp -n 1000 http://localhost:8080

Benchmark 1000 requests with maximum 10 concurrently running requests

    benchttp -n 1000 -c 10 http://localhost:8080

Benchmark server for 1s

    benchttp -d 1s http://localhost:8080

Benchmark server for 1s with max 10 concurrently running requests

    benchttp -d 1s -c 10 http://localhost:8080

## Output

     Duration: 2.238s
     Requests: 10000 (4468.7/s)
    Responses: 10000 (4468.7/s)
        [200]: 10000

## Options

* `-d duration`, e.g., `-d 10s`
* `-n number-of-requests`, e.g., `-n 1000`
* `-c max-concurrent-requests`, e.g. `-c 100`
* `-u admin:pass` supply basic authentication
* `-H "key: value"` custom header
* `-i` do HEAD requests instead of GET

## Contribute

Issues and PRs are welcome.

## Thanks

Thanks @Deleplace for testing and reviewing the code.

## License

MIT
