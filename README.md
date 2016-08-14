# benchttp

Benchttp implements the most commonly used features of `ab`.

## Install

    go get github.com/siadat/benchttp

## Usage

Benchmark 1000 requests

    benchttp -n 1000 http://localhost:8080

Benchmark 1000 requests with maximum 10 concurrently running requests

    benchttp -n 1000 -c 10 http://localhost:8080

Benchmark server for 1s

    benchttp -d 1s http://localhost:8080

Benchmark server for 1s with max 10 concurrently running requests

    benchttp -d 1s -c 10 http://localhost:8080

## Options

* `-d duration`, e.g. `-d 10s`
* `-n max-number-of-requests`
* `-c max-concurrent-requests`
* `-u admin:pass` supply basic authentication
* `-H "key: value"` custom header
* `-i` do HEAD requests instead of GET
* `-v` print errors and their frequencies

## License

MIT
