# benchttp

Benchttp implements the most commonly used features of `ab`.
Benchmarks could be limited by either the number of requests (`-n`) or total duration (`-d`).

**Note**: This project is under active development and there might be bugs.
File an issue if you've found one.
Thank you!

## Install

    go get -u github.com/siadat/benchttp

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

The output for

    benchttp -n 10000 -c 100 localhost:8080

is

     Duration: 2.238s
     Requests: 10000 (4468.7/s)
       Errors: 0 (%0)
    Responses: 10000 (4468.7/s)
        [200]: 10000 (%100.0)

## Options

* `-d duration`, e.g. `-d 10s`
* `-n number-of-requests`
* `-c max-concurrent-requests`
* `-u admin:pass` supply basic authentication
* `-H "key: value"` custom header
* `-i` do HEAD requests instead of GET
* `-v` print errors and their frequencies

## License

MIT
