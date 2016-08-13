# benchttp

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

* `-c max-concurrent-requests`
* `-d duration`
* `-n max-number-of-requests`
* `-v` print errors and their frequencies
* `-i` do HEAD requests instead of GET (default true)
* `-u` supply basic authentication, e.g. admin:pass

## License

MIT
